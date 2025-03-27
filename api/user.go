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

func GetMyCommissionSummary(userInfo model.GetUserResp) ([]byte, error) {
	header := makeHeader()
	AddBeaer(&header, userInfo)
	reqData := map[string]any{}

	log.Debug().Interface("getMyCommissionSummaryHistory requestData", reqData).Send()
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		log.Error().Err(err).Msg("list trade history marshal reqData err")
		return nil, err
	}

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/internal/tgUser/getMyCommissionSummary",
		Method:  "POST",
		Headers: header,
		Body:    jsonData,
	}
	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Debug().Err(err).Msg("failed to send trade history request")
		return nil, err
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, err
	}
	log.Debug().RawJSON("getMyCommissionSummaryHistory result", data).Send()

	if gjson.GetBytes(data, "code").Int() == 404 {
		return nil, errors.New("not 200")
	}

	return data, nil
}

func GetMyCommissionDetail(userInfo model.GetUserResp) ([]byte, error) {
	header := makeHeader()
	AddBeaer(&header, userInfo)
	reqData := map[string]any{}

	log.Debug().Interface("getCommissionDetailHistory requestData", reqData).Send()
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		log.Error().Err(err).Msg("list trade history marshal reqData err")
		return nil, err
	}

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/internal/tgUser/getCommissionDetail",
		Method:  "POST",
		Headers: header,
		Body:    jsonData,
	}
	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Debug().Err(err).Msg("failed to send trade history request")
		return nil, err
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, err
	}
	log.Debug().RawJSON("getCommissionDetailHistory result", data).Send()

	if gjson.GetBytes(data, "code").Int() == 404 {
		return nil, errors.New("not 200")
	}

	return data, nil
}

func SubmitWithdraw(chainCode string, walletAddress string, amount int64, userInfo model.GetUserResp) ([]byte, error) {
	header := makeHeader()
	AddBeaer(&header, userInfo)
	reqData := map[string]any{
		"chainCode":     chainCode,
		"walletAddress": walletAddress,
		"amount":        amount,
	}

	log.Debug().Interface("submitWithdrawHistory requestData", reqData).Send()
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		log.Error().Err(err).Msg("list trade history marshal reqData err")
		return nil, err
	}

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/internal/tgUser/submitWithdraw",
		Method:  "POST",
		Headers: header,
		Body:    jsonData,
	}
	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Debug().Err(err).Msg("failed to send trade history request")
		return nil, err
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, err
	}
	log.Debug().RawJSON("submitWithdrawHistory result", data).Send()

	if gjson.GetBytes(data, "code").Int() == 404 {
		return nil, errors.New("not 200")
	}

	return data, nil
}
