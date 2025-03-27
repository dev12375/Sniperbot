package queue

import (
	"context"
	"time"

	"github.com/go-telegram/bot"
	"github.com/hellodex/tradingbot/store"
	"github.com/rs/zerolog/log"
)

var messageQueue = make(chan *bot.SendMessageParams, 1024)

func RetryPushMessage(msg *bot.SendMessageParams) {
	messageQueue <- msg
}

func InitPushMessage(b *bot.Bot) {
	for message := range messageQueue {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		store.BotMessageAdd()
		_, err := b.SendMessage(ctx, message)
		if err != nil {
			if bot.IsTooManyRequestsError(err) {
				log.Error().Msg("too many req waitting......")
				time.Sleep(time.Duration(err.(*bot.TooManyRequestsError).RetryAfter))
			}
		}
		cancel()
	}
}
