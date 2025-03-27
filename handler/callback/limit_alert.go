package callback

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/logger"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/util"
)

func botButton(name string, url string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text: name,
		URL:  url,
	}
}

func Alert(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	message := "当前bot 繁忙，回复延迟大请使用以下机器人"

	botStatus, err := store.GetBotsStatus()
	if err != nil {
		logger.StdLogger().Error().Err(err).Msg("获取机器人状态失败")
		return
	}

	if len(botStatus) == 0 {
		store.BotMessageAdd()
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatId,
			Text:   message + "，但目前没有可用的替代机器人信息。",
		})
		return
	}

	kb := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{},
	}

	for k, v := range botStatus {
		url := fmt.Sprintf("https://t.me/%s", k)
		logger.StdLogger().Info().Str("url", url).Msg("添加按钮")

		status := getSimpleBotStatus(v)
		buttonText := fmt.Sprintf("@%s - %s", k, status)

		buttonRow := []models.InlineKeyboardButton{
			botButton(buttonText, url),
		}
		kb.InlineKeyboard = append(kb.InlineKeyboard, buttonRow)
	}

	if len(kb.InlineKeyboard) == 0 {
		store.BotMessageAdd()
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatId,
			Text:   message + "，但目前没有可用的替代机器人。",
		})
		return
	}

	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        message,
		ReplyMarkup: &kb,
	})
}

func getSimpleBotStatus(count string) string {
	s, err := strconv.Atoi(count)
	if err != nil {
		return "状态未知"
	}

	switch {
	case s > 55:
		return "⚠️ 繁忙"
	case s > 30:
		return "⚡ 拥挤"
	default:
		return "✅ 流畅"
	}
}
