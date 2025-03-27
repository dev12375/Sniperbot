package callback

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/util"
)

func HelpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := util.EffectId(update)

	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      "这里是帮助信息",
		ParseMode: "HTML",
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
}
