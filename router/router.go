package router

import (
	"context"
	"net/http"

	"github.com/go-telegram/bot"
	"github.com/hellodex/tradingbot/config"
)

// use webhook to get update
func SetWebhook(ctx context.Context, b *bot.Bot) {
	b.SetWebhook(ctx, &bot.SetWebhookParams{
		DropPendingUpdates: true,
		URL:                config.YmlConfig.Env.TgHook,
		SecretToken:        config.YmlConfig.Env.TgHookToken,
	})
	go func() {
		http.ListenAndServe(config.YmlConfig.Env.LocalHost, wrapWebhookHandler(b))
	}()
	b.StartWebhook(ctx)
}

func wrapWebhookHandler(bot *bot.Bot) http.HandlerFunc {
	originalHandler := bot.WebhookHandler()
	return func(w http.ResponseWriter, r *http.Request) {
		originalHandler(w, r)
		w.WriteHeader(http.StatusNoContent)
	}
}
