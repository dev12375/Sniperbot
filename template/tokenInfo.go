package template

import (
	"github.com/flosch/pongo2/v6"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
)

var tokenInfoTemplate = `
<a href="https://hellodex.io/k/{{ data.PairAddress }}?chainCode={{ data.ChainCode }}&timeType=15m">{{ data.BaseToken.Symbol }}</a> ({{ data.ChainCode }})
{% if view_token %}<code>{{ view_token }}</code>{% endif %}

💳持币:
--余额: {{ data.Amount | formatNumber }}
--价格: ${{ data.Price | formatNumber }}
--总金额: ${{ data.Volume | formatNumber }}
--池子: <code>{{data.PairAddress}}</code>

💴仓位:
--总买入: {{ data.TotalBuyAmount | formatNumber }}
--总买入金额: ${{ data.TotalBuyVolume | formatNumber }}
--平均买入价格: ${{ data.AveragePrice | formatNumber }}
--总卖出: {{ data.TotalSellAmount | formatNumber }}
--总卖出金额: ${{ data.TotalSellVolume | formatNumber }}
--累计收益: {{ data.TotalEarn | formatNumber }}
--收益率: {{ data.TotalEarnRate | formatNumber }}%`

func RanderTokenInfo(data model.PositionByWalletAddress) (string, error) {
	// Compile the template first (i. e. creating the AST)
	tpl, err := pongo2.FromString(tokenInfoTemplate)
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	view_token := func() string {
		if !util.IsNativeCoion(data.Data.BaseToken.Address) {
			return data.Data.BaseToken.Address
		}
		return ""
	}

	// Now you can render the template with the given
	// pongo2.Context how often you want to.
	out, err := tpl.Execute(pongo2.Context{
		"data":       data.Data,
		"view_token": view_token,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	return out, nil
}
