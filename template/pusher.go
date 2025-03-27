package template

import (
	"github.com/flosch/pongo2/v6"
	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

var tgUserTokenPushTemplate = `
HelloDex: AI监控价格

{{ symbol }} ({{ chainCode | getChainName }})

{% if baseAddress %}<code>{{ baseAddress }}</code>{% endif %}

价格已到: ${{ price | formatNumber }}
交易数量: {{ amount | formatNumber }}
交易总额: ${{ volume | formatNumber }}
交易方向: {% if flag == 0 %}买入{% else %}卖出{% endif %}
`

func RenderTgUserTokenPush(data []byte) (string, error) {
	tpl, err := pongo2.FromString(tgUserTokenPushTemplate)
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	allParts := gjson.GetManyBytes(
		data,
		"payload.symbol",
		"payload.chainCode",
		"payload.baseAddress",
		"payload.price",
		"payload.amount",
		"payload.volume",
		"payload.flag",
	)

	dataMap := map[string]any{
		"symbol":      allParts[0].String(),
		"chainCode":   allParts[1].String(),
		"baseAddress": allParts[2].String(),
		"price":       allParts[3].String(),
		"amount":      allParts[4].String(),
		"volume":      allParts[5].String(),
		"flag":        allParts[5].Int(),
	}

	out, err := tpl.Execute(pongo2.Context(dataMap))
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}
	return out, nil
}
