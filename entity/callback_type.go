package entity

import (
	"fmt"
	"strings"

	"github.com/go-telegram/bot/models"
)

type BOT_CALLBACK_DATA_CODE = string

const (
	BUY_SELL              BOT_CALLBACK_DATA_CODE = "code::buy_sell"
	ASSETS                BOT_CALLBACK_DATA_CODE = "code::assets"
	SETTING               BOT_CALLBACK_DATA_CODE = "code::setting"
	WALLET                BOT_CALLBACK_DATA_CODE = "code::wallet"
	SWITCH_DEFAULT_WALLET BOT_CALLBACK_DATA_CODE = "code::switch_default_wallet"
	SWITCH_PUBLIC_CHAIN   BOT_CALLBACK_DATA_CODE = "code::switch_public_chain"
	TRANSFER_OUT          BOT_CALLBACK_DATA_CODE = "code::transfer_out"
	SETTING_SLIPPY        BOT_CALLBACK_DATA_CODE = "code::setting_slippy"

	ORDER_FOLLOW          BOT_CALLBACK_DATA_CODE = "code::order_follow"
	ADD_ORDER_FOLLOW      BOT_CALLBACK_DATA_CODE = "code::add_order_follow"
	ORDER_FOLLOW_ALL_STOP BOT_CALLBACK_DATA_CODE = "code::order_follow_all_stop"

	ORDER_PENDING             BOT_CALLBACK_DATA_CODE = "code::order_pending"
	ADD_ORDER_PENDING         BOT_CALLBACK_DATA_CODE = "code::add_order_pending"
	ORDER_PENDING_IN_Progress BOT_CALLBACK_DATA_CODE = "code::order_pending_in_progress"

	HELP                BOT_CALLBACK_DATA_CODE = "code::help"
	LANG                BOT_CALLBACK_DATA_CODE = "code::lang"
	INVITE              BOT_CALLBACK_DATA_CODE = "code::invite"
	CHANGE_BOT          BOT_CALLBACK_DATA_CODE = "code::change_bot"
	ETH_BOT             BOT_CALLBACK_DATA_CODE = "code::eth_bot"
	BASE_BOT            BOT_CALLBACK_DATA_CODE = "code::base_bot"
	INVITE_DETIAL       BOT_CALLBACK_DATA_CODE = "code::invite_detial"
	Withdrawal          BOT_CALLBACK_DATA_CODE = "code::withdrawal"
	HistoryOrder        BOT_CALLBACK_DATA_CODE = "code::history_order"
	OrderTrade          BOT_CALLBACK_DATA_CODE = "code::history_trade"
	HistoryTransfer     BOT_CALLBACK_DATA_CODE = "code::history_transfer"
	AppDownload         BOT_CALLBACK_DATA_CODE = "code::app_download"
	AdminUrl            BOT_CALLBACK_DATA_CODE = "code::admin_url"
	Other               BOT_CALLBACK_DATA_CODE = "code::other"
	ReflashTokenInfo    BOT_CALLBACK_DATA_CODE = "code::reflashTokenInfo"
	RefalshStartBalacne BOT_CALLBACK_DATA_CODE = "code::reflashStartBalance"

	AiMonitorTokenInfoSetting     BOT_CALLBACK_DATA_CODE = "code::aiMonitorSetting"
	AddAiMonitor                  BOT_CALLBACK_DATA_CODE = "code::addAiMonitor"
	AiMonitorList                 BOT_CALLBACK_DATA_CODE = "code::aiMonitorList"
	InviteButton                  BOT_CALLBACK_DATA_CODE = "code::inviteButton"
	AIMonitorButton               BOT_CALLBACK_DATA_CODE = "code::aiMonitorButton"
	_BOT_CALLBACK_DATA_CODE_COUNT                        = iota
)

var CallbackTextMap = map[BOT_CALLBACK_DATA_CODE]string{
	BUY_SELL:              "👉买/卖",
	ASSETS:                "💰资产",
	SETTING:               "⚙️设置",
	WALLET:                "💳钱包",
	SWITCH_DEFAULT_WALLET: "切换默认钱包",
	SWITCH_PUBLIC_CHAIN:   "切换公链",
	TRANSFER_OUT:          "转出",
	SETTING_SLIPPY:        "滑点设置",

	ORDER_FOLLOW:          "跟单",
	ADD_ORDER_FOLLOW:      "新增跟单",
	ORDER_FOLLOW_ALL_STOP: "全部暂停",

	ORDER_PENDING:             "挂单",
	ADD_ORDER_PENDING:         "添加挂单",
	ORDER_PENDING_IN_Progress: "进行中的挂单",

	HELP:                      "帮助",
	LANG:                      "语言",
	INVITE:                    "邀请好友",
	CHANGE_BOT:                "切换机器人",
	ETH_BOT:                   "ETH_Bot",
	BASE_BOT:                  "Base_Bot",
	INVITE_DETIAL:             "邀请详情",
	Withdrawal:                "提现返佣",
	HistoryOrder:              "🏷️挂单记录",
	OrderTrade:                "📈挂单交易",
	HistoryTransfer:           "📋交易记录",
	AppDownload:               "📱APP下载",
	AdminUrl:                  "💬联系客服",
	Other:                     "❤️主导Web3变革利润80%给用户",
	ReflashTokenInfo:          "♻️刷新",
	RefalshStartBalacne:       "刷新余额",
	AiMonitorTokenInfoSetting: "Token监控设置",
	AiMonitorList:             "监控列表",
	AddAiMonitor:              "添加监控",
	InviteButton:              "邀请返佣",
	AIMonitorButton:           "AI监控",
}

// check text map and code count
var _ = func() any {
	if len(CallbackTextMap) != _BOT_CALLBACK_DATA_CODE_COUNT {
		panic(fmt.Sprintf(
			"CallbackTextMap size mismatch: got %d, want %d",
			len(CallbackTextMap),
			_BOT_CALLBACK_DATA_CODE_COUNT,
		))
	}
	return nil
}()

type CallbackButton = models.InlineKeyboardButton

// build callback button
func GetCallbackButton(code BOT_CALLBACK_DATA_CODE) CallbackButton {
	return CallbackButton{
		CallbackData: code,
		Text:         CallbackTextMap[code],
	}
}

// split callback data
func SplitCallbackData(code BOT_CALLBACK_DATA_CODE) []string {
	return strings.Split(code, "::")
}
