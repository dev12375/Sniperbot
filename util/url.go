package util

import "strings"

func GetChainScanUrl(chainCode string, hash string) string {
	var baseUrl string

	switch strings.ToUpper(chainCode) {
	case "SOLANA":
		baseUrl = "https://solscan.io/tx/"
	case "BSC":
		baseUrl = "https://bscscan.com/tx/"
	case "ETH":
		baseUrl = "https://etherscan.io/tx/"
	case "BASE":
		baseUrl = "https://basescan.org/tx/"
	case "ARBITRUM":
		baseUrl = "https://arbiscan.io/tx/"
	case "OPTIMISM":
		baseUrl = "https://optimistic.etherscan.io/tx/"
	default:
		return ""
	}

	return baseUrl + hash
}

var AdminUrl string = `<a href="https://t.me/HelloDex_cn">点击联系客服</a>`
