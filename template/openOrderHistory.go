package template

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flosch/pongo2/v6"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

var listOpenOrdersHistoryTemplate = `
{% for order in openOrders | slice:slice_range %}
{{ order.ChainCode | getChainName }}
委托信息: <b>{{ order.FromTokenSymbol }}/{{ order.ToTokenSymbol }}</b>
{{ limitTypeStr }}
{{ trigger }}: $<b>{{ triggerAmount | formatNumber }}</b>
委托数量: <b>{{ order.Amount|formatNumber }} {{ order.FromTokenSymbol }}</b>
交易额: <b>${{ order.Volume|formatNumber }}</b>
状态: <b>{{ order.OrderStatusUI }}</b>
订单号: <code><b>{{ order.OrderNo }}</b></code>
{%- if order.Tx %}
交易哈希: <code>{{ order.Tx }}</code>{% endif %}
时间: <b>{{ order.Timestamp|formatTime }}</b>
{% endfor %}
`

func RanderOpenOrderInlineKeyboard(openOrders []model.OpenOrderInner) (models.InlineKeyboardMarkup, error) {
	kb := models.InlineKeyboardMarkup{
		InlineKeyboard: make([][]models.InlineKeyboardButton, 0, len(openOrders)),
	}

	for _, order := range openOrders {
		buttonText := fmt.Sprintf("%s-%v-%v",
			util.FormatNumber(order.FromTokenSymbol),
			util.FormatNumber(order.Amount),
			util.FormatNumber(order.Price))

		callbackData := fmt.Sprintf("view_order::%s", order.OrderNo)

		buttonRow := []models.InlineKeyboardButton{
			{
				Text:         buttonText,
				CallbackData: callbackData,
			},
		}

		kb.InlineKeyboard = append(kb.InlineKeyboard, buttonRow)
	}

	return kb, nil
}

func CallbackHandlerViewOrder(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	callbackData := u.CallbackQuery.Data
	orderNo := strings.TrimPrefix(callbackData, "view_order::")

	kb := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         "取消委托",
					CallbackData: "cancelOrder::" + orderNo,
				},
			},
			{
				{
					Text:         "返回上一级",
					CallbackData: "backToOrderList",
				},
				{
					Text:         "返回主菜单",
					CallbackData: "backToMainMenu",
				},
			},
		},
	}

	// get orderDetail info from redis cache
	data, has := store.UserGetOrderHistory(chatId)
	if !has {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	var orderList []model.OpenOrderInner
	err := json.Unmarshal(data, &orderList)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	n := 0
	for _, x := range orderList {
		if x.OrderNo == orderNo {
			orderList[n] = x
			n++
		}
	}
	orderList = orderList[:n]

	orderDetail, err := RanderOpenOrdersHistory(orderList, 1, 5)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        orderDetail,
		ReplyMarkup: kb,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: bot.True(),
		},
		ParseMode: models.ParseModeHTML,
	})
}

// cancelOrder::
func CallabckOrderCancelOrder(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	callbackData := u.CallbackQuery.Data
	orderNo := strings.TrimPrefix(callbackData, "cancelOrder::")
	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}
	_, err = api.CancelOrder(orderNo, userInfo)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		Text:   fmt.Sprintf("取消: %s 成功", orderNo),
		ChatID: chatId,
	})
}

func RanderOpenOrdersHistory(openOrders []model.OpenOrderInner, page, pageSize int) (string, error) {
	// Compile the template first (i. e. creating the AST)
	tpl, err := pongo2.FromString(listOpenOrdersHistoryTemplate)
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	sliceRange := fmt.Sprintf("%d:%d", start, end)

	limitTypeStr := ""
	for _, order := range openOrders {
		limitType := order.LimitType
		if limitType == "1" || limitType == "5" {
			limitTypeStr = "高于价格后买入"
		} else if limitType == "2" || limitType == "6" {
			limitTypeStr = "抄底"
		} else if limitType == "3" || limitType == "7" {
			limitTypeStr = "止盈"
		} else if limitType == "4" || limitType == "8" {
			limitTypeStr = "止损"
		}
		if order.FromOrderNo != "" {
			profitFlag := cast.ToInt64(order.ProfitFlag) * 100
			limitTypeStr = "自定义涨幅" + cast.ToString(profitFlag) + "%出本"
		}
	}

	trigger := ""
	triggerAmount := ""
	for _, order := range openOrders {
		limitType := cast.ToInt(order.LimitType)
		if limitType > 4 {
			trigger = "触发市值"
			triggerAmount = order.MarketCap
		} else {
			trigger = "触发价格"
			triggerAmount = order.Price
		}
		if order.FromOrderNo != "" {
			trigger = "触发价格"
			triggerAmount = order.Price
		}

	}
	// Now you can render the template with the given
	// pongo2.Context how often you want to.
	out, err := tpl.Execute(pongo2.Context{
		"openOrders":    openOrders,
		"slice_range":   sliceRange,
		"size":          pageSize,
		"limitTypeStr":  limitTypeStr,
		"trigger":       trigger,
		"triggerAmount": triggerAmount,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	return out, nil
}
