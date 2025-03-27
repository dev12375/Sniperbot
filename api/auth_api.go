package api

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/duke-git/lancet/v2/netutil"
	"github.com/hellodex/tradingbot/logger"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
)

var ErrTransferToAmount = errors.New("转出 SOL 数量不能低于0.001")
var ErrTransferFail = errors.New("转出交易失败")

type TransferTo struct {
	RawAmount    string
	ToAddress    string
	TokenAddress string
	WalletId     string
	WalletKey    string
	UserInfo     model.GetUserResp
}

func (t *TransferTo) IsAmountSet() bool {
	return t.RawAmount != ""
}

func (t *TransferTo) IsToAddressSet() bool {
	return t.ToAddress != ""
}

func (t *TransferTo) Send() (string, error) {
	result := ""

	if util.IsNativeCoion(t.TokenAddress) {
		t.TokenAddress = ""
	}

	// transfer native coin
	if t.TokenAddress == "" && t.RawAmount < "1000000" {
		log.Debug().Func(func(e *zerolog.Event) {
			logger.WithTxCategory(e).Err(ErrTransferToAmount).Send()
		})
		return result, ErrTransferToAmount
	}

	header := makeHeader()
	AddBeaer(&header, t.UserInfo)

	intWalletID := cast.ToInt(t.WalletId)
	requestBody := map[string]any{
		"amount":    t.RawAmount,
		"to":        t.ToAddress,
		"token":     t.TokenAddress,
		"walletId":  intWalletID,
		"walletKey": t.WalletKey,
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}
	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/trade/transferTo",
		Method:  "POST",
		Headers: header,
		Body:    bodyBytes,
	}
	// log requestBody
	log.Debug().Func(func(e *zerolog.Event) {
		logger.WithTxCategory(e).Func(func(e *zerolog.Event) {
			jsonD, err := json.Marshal(requestBody)
			if err != nil {
				e.Err(err).Msg("err in transferTo marshal")
				return
			}
			e.Str("route", "transferTo").RawJSON("requestBody", jsonD).Send()
		})
	})

	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}

	tx := gjson.GetManyBytes(data, "code", "data.tx", "msg")
	if len(tx) == 0 && len(tx) < 3 {
		return result, ErrTransferFail
	}

	switch tx[0].Int() {
	case 200:
		result = tx[1].String()
		return result, nil
	case 404:
		return result, errors.New(tx[2].String())
	}

	// log result
	log.Debug().Func(func(e *zerolog.Event) {
		logger.WithTxCategory(e).Func(func(e *zerolog.Event) {
			e.Str("route", "transferTo").RawJSON("result", data).Send()
		})
	})

	return result, nil
}
