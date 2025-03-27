package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
)

var withdrawlTxt = `
可提现金额: $%s, 请输入提现金额
`

func CallbackWithdrawl(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)

	reply := models.ForceReply{
		ForceReply:            true,
		InputFieldPlaceholder: "10",
	}
	userInfo, err := api.GetUserProfile(chatId)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 错误代码：getuser", util.AdminUrl))
		return
	}
	cmInfo, err := api.GetMyCommissionSummary(userInfo)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}
	dw, _, _ := UserDefaultWalletInfo(userInfo)
	reqData := map[string]any{
		"chainCode":     dw.ChainCode,
		"walletAddress": "",
		"amount":        "",
	}

	dd, err := json.Marshal(reqData)
	if err != nil {
		util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
		return
	}

	store.UserSetCommissionInfo(chatId, dd)

	withdrawableCommissionAmount := gjson.GetBytes(cmInfo, "data.withdrawableCommissionAmount").String()

	store.BotMessageAdd()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		ReplyMarkup: reply,
		Text:        fmt.Sprintf(withdrawlTxt, withdrawableCommissionAmount),
	})
}

// button("取消提现", "withdrawal_no"),
// button("确认提现", "withdrawal_yes"),
func CallBackWithdrawlButton(ctx context.Context, b *bot.Bot, u *models.Update) {
	chatId := util.EffectId(u)
	callbackData := u.CallbackQuery.Data

	action := strings.TrimPrefix(callbackData, "withdrawal_")
	switch action {
	case "yes":
		data, has := store.UserGetCommissionInfo(chatId)
		if !has {
			log.Debug().Msg("user has not commission info")
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}

		var sq map[string]string
		err := json.Unmarshal(data, &sq)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}

		amount := cast.ToInt64(sq["amount"])
		chainCode := sq["chainCode"]
		walletAddress := sq["walletAddress"]

		userInfo, err := api.GetUserProfile(chatId)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s, 错误代码：getuser", util.AdminUrl))
			return
		}
		_, err = api.SubmitWithdraw(chainCode, walletAddress, amount, userInfo)
		if err != nil {
			util.QuickMessage(ctx, b, chatId, fmt.Sprintf("出错了,%s", util.AdminUrl))
			return
		}
		store.BotMessageAdd()
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:      "提现已提交，审核通过后立即发放",
			ChatID:    chatId,
			ParseMode: "HTML",
		})
		store.UserDeleteCommissionInfo(chatId)
	case "no":
		store.UserDeleteCommissionInfo(chatId)
		store.BotMessageAdd()
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:      "提现已取消",
			ChatID:    chatId,
			ParseMode: "HTML",
		})
	}
}
