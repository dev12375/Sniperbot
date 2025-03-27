package api

func GetChainNameFallbackCode(chainCode string) string {
	chainConfigs, err := GetChainConfigs()
	if err != nil {
		return chainCode
	}
	for _, chain := range chainConfigs.Data {
		if chainCode == chain.ChainCode {
			return chain.Chain
		}
	}
	return chainCode
}
