package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/handler/callback"
	"github.com/hellodex/tradingbot/logger"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/queue"
	"github.com/hellodex/tradingbot/rpc"
	"github.com/hellodex/tradingbot/session"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/template"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
)

var (
	replayBuyMsgCache       = make(map[int64]int)
	replaySellMsgCache      = make(map[int64]int)
	replaySellMsgCacheIsNum = make(map[int64]bool)
)

func setReplaySellMsgCacheIsNum(userID int64, isNum bool) {
	tradingLock.Lock()
	defer tradingLock.Unlock()
	replaySellMsgCacheIsNum[userID] = isNum
}

func getReplaySellMsgCacheIsNum(userID int64) (isNum bool, ok bool) {
	tradingLock.Lock()
	defer tradingLock.Unlock()
	info, ok := replaySellMsgCacheIsNum[userID]
	return info, ok
}

var (
	BUY_BUTTON  = "buy_%s_%s_"
	SELL_BUTTON = "sell_%s_%s_"
)

var tradingLock = sync.Mutex{}

func TokenInfoHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	var emptyPinMessage models.MaybeInaccessibleMessage
	if update.Message.PinnedMessage != emptyPinMessage {
		return
	}

	chatId := util.EffectId(update)
	if update.Message.ReplyToMessage != nil {
		if replayBuyMsgCache[update.Message.Chat.ID] == update.Message.ReplyToMessage.ID {
			num := cast.ToFloat64(update.Message.Text)
			processSwap(ctx, b, update, true, num)
			return
		}
		if replaySellMsgCache[update.Message.Chat.ID] == update.Message.ReplyToMessage.ID {
			num := cast.ToFloat64(update.Message.Text)
			processSwap(ctx, b, update, false, num)
			return
		}

		if v, exists := session.GetSessionManager().Get(chatId, session.UserSessionState); exists {
			if exists {
				if v.(string) == session.TransferToState {
					processTransferTo(ctx, b, update)
					return
				}
			}
		}

		if v, exists := session.GetSessionManager().Get(chatId, session.UserSessionState); exists {
			if exists {
				if v.(string) == session.LimitOrderState {
					processLimitOrder(ctx, b, update)
					return
				}
			}
		}
		if _, exists := store.UserGetAiMonitorInfo(chatId); exists {
			if state, exists := store.RedisGetState(chatId, "edit_current"); exists {
				logger.StdLogger().Info().Str("state", state).Send()
				processCurrentAimonitor(ctx, b, update)
				return
			}
			processAImonitor(ctx, b, update)
			return
		}

		if _, exists := store.UserGetCommissionInfo(chatId); exists {
			processWithdrawl(ctx, b, update)
			return
		}

	}

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
		return
	}

	waithSelectChain := false

	// check if come from select
	// sw := func() (w model.Wallet, comeFromSelect bool) {
	// sM := session.GetSessionManager()
	// value, has := sM.Get(chatId, session.UserSelectWalletCache)
	// if has {
	// 	w, ok := value.(model.Wallet)
	// 	if ok {
	// 		return w, true
	// 	}
	// }

	// 	return model.Wallet{}, false
	// }

	// get tokeninfo
	tokenInfo := func() model.PositionByWalletAddress {
		// from select
		// selectWallet, fromSelect := sw()
		// if fromSelect {
		// 	pwa, err := api.GetPositionByWalletAddress(
		// 		selectWallet.Wallet,
		// 		update.Message.Text,
		// 		selectWallet.ChainCode,
		// 		userInfo,
		// 	)
		// 	if err != nil {
		// 		return model.PositionByWalletAddress{}
		// 	}
		//
		// 	return pwa
		// }

		// 不来自钱包，直接是个地址，就得检查地址
		// 这里有两种情况，如果是select chain已经选择了，就直接拿那个链默认钱包返回这些信息，如果chain没设置
		// 返回选择chain
		// 如果是 solana地址，直接进行它的默认钱包进行操作即可
		// 检查地址，返回相应逻辑，是sol直接发送solana的查询token ，如果不是让他选择EVM哪个链再查询token
		isSolana, err := util.CheckValidAddress(update.Message.Text)
		if err != nil {
			log.Error().Err(err).Send()
			// util.QuickMessage(ctx, b, chatId, "输入的代币合约不正确，无法快速买入")
			return model.PositionByWalletAddress{}
		}
		if isSolana {
			dW, _, chain := callback.UserDefaultWalletInfo(userInfo)

			// the token is solana but your default wallet id is not solana

			if dW.ChainCode != "SOLANA" {
				CallbackSwitchWalletInChain(ctx, b, update, "SOLANA")
				return model.PositionByWalletAddress{}
			}
			pwa, err := api.GetPositionByWalletAddress(
				dW.Wallet,
				update.Message.Text,
				chain,
				userInfo,
			)
			if err != nil {
				log.Error().Err(err).Send()
				return model.PositionByWalletAddress{}
			}

			return pwa
		}

		supportEVMchainData, support := func() ([]model.ChainConfig, bool) {
			chainCfgs, err := api.GetChainConfigs()
			if err != nil {
				log.Error().Err(err).Send()
				return nil, false
			}
			cfgs := make([]model.ChainConfig, 0, len(chainCfgs.Data))
			for _, c := range chainCfgs.Data {
				if c.ChainCode != "SOLANA" {
					cfgs = append(cfgs, c)
				}
			}
			// sort by Sort key
			slices.SortFunc(cfgs, func(a, b model.ChainConfig) int {
				if a.Sort < b.Sort {
					return -1
				}
				if a.Sort > b.Sort {
					return 1
				}
				return 0
			})

			return cfgs, true
		}()

		if support {
			var buttons [][]models.InlineKeyboardButton
			for _, chainCfg := range supportEVMchainData {
				callbackData := fmt.Sprintf(session.UserSelectChainCache+"::%v::%s", chatId, chainCfg.ChainCode)
				buttonRow := []models.InlineKeyboardButton{
					{
						Text:         chainCfg.Chain,
						CallbackData: callbackData,
					},
				}
				buttons = append(buttons, buttonRow)
			}

			store.BotMessageAdd()
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    chatId,
				Text:      "请选择对应的公链：",
				ParseMode: "HTML",
				ReplyMarkup: &models.InlineKeyboardMarkup{
					InlineKeyboard: buttons,
				},
			})
			if err != nil {
				log.Error().Err(err).Send()
				return model.PositionByWalletAddress{}
			}
		}

		waithSelectChain = true
		return model.PositionByWalletAddress{}
	}()

	// empty
	if tokenInfo == (model.PositionByWalletAddress{}) {
		return
	}

	// 正在等待用户选择，所以退出了这个界面，会由 callbck select chain来操作
	// @@@selectChain
	if waithSelectChain {
		return
	}

	// get token info from map
	// setUserTokenInfo(chatId, tokenInfo)
	session.GetSessionManager().Set(chatId, session.UserLastSelectTokenCache, &tokenInfo)

	// setting button callbackData
	buyCallBackData := fmt.Sprintf(BUY_BUTTON, tokenInfo.Data.PairAddress, tokenInfo.Data.ChainCode)
	sellCallBackData := fmt.Sprintf(SELL_BUTTON, tokenInfo.Data.PairAddress, tokenInfo.Data.ChainCode)

	kb := util.BuySellKeyBoard(util.BuySellKeyBoardData{
		PoolAddress:         tokenInfo.Data.PairAddress,
		BuyCallBackData:     buyCallBackData,
		SellCallBackData:    sellCallBackData,
		BaseToken:           tokenInfo.Data.BaseToken.Symbol,
		BaseTokenChainCode:  tokenInfo.Data.BaseToken.ChainCode,
		BaseTokenAddress:    tokenInfo.Data.BaseToken.Address,
		QuoteToken:          tokenInfo.Data.QuoteToken.Symbol,
		QuoteTokenChainCode: tokenInfo.Data.QuoteToken.ChainCode,
		QuoteTokenAddress:   tokenInfo.Data.QuoteToken.Address,
	})

	// check PairAddress
	if tokenInfo.Data.BaseToken.Address == "" {
		// util.QuickMessage(ctx, b, chatId, "输入的代币合约不正确，无法快速买入")
		return
	}

	session.GetSessionManager().Set(chatId, session.UserSelectTokenAddressCache, tokenInfo.Data.BaseToken.Address)

	textTemplate, err := template.RanderTokenInfo(tokenInfo)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
		return
	}
	store.BotMessageAdd()
	message, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        textTemplate,
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	messageWrap := model.NewMessageWrap(chatId, *message, tokenInfo)

	// clean session data
	defer func() {
		sm := session.GetSessionManager()
		sm.Set(chatId, session.UserLastSwapMessage, messageWrap)
		sm.Delete(chatId, session.UserSelectChainCache)
		// sm.Delete(chatId, session.UserSelectWalletCache)
		// WARN: user select token
		// sm.Delete(chatId, session.UserSelectTokenAddressCache)
	}()
}

// @@@selectChain
// 这个函数会拿到 select 的chain。这个时候，需要那用户这个链的默认钱包结合token来进行操作了
func CallBackUserSelectChainForToken(ctx context.Context, b *bot.Bot, update *models.Update) {
	splitData := strings.Split(update.CallbackQuery.Data, "::")
	chatId := util.EffectId(update)

	// get UserDefaultWalletInfo
	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	selectChain := splitData[2]

	// firstWalletInSelectChain := func() model.Wallet {
	// 	// 首先检查 splitData 的长度
	// 	if len(splitData) < 3 {
	// 		return model.Wallet{}
	// 	}
	//
	// 	// 检查目标链的钱包切片
	// 	if wallets, exists := userInfo.Data.Wallets[splitData[2]]; exists {
	// 		// 确保切片非空
	// 		if len(wallets) > 0 {
	// 			log.Debug().Interface("firstWalletInSelectChain", wallets[0]).Send()
	// 			return wallets[0]
	// 		}
	// 	}
	//
	// 	return model.Wallet{}
	// }()
	dw, _, _ := callback.UserDefaultWalletInfo(userInfo)
	if dw.ChainCode != selectChain {
		CallbackSwitchWalletInChain(ctx, b, update, selectChain)
		store.UserSetState(chatId, "selectForTrade")
		return
	}

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

	tokenInfo := func() model.PositionByWalletAddress {
		v, err := api.GetPositionByWalletAddress(
			dw.Wallet,
			tokenAddress,
			dw.ChainCode,
			userInfo,
		)
		if err != nil {
			return model.PositionByWalletAddress{}
		}
		return v
	}()

	// setUserTokenInfo(chatId, tokenInfo)
	session.GetSessionManager().Set(chatId, session.UserLastSelectTokenCache, &tokenInfo)

	// setting button callbackData
	buyCallBackData := fmt.Sprintf(BUY_BUTTON, tokenInfo.Data.PairAddress, tokenInfo.Data.ChainCode)
	sellCallBackData := fmt.Sprintf(SELL_BUTTON, tokenInfo.Data.PairAddress, tokenInfo.Data.ChainCode)

	kb := util.BuySellKeyBoard(util.BuySellKeyBoardData{
		PoolAddress:         tokenInfo.Data.PairAddress,
		BuyCallBackData:     buyCallBackData,
		SellCallBackData:    sellCallBackData,
		BaseToken:           tokenInfo.Data.BaseToken.Symbol,
		BaseTokenChainCode:  tokenInfo.Data.BaseToken.ChainCode,
		BaseTokenAddress:    tokenInfo.Data.BaseToken.Address,
		QuoteToken:          tokenInfo.Data.QuoteToken.Symbol,
		QuoteTokenChainCode: tokenInfo.Data.QuoteToken.ChainCode,
		QuoteTokenAddress:   tokenInfo.Data.QuoteToken.Address,
	})
	textTemplate, err := template.RanderTokenInfo(tokenInfo)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
		return
	}
	store.BotMessageAdd()
	message, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        textTemplate,
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	messageWrap := model.NewMessageWrap(chatId, *message, tokenInfo)
	defer func() {
		sm := session.GetSessionManager()
		sm.Set(chatId, session.UserLastSwapMessage, messageWrap)
		sm.Delete(chatId, session.UserSelectChainCache)
		// sm.Delete(chatId, session.UserSelectWalletCache)
		// WARN: user select token address
		// sm.Delete(chatId, session.UserSelectTokenAddressCache)
	}()
}

func CallbackSwitchWalletInChain(ctx context.Context, b *bot.Bot, u *models.Update, chainCode string) {
	userId := util.EffectId(u)

	profile, err := api.GetUserProfile(userId)
	if err != nil {
		log.Error().Err(err).Msg("err in CallbackSwitchWalletInChain")
	}

	var ws []model.Wallet
	func() {
		for c, w := range profile.Data.Wallets {
			if chainCode == c {
				ws = w
				return
			}
		}
	}()

	if len(ws) == 0 {
		log.Debug().Msg("user wallet empty in CallbackSwitchWalletInChain")
		return
	}
	var buttons [][]models.InlineKeyboardButton
	for index, wallet := range ws {
		addr := wallet.Wallet
		if len(addr) > 10 {
			displayText := fmt.Sprintf("钱包%d %s....%s", index+1, addr[:4], addr[len(addr)-4:])
			callbackData := fmt.Sprintf("selectForTrade::%v::%s", userId, wallet.WalletId)

			button := models.InlineKeyboardButton{
				Text:         displayText,
				CallbackData: callbackData,
			}
			buttons = append(buttons, []models.InlineKeyboardButton{button})
		}
	}
	keyboard := models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	store.BotMessageAdd()
	message, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userId,
		Text:        fmt.Sprintf("你的默认钱包不是当前交易的链,请选择 %s链的钱包", chainCode),
		ReplyMarkup: keyboard,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("err in CallbackSwitchWalletInChain")
		return
	}

	_ = message
}

func MessageComfromSellectWallet(ctx context.Context, b *bot.Bot, u *models.Update) {
	callbackData := u.CallbackQuery.Data

	params := strings.Split(callbackData, "::")

	selectWalletId := params[len(params)-1]

	chatId := util.EffectId(u)
	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	var dw model.Wallet
	for _, wallets := range userInfo.Data.Wallets {
		if wallet, has := lo.Find(wallets, func(w model.Wallet) bool {
			return selectWalletId == w.WalletId
		}); has {
			dw = wallet
			break
		}
	}

	if dw.Wallet == "" {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	session.GetSessionManager().Set(chatId, session.UserSelectWalletCache, dw)
	log.Debug().Interface("userwallet", dw).Send()

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

	tokenInfo := func() model.PositionByWalletAddress {
		v, err := api.GetPositionByWalletAddress(
			dw.Wallet,
			tokenAddress,
			dw.ChainCode,
			userInfo,
		)
		if err != nil {
			return model.PositionByWalletAddress{}
		}
		return v
	}()

	// setUserTokenInfo(chatId, tokenInfo)
	session.GetSessionManager().Set(chatId, session.UserLastSelectTokenCache, &tokenInfo)

	// setting button callbackData
	buyCallBackData := fmt.Sprintf(BUY_BUTTON, tokenInfo.Data.PairAddress, tokenInfo.Data.ChainCode)
	sellCallBackData := fmt.Sprintf(SELL_BUTTON, tokenInfo.Data.PairAddress, tokenInfo.Data.ChainCode)

	kb := util.BuySellKeyBoard(util.BuySellKeyBoardData{
		PoolAddress:         tokenInfo.Data.PairAddress,
		BuyCallBackData:     buyCallBackData,
		SellCallBackData:    sellCallBackData,
		BaseToken:           tokenInfo.Data.BaseToken.Symbol,
		BaseTokenChainCode:  tokenInfo.Data.BaseToken.ChainCode,
		BaseTokenAddress:    tokenInfo.Data.BaseToken.Address,
		QuoteToken:          tokenInfo.Data.QuoteToken.Symbol,
		QuoteTokenChainCode: tokenInfo.Data.QuoteToken.ChainCode,
		QuoteTokenAddress:   tokenInfo.Data.QuoteToken.Address,
	})
	textTemplate, err := template.RanderTokenInfo(tokenInfo)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
		return
	}
	store.BotMessageAdd()
	message, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        textTemplate,
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	messageWrap := model.NewMessageWrap(chatId, *message, tokenInfo)
	defer func() {
		sm := session.GetSessionManager()
		sm.Set(chatId, session.UserLastSwapMessage, messageWrap)
		sm.Delete(chatId, session.UserSelectChainCache)
		// sm.Delete(chatId, session.UserSelectWalletCache)
		// WARN: user select token address
		// sm.Delete(chatId, session.UserSelectTokenAddressCache)
	}()
}

func quickSwap(d SwapCallbackData, ctx context.Context, b *bot.Bot, update *models.Update) {
	num := cast.ToFloat64(d.Amount)
	if d.Action == "buy" {
		processSwap(ctx, b, update, true, num)
	} else {
		processSwap(ctx, b, update, false, num)
	}
}

func BuyCallBackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	if update.CallbackQuery == nil {
		return
	}
	swapData := extractSwapData(update.CallbackQuery.Data)
	if swapData.Amount == "x" {
		reply := models.ForceReply{
			ForceReply:            true,
			InputFieldPlaceholder: "20",
		}
		symbol := func() string {
			v, ok := session.GetSessionManager().Get(chatId, session.UserLastSelectTokenCache)
			if !ok {
				log.Debug().Msg("get userTokenInfo err by tradingLock")
				return ""
			}

			tokenInfo, ok := v.(*model.PositionByWalletAddress)
			// tokenInfo, ok := getUserTokenInfo(chatId)
			if !ok {
				log.Debug().Msg("get userTokenInfo err by tradingLock")
				return ""
			}
			return tokenInfo.Data.QuoteToken.Symbol
		}()

		store.BotMessageAdd()
		message, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatId,
			Text:        fmt.Sprintf("请输入买入数量，如20则购买20 %s，输入数量后立刻购买 %s ", symbol, symbol),
			ReplyMarkup: reply,
		})
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		replayBuyMsgCache[chatId] = message.ID
		return
	} else {
		quickSwap(swapData, ctx, b, update)
	}
}

func SellCallBackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	if update.CallbackQuery == nil {
		return
	}
	swapData := extractSwapData(update.CallbackQuery.Data)
	if swapData.Amount == "y" {
		reply := models.ForceReply{
			ForceReply:            true,
			InputFieldPlaceholder: "20",
		}
		symbol := func() string {
			v, ok := session.GetSessionManager().Get(chatId, session.UserLastSelectTokenCache)
			if !ok {
				log.Debug().Msg("get userTokenInfo err by tradingLock")
				return ""
			}

			tokenInfo, ok := v.(*model.PositionByWalletAddress)
			// tokenInfo, ok := getUserTokenInfo(chatId)
			if !ok {
				log.Debug().Msg("get userTokenInfo err by tradingLock")
				return ""
			}
			return tokenInfo.Data.BaseToken.Symbol
		}()
		store.BotMessageAdd()
		message, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatId,
			Text:        fmt.Sprintf("请输入卖出数量，如20则卖出20 %s，输入数量后立刻卖出 %s ", symbol, symbol),
			ReplyMarkup: reply,
		})
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		replaySellMsgCache[chatId] = message.ID

		// WARN: 记得删除状态
		setReplaySellMsgCacheIsNum(chatId, true)
		return
	} else if swapData.Amount == "x" {
		reply := models.ForceReply{
			ForceReply:            true,
			InputFieldPlaceholder: "20",
		}
		symbol := func() string {
			v, ok := session.GetSessionManager().Get(chatId, session.UserLastSelectTokenCache)
			if !ok {
				log.Debug().Msg("get userTokenInfo err by tradingLock")
				return ""
			}

			tokenInfo, ok := v.(*model.PositionByWalletAddress)
			// tokenInfo, ok := getUserTokenInfo(chatId)
			if !ok {
				log.Debug().Msg("get userTokenInfo err by tradingLock")
				return ""
			}
			return tokenInfo.Data.BaseToken.Symbol
		}()
		store.BotMessageAdd()
		message, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatId,
			Text:        fmt.Sprintf("请输入卖出百分比，如20则卖出20%% %s，输入数量后立刻卖出 %s ", symbol, symbol),
			ReplyMarkup: reply,
		})
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		replaySellMsgCache[chatId] = message.ID
		return
	} else {
		quickSwap(swapData, ctx, b, update)
	}
}

func TransferToCallBack(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	if update.CallbackQuery == nil {
		return
	}

	tokenAddress, ok := strings.CutPrefix(update.CallbackQuery.Data, "tx_")
	if !ok {
		return
	}

	reply := models.ForceReply{
		ForceReply:            true,
		InputFieldPlaceholder: "10",
	}
	symbol := func() string {
		v, ok := session.GetSessionManager().Get(chatId, session.UserLastSelectTokenCache)
		if !ok {
			log.Debug().Msg("get userTokenInfo err by tradingLock")
			return ""
		}

		tokenInfo, ok := v.(*model.PositionByWalletAddress)
		// tokenInfo, ok := getUserTokenInfo(chatId)
		if !ok {
			log.Debug().Msg("get userTokenInfo err by tradingLock")
			return ""
		}
		return tokenInfo.Data.BaseToken.Symbol
	}()

	// set transferTo data
	transfer := api.TransferTo{
		TokenAddress: tokenAddress,
	}
	store.BotMessageAdd()
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        fmt.Sprintf("请输入转出 %s 数量", symbol),
		ReplyMarkup: reply,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	session.GetSessionManager().Set(chatId, session.UserSessionState, session.TransferToState)
	session.GetSessionManager().Set(chatId, session.UserInTransferToCache, &transfer)
}

const (
	BUY        string  = "0"
	SELL       string  = "1"
	TRADE_TYPE string  = "M"
	PROFITFLAG float64 = 0
)

func processSwap(ctx context.Context, b *bot.Bot, update *models.Update, isBuy bool, numberHandle float64) {
	chatId := util.EffectId(update)
	v, ok := session.GetSessionManager().Get(chatId, session.UserLastSelectTokenCache)
	if !ok {
		log.Debug().Msg("get userTokenInfo err by tradingLock")
		return
	}

	tokenInfo, ok := v.(*model.PositionByWalletAddress)
	// tokenInfo, ok := getUserTokenInfo(chatId)
	if !ok {
		log.Debug().Msg("get userTokenInfo err by tradingLock")
		return
	}

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Debug().Msg("get GetUserProfile err in trading")
		return
	}

	var swap model.Swap
	swap.Price = tokenInfo.Data.Price
	swap.Slippage = userInfo.Data.Slippage
	swap.TradeType = TRADE_TYPE
	swap.ProfitFlag = PROFITFLAG

	baseToken := tokenInfo.Data.BaseToken
	quoteToken := tokenInfo.Data.QuoteToken

	// If buying base token
	if isBuy {
		// When buying base token, we're selling quote token
		swap.FromTokenAddress = quoteToken.Address
		swap.FromTokenDecimals = cast.ToInt(quoteToken.Decimals)
		swap.ToTokenAddress = baseToken.Address
		swap.ToTokenDecimals = cast.ToInt(baseToken.Decimals)
		swap.Type = BUY
	} else {
		// When selling base token, we're selling base token for quote token
		swap.FromTokenAddress = baseToken.Address
		swap.FromTokenDecimals = cast.ToInt(baseToken.Decimals)
		swap.ToTokenAddress = quoteToken.Address
		swap.ToTokenDecimals = cast.ToInt(quoteToken.Decimals)
		swap.Type = SELL
	}

	// check if come from select
	wallet, _ := func() (w *model.Wallet, comeFromSelect bool) {
		sM := session.GetSessionManager()
		value, has := sM.Get(chatId, session.UserSelectWalletCache)
		if has {
			w, ok := value.(model.Wallet)
			if ok {
				log.Debug().Interface("user wallet cache ", w).Msg("hit message")
				return &w, true
			}
		}

		dW, _, _ := callback.UserDefaultWalletInfo(userInfo)
		return &dW, false
	}()
	// wallets := api.ListUserDefaultWalletsSwitch(userInfo, "SOLANA")
	// selectedWallet[userId] = &dW
	// wallet := selectedWallet[userId]
	if wallet == nil {
		store.BotMessageAdd()
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatId,
			Text:   "请选择钱包",
		})
		return
	}

	// setting payer
	swap.WalletId = wallet.WalletId
	swap.WalletKey = wallet.WalletKey

	// setting decimals
	func() {
		switch swap.FromTokenAddress {
		case tokenInfo.Data.QuoteToken.Address:
			swap.FromTokenDecimals = cast.ToInt(tokenInfo.Data.QuoteToken.Decimals)
		case tokenInfo.Data.BaseToken.Address:
			swap.FromTokenDecimals = cast.ToInt(tokenInfo.Data.BaseToken.Decimals)
		}

		switch swap.ToTokenAddress {
		case tokenInfo.Data.QuoteToken.Address:
			swap.ToTokenDecimals = cast.ToInt(tokenInfo.Data.QuoteToken.Decimals)
		case tokenInfo.Data.BaseToken.Address:
			swap.ToTokenDecimals = cast.ToInt(tokenInfo.Data.BaseToken.Decimals)
		}
	}()

	// TestPrint
	log.Debug().Func(func(e *zerolog.Event) {
		for _, v := range userInfo.Data.Wallets {
			for _, w := range v {
				if w.WalletKey == swap.WalletKey {
					logger.WithTxCategory(e).
						Str("wallet", w.Wallet).
						Str("uuid", userInfo.Data.UUID)
				}
			}
		}
	}).Msg("swap trading userInfo")

	// setting swap.Amount
	userInputAmount := cast.ToString(numberHandle)
	insufficientBalance := false
	func() {
		if isBuy {
			amount := decimal.NewFromFloat(numberHandle).Mul(
				decimal.New(1, cast.ToInt32(swap.FromTokenDecimals)),
			)
			swap.Amount = amount.String()
		} else {
			percentage := decimal.NewFromFloat(numberHandle).Div(decimal.NewFromInt(100))
			tokenInfo, err := api.GetTokenInfoByWalletAddress(
				swap.FromTokenAddress,
				wallet.Wallet,
				wallet.ChainCode,
				userInfo,
			)

			log.Debug().Func(func(e *zerolog.Event) {
				logger.WithTxCategory(e).Interface("tokeninfo in swap", tokenInfo).Send()
			})

			if err != nil {
				log.Error().Err(err).Send()
				util.QuickMessage(ctx, b, chatId, "出错了，请联系客服！")
			}

			hasAmount, _ := decimal.NewFromString(tokenInfo.Amount)
			amount := hasAmount.Mul(percentage)
			swap.Amount = util.CutPointRight(amount.String())

			// WARN: the input is already shiftleft
			// user input amount str not raw amount
			userInputAmount = util.ShiftLeftStr(swap.Amount, tokenInfo.Decimals)

			// if percentage == 100% sell All
			if numberHandle == 100 {
				swap.Amount = tokenInfo.Amount
			}

			// if numberHandle is num y
			if isNum, ok := getReplaySellMsgCacheIsNum(chatId); ok && isNum {
				amount := decimal.NewFromFloat(numberHandle).Mul(
					decimal.New(1, cast.ToInt32(swap.FromTokenDecimals)),
				)
				swap.Amount = amount.String()

				// check sell num not greater then balance
				balance, _ := decimal.NewFromString(tokenInfo.Amount)
				if balance.LessThan(amount) {
					insufficientBalance = true
				}
			}

			defer setReplaySellMsgCacheIsNum(chatId, false)
		}
	}()

	// WARN: 无法知道是否能够覆盖这个 swap 的手续费问题
	//  sell       not enough balance
	if !isBuy && insufficientBalance {
		util.QuickMessage(ctx, b, chatId, "你卖出余额不足！")
		return
	}
	// WARN:
	msgqq := func() string {
		if isBuy {
			return fmt.Sprintf("🚀 %s 买 %s %s，正在交易中", baseToken.Symbol, userInputAmount, quoteToken.Symbol)
		}
		return fmt.Sprintf("🚀 %s 卖 %s %s，正在交易中", baseToken.Symbol, userInputAmount, baseToken.Symbol)
	}
	util.QuickMessage(ctx, b, chatId, msgqq())

	sp := queue.SwapPayload{
		B:               b,
		SwapBody:        swap,
		UserInfo:        userInfo,
		UserID:          chatId,
		HandleWallet:    *wallet,
		BaseToken:       baseToken,
		QuoteToken:      quoteToken,
		UserInputAmount: userInputAmount,
	}

	sm := session.GetSessionManager()
	sm.Delete(chatId, session.UserSelectWalletCache)
	queue.AddProcessingSwapQueue(&sp)
}

func processTransferTo(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	// 如果不是回复消息则不处理
	if update.Message.ReplyToMessage == nil {
		session.GetSessionManager().Delete(chatId, session.UserInTransferToCache)
		return
	}

	// tokenInfo := lastTokenSelect[chatId]
	v, ok := session.GetSessionManager().Get(chatId, session.UserLastSelectTokenCache)
	if !ok {
		log.Debug().Msg("get userTokenInfo err by tradingLock")
		return
	}

	tokenInfo, ok := v.(*model.PositionByWalletAddress)
	// tokenInfo, ok := getUserTokenInfo(chatId)
	if !ok {
		log.Debug().Msg("get userTokenInfo err in transferToState trigger by tradingLock")
		return
	}
	if v, exists := session.GetSessionManager().Get(chatId, session.UserInTransferToCache); exists {
		transfer, ok := v.(*api.TransferTo)
		if !ok {
			session.GetSessionManager().Delete(chatId, session.UserInTransferToCache)
			return
		}

		// 先处理金额
		if !transfer.IsAmountSet() {
			f, err := cast.ToFloat64E(update.Message.Text)
			if err != nil {
				util.QuickMessage(ctx, b, chatId, "数量不能是 0 或负数，请重新转账")
				return
			}
			if f < 0 {
				util.QuickMessage(ctx, b, chatId, "数量不能是 0 或负数，请重新转账")
				return
			}
			amount := util.ShiftRightStr(update.Message.Text, tokenInfo.Data.BaseToken.Decimals)
			transfer.RawAmount = amount
			// 金额设置完后，请求输入地址
			reply := models.ForceReply{
				ForceReply:            true,
				InputFieldPlaceholder: "xxxxxxxxxxxxxx",
			}
			store.BotMessageAdd()
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatId,
				Text:        "请输入接收地址：",
				ReplyMarkup: reply,
			})
			// 保存更新后的 transfer
			session.GetSessionManager().Set(chatId, session.UserInTransferToCache, transfer)
			return
		}

		// 如果金额已设置，则处理地址
		if !transfer.IsToAddressSet() {
			address := update.Message.Text

			isSolana, err := util.CheckValidAddress(address)
			if err != nil {

				// util.QuickMessage(ctx, b, chatId, "接收钱包格式不正确，请检查后重新点击转出")
				line := []models.InlineKeyboardButton{
					{
						Text:         "🔴转出",
						CallbackData: "tx_" + tokenInfo.Data.BaseToken.Address,
					},
				}
				keyboard := [][]models.InlineKeyboardButton{line}
				store.BotMessageAdd()
				message, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatId,
					Text:   "接收钱包格式不正确，请检查后重新点击转出",
					ReplyMarkup: models.InlineKeyboardMarkup{
						InlineKeyboard: keyboard,
					},
				})

				messageWrap := model.NewMessageWrap(chatId, *message, *tokenInfo)
				sm := session.GetSessionManager()
				sm.Set(chatId, session.UserLastSwapMessage, messageWrap)
				if err != nil {
					log.Debug().Err(err).Send()
				}
				return
			}

			if isSolana {
				transfer.ToAddress = update.Message.Text
				// check valid evm address
			} else if common.IsHexAddress(address) {
				transfer.ToAddress = update.Message.Text
			} else {
				// util.QuickMessage(ctx, b, chatId, "接收钱包格式不正确，请检查后重新点击转出")
				line := []models.InlineKeyboardButton{
					{
						Text:         "🔴转出",
						CallbackData: "tx_" + tokenInfo.Data.BaseToken.Address,
					},
				}
				keyboard := [][]models.InlineKeyboardButton{line}
				store.BotMessageAdd()
				message, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatId,
					Text:   "接收钱包格式不正确，请检查后重新点击转出",
					ReplyMarkup: models.InlineKeyboardMarkup{
						InlineKeyboard: keyboard,
					},
				})

				messageWrap := model.NewMessageWrap(chatId, *message, *tokenInfo)
				sm := session.GetSessionManager()
				sm.Set(chatId, session.UserLastSwapMessage, messageWrap)
				if err != nil {
					log.Debug().Err(err).Send()
				}
				return
			}

		}

		userInfo, err := api.GetUserProfile(chatId)
		if err != nil {
			log.Error().Err(err).Send()
			util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
			return
		}

		dw, _, _ := callback.UserDefaultWalletInfo(userInfo)
		if dw.ChainCode != tokenInfo.Data.ChainCode {
			log.Debug().Msg("chainCode not match")
			value, has := session.GetSessionManager().Get(chatId, session.UserSelectWalletCache)
			if has {
				wallet, ok := value.(model.Wallet)
				if ok {
					dw = wallet
				}
			}
		}
		transfer.UserInfo = userInfo
		transfer.WalletId = dw.WalletId
		transfer.WalletKey = dw.WalletKey

		// check balance
		from := cast.ToFloat64(transfer.RawAmount)
		has := cast.ToFloat64(tokenInfo.Data.RawAmount)
		fromView := util.ShiftLeftStr(transfer.RawAmount, cast.ToString(tokenInfo.Data.BaseToken.Decimals))
		hasView := util.ShiftLeftStr(tokenInfo.Data.RawAmount, tokenInfo.Data.BaseToken.Decimals)
		if from > has {
			msg := fmt.Sprintf("%s余额不足，余额：%s，转出数量：%s", tokenInfo.Data.BaseToken.Symbol, hasView, fromView)
			util.QuickMessage(ctx, b, chatId, msg)
			return
		}

		util.QuickMessage(ctx, b, chatId, "正在转出中")

		tx, err := transfer.Send()
		if err != nil {
			log.Error().Err(err).Send()
			if errors.Is(err, api.ErrTransferToAmount) {
				util.QuickMessage(ctx, b, chatId, err.Error())
				return
			} else if errors.Is(err, api.ErrTransferFail) {
				util.QuickMessage(ctx, b, chatId, err.Error())
				return
			}
			// util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
			errMsg := fmt.Sprintf("出错了，%s", util.AdminUrl)
			util.QuickMessage(ctx, b, chatId, errMsg)
			return
		}

		// PollTransactionStatus
		msg := fmt.Sprintf("交易hash：\n <code>%s</code>", tx)
		chainCode := func() string {
			for _, w := range userInfo.Data.Wallets {
				for _, wallet := range w {
					if transfer.WalletId == wallet.WalletId {
						log.Debug().Interface("transfer wallet", wallet).Send()
						return wallet.ChainCode
					}
				}
			}
			return ""
		}()
		scanUrl := util.GetChainScanUrl(chainCode, tx)
		button := util.UrlButton("点击打开区块浏览器", scanUrl)
		util.QuickMessageWithButton(ctx, b, chatId, msg, button)

		util.QuickMessage(ctx, b, chatId, "链上确认中")

		go func() {
			if chainCode == "" {
				log.Error().Err(errors.New("get user wallet chainCode err in transferTo")).Send()
				util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
				return
			}
			err := rpc.PollTransactionStatus(chainCode, tx)
			// err := rpc.SOL_PollTransactionStatus(tx)
			if err != nil {
				if errors.Is(err, rpc.ErrPollTxMaxRetry) {
					util.QuickMessage(ctx, b, chatId, err.Error())
				}
				return
			}
			util.QuickMessage(ctx, b, chatId, "交易成功")
		}()
		return
	}

	session.GetSessionManager().Delete(chatId, session.UserInTransferToCache)
	session.GetSessionManager().Delete(chatId, session.UserSelectWalletCache)
}

func processWithdrawl(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	// 如果不是回复消息则不处理
	if update.Message.ReplyToMessage == nil {
		store.UserDeleteCommissionInfo(chatId)
		return
	}

	v, ok := store.UserGetCommissionInfo(chatId)
	if !ok {
		log.Debug().Msg("user not in withdrawal")
		return
	}

	subReq := map[string]string{}
	err := json.Unmarshal(v, &subReq)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	cmInfo, err := api.GetMyCommissionSummary(userInfo)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}
	withdrawableCommissionAmount := gjson.GetBytes(cmInfo, "data.withdrawableCommissionAmount").String()
	if subReq["amount"] == "" {
		f, err := cast.ToFloat64E(update.Message.Text)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, "目标价格不能是 0 或负数，请重新提现")
			store.UserDeleteCommissionInfo(chatId)
			return
		}
		if f < 0 {
			util.QuickMessage(ctx, b, chatId, "目标价格不能是 0 或负数，请重新提现")
			store.UserDeleteCommissionInfo(chatId)
			return
		}
		if f < 10 {
			util.QuickMessage(ctx, b, chatId, "最少提现 10 U，请重新提现")
			store.UserDeleteCommissionInfo(chatId)
			return
		}
		if f > cast.ToFloat64(withdrawableCommissionAmount) {
			store.UserDeleteCommissionInfo(chatId)
			util.QuickMessage(ctx, b, chatId, "你的可提现余额不足，请重新提现")
			return
		}

		subReq["amount"] = update.Message.Text
		// 金额设置完后，请求输入地址
		reply := models.ForceReply{
			ForceReply:            true,
			InputFieldPlaceholder: "xxxxxxxxxxxxxx",
		}
		store.BotMessageAdd()
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatId,
			Text:        "请输入接收地址：",
			ReplyMarkup: reply,
		})
		// 保存更新后的 transfer
		ddn, err := json.Marshal(subReq)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}
		store.UserSetCommissionInfo(chatId, ddn)
		return
	}

	if subReq["walletAddress"] == "" {
		address := update.Message.Text
		_, err := util.CheckValidAddress(address)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, "地址不正确,无法完成提现,请重新提现")
			store.UserDeleteCommissionInfo(chatId)
			return
		}
		subReq["walletAddress"] = address
		ddn, err := json.Marshal(subReq)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			store.UserDeleteCommissionInfo(chatId)
			return
		}
		store.UserSetCommissionInfo(chatId, ddn)

		text := `
提现金额: $%s
提现网络: %s
提现地址: <code>%s</code>
到账时间: 每天晚上审核通过后
		`
		kb := util.WithdrawalKeyBoard()

		store.BotMessageAdd()
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:        fmt.Sprintf(text, subReq["amount"], api.GetChainNameFallbackCode(subReq["chainCode"]), subReq["walletAddress"]),
			ChatID:      chatId,
			ReplyMarkup: kb,
			ParseMode:   "HTML",
		})
	}
}

func processAImonitor(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	if update.Message.ReplyToMessage == nil {
		store.UserDeleteAiMonitorInfo(chatId)
		return
	}

	v, ok := store.UserGetAiMonitorInfo(chatId)
	if !ok {
		log.Debug().Msg("user not in ai monitor")
		return
	}

	subReq := &model.AISubscribeReqData{}
	err := json.Unmarshal(v, subReq)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	if subReq.BaseAddress == "" {
		subReq.BaseAddress = update.Message.Text

		_, err := util.CheckValidAddress(update.Message.Text)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, "输入的代币合约不正确")
			return
		}

		userInfo, err := api.GetUserProfile(chatId)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}

		monitorType := subReq.MonitorType
		result, hasSubScribe, err := api.GetUserTokenSubscribe(chatId, subReq.ChainCode, subReq.BaseAddress, monitorType, userInfo)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}

		templ := `
%s
当前价格:$%s
		`

		log.Debug().RawJSON("getUserTokenSubscribe info", result).Bool("has", hasSubScribe).Send()

		if hasSubScribe {

			monitorType := gjson.GetBytes(result, "data.subscribe.type").String()
			freq_Map := map[int64]string{
				1: "once",
				2: "daily",
				3: "every",
			}

			byteStr := gjson.GetBytes(result, "data.subscribe").Raw

			var subTokenInfo model.TokenSubscribeInfo
			err = json.Unmarshal([]byte(byteStr), &subTokenInfo)
			if err != nil {
				log.Error().Err(err).Send()
			}
			keyBoardData := util.SettingsKeyBoardData{
				EnableTG:  lo.Contains(userInfo.Data.SubscribeSetting, "telegram"),
				EnableWeb: lo.Contains(userInfo.Data.SubscribeSetting, "web"),
				EnableApp: lo.Contains(userInfo.Data.SubscribeSetting, "app"),
				Frequency: freq_Map[subTokenInfo.NoticeType],
			}
			kb := util.AiMonitor_EditSettingsKeyBoard(keyBoardData, monitorType, int(subTokenInfo.NoticeType))
			tmplll := func() string {
				switch monitorType {
				case "price":
					return `
HelloDex: AI监控
当前价格: $%s
目标价格: $%s
						`
				case "chg":
					return `
HelloDex: AI监控
目标涨跌幅: %s
						`
				case "buy":
					return `
HelloDex: AI监控
买入交易额: %s
						`
				case "sell":
					return `
HelloDex: AI监控
卖出交易额: %s
						`
				}
				return ""
			}()
			aiMonitoryInfoByte, has := store.UserGetAiMonitorInfo(chatId)
			if !has {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}

			var subReq model.AISubscribeReqData

			err := json.Unmarshal(aiMonitoryInfoByte, &subReq)
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}

			subReq.UserId = chatId
			subReq.CurrentPrice = subTokenInfo.StartPrice
			subReq.ChainCode = subTokenInfo.ChainCode
			subReq.BaseAddress = subTokenInfo.BaseAddress
			subReq.Symbol = subTokenInfo.Symbol
			subReq.NoticeType = int(subTokenInfo.NoticeType)
			subReq.TargetPrice = subTokenInfo.TargetPrice
			subReq.Data = subTokenInfo.Data
			currentPrice := util.FormatNumber(subTokenInfo.StartPrice)
			targetPrice := util.FormatNumber(subTokenInfo.TargetPrice)

			if monitorType == "price" {
				sendParams := &bot.SendMessageParams{
					ChatID:      chatId,
					Text:        fmt.Sprintf(tmplll, currentPrice, targetPrice),
					ReplyMarkup: kb,
				}
				store.BotMessageAdd()
				msg, err := b.SendMessage(ctx, sendParams)
				if err != nil {
					log.Error().Err(err).Send()
					util.QuickMessage(ctx, b, chatId, "出错了,请联系客服")
					return
				}
				subReq.SessionMessageID = msg.ID
				dataSendParams, err := json.Marshal(sendParams)
				if err != nil {
					log.Error().Err(err).Send()
					util.QuickMessage(ctx, b, chatId, "出错了,请联系客服")
					return
				}
				store.RedisSetSendMessageParams(chatId, msg.ID, dataSendParams)
			} else {
				sendParams := &bot.SendMessageParams{
					ChatID:      chatId,
					Text:        fmt.Sprintf(tmplll, subReq.Data),
					ReplyMarkup: kb,
				}
				store.BotMessageAdd()
				msg, err := b.SendMessage(ctx, sendParams)
				if err != nil {
					log.Error().Err(err).Send()
					util.QuickMessage(ctx, b, chatId, "出错了,请联系客服")
					return
				}
				subReq.SessionMessageID = msg.ID
				dataSendParams, err := json.Marshal(sendParams)
				if err != nil {
					log.Error().Err(err).Send()
					util.QuickMessage(ctx, b, chatId, "出错了,请联系客服")
					return
				}
				store.RedisSetSendMessageParams(chatId, msg.ID, dataSendParams)
			}
			log.Debug().Msg("user already subscribe token")
			newData, err := subReq.JsonB()
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}
			store.UserSetAiMonitorInfo(chatId, newData)
			return
		}

		// don't has subscribe
		many := gjson.GetManyBytes(result, "data.info.baseToken.symbol", "data.info.baseToken.price")
		symbol := many[0].String()
		price := util.FormatNumber(many[1].String())

		// set symbol
		subReq.Symbol = symbol
		subReq.CurrentPrice = price

		var reply models.ForceReply
		switch subReq.MonitorType {
		case "price":
			reply = models.ForceReply{
				ForceReply:            true,
				InputFieldPlaceholder: "请输入目标价格，到达后将会推送消息",
			}
		case "chg":
			reply = models.ForceReply{
				ForceReply:            true,
				InputFieldPlaceholder: "请输入涨跌幅，可以是负数，到达后将会推送消息",
			}

		case "buy":
			reply = models.ForceReply{
				ForceReply:            true,
				InputFieldPlaceholder: "请输入交易额，单笔买入触达后 会推送消息",
			}
		case "sell":
			reply = models.ForceReply{
				ForceReply:            true,
				InputFieldPlaceholder: "请输入交易额，单笔卖出触达后 会推送消息",
			}
		}

		store.BotMessageAdd()
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatId,
			Text:        fmt.Sprintf(templ, symbol, price),
			ReplyMarkup: reply,
		})
		// 保存更新后的
		newData, err := subReq.JsonB()
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}

		store.UserSetAiMonitorInfo(chatId, newData)
		return
	}

	switch subReq.MonitorType {
	// handle price type
	case "price":
		if subReq.TargetPrice == "" {
			f, err := cast.ToFloat64E(update.Message.Text)
			if err != nil {
				util.QuickMessage(ctx, b, chatId, "目标价格不能是 0 或负数，请重新添加监听")
				return
			}
			if f < 0 {
				util.QuickMessage(ctx, b, chatId, "目标价格不能是 0 或负数，请重新添加监听")
				return
			}

			subReq.TargetPrice = update.Message.Text
			subReq.NoticeType = 1
			subReq.MonitorType = "price"
			subReq.UserId = chatId

			userInfo, err := api.GetUserProfile(chatId)
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}

			keyBoardData := util.SettingsKeyBoardData{
				EnableTG:  true,
				EnableWeb: lo.Contains(userInfo.Data.SubscribeSetting, "web"),
				EnableApp: lo.Contains(userInfo.Data.SubscribeSetting, "app"),
				Frequency: "once", // for default
			}
			kb := util.AiMonitorSettingsKeyBoard(keyBoardData)

			tmplll := `
HelloDex: AI监控
当前价格: $%s
目标价格: $%s
		`

			store.BotMessageAdd()
			msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatId,
				Text:        fmt.Sprintf(tmplll, subReq.CurrentPrice, subReq.TargetPrice),
				ReplyMarkup: kb,
			})
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}
			subReq.SessionMessageID = msg.ID
			newData, err := subReq.JsonB()
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}
			store.UserSetAiMonitorInfo(chatId, newData)
			return
		}
	// handle chg type
	case "chg":
		if subReq.Data == "" {
			f, err := cast.ToFloat64E(update.Message.Text)
			if err != nil {
				util.QuickMessage(ctx, b, chatId, "目标涨幅不能是 0 或负数，请重新添加监听")
				return
			}
			if f < 0 {
				util.QuickMessage(ctx, b, chatId, "目标涨幅不能是 0 或负数，请重新添加监听")
				return
			}

			subReq.Data = update.Message.Text
			subReq.NoticeType = 1
			subReq.MonitorType = "chg"
			subReq.UserId = chatId

			userInfo, err := api.GetUserProfile(chatId)
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}

			keyBoardData := util.SettingsKeyBoardData{
				EnableTG:  true,
				EnableWeb: lo.Contains(userInfo.Data.SubscribeSetting, "web"),
				EnableApp: lo.Contains(userInfo.Data.SubscribeSetting, "app"),
				Frequency: "once", // for default
			}
			kb := util.AiMonitorSettingsKeyBoard(keyBoardData)

			tmplll := `
HelloDex: AI监控
当前价格: $%s
目标涨跌幅: %s
		`

			store.BotMessageAdd()
			msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatId,
				Text:        fmt.Sprintf(tmplll, subReq.CurrentPrice, subReq.Data),
				ReplyMarkup: kb,
			})
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}
			subReq.SessionMessageID = msg.ID
			newData, err := subReq.JsonB()
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}
			store.UserSetAiMonitorInfo(chatId, newData)
			return
		}
	case "buy":
		if subReq.Data == "" {
			f, err := cast.ToFloat64E(update.Message.Text)
			if err != nil {
				util.QuickMessage(ctx, b, chatId, "目标交易额不能是 0 或负数，请重新添加监听")
				return
			}
			if f < 0 {
				util.QuickMessage(ctx, b, chatId, "目标交易额不能是 0 或负数，请重新添加监听")
				return
			}

			subReq.Data = update.Message.Text
			subReq.NoticeType = 1
			subReq.MonitorType = "buy"
			subReq.UserId = chatId

			userInfo, err := api.GetUserProfile(chatId)
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}

			keyBoardData := util.SettingsKeyBoardData{
				EnableTG:  true,
				EnableWeb: lo.Contains(userInfo.Data.SubscribeSetting, "web"),
				EnableApp: lo.Contains(userInfo.Data.SubscribeSetting, "app"),
				Frequency: "once", // for default
			}
			kb := util.AiMonitorSettingsKeyBoard(keyBoardData)

			tmplll := `
HelloDex: AI监控
当前价格: $%s
买入交易额: %s
		`

			store.BotMessageAdd()
			msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatId,
				Text:        fmt.Sprintf(tmplll, subReq.CurrentPrice, subReq.Data),
				ReplyMarkup: kb,
			})
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}
			subReq.SessionMessageID = msg.ID
			newData, err := subReq.JsonB()
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}
			store.UserSetAiMonitorInfo(chatId, newData)
			return
		}
	case "sell":
		if subReq.Data == "" {
			f, err := cast.ToFloat64E(update.Message.Text)
			if err != nil {
				util.QuickMessage(ctx, b, chatId, "目标交易额不能是 0 或负数，请重新添加监听")
				return
			}
			if f < 0 {
				util.QuickMessage(ctx, b, chatId, "目标交易额不能是 0 或负数，请重新添加监听")
				return
			}

			subReq.Data = update.Message.Text
			subReq.NoticeType = 1
			subReq.MonitorType = "sell"
			subReq.UserId = chatId

			userInfo, err := api.GetUserProfile(chatId)
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}

			keyBoardData := util.SettingsKeyBoardData{
				EnableTG:  true,
				EnableWeb: lo.Contains(userInfo.Data.SubscribeSetting, "web"),
				EnableApp: lo.Contains(userInfo.Data.SubscribeSetting, "app"),
				Frequency: "once", // for default
			}
			kb := util.AiMonitorSettingsKeyBoard(keyBoardData)

			tmplll := `
HelloDex: AI监控
当前价格: $%s
卖出交易额: %s
		`

			store.BotMessageAdd()
			msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatId,
				Text:        fmt.Sprintf(tmplll, subReq.CurrentPrice, subReq.Data),
				ReplyMarkup: kb,
			})
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}
			subReq.SessionMessageID = msg.ID
			newData, err := subReq.JsonB()
			if err != nil {
				util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
				return
			}
			store.UserSetAiMonitorInfo(chatId, newData)
			return
		}
	}
}

type SwapCallbackData struct {
	Action   string // "buy" or "sell"
	PairAddr string
	Chain    string
	Amount   string
}

func extractSwapData(callbackData string) SwapCallbackData {
	parts := strings.Split(callbackData, "_")
	if len(parts) < 4 {
		return SwapCallbackData{}
	}

	return SwapCallbackData{
		Action:   parts[0],
		PairAddr: parts[1],
		Chain:    parts[2],
		Amount:   parts[3],
	}
}

func CallbackLimitOrder(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	kb := util.LimitOrderKeyBoard()
	store.BotMessageAdd()
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        "请选择挂单类型：\n\n选择后进行输入数量即可",
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
}

func ConfirmLimitOrder(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	callbackData := update.CallbackQuery.Data
	list := strings.Split(callbackData, "_")
	action, limitType := list[1], list[2]

	tokenInfo := func() *model.PositionByWalletAddress {
		v, has := session.GetSessionManager().Get(chatId, session.UserLastSelectTokenCache)
		if has {
			tokenInfoInner, ok := v.(*model.PositionByWalletAddress)
			if ok {
				return tokenInfoInner
			}
		}
		return nil
	}()
	if tokenInfo == nil {
		util.QuickMessage(ctx, b, chatId, "出错了，请联系客服！")
		return
	}
	textTemplate := `
当前 %s 币 价格 $%s
请输入%s%s
	`

	text := fmt.Sprintf(
		textTemplate,
		tokenInfo.Data.BaseToken.Symbol,
		util.FormatNumber(tokenInfo.Data.Price),
		util.GetLimitOrderPrefixText(callbackData),
		"价格($)",
	)

	log.Debug().Str("action", action).Str("limitType", limitType).Msg("user ConfirmLimitOrder")

	reply := models.ForceReply{
		ForceReply:            true,
		InputFieldPlaceholder: "20",
	}
	store.BotMessageAdd()
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        text,
		ReplyMarkup: reply,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	order := &api.Order{
		LimitOrderType: cast.ToInt(limitType),
		OrderType: func() int {
			if action == "buy" {
				return 0
			} else {
				return 1
			}
		}(),
	}
	session.GetSessionManager().Set(chatId, session.UserSessionState, session.LimitOrderState)
	session.GetSessionManager().Set(chatId, session.UserInLimitOrderCache, order)
}

func processLimitOrder(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	// 如果不是回复消息则不处理
	if update.Message.ReplyToMessage == nil {
		session.GetSessionManager().Delete(chatId, session.UserInLimitOrderCache)
		return
	}
	// tokenInfo := lastTokenSelect[chatId]
	v, ok := session.GetSessionManager().Get(chatId, session.UserLastSelectTokenCache)
	if !ok {
		log.Debug().Msg("get userTokenInfo err by tradingLock")
		return
	}

	tokenInfo, ok := v.(*model.PositionByWalletAddress)
	// tokenInfo, ok := getUserTokenInfo(chatId)
	if !ok {
		log.Debug().Msg("get userTokenInfo err by tradingLock")
		return
	}

	if v, exists := session.GetSessionManager().Get(chatId, session.UserInLimitOrderCache); exists {
		order, ok := v.(*api.Order)
		if !ok {
			session.GetSessionManager().Delete(chatId, session.UserInLimitOrderCache)
			return
		}
		// 先处理金额 ($)
		if !order.IsTargetPriceSet() {
			targetPrice := update.Message.Text
			f, err := cast.ToFloat64E(targetPrice)
			if err != nil {
				util.QuickMessage(ctx, b, chatId, "金额不能是 0 或负数，请重新转账")
				return
			}
			if f < 0 {
				util.QuickMessage(ctx, b, chatId, "金额不能是 0 或负数，请重新转账")
				return
			}
			order.TargetPrice = targetPrice
			reply := models.ForceReply{
				ForceReply:            true,
				InputFieldPlaceholder: "100",
			}
			store.BotMessageAdd()
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatId,
				Text:        "请输入数量",
				ReplyMarkup: reply,
			})
			// 保存更新后的 transfer
			session.GetSessionManager().Set(chatId, session.UserInLimitOrderCache, order)
			return
		}

		// 如果金额已经处理，处理数额
		if !order.IsAmountSet() {
			f, err := cast.ToFloat64E(update.Message.Text)
			if err != nil {
				util.QuickMessage(ctx, b, chatId, "数量不能是 0 或负数，请重新转账")
				return
			}
			if f < 0 {
				util.QuickMessage(ctx, b, chatId, "数量不能是 0 或负数，请重新转账")
				return
			}
			amount := util.ShiftRightStr(update.Message.Text, tokenInfo.Data.BaseToken.Decimals)
			order.FromTokenAmount = amount
		}

		userInfo, err := api.GetUserProfile(chatId)
		if err != nil {
			log.Debug().Msg("get GetUserProfile err in trading")
			return
		}

		isBuy := order.OrderType == 0
		baseToken := tokenInfo.Data.BaseToken
		quoteToken := tokenInfo.Data.QuoteToken

		// If buying base token
		if isBuy {
			if order.LimitOrderType == 2 {
				order.FromTokenAddress = quoteToken.Address
				order.FromTokenDecimals = cast.ToInt(quoteToken.Decimals)
				order.ToTokenAddress = baseToken.Address
				order.ToTokenDecimals = cast.ToInt(baseToken.Decimals)
			}
			if order.LimitOrderType == 1 {
				order.FromTokenAddress = quoteToken.Address
				order.FromTokenDecimals = cast.ToInt(quoteToken.Decimals)
				order.ToTokenAddress = baseToken.Address
				order.ToTokenDecimals = cast.ToInt(baseToken.Decimals)
			}
		} else {
			if order.LimitOrderType == 3 {
				order.FromTokenAddress = baseToken.Address
				order.FromTokenDecimals = cast.ToInt(baseToken.Decimals)
				order.ToTokenAddress = quoteToken.Address
				order.ToTokenDecimals = cast.ToInt(quoteToken.Decimals)
			}

			if order.LimitOrderType == 4 {
				order.FromTokenAddress = baseToken.Address
				order.FromTokenDecimals = cast.ToInt(baseToken.Decimals)
				order.ToTokenAddress = quoteToken.Address
				order.ToTokenDecimals = cast.ToInt(quoteToken.Decimals)
			}
		}

		dw, _, _ := callback.UserDefaultWalletInfo(userInfo)

		if dw.ChainCode != tokenInfo.Data.ChainCode {
			log.Debug().Msg("chainCode not match")
			value, has := session.GetSessionManager().Get(chatId, session.UserSelectWalletCache)
			if has {
				wallet, ok := value.(model.Wallet)
				if ok {
					dw = wallet
				}
			}
		}
		order.WalletID = dw.WalletId
		order.WalletKey = dw.WalletKey
		order.ChainCode = dw.ChainCode

		if !isBuy {
			from := cast.ToFloat64(order.FromTokenAmount)
			has := cast.ToFloat64(tokenInfo.Data.RawAmount)

			fromView := util.ShiftLeftStr(order.FromTokenAmount, cast.ToString(order.FromTokenDecimals))
			hasView := util.ShiftLeftStr(tokenInfo.Data.RawAmount, tokenInfo.Data.BaseToken.Decimals)
			if from > has {
				msg := fmt.Sprintf("%s余额不足，余额：%s，交易数量：%s", baseToken.Symbol, hasView, fromView)
				util.QuickMessage(ctx, b, chatId, msg)
				return
			}
		}

		err = order.SendOrder(userInfo)
		if err != nil {
			log.Error().Err(err).Send()
			if errors.Is(err, api.ErrNewOrder) {
				util.QuickMessage(ctx, b, chatId, api.ErrNewOrder.Error())
				return
			}
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了，%s", util.AdminUrl))
			return
		} else {
			util.QuickMessage(ctx, b, chatId, "挂单成功")
			return
		}
	}
	session.GetSessionManager().Delete(chatId, session.UserInLimitOrderCache)
	session.GetSessionManager().Delete(chatId, session.UserSelectWalletCache)
}

func processCurrentAimonitor(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatId := util.EffectId(update)
	if update.Message.ReplyToMessage == nil {
		return
	}
	state, has := store.RedisGetState(chatId, "edit_current")
	if !has {
		util.QuickMessage(ctx, b, chatId, "消息过期,请重新编辑")
		return
	}
	logger.StdLogger().Info().Str("state", state).Send()

	var subReq model.AISubscribeReqData
	dataB, exists := store.UserGetAiMonitorInfo(chatId)
	if !exists {
		util.QuickMessage(ctx, b, chatId, "出错了看日志")
		return
	}
	err := json.Unmarshal(dataB, &subReq)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, "出错了看日志")
		return
	}
	switch state {
	case "price":
		f, err := cast.ToFloat64E(update.Message.Text)
		if err != nil {
			logger.StdLogger().Error().Err(err).Send()
			util.QuickMessage(ctx, b, chatId, "目标价格不能是 0 或负数，请重新编辑监听")
			return
		}
		if f < 0 {
			logger.StdLogger().Error().Err(err).Send()
			util.QuickMessage(ctx, b, chatId, "目标价格不能是 0 或负数，请重新编辑监听")
			return
		}
		subReq.TargetPrice = update.Message.Text
		newData, err := subReq.JsonB()
		if err != nil {
			logger.StdLogger().Error().Err(err).Send()
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}
		store.UserSetAiMonitorInfo(chatId, newData)

		sendMessageByte, err := store.RedisGetSendMessageParams(chatId, subReq.SessionMessageID)
		if err != nil {
			logger.StdLogger().Error().Err(err).Send()
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}
		var sendMsg bot.SendMessageParams
		err = json.Unmarshal(sendMessageByte, &sendMsg)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			logger.StdLogger().Error().Err(err).Send()
			return
		}

		textTmpl := edit_current_tmpl(subReq.MonitorType)
		textSend := ""
		currentPrice := util.FormatNumber(subReq.CurrentPrice)
		targetPrice := util.FormatNumber(subReq.TargetPrice)
		if subReq.MonitorType == "price" {
			textSend = fmt.Sprintf(textTmpl, currentPrice, targetPrice)
		} else {
			textSend = fmt.Sprintf(textTmpl, subReq.Data)
		}

		sendMsg.Text = textSend
		store.BotMessageAdd()
		_, err = b.SendMessage(ctx, &sendMsg)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			logger.StdLogger().Error().Err(err).Send()
			return
		}

		b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
			ChatID:     chatId,
			MessageIDs: []int{subReq.SessionMessageID},
		})

	case "chg":
		subReq.Data = update.Message.Text
		newData, err := subReq.JsonB()
		if err != nil {
			logger.StdLogger().Error().Err(err).Send()
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}
		store.UserSetAiMonitorInfo(chatId, newData)

		sendMessageByte, err := store.RedisGetSendMessageParams(chatId, subReq.SessionMessageID)
		if err != nil {
			logger.StdLogger().Error().Err(err).Send()
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}
		var sendMsg bot.SendMessageParams
		err = json.Unmarshal(sendMessageByte, &sendMsg)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			logger.StdLogger().Error().Err(err).Send()
			return
		}

		textTmpl := edit_current_tmpl(subReq.MonitorType)
		textSend := ""
		currentPrice := util.FormatNumber(subReq.CurrentPrice)
		targetPrice := util.FormatNumber(subReq.TargetPrice)
		if subReq.MonitorType == "price" {
			textSend = fmt.Sprintf(textTmpl, currentPrice, targetPrice)
		} else {
			textSend = fmt.Sprintf(textTmpl, subReq.Data)
		}

		sendMsg.Text = textSend
		store.BotMessageAdd()
		_, err = b.SendMessage(ctx, &sendMsg)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			logger.StdLogger().Error().Err(err).Send()
			return
		}

		b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
			ChatID:     chatId,
			MessageIDs: []int{subReq.SessionMessageID},
		})

	case "buy", "sell":
		f, err := cast.ToFloat64E(update.Message.Text)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, "交易额不能是 0 或负数，请重新编辑监听")
			return
		}
		if f < 0 {
			util.QuickMessage(ctx, b, chatId, "交易额不能是 0 或负数，请重新编辑监听")
			return
		}
		subReq.Data = update.Message.Text
		newData, err := subReq.JsonB()
		if err != nil {
			logger.StdLogger().Error().Err(err).Send()
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}
		store.UserSetAiMonitorInfo(chatId, newData)

		sendMessageByte, err := store.RedisGetSendMessageParams(chatId, subReq.SessionMessageID)
		if err != nil {
			logger.StdLogger().Error().Err(err).Send()
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}
		var sendMsg bot.SendMessageParams
		err = json.Unmarshal(sendMessageByte, &sendMsg)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			logger.StdLogger().Error().Err(err).Send()
			return
		}

		textTmpl := edit_current_tmpl(subReq.MonitorType)
		textSend := ""
		currentPrice := util.FormatNumber(subReq.CurrentPrice)
		targetPrice := util.FormatNumber(subReq.TargetPrice)
		if subReq.MonitorType == "price" {
			textSend = fmt.Sprintf(textTmpl, currentPrice, targetPrice)
		} else {
			textSend = fmt.Sprintf(textTmpl, subReq.Data)
		}

		sendMsg.Text = textSend
		store.BotMessageAdd()
		_, err = b.SendMessage(ctx, &sendMsg)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			logger.StdLogger().Error().Err(err).Send()
			return
		}

		b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
			ChatID:     chatId,
			MessageIDs: []int{subReq.SessionMessageID},
		})
	}
}

func edit_current_tmpl(monitorType string) string {
	switch monitorType {
	case "price":
		return `
HelloDex: AI监控
当前价格: $%s
目标价格: $%s
推送设置已变动，请点击【保存更新】
						`
	case "chg":
		return `
HelloDex: AI监控
目标涨跌幅: %s
推送设置已变动，请点击【保存更新】
						`
	case "buy":
		return `
HelloDex: AI监控
买入交易额: %s
推送设置已变动，请点击【保存更新】
						`
	case "sell":
		return `
HelloDex: AI监控
卖出交易额: %s
推送设置已变动，请点击【保存更新】
						`
	}
	return ""
}
