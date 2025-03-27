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

func ListUserTokenSubscribe(chainCode string, userInfo model.GetUserResp) (any, error) {
	var result map[string]any

	header := makeHeader()
	AddBeaer(&header, userInfo)
	reqData := map[string]any{
		"chainCode": chainCode,
	}

	log.Debug().Interface("listUserTokenSubscribe requestData", reqData).Send()
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		log.Error().Err(err).Msg("list trade history marshal reqData err")
		return result, err
	}

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/sub/listUserTokenSubscribe",
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
	log.Debug().RawJSON("listUserTokenSubscribe result", data).Send()

	if gjson.GetBytes(data, "code").Int() == 404 {
		return model.OpenOrdersHistory{}, errors.New("not 200")
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}

	return data, nil
}

func GetUserTokenSubscribe(userId int64, chainCode string, baseAddress string, monitorType string, userInfo model.GetUserResp) (data []byte, hasSubscribe bool, err error) {
	header := makeHeader()
	AddBeaer(&header, userInfo)
	reqData := map[string]any{
		"chainCode":   chainCode,
		"baseAddress": baseAddress,
		"type":        monitorType,
	}

	log.Debug().Interface("getUserTokenSubscribe requestData", reqData).Send()
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		log.Error().Err(err).Msg("list trade history marshal reqData err")
		return nil, false, err
	}

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/sub/getUserSubscribe",
		Method:  "POST",
		Headers: header,
		Body:    jsonData,
	}
	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Debug().Err(err).Msg("failed to send trade history request")
		return nil, false, err
	}

	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, false, err
	}
	log.Debug().RawJSON("getUserTokenSubscribe result", data).Send()

	if gjson.GetBytes(data, "code").Int() == 404 {
		return nil, false, errors.New("not 200")
	}

	info := gjson.GetBytes(data, "data.info")
	baseTokenSymbol := info.Get("baseToken.symbol").String()
	log.Debug().Str("baseTokenSymbol", baseTokenSymbol).Send()
	subscribe := gjson.GetBytes(data, "data.subscribe")

	// handle subscribe info
	if subscribe.Exists() && subscribe.IsObject() {
		log.Debug().Msg("subscribe exists && IsObject")

		if subscribe.Raw == "{}" || len(subscribe.Map()) == 0 {
			log.Debug().Msg("subscribe object is empty")
			return data, false, nil
		}

		log.Debug().Interface("user subscribe info", subscribe.String()).Send()
		return data, true, nil
	}

	return data, false, nil
}

func UpdateCommonSubscribe(reqData model.AISubscribeReqData, userInfo model.GetUserResp) ([]byte, error) {
	header := makeHeader()
	AddBeaer(&header, userInfo)
	chainCode := reqData.ChainCode
	baseAddress := reqData.BaseAddress
	symbol := reqData.Symbol
	noticeType := reqData.NoticeType
	monitorType := reqData.MonitorType
	targetPrice := reqData.TargetPrice
	optionData := reqData.Data
	reqDataMap := map[string]any{
		"chainCode":   chainCode,
		"baseAddress": baseAddress,
		"symbol":      symbol,
		"noticeType":  noticeType,
		// "startPrice":  "0.00000001",
		"type":   monitorType,
		"status": 1,
	}
	if reqData.MonitorType == "price" {
		reqDataMap["targetPrice"] = targetPrice
	} else {
		reqDataMap["data"] = optionData
	}

	log.Debug().Interface("updateCommonSubscribe requestData", reqDataMap).Send()
	jsonData, err := json.Marshal(reqDataMap)
	if err != nil {
		log.Error().Err(err).Msg("list trade history marshal reqData err")
		return nil, err
	}

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/sub/updateCommonSubscribe",
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
	log.Debug().RawJSON("updateCommonSubscribe result", data).Send()

	return data, nil
}

func UpdateUserSubscribeSetting(channels []string, userInfo model.GetUserResp) (any, error) {
	var result map[string]any

	header := makeHeader()
	AddBeaer(&header, userInfo)
	reqData := map[string]any{
		"channels": channels,
	}

	log.Debug().Interface("updateUserSubscribeSetting requestData", reqData).Send()
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		log.Error().Err(err).Msg("list trade history marshal reqData err")
		return result, err
	}

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/sub/updateUserSubscribeSetting",
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
	log.Debug().RawJSON("updateUserSubscribeSetting result", data).Send()

	if gjson.GetBytes(data, "code").Int() == 404 {
		return model.OpenOrdersHistory{}, errors.New("not 200")
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}

	return data, nil
}

//	{
//	 "chainCode": "SOLANA",
//	 "baseAddress": "asfaef3fcsf42twef",
//	 "type": "chg"
//	}
func PauseUserTokenSubscribe(reqData map[string]string, userInfo model.GetUserResp) (any, error) {
	var result map[string]any

	header := makeHeader()
	AddBeaer(&header, userInfo)
	log.Debug().Interface("pauseUserTokenSubscribe requestData", reqData).Send()

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		log.Error().Err(err).Msg("list trade history marshal reqData err")
		return result, err
	}

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/sub/pauseUserTokenSubscribe",
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
	log.Debug().RawJSON("pauseUserTokenSubscribe result", data).Send()

	if gjson.GetBytes(data, "code").Int() == 404 {
		return model.OpenOrdersHistory{}, errors.New("not 200")
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}

	return data, nil
} //	{
//	 "chainCode": "SOLANA",
//	 "baseAddress": "asfaef3fcsf42twef",
//	 "type": "chg"
//	}
func DeleteUserTokenSubscribe(reqData map[string]string, userInfo model.GetUserResp) (any, error) {
	var result map[string]any

	header := makeHeader()
	AddBeaer(&header, userInfo)
	log.Debug().Interface("deleteUserTokenSubscribe requestData", reqData).Send()

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		log.Error().Err(err).Msg("list trade history marshal reqData err")
		return result, err
	}

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/sub/deleteUserTokenSubscribe",
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
	log.Debug().RawJSON("deleteUserTokenSubscribe result", data).Send()

	if gjson.GetBytes(data, "code").Int() == 404 {
		return model.OpenOrdersHistory{}, errors.New("not 200")
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}

	return data, nil
}
