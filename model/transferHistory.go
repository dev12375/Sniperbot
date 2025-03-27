package model

import "encoding/json"

func UnmarshalTransferHistory(data []byte) (TransferHistory, error) {
	var r TransferHistory
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *TransferHistory) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type TransferHistory struct {
	Code int64                  `json:"code"`
	Msg  string                 `json:"msg"`
	Data []TransferHistoryInner `json:"data"`
}

type TransferHistoryInner struct {
	Chain         string `json:"chain"`
	ChainCode     string `json:"chainCode"`
	ChainID       string `json:"chainId"`
	WalletAddress string `json:"walletAddress"`
	TokenAddress  string `json:"tokenAddress"`
	ToAddress     string `json:"toAddress"`
	Number        string `json:"number"`
	Fee           string `json:"fee"`
	Remark        string `json:"remark"`
	Nonce         string `json:"nonce"`
	Amount        string `json:"amount"`
	Status        int64  `json:"status"`
	Hash          string `json:"hash"`
	Timestamp     int64  `json:"timestamp"`
	OrderStatusUI string `json:"orderStatusUI"`
}
