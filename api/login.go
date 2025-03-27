package api

import (
	"encoding/json"
	"io"
	"time"

	"github.com/duke-git/lancet/v2/netutil"
	"github.com/hellodex/tradingbot/config"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

func GetTg2WebLoginToken(userID int64, platform string) (any, error) {
	var result any

	ec, err := util.Encrypt(userID)
	if err != nil {
		log.Debug().Err(err).Msg("encrypt userID err")
		return result, err
	}

	header := makeHeader()
	ts := cast.ToString(time.Now().UnixMilli())
	signTs := signAppInfo(ts)
	requestBody := map[string]any{
		"accountId": ec,
		"platform":  platform,
		"ver":       config.YmlConfig.App.Ver,
		"appId":     config.YmlConfig.App.Appid,
		"sign":      signTs,
		"ts":        ts,
	}
	header.Add("SIG", signTs)
	header.Add("TS", ts)
	header.Add("VER", config.YmlConfig.App.Ver)
	header.Add("APP_ID", config.YmlConfig.App.Appid)

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		log.Debug().Err(err).Msg("marshal requestBody err")
		return result, err
	}

	log.Trace().RawJSON("getTg2WebLoginToken requestBody", bodyBytes).Send()

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/internal/tgUser/getTg2WebLoginToken",
		Method:  "POST",
		Headers: header,
		Body:    bodyBytes,
	}

	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	log.Trace().RawJSON("getTg2WebLoginToken raw result", data).Send()

	return data, nil
}
