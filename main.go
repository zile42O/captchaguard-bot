package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	BOT_DEBUG         = true
	BOT_UPTIME        time.Time
	CAPTCHA_IMAGE_API = "https://api.zile42o.dev/captcha/captcha.php?code=%d"
	BOT_API_TOKEN     = ""
)

var (
	ETH_DONATE      = ""
	BITCOIN_DONATE  = ""
	LITECOIN_DONATE = ""
	MONERO_DONATE   = ""
)

var (
	CaptchaChatID          = make(map[int64]int64)
	CaptchaTime            = make(map[int64]int64)
	CaptchaCode            = make(map[int64]int)
	CaptchaMessageID       = make(map[int64]int)
	VerifyingCaptchaStatus = make(map[int64]bool)
	inviteLinkRegex        = regexp.MustCompile(`(https?:\/\/)?(www\.)?(t\.(me)|telegram\.)\/.+[a-z]`)
)

func checkVerifyMember(bot *tgbotapi.BotAPI, user int64, chat int64) {
	if VerifyingCaptchaStatus[user] {
		timein := time.Now().Add(time.Minute * 1)
		banChatMember := tgbotapi.BanChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{
				ChatID: chat,
				UserID: user,
			},
			UntilDate: timein.Unix(),
		}
		_, _ = bot.Request(banChatMember)

		msgToDelete := tgbotapi.DeleteMessageConfig{
			ChatID:    CaptchaChatID[user],
			MessageID: CaptchaMessageID[user],
		}
		_, _ = bot.Request(msgToDelete)
		VerifyingCaptchaStatus[user] = false
		color.Red("[Debug]: System banned [%d] because didn't completed captcha", user)
	}
}

func main() {
	bot, err := tgbotapi.NewBotAPI(BOT_API_TOKEN)
	if err != nil {
		color.Red("[Error]: %s", err)
	}
	bot.Debug = BOT_DEBUG
	color.Green("[Success]: Authorized on account %s", bot.Self.UserName)
	BOT_UPTIME = time.Now()
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "chat_member", "callback_query"}
	updates := bot.GetUpdatesChan(u)
	time.Sleep(time.Millisecond * 10000)
	updates.Clear()
	for update := range updates {
		if update.ChatMember != nil {
			if update.ChatMember.OldChatMember.Status == "left" && update.ChatMember.NewChatMember.Status != "left" && !update.ChatMember.NewChatMember.User.IsBot {
				color.Blue("Member %s joined to group %s", update.ChatMember.NewChatMember.User.UserName, update.ChatMember.Chat.Title)
				if !VerifyingCaptchaStatus[update.ChatMember.NewChatMember.User.ID] {
					VerifyingCaptchaStatus[update.ChatMember.NewChatMember.User.ID] = true
					CaptchaChatID[update.ChatMember.NewChatMember.User.ID] = update.ChatMember.Chat.ID
					t := time.Now().Unix()
					CaptchaTime[update.ChatMember.NewChatMember.User.ID] = t + 120
					rand.Seed(time.Now().UnixNano())
					answercode := rand.Intn(9999)
					CaptchaCode[update.ChatMember.NewChatMember.User.ID] = answercode
					strcode := fmt.Sprintf(CAPTCHA_IMAGE_API, answercode)
					msg := tgbotapi.NewPhoto(update.ChatMember.Chat.ID, tgbotapi.FileURL(strcode))
					if len(update.ChatMember.NewChatMember.User.UserName) > 0 {
						msg.Caption = fmt.Sprintf("Hello *%s* (@%s), welcome to %s!\nPlease verify yourself by typing bellow the text from captcha image.\nIf you don't answer of this verification, after 120 seconds you will be kicked.\n\nThis group is `protected` by *Captcha Guard* Bot.", update.ChatMember.NewChatMember.User.FirstName, update.ChatMember.NewChatMember.User.UserName, update.ChatMember.Chat.Title)
					} else {
						msg.Caption = fmt.Sprintf("Hello *%s*, welcome to %s!\nPlease verify yourself by typing bellow the text from captcha image.\nIf you don't answer of this verification, after 120 seconds you will be kicked.\n\nThis group is `protected` by *Captcha Guard* Bot.", update.ChatMember.NewChatMember.User.FirstName, update.ChatMember.Chat.Title)
					}
					msg.ParseMode = "markdown"
					mss, _ := bot.Send(msg)
					CaptchaMessageID[update.ChatMember.NewChatMember.User.ID] = mss.MessageID
					continue
				}
			} else if update.ChatMember.NewChatMember.Status != "member" || !update.ChatMember.NewChatMember.IsMember && update.ChatMember.OldChatMember.IsMember {
				if VerifyingCaptchaStatus[update.ChatMember.OldChatMember.User.ID] {
					msgToDelete := tgbotapi.DeleteMessageConfig{
						ChatID:    CaptchaChatID[update.ChatMember.OldChatMember.User.ID],
						MessageID: CaptchaMessageID[update.ChatMember.OldChatMember.User.ID],
					}
					_, _ = bot.Request(msgToDelete)
					color.Yellow("[Debug]: Deleted captcha message in chat id [%d] of user id [%d] because he left verification", CaptchaChatID[update.ChatMember.OldChatMember.User.ID], CaptchaMessageID[update.ChatMember.OldChatMember.User.ID])
				}
				color.Blue("Member %s left group %s", update.ChatMember.OldChatMember.User.UserName, update.ChatMember.Chat.Title)
			}
		} else if update.Message != nil {
			color.Yellow("[Debug]: Message update in %s", update.Message.Chat.Title)
			color.Yellow("[Debug]: Checking verify captchas...")
			for i, v := range VerifyingCaptchaStatus {
				if v {
					userID := i
					chatID := CaptchaChatID[userID]
					if chatID != -1 {
						captchaTime := CaptchaTime[userID]
						t := time.Now().Unix()
						if t > captchaTime {
							color.Yellow("[Debug]: Checking captcha for user [%d]", userID)
							checkVerifyMember(bot, userID, chatID)
						}
					}
				}
			}
			color.Yellow("[Debug]: Checking verify captchas done...")
			if len(update.Message.Text) < 1 {
				if update.Message.Contact == nil && update.Message.VideoNote == nil && update.Message.Document == nil && update.Message.Animation == nil && update.Message.Voice == nil && update.Message.Sticker == nil && update.Message.Photo == nil && update.Message.Video == nil {
					color.Yellow("[Debug]: Removed system message in [%s]", update.Message.Chat.Title)
					msgToDelete := tgbotapi.DeleteMessageConfig{
						ChatID:    update.Message.Chat.ID,
						MessageID: update.Message.MessageID,
					}
					_, _ = bot.Request(msgToDelete)
				}
			}
			if VerifyingCaptchaStatus[update.Message.From.ID] {
				convert_int, _ := strconv.Atoi(update.Message.Text)
				if convert_int == CaptchaCode[update.Message.From.ID] {
					msgToDelete := tgbotapi.DeleteMessageConfig{
						ChatID:    CaptchaChatID[update.Message.From.ID],
						MessageID: CaptchaMessageID[update.Message.From.ID],
					}
					_, _ = bot.Request(msgToDelete)
					VerifyingCaptchaStatus[update.Message.From.ID] = false
					CaptchaChatID[update.Message.From.ID] = -1
					color.Yellow("[Debug]: User [%d] finished captcha verify", update.Message.From.ID)
				}
				msgToDelete := tgbotapi.DeleteMessageConfig{
					ChatID:    update.Message.Chat.ID,
					MessageID: update.Message.MessageID,
				}
				_, _ = bot.Request(msgToDelete)
				continue
			}
			submatch := inviteLinkRegex.FindStringSubmatch(update.Message.Text)
			if len(submatch) != 0 {
				if !checkAdmin(bot, update.Message.Chat, update.Message.From) {
					msgToDelete := tgbotapi.DeleteMessageConfig{
						ChatID:    update.Message.Chat.ID,
						MessageID: update.Message.MessageID,
					}
					_, err = bot.Request(msgToDelete)
				}
			}
			f, err := os.Open("censured.txt")
			if err == nil {
				defer f.Close()
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					lineSplit := strings.SplitN(scanner.Text(), "|", -1)
					id64, _ := strconv.ParseInt(lineSplit[0], 10, 64)
					if id64 == update.Message.Chat.ID {
						if strings.Contains(update.Message.Text, lineSplit[1]) {
							if !checkAdmin(bot, update.Message.Chat, update.Message.From) {
								msgToDelete := tgbotapi.DeleteMessageConfig{
									ChatID:    update.Message.Chat.ID,
									MessageID: update.Message.MessageID,
								}
								_, err = bot.Request(msgToDelete)
							}
						}
					}
				}
			}
			if !update.Message.IsCommand() {
				continue
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			_, _ = bot.Request(tgbotapi.NewChatAction(update.Message.Chat.ID, tgbotapi.ChatTyping))
			switch update.Message.Command() {
			case "start":
				msg.Text = "Hi! To start use this bot check /help for commands. To check about bot info use /about"
			case "about":
				msg.Text = "*Captcha Guard Bot*\nThis bot is made to protect your *group/supergroup*."
			case "help":
				msg.Text = "*Command list*\n\n/version\n/uptime\n/hi\n/purge\n/botfather\n/support\n/ban\n/mute\n/censure\n/censurelist"
			case "version":
				msg.Text = "*v1.1 (Stable)*"
			case "uptime":
				msg.Text = fmt.Sprint("*Uptime:* ", botUptime())
			case "donate":
				msg.Text = "You can support this bot via crypto payment!\n\n*Eth Address:* `" + ETH_DONATE + "`\n*Bitcoin Address:* `" + BITCOIN_DONATE + "`\n*Litecoin Address:* `" + LITECOIN_DONATE + "`\n*Monero Address:* `" + MONERO_DONATE + "`"
			case "censurelist":
				if update.Message.Chat.Type == "supergroup" || update.Message.Chat.Type == "group" {
					if checkAdmin(bot, update.Message.Chat, update.Message.From) {
						f, err := os.Open("censured.txt")
						if err == nil {
							defer f.Close()
							scanner := bufio.NewScanner(f)
							total_words := 0
							words_list := ""
							for scanner.Scan() {
								lineSplit := strings.SplitN(scanner.Text(), "|", -1)
								id64, _ := strconv.ParseInt(lineSplit[0], 10, 64)
								if id64 == update.Message.Chat.ID {
									words_list += " *" + lineSplit[1] + "* "
									total_words++
								}
							}
							if total_words < 1 {
								msg.Text = "❌ Sorry, but not found censured words for this group, try add it /censure!"
							} else {
								msg.Text = words_list + "\nTotal *" + strconv.FormatInt(int64(total_words), 10) + "* words in this group!"
							}
						}
					} else {
						msg.Text = "❌ You are not an Administrator!"
					}
				} else {
					msg.Text = "❌ This chat is not a group/supergroup!"
				}
			case "censure":
				if update.Message.Chat.Type == "supergroup" || update.Message.Chat.Type == "group" {
					if checkAdmin(bot, update.Message.Chat, update.Message.From) {
						arg := strings.Fields(update.Message.Text)
						if len(arg) < 2 {
							msg.Text = "❌ Missing parameter: word!"
						} else {
							f, err := os.OpenFile("censured.txt",
								os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
							if err != nil {
								log.Println(err)
							}
							defer f.Close()
							string_ID_chat := strconv.FormatInt(int64(update.Message.Chat.ID), 10)
							result := string_ID_chat + "|" + arg[1] + "\n"
							if _, err := f.WriteString(result); err != nil {
								log.Println(err)
							}
							msg.Text = "✔️ Word *" + arg[1] + "* successfully added into censure filter."
						}
					} else {
						msg.Text = "❌ You are not an Administrator!"
					}
				} else {
					msg.Text = "❌ This chat is not a group/supergroup!"
				}
			case "hi":
				msgToDelete := tgbotapi.DeleteMessageConfig{
					ChatID:    update.Message.Chat.ID,
					MessageID: update.Message.MessageID,
				}
				_, err = bot.Request(msgToDelete)
				msg.Text = "👋🏽 Hi :)"
			case "mute":
				if update.Message.Chat.Type == "supergroup" || update.Message.Chat.Type == "group" {
					if checkAdmin(bot, update.Message.Chat, update.Message.From) {
						if update.Message.ReplyToMessage == nil {
							msg.Text = "❌ You must reply to target last message!"
						} else {
							if checkAdmin(bot, update.Message.Chat, update.Message.ReplyToMessage.From) {
								msg.Text = "❌ The target is Administrator, you can't mute him!"
							} else {
								arg := strings.Fields(update.Message.Text)
								if len(arg) < 1 {
									msg.Text = "❌ Missing parameter type!"
								} else {
									if string(arg[1]) == "?" {
										msg.Text = "/mute time type interval\n*Example:* /mute d 5\n*Types:* h (hours) m (minutes)"
									} else {
										switch string(arg[1]) {
										case "m":
										case "h":
										default:
											msg.Text = "❌ Invalid type, check /mute *?*"
										}
										if msg.Text != "❌ Invalid type, check /mute *?*" {
											if len(arg) < 2 {
												msg.Text = "❌ Missing parameter interval!"
											} else {
												number, _ := strconv.Atoi(string(arg[2]))
												if number < 1 {
													msg.Text = "❌ Number can't be less then 1"
												} else {
													inverval := time.Duration(number)
													timein := time.Now().Add(time.Minute * 1200)
													switch string(arg[1]) {
													case "m":
														timein = time.Now().Add(time.Minute * inverval)
													case "d":
														timein = time.Now().Add(time.Hour * inverval * 24)
													case "h":
														timein = time.Now().Add(time.Hour * inverval)
													}
													if update.Message.ReplyToMessage.From.IsBot {
														msg.Text = "❌ User is bot!"
													} else {
														restrictChatMember := tgbotapi.RestrictChatMemberConfig{
															ChatMemberConfig: tgbotapi.ChatMemberConfig{
																ChatID: update.Message.Chat.ID,
																UserID: update.Message.ReplyToMessage.From.ID,
															},
															UntilDate: timein.Unix(),
															Permissions: &tgbotapi.ChatPermissions{
																CanSendMessages:      false,
																CanSendMediaMessages: false,
															},
														}
														_, _ = bot.Request(restrictChatMember)
														msg.Text = fmt.Sprintf("✔️ User muted successfully, mute expire: *%s*!", timein.Format("2006-01-02 3:4:5 pm"))
													}
												}
											}
										}
									}
								}
							}
						}
					} else {
						msg.Text = "❌ You are not Administrator!"
					}
				} else {
					msg.Text = "❌ This chat is not a group/supergroup!"
				}
			case "ban":
				if update.Message.Chat.Type == "supergroup" || update.Message.Chat.Type == "group" {
					if checkAdmin(bot, update.Message.Chat, update.Message.From) {
						if update.Message.ReplyToMessage == nil {
							msg.Text = "❌ You must reply to target last message!"
						} else {
							if checkAdmin(bot, update.Message.Chat, update.Message.ReplyToMessage.From) {
								msg.Text = "❌ The target is Administrator, you can't ban him!"
							} else {
								arg := strings.Fields(update.Message.Text)
								if len(arg) < 1 {
									msg.Text = "❌ Missing parameter type!"
								} else {
									if string(arg[1]) == "?" {
										msg.Text = "/ban time type interval\n*Example:* /ban d 5\n*Types:* d (days) h (hours) m (minutes)"
									} else {
										switch string(arg[1]) {
										case "m":
										case "d":
										case "h":
										default:
											msg.Text = "❌ Invalid type, check /ban *?*"
										}
										if msg.Text != "❌ Invalid type, check /ban *?*" {
											if len(arg) < 2 {
												msg.Text = "❌ Missing parameter interval!"
											} else {
												number, _ := strconv.Atoi(string(arg[2]))
												if number < 1 {
													msg.Text = "❌ Number can't be less then 1"
												} else {
													inverval := time.Duration(number)
													timein := time.Now().Add(time.Minute * 1200)
													switch string(arg[1]) {
													case "m":
														timein = time.Now().Add(time.Minute * inverval)
													case "d":
														timein = time.Now().Add(time.Hour * inverval * 24)
													case "h":
														timein = time.Now().Add(time.Hour * inverval)
													}
													if update.Message.ReplyToMessage.From.IsBot {
														msg.Text = "❌ User is bot!"
													} else {
														banChatMember := tgbotapi.BanChatMemberConfig{
															ChatMemberConfig: tgbotapi.ChatMemberConfig{
																ChatID: update.Message.Chat.ID,
																UserID: update.Message.ReplyToMessage.From.ID,
															},
															UntilDate: timein.Unix(),
														}
														_, _ = bot.Request(banChatMember)
														msg.Text = fmt.Sprintf("✔️ User banned successfully, ban expire: *%s*!", timein.Format("2006-01-02 3:4:5 pm"))
													}
												}
											}
										}
									}
								}
							}
						}
					} else {
						msg.Text = "❌ You are not Administrator!"
					}
				} else {
					msg.Text = "❌ This chat is not a group/supergroup!"
				}
			case "purge":
				if update.Message.Chat.Type == "supergroup" || update.Message.Chat.Type == "group" {
					if checkAdmin(bot, update.Message.Chat, update.Message.From) {
						if update.Message.ReplyToMessage == nil {
							msg.Text = "❌ You must reply to message from where you want to start delete!"
						} else {
							msgToDelete := tgbotapi.DeleteMessageConfig{
								ChatID:    update.Message.Chat.ID,
								MessageID: update.Message.MessageID,
							}
							_, err = bot.Request(msgToDelete)
							chatID := update.Message.Chat.ID
							startID := update.Message.ReplyToMessage.MessageID
							arg := update.Message.CommandArguments()
							number, err := strconv.Atoi(arg)
							if err != nil {
								msg.Text = "❌ Can't delete message, input the number of messages! 1 - 100"
							}
							if number < 1 || number > 100 {
								msg.Text = "❌ Can't delete message, input the number of messages! 1 - 100"
							}
							if msg.Text != "❌ Can't delete message, input the number of messages! 1 - 100" {
								endID := update.Message.ReplyToMessage.MessageID - number
								var deleted = 0
								for endID <= startID {
									msgToDelete := tgbotapi.DeleteMessageConfig{
										ChatID:    chatID,
										MessageID: startID,
									}
									_, err = bot.Request(msgToDelete)
									startID--
									if err == nil {
										deleted++
									}
								}
								if deleted > 0 {
									msg.Text = fmt.Sprintf("✔️ Successfully deleted %d messages", deleted)
								} else {
									msg.Text = "❌ No deleted messages!"
								}
							}
						}
					} else {
						msg.Text = "❌ You are not Administrator!"
					}
				} else {
					msg.Text = "❌ This chat is not a group/supergroup!"
				}
			case "botfather":
				msg.Text = fmt.Sprintf("Bot made by *Zile42O*\n(@zile42O) (github.com/zile42O)")
			}
			msg.ParseMode = "markdown"
			if _, err := bot.Send(msg); err != nil {
				color.Red("[Error]: %s", err)
			}
		}
	}
}

func checkAdmin(bot *tgbotapi.BotAPI, chat *tgbotapi.Chat, user *tgbotapi.User) bool {
	var chatconfig = chat.ChatConfig()
	member, err := bot.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: chatconfig.ChatID,
			UserID: user.ID,
		},
	},
	)
	if err != nil {
		log.Fatal(err)
	} else if member.IsAdministrator() {
		return true
	}
	if member.IsCreator() {
		return true
	}
	return false
}

func botUptime() time.Duration {
	return time.Since(BOT_UPTIME)
}
