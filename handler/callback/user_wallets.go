package callback

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/entity"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
)

// trigger by entity.WALLET
func WalletMenuHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := util.EffectId(update)

	userInfo, err := api.GetUserProfile(chatID)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, "你的账户信息不正常，请联系管理员！")
		return
	}

	// TODO: default wallet
	defaultW, chainWallets, defaultChain := UserDefaultWalletInfo(userInfo)

	chanName := api.GetChainNameFallbackCode(defaultChain)

	kb := buildWalletMenuSelect(chainWallets, chatID, userInfo)
	switchWalletButton := models.InlineKeyboardButton{
		Text:         "切换钱包",
		CallbackData: entity.SWITCH_DEFAULT_WALLET,
	}
	lastLineButton := []models.InlineKeyboardButton{
		entity.GetCallbackButton(entity.SWITCH_PUBLIC_CHAIN),
		// entity.GetCallbackButton(entity.SWITCH_DEFAULT_WALLET),
		switchWalletButton,
	}
	kb.InlineKeyboard = append(kb.InlineKeyboard, lastLineButton)

	store.BotMessageAdd()
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        walletInfo(defaultW, chanName),
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

func walletInfo(w model.Wallet, chain string) string {
	textTemplate := `
当前钱包：（点击地址切换默认钱包）
<code>%s</code> (点击复制)

当前公链：%s
`
	textTemplate = strings.TrimSpace(textTemplate)
	return fmt.Sprintf(textTemplate, w.Wallet, chain)
}

// trigger by SWITCH_DEFAULLT_CHAIN_WALLETS
func CallbackSelectDefaultChainWallet(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	userInfo, err := api.GetUserProfile(chatID)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	_, chainWallets, chainCode := UserDefaultWalletInfo(userInfo)
	chanName := api.GetChainNameFallbackCode(chainCode)
	kb := buildWalletMenuSelect(chainWallets, chatID, userInfo)
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		Text:        fmt.Sprintf("当前 %s 链的钱包有如下：\n", chanName),
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.False(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
}

// select handler trigger by SWITCH_DEFAULT_WALLET
func CallbackSelectChainHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID

	userInfo, err := api.GetUserProfile(chatID)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, "请联系管理员！")
		return
	}

	kb, err := BuildChainsMenuSelect(chatID, userInfo)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, "请联系管理员！")
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		Text:        "选择你要设置的链",
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.False(),
		},
	})
}

func BuildChainsMenuSelect(chatID any, userInfo model.GetUserResp) (models.InlineKeyboardMarkup, error) {
	var buttons [][]models.InlineKeyboardButton

	chainCfgs, err := api.GetChainConfigs()
	if err != nil {
		log.Error().Err(err).Send()
		return models.InlineKeyboardMarkup{}, err
	}
	slices.SortFunc(chainCfgs.Data, func(a, b model.ChainConfig) int {
		if a.Sort < b.Sort {
			return -1
		}
		if a.Sort > b.Sort {
			return 1
		}
		return 0
	})

	for _, chainCfg := range chainCfgs.Data {
		displayText := chainCfg.Chain

		dw, _, _ := UserDefaultWalletInfo(userInfo)
		if dw.ChainCode == chainCfg.ChainCode {
			displayText = "✅" + displayText
		}

		callbackData := fmt.Sprintf("swchain::%v::%s", chatID, chainCfg.ChainCode)

		button := models.InlineKeyboardButton{
			Text:         displayText,
			CallbackData: callbackData,
		}
		buttons = append(buttons, []models.InlineKeyboardButton{button})
	}

	return models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}, nil
}

// /////////////////////////////////////////////////////////////////////////////

// trigger by "swchain" prefix message
func CallbackListSelectWalletHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	selectChain := func() string {
		splitData := strings.Split(update.CallbackQuery.Data, "::")
		if len(splitData) > 0 {
			return splitData[2]
		}
		return ""
	}
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	userInfo, err := api.GetUserProfile(chatID)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	wallets := api.ListUserDefaultWalletsSwitch(userInfo, selectChain())
	if wallets == nil {
		util.QuickMessage(ctx, b, chatID, "没有钱包")
		return
	}

	kb := buildWalletMenuSelect(wallets, chatID, userInfo)

	chainName := api.GetChainNameFallbackCode(selectChain())

	text := fmt.Sprintf("当前选择链：%s\n\n现在选择你的钱包", chainName)

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.False(),
		},
	})
	if err != nil {
		log.Error().Err(err).Send()
	}
}

// Format:
// callbackData: switchwallet_7148949927_testAddress1
// text: testA....ess1
func buildWalletMenuSelect(wallets []model.Wallet, chatID any, userInfo model.GetUserResp) models.InlineKeyboardMarkup {
	var buttons [][]models.InlineKeyboardButton

	for index, wallet := range wallets {
		addr := wallet.Wallet
		if len(addr) > 10 {
			displayText := fmt.Sprintf("钱包%d %s....%s", index+1, addr[:4], addr[len(addr)-4:])
			if wallet.WalletId == userInfo.Data.TgDefaultWalletId {
				displayText = "✅" + displayText
			}

			callbackData := fmt.Sprintf("swW::%v::%s", chatID, wallet.WalletId)

			button := models.InlineKeyboardButton{
				Text:         displayText,
				CallbackData: callbackData,
			}
			buttons = append(buttons, []models.InlineKeyboardButton{button})
		}
	}

	return models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}

func CallbackConfirmSelectWallet(ctx context.Context, b *bot.Bot, update *models.Update) {
	confirmWalletID := func() string {
		splitData := strings.Split(update.CallbackQuery.Data, "::")
		if len(splitData) > 0 {
			return splitData[2]
		}
		return ""
	}()
	chatID := util.EffectId(update)
	userInfo, err := api.GetUserProfile(chatID)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, "出错了，请联系管理员！")
		return
	}

	// find the confirmWallet WalletId and setting it
	userInfo.Data.TgDefaultWalletId = confirmWalletID

	err = api.UpdateUserProfile(chatID, userInfo)
	if err != nil {
		log.Error().Err(err).Send()
		util.QuickMessage(ctx, b, chatID, "更新出错，请联系客服")
		return
	}

	selectWallet := func() model.Wallet {
		for _, wallets := range userInfo.Data.Wallets {
			for _, w := range wallets {
				if w.WalletId == confirmWalletID {
					return w
				}
			}
		}
		return model.Wallet{}
	}()

	AssetsSelectByAddressHandler(ctx, b, update, selectWallet)
}

// helper to get user default wallet and default chain
func UserDefaultWalletInfo(userInfo model.GetUserResp) (default_wallet model.Wallet, chainWallets []model.Wallet, default_chain string) {
	defaultID := userInfo.Data.TgDefaultWalletId

	for chain, wallets := range userInfo.Data.Wallets {
		for _, w := range wallets {
			if w.WalletId == defaultID {
				return w, wallets, chain
			}
		}
	}

	return model.Wallet{}, nil, ""
}
