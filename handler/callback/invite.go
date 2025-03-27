package callback

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/entity"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/template"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
)

func InviteHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := util.EffectId(update)
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			// line 1
			{
				// entity.GetCallbackButton(entity.INVITE_DETIAL),
				entity.GetCallbackButton(entity.Withdrawal),
			},
		},
	}

	userInfo, err := api.GetUserProfile(chatID)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}
	data, err := api.GetMyCommissionSummary(userInfo)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	store.UserSetCommissionInfo(chatID, data)

	var body map[string]any
	err = json.Unmarshal(data, &body)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	log.Debug().Interface("commission body", body).Send()
	botUserName := store.GetEnv(store.BOT_USERNAME)

	text, err := template.RanderMyCommissionSummary(userInfo.Data.InviteCode, botUserName, body)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}
	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
}
