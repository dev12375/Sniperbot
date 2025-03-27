package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/duke-git/lancet/v2/netutil"
	"github.com/hellodex/tradingbot/config"
	"github.com/hellodex/tradingbot/logger"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

func BuildBasicUrl() string {
	return config.YmlConfig.Env.ApiEndpoint
}

func HttpGet(path string) *netutil.HttpRequest {
	return &netutil.HttpRequest{
		RawURL: BuildBasicUrl() + path,
		Method: "GET",
	}
}

func signAppInfo(ts string) string {
	sha256Hash := sha256.New()
	params := config.YmlConfig.App.Appid + ts + config.YmlConfig.App.Ver + config.YmlConfig.App.Appkey

	sha256Hash.Write([]byte(params))
	digest := sha256Hash.Sum(nil)

	return hex.EncodeToString(digest)
}

func calculateSign(ts int64) string {
	code := config.YmlConfig.Channel.Code
	version := config.YmlConfig.Channel.Version
	key := config.YmlConfig.Channel.Key
	sign := cryptor.Sha256(code + convertor.ToString(ts) + version + key)
	return sign
}

func makeHeader() http.Header {
	ts := time.Now().UnixMilli()
	sign := calculateSign(ts)

	header := http.Header{}
	header.Add("Content-Type", "application/json")
	header.Add("Sign", sign)
	header.Add("Ts", cast.ToString(ts))
	header.Add("Channel", config.YmlConfig.Channel.Code)
	header.Add("Version", config.YmlConfig.Channel.Version)

	return header
}

func AddBeaer(header *http.Header, userInfo model.GetUserResp) *http.Header {
	header.Add("Authorization", "Bearer "+userInfo.Data.TokenInfo.TokenValue)
	return header
}

func fetchUserProfile(userID int64) (model.GetUserResp, error) {
	var result model.GetUserResp

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

	log.Trace().RawJSON("fetchUserProfile requestBody", bodyBytes).Send()

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/internal/tgUser/getUser",
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

	err = json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}
	log.Trace().RawJSON("fetchUserProfile raw result", data).Send()

	logger.NewStdLog("/internal/tgUser/getUser", bodyBytes, data)

	return result, nil
}

func BindUserInvitationCode(userID int64, code string) (model.GetUserResp, error) {
	var result model.GetUserResp

	ec, err := util.Encrypt(userID)
	if err != nil {
		log.Debug().Err(err).Msg("encrypt userID err")
		return result, err
	}

	header := makeHeader()
	ts := cast.ToString(time.Now().UnixMilli())
	signTs := signAppInfo(ts)
	requestBody := map[string]any{
		"accountId":      ec,
		"ver":            config.YmlConfig.App.Ver,
		"appId":          config.YmlConfig.App.Appid,
		"sign":           signTs,
		"ts":             ts,
		"invitationCode": code,
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

	log.Trace().RawJSON("fetchUserProfile requestBody", bodyBytes).Send()

	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/internal/tgUser/getUser",
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

	err = json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}
	log.Trace().RawJSON("fetchUserProfile raw result", data).Send()

	store.RedisDeleteUserProfile(userID)

	return result, nil
}
