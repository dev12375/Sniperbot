package model

import "encoding/json"

func UnmarshalOpenOrders(data []byte) (OpenOrdersHistory, error) {
	var r OpenOrdersHistory
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *OpenOrdersHistory) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type OpenOrdersHistory struct {
	Code int64            `json:"code"`
	Msg  string           `json:"msg"`
	Data []OpenOrderInner `json:"data"`
}

type OpenOrderInner struct {
	MarketCap         string `json:"marketCap"`
	OrderNo           string `json:"orderNo"`
	Price             string `json:"price"`
	Amount            string `json:"amount"`
	Volume            string `json:"volume"`
	FromTokenAddress  string `json:"fromTokenAddress"`
	FromTokenAmount   string `json:"fromTokenAmount"`
	FromTokenSymbol   string `json:"fromTokenSymbol"`
	FromTokenDecimals int64  `json:"fromTokenDecimals"`
	ToTokenDecimals   int64  `json:"toTokenDecimals"`
	ToTokenAddress    string `json:"toTokenAddress"`
	ToTokenAmount     string `json:"toTokenAmount"`
	ToTokenSymbol     string `json:"toTokenSymbol"`
	BaseSymbol        string `json:"baseSymbol"`
	BaseAddress       string `json:"baseAddress"`
	Logo              string `json:"logo"`
	ChainCode         string `json:"chainCode"`
	LimitType         string `json:"limitType"`
	Timestamp         string `json:"timestamp"`
	LastUpdateTime    string `json:"lastUpdateTime"`
	Status            string `json:"status"`
	Tx                string `json:"tx"`
	OrderType         string `json:"orderType"`
	TradeType         string `json:"tradeType"`
	FromOrderNo       string `json:"fromOrderNo"`
	ProfitFlag        string `json:"profitFlag"`
	UIType            int64  `json:"uiType"`
	OrderStatusUI     string `json:"orderStatusUI"`
}
