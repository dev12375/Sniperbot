package model

type TradeHistory struct {
	Code int                 `json:"code"`
	Msg  string              `json:"msg"`
	Data []TradeHistoryInner `json:"data"`
}

type TradeHistoryInner struct {
	BaseSymbol    string `json:"baseSymbol"`
	ChainCode     string `json:"chainCode"`
	QuoteSymbol   string `json:"quoteSymbol"`
	MarketCap     string `json:"marketCap"`
	Price         string `json:"price"`
	Amount        string `json:"amount"`
	TradeType     string `json:"tradeType"`
	Direction     string `json:"direction"`
	FromOrderNo   string `json:"fromOrderNo"`
	Volume        string `json:"volume"`
	Logo          string `json:"logo"`
	Status        string `json:"status"`
	Timestamp     string `json:"timestamp"`
	Tx            string `json:"tx"`
	ProfitFlag    string `json:"profitFlag"`
	OrderNo       string `json:"orderNo"`
	UIType        int64  `json:"uiType"`
	OrderStatusUI string `json:"orderStatusUI"`
}
