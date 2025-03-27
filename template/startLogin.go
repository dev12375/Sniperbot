package template

import (
	"github.com/flosch/pongo2/v6"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
)

var startLoginTemplate = `
登录 HelloDex 体验秒级交易 🤘🏻
开创和主导Web3变革，教育平台利润80%分给用户

💳 {{ chainCode | getChainName }}: {{ balance | formatNumber }} {{ symbol }} {% if balance == 0 %}(余额不足, 请充值👇🏻){% endif %}
<code>{{ address }}</code> (点击复制) 

如有问题，{{ adminUrl | safe }}

TG BOT邀请返佣：
<code>https://t.me/{{ botUserName }}?start=I_{{ invitationCode  }}</code> 

网页邀请返佣：
<code>https://hellodex.io/Refer?invitationCode={{ invitationCode  }}</code> 

有史以来都是资本联合平台赚用户的钱、 请各位用户一起加入变革。
`

func RanderStartLogin(userInfo model.GetUserResp, commissionUrl string) (string, error) {
	// Compile the template first (i. e. creating the AST)
	tpl, err := pongo2.FromString(startLoginTemplate)
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	wallet := func() *model.Wallet {
		for _, wallets := range userInfo.Data.Wallets {
			for _, w := range wallets {
				if userInfo.Data.TgDefaultWalletId == w.WalletId {
					return &w
				}
			}
		}
		return nil
	}()

	// Now you can render the template with the given
	// pongo2.Context how often you want to.
	out, err := tpl.Execute(pongo2.Context{
		"balance":        userInfo.Data.MainnetToken.Balance,
		"symbol":         userInfo.Data.MainnetToken.Symbol,
		"chainCode":      wallet.ChainCode,
		"address":        wallet.Wallet,
		"adminUrl":       util.AdminUrl,
		"commissionUrl":  commissionUrl,
		"invitationCode": userInfo.Data.InviteCode,
		"botUserName":    store.GetEnv(store.BOT_USERNAME),
	})
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	return out, nil
}
