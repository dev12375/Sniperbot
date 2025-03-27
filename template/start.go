package template

import (
	"github.com/flosch/pongo2/v6"
	"github.com/rs/zerolog/log"
)

var StartTemPlate = `
登录 HelloDex 体验秒级交易 🤘🏻
开创和主导Web3变革，教育平台利润80%分给用户

开始交易之前，请发送一些 {{ symbol }} 到您的HelloDex 默认钱包地址:

<code>{{ walletAddress }}</code> 
钱包余额：{{ native_coin }} {{ symbol }} (${{ usd }})

发送 {{ symbol }} 后，点击刷新查看余额

支持全终端：
已推出ios、安卓、网页、TG 机器人、TG小程序、代码已开源在 <a href="https://github.com/hellodex">github</a>

TG BOT邀请返佣：
<code>https://t.me/{{ botUserName }}?start=I_{{ invitationCode  }}</code>

网页邀请返佣：
<code>https://hellodex.io/Refer?invitationCode={{ invitationCode  }}</code> 

<a href="https://t.me/HelloDex_cn">帮助</a> • <a href="https://t.me/HelloDex_cn">中文社区</a>

有史以来都是资本联合平台赚用户的钱、 请各位用户一起加入变革。
`

func RanderStart(symbol string, walletAddress string, native_coin string, usd string, invitationCode string, botUserName string) (string, error) {
	// Compile the template first (i. e. creating the AST)
	tpl, err := pongo2.FromString(StartTemPlate)
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	// Now you can render the template with the given
	// pongo2.Context how often you want to.
	out, err := tpl.Execute(pongo2.Context{
		"symbol":         symbol,
		"walletAddress":  walletAddress,
		"native_coin":    native_coin,
		"usd":            usd,
		"invitationCode": invitationCode,
		"botUserName":    botUserName,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	return out, nil
}
