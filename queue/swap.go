package queue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/rpc"
	"github.com/hellodex/tradingbot/session"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/template"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
)

var swapQueue = make(chan *SwapPayload, 1024)

type SwapPayload struct {
	B               *bot.Bot
	SwapBody        model.Swap
	BaseToken       model.TokenInner
	QuoteToken      model.TokenInner
	UserInfo        model.GetUserResp
	MessageID       int
	UserID          int64
	HandleWallet    model.Wallet
	Status          Event
	Tx              string
	UserInputAmount string
}

// AddProcessingSwapQueue 添加交易到队列，带有重试机制
func AddProcessingSwapQueue(sp *SwapPayload) error {
	// make it init
	sp.Status = Processing

	// 立即尝试发送
	select {
	case swapQueue <- sp:
		log.Info().
			Int("messageID", sp.MessageID).
			Int64("userID", sp.UserID).
			Msg("Successfully added swap to queue")
		return nil
	default:
		// 队列满，记录警告
		log.Warn().
			Int("messageID", sp.MessageID).
			Int64("userID", sp.UserID).
			Msg("Queue is full, retrying...")

		// 使用带超时的重试
		timer := time.NewTimer(100 * time.Millisecond)
		defer timer.Stop()

		select {
		case swapQueue <- sp:
			log.Info().
				Int("messageID", sp.MessageID).
				Int64("userID", sp.UserID).
				Msg("Successfully added swap to queue after retry")
			return nil
		case <-timer.C:
			log.Error().
				Int("messageID", sp.MessageID).
				Int64("userID", sp.UserID).
				Msg("Failed to add swap to queue after retry")
			return ErrQueueFull
		}
	}
}

// AddFailedSwapQueue
func AddFailedSwapQueue(sp *SwapPayload) error {
	// make it failed
	sp.Status = Failed

	// 立即尝试发送
	select {
	case swapQueue <- sp:
		log.Info().
			Int("messageID", sp.MessageID).
			Int64("userID", sp.UserID).
			Msg("Successfully added swap to queue")
		return nil
	default:
		// 队列满，记录警告
		log.Warn().
			Int("messageID", sp.MessageID).
			Int64("userID", sp.UserID).
			Msg("Queue is full, retrying...")

		// 使用带超时的重试
		timer := time.NewTimer(100 * time.Millisecond)
		defer timer.Stop()

		select {
		case swapQueue <- sp:
			log.Info().
				Int("messageID", sp.MessageID).
				Int64("userID", sp.UserID).
				Msg("Successfully added swap to queue after retry")
			return nil
		case <-timer.C:
			log.Error().
				Int("messageID", sp.MessageID).
				Int64("userID", sp.UserID).
				Msg("Failed to add swap to queue after retry")
			return ErrQueueFull
		}
	}
}

func InitSwapConsumers(ctx context.Context) {
	for i := 0; i < workerNum; i++ {
		go func(workerID int) {
			HandleSwapQueue(ctx, workerID)
		}(i)
	}
}

// HandleSwapQueue
func HandleSwapQueue(ctx context.Context, workerID int) {
	log.Info().Msgf("Swap consumer %d started and watching queue", workerID)

	// 持续运行
	for {
		select {
		case sp := <-swapQueue:
			processSwap(sp)

		case <-ctx.Done():
			log.Info().Msgf("Swap consumer %d shutting down due to context cancellation", workerID)
			return
		}
	}
}

// processSwap
func processSwap(sp *SwapPayload) {
	if sp == nil {
		return
	}

	switch sp.Status {
	case Processing:
		log.Info().Msg("Processing")
		processingSwap(sp)
	case Success:
		log.Info().Msg("Success")
		successSwap(sp)
	case Failed:
		log.Info().Msg("Failed")
		failedSwap(sp)
	}
}

func successSwap(sp *SwapPayload) {
	ctx := context.Background()
	baseToken := sp.BaseToken
	quoteToken := sp.QuoteToken
	amount := sp.UserInputAmount
	msgqq := func() string {
		// is buy
		if sp.SwapBody.Type == "0" {
			return fmt.Sprintf("✅ %s 买 %s %s，交易成功，", baseToken.Symbol, amount, quoteToken.Symbol)
		}
		return fmt.Sprintf("✅ %s 卖 %s %s，交易成功，", baseToken.Symbol, amount, baseToken.Symbol)
	}()
	scanUrl := util.GetChainScanUrl(sp.HandleWallet.ChainCode, sp.Tx)
	viewUrl := fmt.Sprintf(`<a href="%s">%s</a>`, scanUrl, " 点击查看区块浏览器")
	util.QuickMessage(ctx, sp.B, sp.UserID, msgqq+viewUrl)

	time.Sleep(3 * time.Second)

	tokenInfo, err := api.GetPositionByWalletAddress(
		sp.HandleWallet.Wallet,
		sp.BaseToken.Address,
		sp.HandleWallet.ChainCode,
		sp.UserInfo,
	)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	lastSwapMessaage := func() *model.MessageWrap {
		v, has := session.GetSessionManager().Get(sp.UserID, session.UserLastSwapMessage)
		if has {
			message, ok := v.(*model.MessageWrap)
			if ok {
				return message
			}
		}
		return nil
	}()

	if lastSwapMessaage == nil {
		return
	}

	log.Debug().Interface("lastSwapMessaage", lastSwapMessaage).Send()

	text, err := template.RanderTokenInfo(tokenInfo)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	msgs, ok := store.Get(sp.UserID, store.WaitCleanMessage)
	if ok {
		msgs, parseOk := msgs.(string)
		if parseOk {
			var messageIDs []int
			strArray := strings.Split(msgs, "_")
			for _, m := range strArray {
				messageIDs = append(messageIDs, cast.ToInt(m))
			}
			sp.B.DeleteMessages(ctx, &bot.DeleteMessagesParams{
				ChatID:     sp.UserID,
				MessageIDs: messageIDs,
			})
		}
	}

	store.BotMessageAdd()
	message, err := sp.B.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      sp.UserID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: lastSwapMessaage.Message.ReplyMarkup,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	messageWrap := model.NewMessageWrap(sp.UserID, *message, tokenInfo)
	sm := session.GetSessionManager()
	sm.Set(sp.UserID, session.UserLastSwapMessage, messageWrap)

	_, err = sp.B.PinChatMessage(ctx, &bot.PinChatMessageParams{
		ChatID:              sp.UserID,
		MessageID:           message.ID,
		DisableNotification: false,
	})
	if err != nil {
		log.Error().Err(err).Send()
	}
}

func failedSwap(sp *SwapPayload) {
	ctx := context.Background()
	baseToken := sp.BaseToken
	quoteToken := sp.QuoteToken
	amount := sp.UserInputAmount
	msgqq := func() string {
		// is buy
		if sp.SwapBody.Type == "0" {
			return fmt.Sprintf("❌ %s 买 %s %s，交易失败，", baseToken.Symbol, amount, quoteToken.Symbol)
		}
		return fmt.Sprintf("❌ %s 卖 %s %s，交易失败，", baseToken.Symbol, amount, baseToken.Symbol)
	}()
	util.QuickMessage(ctx, sp.B, sp.UserID, msgqq+util.AdminUrl)
}

func processingSwap(sp *SwapPayload) {
	ctx := context.Background()
	userInfo := sp.UserInfo
	swap := sp.SwapBody
	chatId := sp.UserID
	b := sp.B

	// check the transfer chainCode and default wallet chain, if not match, go
	// go select
	result, err := api.SendSwap(swap, userInfo)
	if err != nil {
		AddFailedSwapQueue(sp)
		return
	}

	// handle code 102
	if gjson.GetBytes(result, "code").Int() == 102 {
		msg := gjson.GetBytes(result, "msg").String()
		util.QuickMessage(ctx, b, chatId, msg)
		return
	}

	tx := gjson.GetBytes(result, "data.tx").String()
	chainCode := func() string {
		for _, w := range userInfo.Data.Wallets {
			for _, wallet := range w {
				if swap.WalletId == wallet.WalletId {
					log.Debug().Interface("swap wallet", wallet).Send()
					return wallet.ChainCode
				}
			}
		}
		return ""
	}()
	scanUrl := util.GetChainScanUrl(chainCode, tx)
	viewUrl := fmt.Sprintf(`<a href="%s">%s</a>`, scanUrl, "点击查看区块浏览器")

	util.QuickMessage(ctx, b, chatId, fmt.Sprintf("⏳链上确认中 %s", viewUrl))
	// err := rpc.SOL_PollTransactionStatus(tx)
	if chainCode == "" {
		log.Error().Err(errors.New("get user wallet chainCode err in swap")).Send()
		util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
		return
	}
	err = rpc.PollTransactionStatus(chainCode, tx)
	if err != nil {
		if errors.Is(err, rpc.ErrPollTxMaxRetry) {
			util.QuickMessage(ctx, b, chatId, err.Error())
		}
		return
	}
	// util.QuickMessage(ctx, b, chatId, "交易成功")
	// 交易成功重新进去队列
	sp.Status = Success
	sp.Tx = tx
	swapQueue <- sp
}
