package callback

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/session"
	"github.com/hellodex/tradingbot/template"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
)

func Same(old model.PositionByWalletAddress, newData model.PositionByWalletAddress) bool {
	return old.Data.Amount == newData.Data.Amount &&
		old.Data.Price == newData.Data.Price &&
		old.Data.Volume == newData.Data.Volume &&
		old.Data.TotalBuyAmount == newData.Data.TotalBuyAmount &&
		old.Data.TotalBuyVolume == newData.Data.TotalBuyVolume &&
		old.Data.AveragePrice == newData.Data.AveragePrice &&
		old.Data.TotalSellAmount == newData.Data.TotalSellAmount &&
		old.Data.TotalSellVolume == newData.Data.TotalSellVolume &&
		old.Data.TotalEarn == newData.Data.TotalEarn &&
		old.Data.TotalEarnRate == newData.Data.TotalEarnRate
}

func ReflashTokenInfo(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)

	tokenAddress := func() string {
		value, ok := session.GetSessionManager().Get(chatId, session.UserSelectTokenAddressCache)
		if ok {
			v, parseOk := value.(string)
			if parseOk {
				return v
			}
		}

		return ""
	}()

	if tokenAddress == "" || tokenAddress == "1" {
		log.Debug().Msg("ReflashTokenInfo tokenAddress empty")
		return
	}

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	dw, _, _ := UserDefaultWalletInfo(userInfo)
	tokenInfo, err := api.GetPositionByWalletAddress(
		dw.Wallet,
		tokenAddress,
		dw.ChainCode,
		userInfo,
	)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	lastSwapMessaage := func() *model.MessageWrap {
		v, has := session.GetSessionManager().Get(chatId, session.UserLastSwapMessage)
		if has {
			message, ok := v.(*model.MessageWrap)
			if ok {
				return message
			}
		}
		return nil
	}()

	text, err := template.RanderTokenInfo(tokenInfo)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	if lastSwapMessaage == nil {
		return
	}

	if Same(lastSwapMessaage.Tokeninfo, tokenInfo) {
		log.Debug().Msg("text is same no need to update")
		return
	}

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatId,
		Text:        text,
		MessageID:   lastSwapMessaage.Message.ID,
		ParseMode:   "HTML",
		ReplyMarkup: lastSwapMessaage.Message.ReplyMarkup,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
}
