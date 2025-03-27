package store

var UuidMap = make(map[int64]string)

var userWallets = make(map[int64]map[string]float64)

var PreSelectWalletAddress string
