package template

import (
	"fmt"

	"github.com/flosch/pongo2/v6"
	"github.com/hellodex/tradingbot/model"
	"github.com/rs/zerolog/log"
)

var listTradeHistoryTemplate = `
最近{{size}}笔交易记录：
{% for trade in trades | slice:slice_range %}
<b>{{ trade.BaseSymbol }}/{{ trade.QuoteSymbol }}</b> ({{ trade.ChainCode | getChainName}})
📊 交易信息:
|--类型: <b>{% if trade.Direction == '1' %}卖出{% else %}买入{% endif %}</b>
|--数量: <b>{{ trade.Amount }} {{ trade.BaseSymbol }}</b>
|--价格: <b>${{ trade.Price|formatNumber }}</b>
|--交易额: <b>${{ trade.Volume|formatNumber }}</b>
|--订单号: <code><b>{{ trade.OrderNo }}</b></code>
|--状态: <b>{{ trade.OrderStatusUI }}</b>
{%- if trade.Tx %}
|--交易哈希: <code>{{ trade.Tx }}</code>{% endif %}
|--时间: <b>{{ trade.Timestamp|formatTime }}</b>
{% endfor %}
`

func RanderListTradeHistory(trades []model.TradeHistoryInner, page, pageSize int) (string, error) {
	// Compile the template first (i. e. creating the AST)
	tpl, err := pongo2.FromString(listTradeHistoryTemplate)
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	sliceRange := fmt.Sprintf("%d:%d", start, end)

	// Now you can render the template with the given
	// pongo2.Context how often you want to.
	out, err := tpl.Execute(pongo2.Context{
		"trades":      trades,
		"slice_range": sliceRange,
		"size":        pageSize,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	return out, nil
}
