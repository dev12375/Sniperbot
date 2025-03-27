package model

import "encoding/json"

type AISubscribeReqData struct {
	SessionMessageID int
	UserId           int64
	CurrentPrice     string
	ChainCode        string
	BaseAddress      string
	Symbol           string
	NoticeType       int
	MonitorType      string
	TargetPrice      string
	Data             string // for chg,buy,sell
}

func (sq *AISubscribeReqData) JsonB() ([]byte, error) {
	return json.Marshal(sq)
}

func (sq *AISubscribeReqData) Vaild() bool {
	notVaild := false
	// Check if UserId is valid (greater than 0)
	if sq.UserId <= 0 {
		return notVaild
	}

	// Check if string fields are not empty
	if sq.ChainCode == "" ||
		sq.BaseAddress == "" ||
		sq.Symbol == "" ||
		sq.MonitorType == "" {
		return notVaild
	}

	if sq.TargetPrice == "" && sq.Data == "" {
		return notVaild
	}

	// Check if NoticeType is valid (assuming any non-zero value is valid)
	if sq.NoticeType == 0 {
		return notVaild
	}

	// If all checks pass, the data is valid
	return true
}

type PusherHandlerReqData struct {
	ChainCode   string `json:"chainCode"`
	BaseAddress string `json:"baseAddress"`
	MonitorType string `json:"type"`
}

type TokenSubscribeInfo struct {
	BaseAddress    string `json:"baseAddress"`
	ChainCode      string `json:"chainCode"`
	Data           string `json:"data"`
	TargetPrice    string `json:"targetPrice"`
	StartPrice     string `json:"startPrice"`
	Symbol         string `json:"symbol"`
	Logo           string `json:"logo"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	NoticeType     int64  `json:"noticeType"`
	LastNoticeTime int64  `json:"lastNoticeTime"`
	Status         int64  `json:"status"`
}
