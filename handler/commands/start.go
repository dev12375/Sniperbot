package commands

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/entity"
	"github.com/hellodex/tradingbot/handler/callback"
	"github.com/hellodex/tradingbot/logger"
	"github.com/hellodex/tradingbot/queue"
	"github.com/hellodex/tradingbot/session"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/template"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

func UserStartReflashSet(userID int64, messageID int) {
	session.GetSessionManager().Set(userID, session.UserStartMessaageIDkey, messageID)
}

func UserStartReflashGet(userID int64) int {
	v, has := session.GetSessionManager().Get(userID, session.UserStartMessaageIDkey)
	if has {
		return v.(int)
	}
	return 0
}

func StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	// delete cache key make it hard update
	// store.Delete(chatId, api.UserProfilePrefix)
	// delete redis profile cache
	store.RedisDeleteUserProfile(chatId)

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	text := ""

	badUserDWInfo := true
	func() {
		dW, _, _ := callback.UserDefaultWalletInfo(userInfo)
		tokens, err := api.GetTokensByWalletAddress(dW.Wallet, dW.ChainCode, userInfo)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}

		for _, t := range tokens.Data {
			if util.IsNativeCoion(t.Address) {
				nativeCoinBalance := util.FormatNumber(util.ShiftLeftStr(t.Amount, t.Decimals))
				usdTotalAmount := util.FormatNumber(t.TotalAmount)

				botUserName := store.GetEnv(store.BOT_USERNAME)
				t, err := template.RanderStart(
					t.Symbol,
					dW.Wallet,
					nativeCoinBalance,
					usdTotalAmount,
					userInfo.Data.InviteCode,
					botUserName,
				)
				if err != nil {
					log.Error().Err(err).Send()
					return
				}
				text = t
				badUserDWInfo = false
				return
			}
		}
	}()

	if badUserDWInfo {
		log.Debug().Str("err by", "badUserDWInfo").Msg("err in StartHandler")
		util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
		return
	}

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				entity.GetCallbackButton(entity.BUY_SELL),
				entity.GetCallbackButton(entity.OrderTrade),
			},
			{
				entity.GetCallbackButton(entity.WALLET),
				entity.GetCallbackButton(entity.ASSETS),
			},
			{
				entity.GetCallbackButton(entity.HistoryOrder),
				entity.GetCallbackButton(entity.HistoryTransfer),
			},
			{
				util.UrlButton(entity.CallbackTextMap[entity.AppDownload], "https://hellodex.io/Download"),
				entity.GetCallbackButton(entity.RefalshStartBalacne),
			},
			{
				entity.GetCallbackButton(entity.SETTING),
				util.UrlButton(entity.CallbackTextMap[entity.AdminUrl], "https://t.me/HelloDex_cn"),
			},
			{
				entity.GetCallbackButton(entity.InviteButton),
				entity.GetCallbackButton(entity.AIMonitorButton),
			},
			{
				util.UrlButton(entity.CallbackTextMap[entity.Other], "https://t.me/HelloDex_cn"),
			},
		},
	}

	chatID := util.EffectId(update)

	sendParams := &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: kb,
		ParseMode:   models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	}

	store.BotMessageAdd()
	message, err := b.SendMessage(ctx, sendParams)
	if err != nil {
		if bot.IsTooManyRequestsError(err) {
			queue.RetryPushMessage(sendParams)
		}
		log.Error().Err(err).Send()
		return
	}

	UserStartReflashSet(chatID, message.ID)
}

func StartReflashInfo(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	// delete cache key make it hard update
	store.Delete(chatId, api.UserProfilePrefix)

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	text := ""

	badUserDWInfo := true
	func() {
		dW, _, _ := callback.UserDefaultWalletInfo(userInfo)
		tokens, err := api.GetTokensByWalletAddress(dW.Wallet, dW.ChainCode, userInfo)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}

		for _, t := range tokens.Data {
			if util.IsNativeCoion(t.Address) {
				nativeCoinBalance := util.FormatNumber(util.ShiftLeftStr(t.Amount, t.Decimals))
				usdTotalAmount := util.FormatNumber(t.TotalAmount)
				botUserName := store.GetEnv(store.BOT_USERNAME)
				t, err := template.RanderStart(
					t.Symbol,
					dW.Wallet,
					nativeCoinBalance,
					usdTotalAmount,
					userInfo.Data.InviteCode,
					botUserName,
				)
				if err != nil {
					log.Error().Err(err).Send()
					return
				}
				text = t
				badUserDWInfo = false
				return
			}
		}
	}()

	if badUserDWInfo {
		log.Debug().Str("err by", "badUserDWInfo").Msg("err in StartHandler")
		util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
		return
	}

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				entity.GetCallbackButton(entity.BUY_SELL),
				entity.GetCallbackButton(entity.OrderTrade),
			},
			{
				entity.GetCallbackButton(entity.WALLET),
				entity.GetCallbackButton(entity.ASSETS),
			},
			{
				entity.GetCallbackButton(entity.HistoryOrder),
				entity.GetCallbackButton(entity.HistoryTransfer),
			},
			{
				util.UrlButton(entity.CallbackTextMap[entity.AppDownload], "https://hellodex.io/Download"),
				entity.GetCallbackButton(entity.RefalshStartBalacne),
			},
			{
				entity.GetCallbackButton(entity.SETTING),
				util.UrlButton(entity.CallbackTextMap[entity.AdminUrl], "https://t.me/HelloDex_cn"),
			},
			{
				util.UrlButton(entity.CallbackTextMap[entity.Other], "https://t.me/HelloDex_cn"),
			},
		},
	}

	messageID := UserStartReflashGet(chatId)
	if messageID == 0 {
		return
	}
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatId,
		Text:        text,
		MessageID:   messageID,
		ReplyMarkup: kb,
		ParseMode:   models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
}

func StartWrapMiddlewares(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := util.EffectId(update)
		// cache user in bot

		if update.Message != nil && len(update.Message.Entities) > 0 {
			for _, entity := range update.Message.Entities {
				if entity.Type == models.MessageEntityTypeBotCommand {
					cmds := strings.Split(update.Message.Text, " ")

					if cmds[0] == "/start" {
						if len(cmds) > 1 {

							afterStart := cmds[1]
							log.Debug().Str("afterStart", afterStart).Send()

							if mapResult, match, err := MatchCodeAndToken(afterStart); match && err == nil {
								typeIs, ok := mapResult["type"].(int)
								if !ok {
									logger.StdLogger().Error().Str("start=", update.Message.Text).Send()
									return
								}
								switch typeIs {
								case 1:
									token := mapResult["B"].(string)
									code := mapResult["I"].(string)
									api.BindUserInvitationCode(chatID, code)
									newUpdate := BuildNewUpdateForTokenAddress(token, update)
									b.ProcessUpdate(ctx, newUpdate)
									return
								case 2:
									token := mapResult["B"].(string)
									newUpdate := BuildNewUpdateForTokenAddress(token, update)
									b.ProcessUpdate(ctx, newUpdate)
									return
								case 3:
									code := mapResult["I"].(string)
									api.BindUserInvitationCode(chatID, code)
									StartHandler(ctx, b, update)
									return
								}

							} else {
								if !strings.Contains(afterStart, "TS_") {
									return
								}
								// util.QuickMessage(ctx, b, chatID, fmt.Sprintf("正在查询：%s", cmds[1]))
								args := MatchDeeplink(cmds[1])
								platform := args[len(args)-1]
								userInfo, err := api.GetUserProfile(chatID)
								if err != nil {
									util.QuickMessage(ctx, b, chatID, fmt.Sprintf("出错了,%s", util.AdminUrl))
									log.Error().Err(err).Send()
									return
								}
								templ, err := template.RanderStartLogin(userInfo, "https://example.com")
								if err != nil {
									util.QuickMessage(ctx, b, chatID, fmt.Sprintf("出错了,%s", util.AdminUrl))
									log.Error().Err(err).Send()
									return
								}

								log.Debug().Str("platform", platform).Send()

								matchPlatform := strings.Split(platform, "_")
								if len(matchPlatform) == 0 {
									util.QuickMessage(ctx, b, chatID, fmt.Sprintf("出错了,%s", util.AdminUrl))
									log.Error().Err(err).Send()
									return
								}

								result, err := api.GetTg2WebLoginToken(chatID, matchPlatform[len(matchPlatform)-1])
								if err != nil {
									util.QuickMessage(ctx, b, chatID, fmt.Sprintf("出错了,%s", util.AdminUrl))
									log.Error().Err(err).Send()
									return
								}
								url := gjson.GetBytes(result.([]byte), "data.url").String()

								button := util.UrlButton("登陆网站", url)
								MessageWithButton(ctx, b, chatID, templ, button)
								log.Debug().MsgFunc(func() string {
									return fmt.Sprintf("user: %d %s params: %s", chatID, "正在使用deeplink", update.Message.Text)
								})
								// next(ctx, b, update)
								return
							}

						}
					}
				}
			}
		}

		next(ctx, b, update)
	}
}

func MatchDeeplink(link string) []string {
	// link example: "l_1234567890_P_Web"
	params := strings.Split(link, " ")
	log.Debug().Interface("params", params).Send()
	return params
}

// start=invitationCode=xxxxx
func MatchInvitationCode(link string) string {
	return strings.TrimPrefix(link, "invitationCode=")
}

// start=token=xxxxxxxxxxxxx
func MatchTokenAddress(link string, u *models.Update) *models.Update {
	tokenAddress := strings.TrimPrefix(link, "token=")
	newUpdate := &models.Update{
		Message: &models.Message{
			Text: tokenAddress,
			From: u.Message.From,
			Chat: u.Message.Chat,
			ID:   u.Message.ID,
			Date: int(time.Now().Unix()),
		},
		ID: u.ID,
	}
	return newUpdate
}

func BuildNewUpdateForTokenAddress(B string, u *models.Update) *models.Update {
	newUpdate := &models.Update{
		Message: &models.Message{
			Text: B,
			From: u.Message.From,
			Chat: u.Message.Chat,
			ID:   u.Message.ID,
			Date: int(time.Now().Unix()),
		},
		ID: u.ID,
	}
	return newUpdate
}

// start=b=xxxxx_I=xxxx
func MatchTokenAndInvitationCode(link string) ([]string, bool, error) {
	expr := `b=(.*)_I=(.*)`
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, false, err
	}

	if re.MatchString(link) {
		matches := re.FindStringSubmatch(link)

		return matches[1:], true, nil

	}

	return nil, false, nil
}

// 1：交易和邀请
// 2：只交易
// 3：只邀请
func MatchCodeAndToken(link string) (map[string]any, bool, error) {
	data := link
	if data == "" {
		return nil, false, errors.New("empty link")
	}
	result := make(map[string]any)

	if strings.Contains(data, "B_") && strings.Contains(data, "I_") {
		bIndex := strings.Index(data, "B_") + 2
		iIndex := strings.Index(data, "I_")

		result["type"] = 1
		result["B"] = data[bIndex:iIndex]
		result["I"] = data[iIndex+2:]

	} else if strings.Contains(data, "B_") {
		bIndex := strings.Index(data, "B_") + 2
		result["type"] = 2
		result["B"] = data[bIndex:]

	} else if strings.Contains(data, "I_") {
		iIndex := strings.Index(data, "I_") + 2
		result["type"] = 3
		result["I"] = data[iIndex:]
	} else {
		// don't match B_ or I_
		return nil, false, nil
	}

	return result, true, nil
}

func MessageWithButton(ctx context.Context, b *bot.Bot, userID int64, text string, button models.InlineKeyboardButton) {
	line := []models.InlineKeyboardButton{button}
	keyboard := [][]models.InlineKeyboardButton{line}
	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    userID,
		Text:      text,
		ParseMode: "HTML",
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
}
