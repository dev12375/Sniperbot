package model

import (
	"encoding/json"

	"github.com/spf13/cast"
)

type GetUserResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		UUID              string              `json:"uuid"`
		Slippage          string              `json:"slippage"`
		TgDefaultWalletId string              `json:"tgDefaultWalletId"`
		MainnetToken      MainNetToken        `json:"mainnetToken"`
		Wallets           map[string][]Wallet `json:"wallets"`
		TokenInfo         TokenInfo           `json:"tokenInfo"`
		InviteCode        string              `json:"inviteCode"`
		SubscribeSetting  []string            `json:"subscribeSetting"`
	} `json:"data"`
}

// use for setting 2 is 2%
func (userInfo *GetUserResp) ToPercentage(num float64) string {
	return cast.ToString(num / 100)
}

// use for view 0.02 is 2% handle from GetuserResp
func (userInfo *GetUserResp) FromPercentage(num string) string {
	return cast.ToString(cast.ToFloat64(num) * 100)
}

type MainNetToken struct {
	Symbol  string `json:"symbol"`
	Price   string `json:"price"`
	Balance string `json:"balance"`
}

type Wallet struct {
	WalletId  string `json:"walletId"`
	WalletKey string `json:"walletKey"`
	GroupId   int    `json:"groupId"`
	UUID      string `json:"uuid"`
	Wallet    string `json:"wallet"`
	ChainCode string `json:"chainCode"`
	GroupName string `json:"groupName"`
}

func (w *Wallet) String() string {
	jsonB, _ := json.Marshal(w)
	return string(jsonB)
}

type TokenInfo struct {
	TokenName           string `json:"tokenName"`
	TokenValue          string `json:"tokenValue"`
	IsLogin             bool   `json:"isLogin"`
	LoginID             string `json:"loginId"`
	LoginType           string `json:"loginType"`
	TokenTimeout        int    `json:"tokenTimeout"`
	SessionTimeout      int    `json:"sessionTimeout"`
	TokenSessionTimeout int    `json:"tokenSessionTimeout"`
	TokenActiveTimeout  int    `json:"tokenActiveTimeout"`
	LoginDevice         string `json:"loginDevice"`
}
