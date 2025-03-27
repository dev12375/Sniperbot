package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/duke-git/lancet/v2/netutil"
	"github.com/hellodex/tradingbot/logger"
	"github.com/hellodex/tradingbot/model"
	"github.com/hellodex/tradingbot/store"
	"github.com/hellodex/tradingbot/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

const (
	UserProfilePrefix = "_user_profile"
)

func isValidUserProfile(profile model.GetUserResp) bool {
	return profile.Data.TokenInfo.TokenValue != ""
}

func GetUserProfile(userID int64) (model.GetUserResp, error) {
	// if profile, has := store.Get(userID, UserProfilePrefix); has {
	// 	if u, ok := profile.(model.GetUserResp); ok && isValidUserProfile(u) {
	// 		return u, nil
	// 	}
	// 	store.Delete(userID, UserProfilePrefix)
	// }

	// load from redis
	if data, ok := store.RedisGetUserProfile(userID); ok {
		var result model.GetUserResp
		err := json.Unmarshal(data, &result)
		if err != nil {
			return result, errors.New("err Unmarshal redis data to real struct")
		}
		return result, nil
	}

	var result model.GetUserResp
	operation := func() (*model.GetUserResp, error) {
		r, err := fetchUserProfile(userID)
		if err != nil {
			return nil, err
		}

		// 验证获取的数据是否有效
		if !isValidUserProfile(r) {
			return nil, fmt.Errorf("invalid user profile")
		}

		return &r, nil
	}

	b := backoff.NewExponentialBackOff()
	b.MaxInterval = 10 * time.Second
	b.InitialInterval = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := backoff.Retry(ctx, operation, backoff.WithMaxTries(1), backoff.WithBackOff(b))
	if err != nil {
		return result, fmt.Errorf("failed to fetch user profile after retries: %w", err)
	}

	result = *r
	// store.Set(userID, UserProfilePrefix, result, 30*time.Minute)

	// store in redis
	if ok := store.RedisSetUserProfile(userID, result); !ok {
		log.Debug().
			Int64("user_id", userID).
			Msg("Failed to set user profile in Redis")
	}

	return result, nil
}

func UpdateUserProfile(userID int64, userInfo model.GetUserResp) error {
	var result struct {
		Code int
		Msg  string
	}

	header := makeHeader()
	AddBeaer(&header, userInfo)
	requestBody := map[string]any{
		"uuid":              userInfo.Data.UUID,
		"slippage":          userInfo.Data.Slippage,
		"tgDefaultWalletId": userInfo.Data.TgDefaultWalletId,
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		log.Error().Err(err).Send()
		return fmt.Errorf("构建请求体失败: %w", err)
	}
	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/internal/tgUser/updateUserProfile",
		Method:  "POST",
		Headers: header,
		Body:    bodyBytes,
	}

	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Error().Err(err).Send()
		return err
	}

	derr := client.DecodeResponse(resp, &result)
	if derr != nil {
		return derr
	}
	if result.Code != 200 {
		return errors.New("更新失败，请联系管理员")
	}

	// store.Delete(userID, UserProfilePrefix)
	store.RedisDeleteUserProfile(userID)

	return nil
}

func GetTokensByWalletAddress(walletAddress, chainCode string, userInfo model.GetUserResp) (model.GetAddressTokens, error) {
	var result model.GetAddressTokens
	header := makeHeader()
	requestBody := map[string]any{
		"walletAddress": walletAddress,
		"chainCode":     chainCode,
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return result, fmt.Errorf("构建请求体失败: %w", err)
	}
	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/appv2/getTokensByWalletAddress",
		Method:  "POST",
		Headers: header,
		Body:    bodyBytes,
	}

	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		return result, err
	}

	derr := client.DecodeResponse(resp, &result)
	if derr != nil {
		return result, derr
	}

	return result, nil
}

func GetTokenInfoByWalletAddress(tokenAddress, walletAddress, chainCode string, userInfo model.GetUserResp) (model.DataTokenInfo, error) {
	tokens, err := GetTokensByWalletAddress(walletAddress, chainCode, userInfo)
	if err != nil {
		return model.DataTokenInfo{}, err
	}

	for _, token := range tokens.Data {
		if token.Address == tokenAddress {
			return token, nil
		}
	}

	return model.DataTokenInfo{}, nil
}

func GetPositionByWalletAddress(walletAddress, baseAddress, chainCode string, userInfo model.GetUserResp) (model.PositionByWalletAddress, error) {
	var result model.PositionByWalletAddress

	header := makeHeader()
	AddBeaer(&header, userInfo)
	requestBody := map[string]any{
		"walletAddress": walletAddress,
		"baseAddress":   baseAddress,
		"chainCode":     chainCode,
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		log.Error().Err(err).Send()
		return result, fmt.Errorf("构建请求体失败: %w", err)
	}
	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/order/getPositionByWalletAddress",
		Method:  "POST",
		Headers: header,
		Body:    bodyBytes,
	}
	log.Debug().Func(func(e *zerolog.Event) {
		logger.WithTokenCategory(e).Func(func(e *zerolog.Event) {
			jsonD, err := json.Marshal(requestBody)
			if err != nil {
				e.Err(err).Msg("err in getPositionByWalletAddress debug print")
				return
			}
			e.Str("route", "getPositionByWalletAddress").RawJSON("requestBody", jsonD).Send()
		})
	})

	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Error().Err(err).Send()
		return result, err
	}

	derr := client.DecodeResponse(resp, &result)
	if derr != nil {
		log.Error().Err(derr).Send()
		return result, derr
	}

	// log result
	log.Debug().Func(func(e *zerolog.Event) {
		logger.WithTokenCategory(e).Func(func(e *zerolog.Event) {
			jsonD, err := json.Marshal(result)
			if err != nil {
				e.Err(err).Msg("err in getPositionByWalletAddress debug print")
				return
			}
			e.Str("route", "getPositionByWalletAddress").RawJSON("requestBody", jsonD).Send()
		})
	})

	return result, nil
}

func GetChainConfigs() (model.ChainConfigs, error) {
	var result model.ChainConfigs
	header := makeHeader()
	requestBody := map[string]any{}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println(err)
		return result, err
	}
	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/appv2/getChainConfigs",
		Method:  "POST",
		Headers: header,
		Body:    bodyBytes,
	}

	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Error().Err(err).Send()
	}

	derr := client.DecodeResponse(resp, &result)
	if derr != nil {
		log.Error().Err(derr).Send()
		return result, derr

	}

	return result, nil
}

func ListUserDefaultWalletsSwitch(userInfo model.GetUserResp, chainCode string) []model.Wallet {
	if util.IsDebug() {
		log.Debug().Msg("ListUserDefaultWalletsSwitch")
	}
	return ListUserChainWallets(userInfo, chainCode)
}

func ListUserChainWallets(userInfo model.GetUserResp, chainCode string) []model.Wallet {
	for chain, allWallets := range userInfo.Data.Wallets {
		if chainCode == chain {
			return allWallets
		}
	}
	return nil
}

var (
	ErrSendSwap      = errors.New("失败，请联系客服")
	ErrGetSwapResult = errors.New("获取交易结果失败")
)

func SendSwap(swap model.Swap, userInfo model.GetUserResp) ([]byte, error) {
	// change wrapped to symbolAddress like Sol11**** to 111****
	chains, err := GetChainConfigs()
	if err != nil {
		log.Error().Err(err).Send()
		return nil, ErrSendSwap
	}

	// switch wrappedAddress to symbolAddress
	// change from token address
	addressMap := map[string]string{}
	for _, chain := range chains.Data {
		wrap := strings.ToLower(chain.Wrapped)
		symbolAddress := strings.ToLower(chain.SymbolAddress)
		addressMap[wrap] = symbolAddress
	}

	if a, ok := addressMap[strings.ToLower(swap.FromTokenAddress)]; ok {
		swap.FromTokenAddress = a
	}
	if a, ok := addressMap[strings.ToLower(swap.ToTokenAddress)]; ok {
		swap.ToTokenAddress = a
	}

	// WARN:
	log.Debug().Func(func(e *zerolog.Event) {
		txe := logger.WithTxCategory(e)
		jsonD, err := json.Marshal(swap)
		if err != nil {
			txe.Err(err).Send()
			return
		}

		txe.RawJSON("swap requestBody", jsonD).Send()
	})

	jsonData, err := json.Marshal(swap)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, err
	}

	header := makeHeader()
	AddBeaer(&header, userInfo)
	req := &netutil.HttpRequest{
		RawURL:  BuildBasicUrl() + "/api/auth/trade/swap",
		Method:  "POST",
		Headers: header,
		Body:    jsonData,
	}
	client := netutil.NewHttpClient()
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, ErrSendSwap
	}

	defer resp.Body.Close()

	// ignore error return struct, not 200 err
	if resp.StatusCode != http.StatusOK {
		return nil, ErrSendSwap
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Debug().Func(func(e *zerolog.Event) {
		txe := logger.WithTxCategory(e)
		var prettyJSON map[string]any
		err := json.Unmarshal(body, &prettyJSON)
		if err != nil {
			txe.Err(err).Send()
			return
		}
		jsonD, err := json.Marshal(prettyJSON)
		if err != nil {
			txe.Err(err).Send()
			return
		}

		txe.RawJSON("swap result", jsonD).Send()
	})

	if resultCode := gjson.GetBytes(body, "code").Int(); resultCode != 200 && resultCode != 102 {
		return nil, ErrSendSwap
	}

	return body, nil
}
