package bot

import (
	"context"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
)

var (
	bbbb   *bot.Bot
	once   sync.Once
	update chan *models.Update
)

func InitBotWarpServer(b *bot.Bot) {
	once.Do(func() {
		bbbb = b
		ch, err := util.UnsafeGetUpdateChan(b)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		update = ch
	})
}

func SendMessage(chatId int64, message string) (*models.Message, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kb := util.NewAiMonitorPusherButton()
	sendmessage, err := bbbb.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatId,
		Text:      message,
		ParseMode: "HTML",
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
		ReplyMarkup: kb,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return nil, err
	}
	// store in redis for button recall
	return sendmessage, nil
}

func UnsafeNewUpdate(u *models.Update) {
	update <- u
}

func UnsafeGetUpdateChan() chan *models.Update {
	return update
}
