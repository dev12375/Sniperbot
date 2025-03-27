package model

import "encoding/json"

func UnmarshalGetAddressTokens(data []byte) (GetAddressTokens, error) {
	var r GetAddressTokens
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *GetAddressTokens) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type GetAddressTokens struct {
	Msg  string          `json:"msg"`
	Code int64           `json:"code"`
	Data []DataTokenInfo `json:"data"`
}

type DataTokenInfo struct {
	Address     string `json:"address"`
	Price       string `json:"price"`
	Chg1D       string `json:"chg1d"`
	Amount      string `json:"amount"`
	RawAmount   string `json:"rawAmount"`
	Logo        string `json:"logo"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Decimals    string `json:"decimals"`
	ChainCode   string `json:"chainCode"`
	TotalAmount string `json:"totalAmount"`
	Volume      string `json:"-"`
	PairAddress string `json:"pairAddress"`
}
