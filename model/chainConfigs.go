package model

import "encoding/json"

func UnmarshalChainConfigs(data []byte) (ChainConfigs, error) {
	var r ChainConfigs
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ChainConfigs) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type ChainConfigs struct {
	Code int64         `json:"code"`
	Data []ChainConfig `json:"data"`
	Msg  string        `json:"msg"`
}

type ChainConfig struct {
	Address        string `json:"address"`
	ApproveAddress string `json:"approveAddress"`
	Browser        string `json:"browser"`
	Chain          string `json:"chain"`
	ChainCode      string `json:"chainCode"`
	ChainID        int64  `json:"chainId"`
	Decimals       int64  `json:"decimals"`
	Gas            string `json:"gas"`
	GasLimit       int64  `json:"gasLimit"`
	ID             int64  `json:"id"`
	Logo           string `json:"logo"`
	Network        string `json:"network"`
	Proxy          string `json:"proxy"`
	RPC            string `json:"rpc"`
	RPCProxy       string `json:"rpcProxy"`
	SendGasLimit   int64  `json:"sendGasLimit"`
	SendGasPrice   string `json:"sendGasPrice"`
	Sort           int64  `json:"sort"`
	Symbol         string `json:"symbol"`
	SymbolAddress  string `json:"symbolAddress"`
	TokenLogo      string `json:"tokenLogo"`
	Wrapped        string `json:"wrapped"`
	WssRPC         string `json:"wssRpc"`
}
