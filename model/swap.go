package model

type Swap struct {
	Amount            string  `json:"amount"`
	WalletId          string  `json:"walletId"`
	WalletKey         string  `json:"walletKey"`
	FromTokenAddress  string  `json:"fromTokenAddress"`
	FromTokenDecimals int     `json:"fromTokenDecimals"`
	ToTokenAddress    string  `json:"toTokenAddress"`
	ToTokenDecimals   int     `json:"toTokenDecimals"`
	Slippage          string  `json:"slippage"`
	Type              string  `json:"type"`
	TradeType         string  `json:"tradeType"`
	Price             string  `json:"price"`
	ProfitFlag        float64 `json:"profitFlag"` //先阶段设置为 0
}
