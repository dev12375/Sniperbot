package callback

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/util"
)

func HistoryListKeyBoard() models.InlineKeyboardMarkup {
	buttons := []models.InlineKeyboardButton{
		{
			Text:         "交易记录",
			CallbackData: "list_swap",
		},
		{
			Text:         "转账记录",
			CallbackData: "list_transfer",
		},
	}

	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{buttons},
	}
}

func OrdersListKeyBoard() models.InlineKeyboardMarkup {
	buttons := []models.InlineKeyboardButton{
		{
			Text:         "当前委托",
			CallbackData: "list_orders",
		},
		{
			Text:         "委托历史",
			CallbackData: "list_orders_history",
		},
	}

	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{buttons},
	}
}

func OrdersList(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)

	text := "选择你要查看的记录"
	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        text,
		ReplyMarkup: OrdersListKeyBoard(),
		ParseMode:   models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
}

func HistoryList(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)

	text := "选择你要查看的记录"
	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        text,
		ReplyMarkup: HistoryListKeyBoard(),
		ParseMode:   models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
}
