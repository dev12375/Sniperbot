package callback

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
)

func CallbackOrderFollow(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := util.EffectId(update)

	store.BotMessageAdd()
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      "请输入合约地址",
		ParseMode: "HTML",
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
	}
}
