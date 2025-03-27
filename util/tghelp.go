package util

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/config"
	"github.com/hellodex/tradingbot/entity"
	"github.com/hellodex/tradingbot/store"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

func QuickMessage(ctx context.Context, b *bot.Bot, userID int64, text string) {
	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    userID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
	})
}

func NewCallbackDataButton(text, callbackData string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text:         text,
		CallbackData: callbackData,
	}
}

func BackToMainMenu() models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text:         "返回主菜单",
		CallbackData: "backToMainMenu",
	}
}

func QuickMessageWithButton(ctx context.Context, b *bot.Bot, userID int64, text string, button models.InlineKeyboardButton) {
	line := []models.InlineKeyboardButton{button}
	keyboard := [][]models.InlineKeyboardButton{line}
	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    userID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})
}

type BuySellKeyBoardData struct {
	_                   struct{}
	PoolAddress         string
	BuyCallBackData     string
	SellCallBackData    string
	BaseToken           string // 操作的代币，如 SOL
	BaseTokenChainCode  string
	BaseTokenAddress    string
	QuoteToken          string // 对手代币，如 USDC
	QuoteTokenChainCode string
	QuoteTokenAddress   string
}

func UrlButton(text, url string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text: text,
		URL:  url,
	}
}

func button(text, callbackData string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text:         text,
		CallbackData: callbackData,
	}
}

func BuySellKeyBoard(data BuySellKeyBoardData) models.InlineKeyboardMarkup {
	var kb models.InlineKeyboardMarkup
	var titleLine, buyLine, lineTwo, lineThree, sellLine []models.InlineKeyboardButton

	titleLine = []models.InlineKeyboardButton{
		entity.GetCallbackButton(entity.ReflashTokenInfo),
		// button("📊看K线", "k_line"),
		{
			Text: "📊看K线",
			URL:  fmt.Sprintf("%s%s?chainCode=%s", config.YmlConfig.Env.KchartUrl, data.PoolAddress, data.BaseTokenChainCode),
		},
	}

	strTime := cast.ToString(time.Now().Unix())

	buyLine = []models.InlineKeyboardButton{
		button("----🟢买----", strTime),
	}

	sellLine = []models.InlineKeyboardButton{
		// button(fmt.Sprintf("----🔴卖( %s )----", data.BaseToken), "none"),
		button("----🔴卖----", "none"),
	}
	lineTwo = []models.InlineKeyboardButton{
		button("买 0.1 "+data.QuoteToken, data.BuyCallBackData+"0.1"),
		button("买 0.5 "+data.QuoteToken, data.BuyCallBackData+"0.5"),
		button("买 1 "+data.QuoteToken, data.BuyCallBackData+"1"),
	}
	lineThree = []models.InlineKeyboardButton{
		button("买 5 "+data.QuoteToken, data.BuyCallBackData+"5"),
		button("买 10 "+data.QuoteToken, data.BuyCallBackData+"10"),
		button("买 x "+data.QuoteToken, data.BuyCallBackData+"x"),
	}

	// lineFive := []models.InlineKeyboardButton{
	// 	button("卖10%", data.SellCallBackData+"10"),
	// 	button("卖25%", data.SellCallBackData+"25"),
	// 	button("卖x", data.SellCallBackData+"y"),
	// }

	lineSix := []models.InlineKeyboardButton{
		button("卖50%", data.SellCallBackData+"50"),
		button("卖100%", data.SellCallBackData+"100"),
		button("卖x%", data.SellCallBackData+"x"),
	}

	lineTransfer := []models.InlineKeyboardButton{
		// button("🔴转出 "+data.BaseToken, "tx_"+data.BaseTokenAddress),
		// button("📌挂单 "+data.BaseToken, "order_"+data.BaseTokenAddress),
		button("🔴转出", "tx_"+data.BaseTokenAddress),
		button("📌挂单", "order_"+data.BaseTokenAddress),
		button("资产列表", entity.ASSETS),
	}

	kb.InlineKeyboard = [][]models.InlineKeyboardButton{
		titleLine,
		buyLine,
		lineTwo,
		lineThree,
		sellLine,
		// lineFive,
		lineSix,
		lineTransfer,
	}

	// log.Debug().Func(func(e *zerolog.Event) {
	// 	txe := logger.WithTxCategory(e)
	// 	data, err := json.Marshal(kb)
	// 	if err != nil {
	// 		txe.Err(err).Send()
	// 	}
	// 	txe.RawJSON("InlineKeyboard", data).Send()
	// })
	return kb
}

func GetLimitOrderPrefixText(callbackData string) string {
	switch callbackData {
	case "limitOrder_sell_3":
		return "止盈"
	case "limitOrder_sell_4":
		return "止损"
	case "limitOrder_buy_2":
		return "抄底"
	case "limitOrder_buy_1":
		return "高于买入"
	}
	return ""
}

func LimitOrderKeyBoard() models.InlineKeyboardMarkup {
	var kb models.InlineKeyboardMarkup

	kb.InlineKeyboard = append(kb.InlineKeyboard, []models.InlineKeyboardButton{
		button("止盈", "limitOrder_sell_3"),
		button("止损", "limitOrder_sell_4"),
	})
	kb.InlineKeyboard = append(kb.InlineKeyboard, []models.InlineKeyboardButton{
		button("抄底", "limitOrder_buy_2"),
		button("高于买入", "limitOrder_buy_1"),
	})

	// log.Debug().Func(func(e *zerolog.Event) {
	// 	txe := logger.WithTxCategory(e)
	// 	txe.Interface("InlineKeyboard", kb).Send()
	// })
	return kb
}

func PinMessage(ctx context.Context, b *bot.Bot, update *models.Update, chatID int64, messageID int) {
	_, err := b.PinChatMessage(ctx, &bot.PinChatMessageParams{
		ChatID:              chatID,
		MessageID:           messageID,
		DisableNotification: false,
	})
	if err != nil {
		log.Error().Err(err).Send()
	}
}

func CallBackAnswer(ctx context.Context, b *bot.Bot, callbackQuery *models.CallbackQuery) {
	ok, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackQuery.ID,
	})
	if err != nil {
		log.Error().Err(err).Msg("callbackAnswer err")
		return
	}
	if !ok {
		log.Warn().Msg("callbackAnswer not ok")
	}

	log.Info().Msg("callbackAnswer")
}

func NewAiMonitorPusherButton() models.InlineKeyboardMarkup {
	var kb models.InlineKeyboardMarkup

	// 第一行：三个监控操作按钮
	row1 := []models.InlineKeyboardButton{
		button("暂停监控", "pusher_pause"),
		button("编辑监控", "pusher_edit"),
		button("删除监控", "pusher_delete"),
	}

	// 第二行：两个交易相关按钮
	row2 := []models.InlineKeyboardButton{
		button("👉一键交易/挂单交易", "ai_sendNewupdate"),
	}

	kb.InlineKeyboard = [][]models.InlineKeyboardButton{
		row1,
		row2,
	}

	return kb
}

func NewAiMonitorKeyboard() models.InlineKeyboardMarkup {
	var kb models.InlineKeyboardMarkup
	// define delivery method buttons
	one := []models.InlineKeyboardButton{
		button("监控列表", "ai_monitor_list"),
		button("添加监控", "ai_add_monitor"),
	}
	kb.InlineKeyboard = [][]models.InlineKeyboardButton{
		one,
	}
	return kb
}

func AiMonitorDeliverLineMarkup(data SettingsKeyBoardData) models.InlineKeyboardMarkup {
	var kb models.InlineKeyboardMarkup
	// define delivery method buttons
	deliveryline := []models.InlineKeyboardButton{
		button("推送:", "none"),
		GetToggleButton("TG", data.EnableTG),
		GetToggleButton("网页", data.EnableWeb),
		GetToggleButton("APP", data.EnableApp),
	}
	// Combine all lines into keyboard
	kb.InlineKeyboard = [][]models.InlineKeyboardButton{
		deliveryline,
	}

	return kb
}

func AiMonitorSettingsKeyBoard(data SettingsKeyBoardData) models.InlineKeyboardMarkup {
	var kb models.InlineKeyboardMarkup

	// define delivery method buttons
	deliveryline := []models.InlineKeyboardButton{
		button("推送:", "none"),
		GetToggleButton("TG", data.EnableTG),
		GetToggleButton("网页", data.EnableWeb),
		GetToggleButton("APP", data.EnableApp),
	}

	// Define frequency buttons
	frequencyLine := []models.InlineKeyboardButton{
		button("频率:", "none"),
		GetFreqButton("一次", data.Frequency == "once"),
		GetFreqButton("每天1次", data.Frequency == "daily"),
		GetFreqButton("每次", data.Frequency == "every"),
	}

	// Action buttons
	actionLine := []models.InlineKeyboardButton{
		button("添加代币推送", "add_token_alert"),
		button("取消设置", "cancel_settings"),
	}

	// Combine all lines into keyboard
	kb.InlineKeyboard = [][]models.InlineKeyboardButton{
		deliveryline,
		frequencyLine,
		actionLine,
	}

	return kb
}

func AiMonitor_EditSettingsKeyBoard(data SettingsKeyBoardData, monitorType string, noticeType int) models.InlineKeyboardMarkup {
	var kb models.InlineKeyboardMarkup

	var viewText string
	if monitorType == "price" {
		viewText = "目标价格"
	} else if monitorType == "chg" {
		viewText = "目标涨幅"
	} else if monitorType == "buy" {
		viewText = "买入交易额"
	} else {
		viewText = "卖出交易额"
	}

	// define delivery method buttons
	deliveryline := []models.InlineKeyboardButton{
		button("推送:", "none"),
		GetToggleButton("TG", data.EnableTG),
		GetToggleButton("网页", data.EnableWeb),
		GetToggleButton("APP", data.EnableApp),
	}

	// Define frequency buttons
	frequencyLine := []models.InlineKeyboardButton{
		button("频率:", "none"),
		GetFreqButton("一次", data.Frequency == "once"),
		GetFreqButton("每天1次", data.Frequency == "daily"),
		GetFreqButton("每次", data.Frequency == "every"),
	}

	// Action buttons
	actionLine := []models.InlineKeyboardButton{
		button(fmt.Sprintf("编辑%s", viewText), "edit_current_"+monitorType),
		button("保存更新", "save_current_"+monitorType),
	}

	if noticeType == 0 {
		actionLine = append(actionLine, button("启动推送", "enable_current_"+monitorType), button("删除推送", "delete_current_"+monitorType))
	} else {
		actionLine = append(actionLine, button("暂停推送", "pause_current_"+monitorType), button("删除推送", "delete_current_"+monitorType))
	}

	// Combine all lines into keyboard
	kb.InlineKeyboard = [][]models.InlineKeyboardButton{
		deliveryline,
		frequencyLine,
		actionLine,
	}

	return kb
}

func WithdrawalKeyBoard() models.InlineKeyboardMarkup {
	var kb models.InlineKeyboardMarkup

	// define delivery method buttons
	line := []models.InlineKeyboardButton{
		button("取消提现", "withdrawal_no"),
		button("确认提现", "withdrawal_yes"),
	}

	// Combine all lines into keyboard
	kb.InlineKeyboard = [][]models.InlineKeyboardButton{
		line,
	}

	return kb
}

// Helper function to create toggle buttons with checkmark or cross
func GetToggleButton(text string, enabled bool) models.InlineKeyboardButton {
	var displayText string
	if enabled {
		displayText = "✅ " + text
	} else {
		displayText = "❌ " + text
	}
	return button(displayText, "toggle_"+strings.ToUpper(text))
}

// Helper function to create frequency selection buttons
func GetFreqButton(text string, selected bool) models.InlineKeyboardButton {
	var displayText string
	if selected {
		displayText = "✅ " + text
	} else {
		displayText = text
	}
	switch text {
	case "一次":
		return button(displayText, "freq_"+"1")
	case "每天1次":
		return button(displayText, "freq_"+"2")
	case "每次":
		return button(displayText, "freq_"+"3")
	}
	return models.InlineKeyboardButton{}
}

// SettingsKeyBoardData structure to hold the settings state
type SettingsKeyBoardData struct {
	EnableTG  bool
	EnableWeb bool
	EnableApp bool
	Frequency string // "once", "daily", "every"
}
