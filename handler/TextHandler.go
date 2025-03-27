package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/handler/callback"
	"github.com/hellodex/tradingbot/util"
)

func TextHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	if update.Message.Chat.Type != "private" {
		return
	}

	// check if it is command
	if util.IsCommand(update.Message.Text) {
		CommandHandler(ctx, b, update)
		return
	}

	// 检查是否在回复滑点设置
	// chatID := update.Message.Chat.ID
	chatID := util.EffectId(update)
	if update.Message.ReplyToMessage != nil {
		if msgID, exists := callback.ReplaySlippySettingCache[chatID]; exists && msgID == update.Message.ReplyToMessage.ID {
			callback.HandleSlippyReply(ctx, b, update)
			return
		}
	}

	TokenInfoHandler(ctx, b, update)
}
