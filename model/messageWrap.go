package model

import (
	"fmt"
	"time"

	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/store"
)

type MessageWrap struct {
	Message    models.Message
	Tokeninfo  PositionByWalletAddress
	ExpireTime int64
}

// NewMessageWrap 构造函数,设置24小时过期时间
func NewMessageWrap(userID int64, message models.Message, tokeninfo PositionByWalletAddress) *MessageWrap {
	// store message id in cache
	messageWrap := &MessageWrap{
		Message:    message,
		Tokeninfo:  tokeninfo,
		ExpireTime: time.Now().Unix() + 23*60*60,
		// ExpireTime: time.Now().Unix() + 5,
	}

	store.AppendBy_(userID, store.WaitCleanMessage, fmt.Sprintf("%d", message.ID))

	store.Set(userID, store.TradeSession, messageWrap, 25*time.Hour)

	return messageWrap
}

// IsExpired 检查是否过期
func (m *MessageWrap) IsExpired() bool {
	return time.Now().Unix() > m.ExpireTime
}
