package commands

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/entity"
	"github.com/hellodex/tradingbot/handler/callback"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
)

const menuTemplate = `
钱包地址：
<code>%s</code> (点击复制)

钱包余额：%s SOL ($ %s)
`

func MenuHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := util.EffectId(update)
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
				entity.GetCallbackButton(entity.AppDownload),
			},
			{
				entity.GetCallbackButton(entity.SETTING),
				entity.GetCallbackButton(entity.AdminUrl),
			},
			{
				entity.GetCallbackButton(entity.Other),
			},
		},
	}

	text := ""
	chatId := util.EffectId(update)
	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
		return
	}
	badUserDWInfo := true
	func() {
		dW, _, _ := callback.UserDefaultWalletInfo(userInfo)
		tokns, err := api.GetTokensByWalletAddress(dW.Wallet, dW.ChainCode, userInfo)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}

		for _, t := range tokns.Data {
			if util.IsNativeCoion(t.Address) {
				nativeCoinBalance := util.FormatNumber(util.ShiftLeftStr(t.Amount, t.Decimals))
				usdTotalAmount := util.FormatNumber(t.TotalAmount)
				text = fmt.Sprintf(menuTemplate, dW.Wallet, nativeCoinBalance, usdTotalAmount)
				badUserDWInfo = false
				return
			}
		}
	}()

	if badUserDWInfo {
		util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
		return
	}

	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
}
