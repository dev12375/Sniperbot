package api

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/duke-git/lancet/v2/netutil"
	"github.com/hellodex/tradingbot/config"
	"github.com/hellodex/tradingbot/entity"
	test_data "github.com/hellodex/tradingbot/testData"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

type BotConfigResp struct {
	Result []entity.BotConfig `json:"result"`
}

func ListBotConfigsSwitch() []entity.BotConfig {

	if util.IsDebug() {
		botName := config.YmlConfig.Env.BotName
		userID := cast.ToInt64(config.YmlConfig.Env.BotMaker)
		apiKey := config.YmlConfig.Env.BotApiKey
		botConfigs := make([]entity.BotConfig, 0)
		bc := entity.BotConfig{
			UserId:  userID,
			BotName: botName,
			ApiKey:  apiKey,
			Type:    -1,
			Id:      0,
			Cancel: func() {
				log.Info().Msg("somethings error, exit with code 1")
				os.Exit(1)
			},
		}

		botConfigs = append(botConfigs, bc)
		return botConfigs
	}

	// not debug
	return ListBotConfigs()
}

func ListBotConfigs() []entity.BotConfig {
	req := &netutil.HttpRequest{
		RawURL: BuildBasicUrl() + "/internal/tgBot/listBotConfig",
		Method: "GET",
	}
	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Error().Err(err).Send()
	}

	var configs BotConfigResp
	client.DecodeResponse(resp, &configs)
	if err != nil {
		log.Error().Err(err).Msg("参数解析失败")
	}
	return configs.Result
}

func FreshBotConfigs() {
	// entity.BotConfigs = ListBotConfigs()
	entity.BotConfigs = ListBotConfigsSwitch()
}
func AddBotConfig(config entity.BotConfig) int8 {
	jsonData, err1 := json.Marshal(config)
	if err1 != nil {
		log.Error().Err(err1).Msg("json 序列化失败")
	}

	req := &netutil.HttpRequest{
		RawURL: BuildBasicUrl() + "/internal/tgBot/addBotConfig",
		Method: "POST",
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: jsonData,
	}
	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Error().Err(err).Msg("请求失败")
	}

	type Result struct {
		Result int8 `json:"result"`
	}
	var result Result
	derr := client.DecodeResponse(resp, &result)
	if derr != nil {
		return 0
	}
	if err != nil {
		log.Error().Err(err).Msg("参数解析失败")
	}
	return result.Result
}

type TokenInfo struct {
	BaseAddress        string      `json:"baseAddress"`
	PairAddress        string      `json:"pairAddress"`
	ChainCode          string      `json:"chainCode"`
	BaseToken          Token       `json:"baseToken"`
	QuoteToken         Token       `json:"quoteToken"`
	Name               string      `json:"name"`
	Symbol             string      `json:"symbol"`
	Price              string      `json:"price"`
	Chg5m              string      `json:"chg5m"`
	Chg1h              string      `json:"chg1h"`
	Chg4h              string      `json:"chg4h"`
	MarketCap          string      `json:"marketCap"`
	Holders            json.Number `json:"holders"`
	OpenTime           string      `json:"openTime"`
	PumpProgress       json.Number `json:"pumpProgress"`
	Dex                string      `json:"dex"`
	TopTenHolders      []Holder    `json:"topTenHolders"`
	TopTenTotalPercent json.Number `json:"topTenTotalPercent"`
}

type Token struct {
	Address  string `json:"address"`
	Decimals int8   `json:"decimals"`
	Symbol   string `json:"symbol"`
}

type Holder struct {
	Address string      `json:"address"`
	Percent json.Number `json:"percent"`
}

func SearchTokenInfoSwitch(address string) TokenInfo {
	if util.IsDebug() {
		var data TokenInfo
		err := json.Unmarshal([]byte(test_data.TokenInfo_test), &data)
		if err != nil {
			return TokenInfo{}
		}
		return data
	}

	return SearchTokenInfo(address)

}

func SearchTokenInfo(address string) TokenInfo {
	req := &netutil.HttpRequest{
		RawURL: BuildBasicUrl() + "/internal/token/tokenInfo?address=" + address,
		Method: "GET",
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}

	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Error().Err(err).Msg("请求失败")
	}

	var result TokenInfo
	derr := client.DecodeResponse(resp, &result)
	if derr != nil {
		return result
	}

	if err != nil {
		log.Error().Err(err).Msg("参数解析失败")
	}
	return result
}
