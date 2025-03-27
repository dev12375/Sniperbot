package model

import "encoding/json"

func UnmarshalPositionByWalletAddress(data []byte) (PositionByWalletAddress, error) {
	var r PositionByWalletAddress
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *PositionByWalletAddress) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type PositionByWalletAddress struct {
	Code int               `json:"code"`
	Msg  string            `json:"msg"`
	Data PositionDataInner `json:"data"`
}

type PositionDataInner struct {
	PairAddress         string     `json:"pairAddress"`
	Symbol              string     `json:"symbol"`
	ChainCode           string     `json:"chainCode"`
	Price               string     `json:"price"`
	Amount              string     `json:"amount"`
	Volume              string     `json:"volume"`
	RawAmount           string     `json:"rawAmount"`
	TotalBuyAmount      string     `json:"totalBuyAmount"`
	TotalBuyVolume      string     `json:"totalBuyVolume"`
	TotalSellAmount     string     `json:"totalSellAmount"`
	TotalSellVolume     string     `json:"totalSellVolume"`
	AverageBuyVolume    string     `json:"averageBuyVolume"`
	AverageBuyMarketCap string     `json:"averageBuyMarketCap"`
	Chg1D               string     `json:"chg1d"`
	Chg                 string     `json:"chg"`
	BaseToken           TokenInner `json:"baseToken"`
	QuoteToken          TokenInner `json:"quoteToken"`
	NowMarketCap        string     `json:"nowMarketCap"`
	Tvl                 string     `json:"tvl"`
	AveragePrice        string     `json:"averagePrice"`
	TotalEarn           string     `json:"totalEarn"`
	TotalEarnRate       string     `json:"totalEarnRate"`
	Status              string     `json:"status"`
	Logo                string     `json:"logo"`
	UpdateTime          string     `json:"updateTime"`
}

type TokenInner struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Logo        string `json:"logo"`
	Address     string `json:"address"`
	Decimals    string `json:"decimals"`
	TotalSupply string `json:"totalSupply"`
	ChainCode   string `json:"chainCode"`
}
