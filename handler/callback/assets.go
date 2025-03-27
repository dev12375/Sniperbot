package callback

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/entity"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/session"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/template"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

const assetsText = `
当前钱包：<code>%s</code>
当前公链：%s

选择 Token 进行快速交易！！
`

func AssetsSelectByAddressHandler(ctx context.Context, b *bot.Bot, update *models.Update, wallet model.Wallet) {
	chatID := util.EffectId(update)

	sm := session.GetSessionManager()
	sm.Set(chatID, session.UserSelectWalletCache, wallet)

	userInfo, err := api.GetUserProfile(chatID)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	assets, err := api.GetTokensByWalletAddress(wallet.Wallet, wallet.ChainCode, userInfo)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	kb := buildAssetsMenuSelect(assets, chatID)
	switchWalletButton := models.InlineKeyboardButton{
		Text:         "切换钱包",
		CallbackData: entity.SWITCH_DEFAULT_WALLET,
	}
	menuButton := models.InlineKeyboardButton{
		Text:         "主菜单",
		CallbackData: "go_menu",
	}
	lastLineButton := []models.InlineKeyboardButton{
		entity.GetCallbackButton(entity.SWITCH_PUBLIC_CHAIN),
		// entity.GetCallbackButton(entity.SWITCH_DEFAULT_WALLET),
		switchWalletButton,
		menuButton,
	}
	kb.InlineKeyboard = append(kb.InlineKeyboard, lastLineButton)

	chainName := api.GetChainNameFallbackCode(wallet.ChainCode)

	store.BotMessageAdd()
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        fmt.Sprintf(assetsText, wallet.Wallet, chainName),
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
	}
}

func AssetsHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := util.EffectId(update)
	userInfo, err := api.GetUserProfile(chatID)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	defaultW, _, chainCode := UserDefaultWalletInfo(userInfo)
	assets, err := api.GetTokensByWalletAddress(defaultW.Wallet, chainCode, userInfo)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	kb := buildAssetsMenuSelect(assets, chatID)
	switchWalletButton := models.InlineKeyboardButton{
		Text:         "切换钱包",
		CallbackData: entity.SWITCH_DEFAULT_WALLET,
	}
	menuButton := models.InlineKeyboardButton{
		Text:         "主菜单",
		CallbackData: "go_menu",
	}
	lastLineButton := []models.InlineKeyboardButton{
		entity.GetCallbackButton(entity.SWITCH_PUBLIC_CHAIN),
		// entity.GetCallbackButton(entity.SWITCH_DEFAULT_WALLET),
		switchWalletButton,
		menuButton,
	}
	kb.InlineKeyboard = append(kb.InlineKeyboard, lastLineButton)

	chainName := api.GetChainNameFallbackCode(chainCode)

	store.BotMessageAdd()
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        fmt.Sprintf(assetsText, defaultW.Wallet, chainName),
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
	}
}

// Format:
// callbackData: switchwallet_7148949927_testAddress1
// text: testA....ess1
func buildAssetsMenuSelect(addressTokens model.GetAddressTokens, chatID int64) models.InlineKeyboardMarkup {
	var buttons [][]models.InlineKeyboardButton

	for _, asset := range addressTokens.Data {
		addr := asset.Address
		// totalAmount 使用的是USD
		totalAmount := util.FormatNumber(asset.TotalAmount)
		amount := func() string {
			// shift by decimals
			aS := util.ShiftLeftStr(asset.Amount, asset.Decimals)
			return util.FormatNumber(aS)
		}()

		price := util.FormatNumber(asset.Price)

		tem := "%s-%s($%s)-$%s-%s....%s"
		displayText := fmt.Sprintf(tem, asset.Symbol, amount, price, totalAmount, addr[:4], addr[len(addr)-6:])

		displayText = func() string {
			if util.IsNativeCoion(addr) {
				s := strings.Split(displayText, "-")
				if len(s) > 1 {
					s = s[:len(s)-1]
					return strings.Join(s, "-")
				}
			}
			return displayText
		}()

		callbackData := fmt.Sprintf("AS_%s_%d", asset.Address, chatID)

		if len(callbackData) > 64 {
			log.Error().Err(errors.New("callback Data is too long")).Send()
		}

		button := models.InlineKeyboardButton{
			Text:         displayText,
			CallbackData: callbackData,
		}
		buttons = append(buttons, []models.InlineKeyboardButton{button})
	}

	return models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}

var (
	BUY_BUTTON  = "buy_%s_%s_"
	SELL_BUTTON = "sell_%s_%s_"
)

// trigger by AS::
func AssetsSelectHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callbackData := update.CallbackQuery.Data
	callbackSlice := strings.Split(callbackData, "_")
	if len(callbackSlice) < 3 {
		log.Error().Err(errors.New("callbackSlice err len in AS:: handler")).Msg("len is < 3")
		return
	}

	address := callbackSlice[1]
	chatId := cast.ToInt64(callbackSlice[2])

	// setting user select token in callbackData
	session.GetSessionManager().Set(chatId, session.UserSelectTokenAddressCache, address)

	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
		return
	}

	dw, _, _ := UserDefaultWalletInfo(userInfo)
	tokenInfo, err := api.GetPositionByWalletAddress(dw.Wallet, address, dw.ChainCode, userInfo)
	log.Debug().Interface("tokeninfo", tokenInfo).Msg("test to find bug in bnb")
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatId, "出错了，请联系客服")
		return
	}

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

	messageWrap := model.NewMessageWrap(chatId, *message, tokenInfo)

	// clean session data
	defer func() {
		sm := session.GetSessionManager()
		sm.Set(chatId, session.UserLastSwapMessage, messageWrap)
		sm.Delete(chatId, session.UserSelectChainCache)
		// sm.Delete(chatId, session.UserSelectWalletCache)
		// WARN: user select token address
		// sm.Delete(chatId, session.UserSelectTokenAddressCache)
	}()
}
