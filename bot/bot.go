package bot

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/cenkalti/backoff/v5"
	cfg "github.com/hellodex/tradingbot/config"
	"github.com/hellodex/tradingbot/logger"
	"github.com/hellodex/tradingbot/queue"
	"github.com/hellodex/tradingbot/router"
	"github.com/hellodex/tradingbot/store"
	"github.com/spf13/cast"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/entity"
	"github.com/hellodex/tradingbot/handler"
	"github.com/rs/zerolog/log"
)

func InitBots(ctx context.Context) []*bot.Bot {
	bots := make([]*bot.Bot, 0)
	// filterConfigs(&entity.BotConfigs)
	log.Info().Msg("初始化bot中...")

	log.Info().Msgf("bot数量 %v", len(entity.BotConfigs))
	for _, config := range entity.BotConfigs {
		var ops []bot.Option
		switch config.Type {
		case -1:
			ops = initMainBotOptions()
		case 0:
			continue
		case 1:
			continue
		case 2:
			continue
			// ops = initSignalBotOptions(config)
		}
		b, err := bot.New(config.ApiKey, ops...)
		if err != nil {
			log.Error().Err(err).Msg("创建bot失败")
			continue
		}
		bots = append(bots, b)
		me := new(models.User)
		operation := func() (*models.User, error) {
			m, err := b.GetMe(ctx)
			if err != nil {
				return nil, errors.New("get me failed")
			}
			logger.StdLogger().Info().Msg("获取GetMe成功")
			return m, nil
		}
		attemptCount := 0
		notifyFunc := func(err error, backoffDelay time.Duration) {
			attemptCount++
			logger.StdLogger().Error().Msgf("重试第%d次失败: %v. 下一次重试时间: %v...后", attemptCount, err, backoffDelay)
		}
		back := backoff.NewConstantBackOff(500 * time.Millisecond)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		r, err := backoff.Retry(
			ctx,
			operation,
			backoff.WithMaxTries(50),
			backoff.WithBackOff(back),
			backoff.WithNotify(notifyFunc),
		)
		if err != nil {
			logger.StdLogger().Error().Msgf("重试50次后失败: %v\n", err)
			continue
		}
		me = r

		// setting bot id
		botId := cast.ToString(me.ID)
		err = store.NewEnv(store.BOT_ID, botId)
		if err != nil {
			log.Error().Err(err).Msg("获取bot信息失败")
			continue
		}
		logger.StdLogger().Info().Str("bot_id", botId).Msg("获取BotID成功")
		log.Debug().Str("bot id", botId).Send()

		// setting bot userName
		botUserName := me.Username
		err = store.NewEnv(store.BOT_USERNAME, botUserName)
		if err != nil {
			log.Error().Err(err).Msg("获取bot信息失败")
			continue
		}
		logger.StdLogger().Info().Msg("获取BotUserName成功")

		InitBotWarpServer(b)
		go queue.InitPushMessage(b)

		entity.BotMap[me.ID] = b
		entity.BotConfigMap[me.ID] = config
		entity.UserBotConfigMap[config.UserId] = append(entity.UserBotConfigMap[config.UserId], config)
	}
	return bots
}

func StartBots(ctx context.Context, bots []*bot.Bot) {
	os.Setenv("BOT_INIT_TIMESTAMPS", cast.ToString(time.Now().Unix()))
	handler.SetBotCommand(ctx, bots)
	for _, b := range bots {
		me, err := b.GetMe(ctx)
		if err != nil {
			log.Error().Err(err).Msg("获取bot信息失败")
			continue
		}
		log.Info().Msgf("启动bot %v", me.Username)
		botCtx, botCancel := context.WithCancel(ctx)
		config := entity.BotConfigMap[me.ID]
		config.Cancel = botCancel
		entity.BotConfigMap[me.ID] = config
		go func(b *bot.Bot, botCtx context.Context, botCancel context.CancelFunc) {
			defer botCancel()
			// TODO: finish webhook mode
			if cfg.YmlConfig.Env.WebHookOpen {
				b.DeleteWebhook(ctx, &bot.DeleteWebhookParams{
					DropPendingUpdates: true,
				})
				go router.SetWebhook(botCtx, b)
				log.Info().Msg("starting with webhook")
			} else {
				b.DeleteWebhook(ctx, &bot.DeleteWebhookParams{
					DropPendingUpdates: true,
				})
				go b.Start(botCtx)
				log.Info().Msg("starting with long poll")
			}
			<-botCtx.Done()
			log.Info().Msgf("中止bot %v", me.Username)
		}(b, botCtx, botCancel)
	}
}

func filterConfigs(botConfigs *[]entity.BotConfig) {
	var newConfigs []entity.BotConfig
	for _, config := range *botConfigs {
		if config.Type != -1 {
			newConfigs = append(newConfigs, config)
		}
	}
	*botConfigs = newConfigs
}

func initMainBotOptions() []bot.Option {
	var allOptions []bot.Option

	chOpt := bot.WithUpdatesChannelCap(1024)
	workerOpt := bot.WithWorkers(5)

	mainBotOptions := handler.GetCallbackHandler()
	// TODO: 注意是不是命令
	textHandlerOpt := bot.WithDefaultHandler(handler.TextHandler)
	mainBotOptions = append(mainBotOptions, textHandlerOpt)
	botTokenOptions := bot.WithWebhookSecretToken(cfg.YmlConfig.Env.TgHookToken)

	allOptions = append(allOptions, chOpt, workerOpt, botTokenOptions, bot.WithSkipGetMe())
	allOptions = append(allOptions, mainBotOptions...)

	return allOptions
}

func init() {
	api.FreshBotConfigs()
}
