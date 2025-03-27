package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/hellodex/tradingbot/bot"
	"github.com/hellodex/tradingbot/config"
	_ "github.com/hellodex/tradingbot/handler"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/queue"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/template"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var _ = func() any {
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05"
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	return nil
}()

func main() {
	log.Debug().Msg("现在是 debug 日志，记得切换！！！")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	store.InitRedis()

	bots := bot.InitBots(ctx)
	go bot.StartBots(ctx, bots)

	// InitSwapConsumers
	go queue.InitSwapConsumers(ctx)

	// init AI monitor pusher
	aiMessageCh, err := store.SubChannel(config.YmlConfig.RedisPush.MessageCh)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	go func() {
		pusher(aiMessageCh)
	}()
	log.Info().Msg("程序已经启动成功，发送 Ctrl + c 可以 kill 程序")

	// wait
	<-ctx.Done()
	log.Info().Msg("程序已经退出！！")
}

func pusher(ch <-chan *redis.Message) {
	for msg := range ch {
		msgData := []byte(msg.Payload)
		log.Debug().RawJSON("push", msgData).Send()
		uuid := gjson.GetBytes(msgData, "payload.uuid").String()

		if uuid == "" {
			log.Error().Interface("uuid", uuid).Msg("Invalid or missing UUID")
			continue
		}

		value, err := store.UserInBot(uuid)
		if err != nil {
			if err == redis.Nil {
				log.Error().Err(err).Str("uuid", uuid).Msg("Failed to get user in bot reids nil")
				continue
			}
			log.Error().Err(err).Str("uuid", uuid).Msg("Failed to get user in bot")
			continue
		}

		//  "botId::userId"
		parts := strings.Split(value, "::")
		if len(parts) != 2 {
			log.Error().Str("value", value).Msg("Invalid format: expected botId::userId, user not in bot")
			continue
		}

		botId := parts[0]
		userIdStr := parts[1]

		userId, err := strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			log.Error().Err(err).Str("user_id_str", userIdStr).Msg("Failed to parse user ID")
			continue
		}

		currentBotId := store.GetEnv(store.BOT_ID)
		if currentBotId == "" || currentBotId != botId {
			log.Debug().Str("expected_bot", currentBotId).Str("actual_bot", botId).Msg("Bot ID mismatch")
			continue
		}

		messageTmpl, err := template.RenderTgUserTokenPush(msgData)
		if err != nil {
			log.Error().Err(err).Str("bot_id", botId).Int64("user_id", userId).Msg("Failed to render message template")
			continue
		}

		sendMsg, err := bot.SendMessage(userId, messageTmpl)
		if err != nil {
			log.Error().Err(err).Str("bot_id", botId).Int64("user_id", userId).Msg("Failed to send message")
			continue
		}

		allParts := gjson.GetManyBytes(
			msgData,
			"payload.chainCode",
			"payload.baseAddress",
			"payload.topic",
		)

		// in redis
		puserhandlerReq := model.PusherHandlerReqData{
			ChainCode:   allParts[0].String(),
			BaseAddress: allParts[1].String(),
			MonitorType: allParts[2].String(),
		}

		if dbdbd, err := json.Marshal(puserhandlerReq); err == nil {
			store.NewPusherMessage(userId, cast.ToString(sendMsg.ID), string(dbdbd))
		}

	}
}
