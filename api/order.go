package api

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/duke-git/lancet/v2/netutil"
	"github.com/hellodex/tradingbot/logger"
	"github.com/hellodex/tradingbot/model"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

type Order struct {
	WalletID          string `json:"walletId"`
	WalletKey         string `json:"walletKey"`
	ChainCode         string `json:"ChainCode"`
	OrderType         int    `json:"orderType"` // 1:sell 0:buy
	LimitOrderType    int    `json:"limitOrderType"`
	TradeType         string `json:"tradeType"` // L
	FromTokenAddress  string `json:"fromTokenAddress"`
	FromTokenDecimals int    `json:"fromTokenDecimals"`
	ToTokenAddress    string `json:"toTokenAddress"`
	ToTokenDecimals   int    `json:"toTokenDecimals"`
	FromTokenAmount   string `json:"fromTokenAmount"` // sell amount
	TargetPrice       string `json:"targetPrice"`
	UiType            int    `json:"uiType"`
	ProfitFlag        int    `json:"profitFlag"` // 0
}

func (o *Order) IsAmountSet() bool {
	return o.FromTokenAmount != ""
}

func (o *Order) IsTargetPriceSet() bool {
	return o.TargetPrice != ""
}

var (
	ErrNewOrder    = errors.New("挂单失败！")
	ErrCancelOrder = errors.New("取消委托失败! ")
)

func (o *Order) SendOrder(userInfo model.GetUserResp) error {
	// setting order default info
	o.TradeType = "L"
	o.UiType = 1
	o.ProfitFlag = 0

	// change wrapped to symbolAddress like Sol11**** to 111****
	chains, err := GetChainConfigs()
	if err != nil {
		log.Error().Err(err).Send()
		return ErrNewOrder
	}

	// switch wrappedAddress to symbolAddress
	// change from token address
	addressMap := map[string]string{}
	for _, chain := range chains.Data {
		wrap := strings.ToLower(chain.Wrapped)
		symbolAddress := strings.ToLower(chain.SymbolAddress)
		addressMap[wrap] = symbolAddress
	}

	if a, ok := addressMap[strings.ToLower(o.FromTokenAddress)]; ok {
		o.FromTokenAddress = a
	}
	if a, ok := addressMap[strings.ToLower(o.ToTokenAddress)]; ok {
		o.ToTokenAddress = a
	}

	header := makeHeader()
	AddBeaer(&header, userInfo)
	bodyBytes, err := json.Marshal(o)
	if err != nil {
		log.Error().Err(err).Send()
		return ErrNewOrder
	}
	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/trade/createOrder",
		Method:  "POST",
		Headers: header,
		Body:    bodyBytes,
	}
	log.Debug().Func(func(e *zerolog.Event) {
		logger.WithTokenCategory(e).Func(func(e *zerolog.Event) {
			e.Str("route", "createOrder").Interface("request order", o).Send()
		})
	})

	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Error().Err(err).Send()
		return ErrNewOrder
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Send()
		return ErrNewOrder
	}

	// log result
	log.Debug().Func(func(e *zerolog.Event) {
		logger.WithTokenCategory(e).Func(func(e *zerolog.Event) {
			e.Str("route", "createOrder").RawJSON("order result", data).Send()
		})
	})

	results := gjson.GetManyBytes(data, "code", "msg")
	if len(results) < 2 {
		log.Error().Msg("parse results err len less then 2")
		return ErrNewOrder
	}

	if code := results[0].Int(); code == 200 {
		log.Debug().Int("code", int(code)).Send()
		return nil
	} else {
		msg := results[1].String()
		log.Debug().Str("msg", msg).Send()
		return errors.New(msg)
	}
}

func CancelOrder(orderNo string, userInfo model.GetUserResp) ([]byte, error) {
	header := makeHeader()
	AddBeaer(&header, userInfo)
	reqData := map[string]string{
		"orderNo": orderNo,
	}
	bodyBytes, err := json.Marshal(reqData)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, ErrCancelOrder
	}
	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/order/cancelOrder",
		Method:  "POST",
		Headers: header,
		Body:    bodyBytes,
	}
	log.Debug().Func(func(e *zerolog.Event) {
		logger.WithTokenCategory(e).Func(func(e *zerolog.Event) {
			e.Str("route", "cancel order").Interface("request cancel order", bodyBytes).Send()
		})
	})

	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, ErrCancelOrder
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, ErrCancelOrder
	}

	// log result
	log.Debug().Func(func(e *zerolog.Event) {
		logger.WithTokenCategory(e).Func(func(e *zerolog.Event) {
			e.Str("route", "cancel order").RawJSON("cancel order result", data).Send()
		})
	})

	results := gjson.GetManyBytes(data, "code", "msg")
	if len(results) < 2 {
		log.Error().Msg("parse results err len less then 2")
		return nil, ErrCancelOrder
	}

	if code := results[0].Int(); code == 200 {
		log.Debug().Int("code", int(code)).Send()
		return data, nil
	}

	return data, nil
}
