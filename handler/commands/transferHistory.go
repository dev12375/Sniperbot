package commands

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/handler/callback"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/template"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

// include swap trade order transfer
func TransferHistoryHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := util.EffectId(update)
	util.QuickMessage(ctx, b, chatID, "正在查询......")
	userInfo, err := api.GetUserProfile(chatID)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	// list TgDefaultWallet transfer History
	dw, _, _ := callback.UserDefaultWalletInfo(userInfo)

	history, err := api.ListTransferHistory(cast.ToFloat64(dw.WalletId), dw.ChainCode, userInfo)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, "出错了，请联系客服")
		return
	}

	if len(history.Data) == 0 {
		util.QuickMessage(ctx, b, chatID, "没有最近交易记录！")
		return
	}

	var msg string
	msg, err = template.RanderListTransferHistory(history.Data, 1, 5)
	if err != nil {
		log.Error().Err(err).Send()
		msg = err.Error()
	}
	store.BotMessageAdd()
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      msg,
		ParseMode: models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, "出错了，请联系客服！")
	}
}
