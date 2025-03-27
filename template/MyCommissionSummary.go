package template

import (
	"github.com/flosch/pongo2/v6"
	"github.com/rs/zerolog/log"
)

var MyCommissionSummaryTempl = `
返佣比例: {{ body.commissionRate | formatPer }}
邀请码: <code>{{ invitationCode }}</code>

总邀请人数: {{ body.inviteeNum }}
总交易人数: {{ body.InviteeTradingNum }}
邀请人总交易额: ${{ body.inviteeTradingAmount }}
总返佣金额: ${{ body.totalCommissionAmount }}

可提现: ${{ body.withdrawableCommissionAmount }}
审核中: ${{ body.frozenCommissionAmount }}
已提现: ${{ body.issuedCommissionAmount }}

TG BOT邀请返佣：
<code>https://t.me/{{ botUserName }}?start=I_{{ invitationCode  }}</code> 

网页邀请返佣：
<code>https://hellodex.io/Refer?invitationCode={{ invitationCode  }}</code> 


`

func RanderMyCommissionSummary(invitationCode string, botUserName string, body map[string]any) (string, error) {
	// Compile the template first (i. e. creating the AST)
	tpl, err := pongo2.FromString(MyCommissionSummaryTempl)
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	data := body["data"].(map[string]any)

	// Now you can render the template with the given
	// pongo2.Context how often you want to.
	out, err := tpl.Execute(pongo2.Context{
		"invitationCode": invitationCode,
		"botUserName":    botUserName,
		"body":           data,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	return out, nil
}
