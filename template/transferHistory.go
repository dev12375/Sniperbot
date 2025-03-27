package template

import (
	"fmt"

	"github.com/flosch/pongo2/v6"
	"github.com/hellodex/tradingbot/model"
	"github.com/rs/zerolog/log"
)

var listTransferHistoryTemplate = `
最近{{size}}笔交易记录：
{% for trade in transfers | slice:slice_range %}
{{ trade.Chain }}
📊 交易信息:
|--币种: <b>{% if trade.TokenAddress %}{{ trade.TokenAddress }}{% else %}主网货币{% endif %}</b>
|--数量: <b>{{ trade.Amount|formatNumber }}</b>
|--接收地址: <b>{{ trade.ToAddress }}</b>
|--状态: <b>{{ trade.OrderStatusUI }}</b>
{%- if trade.Hash %}
|--交易哈希: <code>{{ trade.Hash }}</code>{% endif %}
|--时间: <b>{{ trade.Timestamp|formatTime }}</b>
{% endfor %}
`

func RanderListTransferHistory(transfers []model.TransferHistoryInner, page, pageSize int) (string, error) {
	// Compile the template first (i. e. creating the AST)
	tpl, err := pongo2.FromString(listTransferHistoryTemplate)
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
		"transfers":   transfers,
		"slice_range": sliceRange,
		"size":        pageSize,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	return out, nil
}
