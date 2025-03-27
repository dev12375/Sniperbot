package model

type UserWalletData struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		ChainCode           string `json:"chainCode"`
		TokenAddress        string `json:"tokenAddress"`
		Symbol              string `json:"symbol"`
		TotalBuyAmount      string `json:"totalBuyAmount"`
		TotalSellAmount     string `json:"totalSellAmount"`
		AverageBuyMarketCap string `json:"averageBuyMarketCap"`
		NowMarketCap        string `json:"nowMarketCap"`
		Amount              string `json:"amount"`
		AveragePrice        string `json:"averagePrice"`
		TotalEarn           string `json:"totalEarn"`
		TotalEarnRate       string `json:"totalEarnRate"`
		Status              int    `json:"status"`
	} `json:"data"`
}
