package api

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/duke-git/lancet/v2/netutil"
	"github.com/hellodex/tradingbot/model"
	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

func ListTradeHistory(walletID float64, userInfo model.GetUserResp) (model.TradeHistory, error) {
	var result model.TradeHistory

	header := makeHeader()
	AddBeaer(&header, userInfo)
	reqData := map[string]any{
		"walletId": walletID,
	}

	log.Debug().Interface("listTradeHistories requestData", reqData).Send()
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		log.Error().Err(err).Msg("list trade history marshal reqData err")
		return result, err
	}

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/order/listTradeHistories",
		Method:  "POST",
		Headers: header,
		Body:    jsonData,
	}
	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Debug().Err(err).Msg("failed to send trade history request")
		return result, err
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}
	log.Debug().RawJSON("listTradeHistories result", data).Send()

	if gjson.GetBytes(data, "code").Int() == 404 {
		return model.TradeHistory{}, errors.New("not 200")
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}

	return result, nil
}

func ListTransferHistory(walletID float64, chainCode string, userInfo model.GetUserResp) (model.TransferHistory, error) {
	var result model.TransferHistory

	header := makeHeader()
	AddBeaer(&header, userInfo)
	reqData := map[string]any{
		"walletId":  walletID,
		"chainCode": chainCode,
	}

	log.Debug().Interface("listTransferHistory requestData", reqData).Send()
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		log.Error().Err(err).Msg("list trade history marshal reqData err")
		return result, err
	}

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/order/listTransferHistory",
		Method:  "POST",
		Headers: header,
		Body:    jsonData,
	}
	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Debug().Err(err).Msg("failed to send trade history request")
		return result, err
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}
	log.Debug().RawJSON("listTransferHistory result", data).Send()

	if gjson.GetBytes(data, "code").Int() == 404 {
		return model.TransferHistory{}, errors.New("not 200")
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}

	return result, nil
}

func ListOpeningOrders(walletID float64, userInfo model.GetUserResp) (model.OpenOrdersHistory, error) {
	var result model.OpenOrdersHistory

	header := makeHeader()
	AddBeaer(&header, userInfo)
	reqData := map[string]any{
		"walletId": walletID,
	}

	log.Debug().Interface("listOpeningOrdersHistory requestData", reqData).Send()
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		log.Error().Err(err).Msg("list trade history marshal reqData err")
		return result, err
	}

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/order/listOpeningOrders",
		Method:  "POST",
		Headers: header,
		Body:    jsonData,
	}
	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Debug().Err(err).Msg("failed to send trade history request")
		return result, err
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}
	log.Debug().RawJSON("listOpeningOrdersHistory result", data).Send()

	if gjson.GetBytes(data, "code").Int() == 404 {
		return model.OpenOrdersHistory{}, errors.New("not 200")
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}

	return result, nil
}

func ListHistoryOrders(walletID float64, userInfo model.GetUserResp) (model.OpenOrdersHistory, error) {
	var result model.OpenOrdersHistory

	header := makeHeader()
	AddBeaer(&header, userInfo)
	reqData := map[string]any{
		"walletId": walletID,
	}

	log.Debug().Interface("listHistoryOrdersHistory requestData", reqData).Send()
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		log.Error().Err(err).Msg("list trade history marshal reqData err")
		return result, err
	}

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/order/listHistoryOrders",
		Method:  "POST",
		Headers: header,
		Body:    jsonData,
	}
	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Debug().Err(err).Msg("failed to send trade history request")
		return result, err
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}
	log.Debug().RawJSON("listHistoryOrdersHistory result", data).Send()

	if gjson.GetBytes(data, "code").Int() == 404 {
		return model.OpenOrdersHistory{}, errors.New("not 200")
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}

	return result, nil
}
