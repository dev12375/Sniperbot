package session

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

// key to session
var UserSelectWalletCache string = "select_wallet"
var UserSelectChainCache string = "select_chain"
var UserSelectTokenAddressCache string = "select_token"
var UserLastSelectTokenCache string = "user_lastSelectToken"
var UserLastSwapMessage string = "user_lastSwapMessage"

var UserSessionState string = "in_state"
var LimitOrderState string = "limitOrder"
var UserInLimitOrderCache string = "user_limitOrder_cache"
var TransferToState = "transferTo"
var UserInTransferToCache string = "user_transferTo_cache"
var UserStartMessaageIDkey = "user_start_reflash"

var SessionType = struct{}{}

type SessionManager struct {
	sessions sync.Map
}

var sessionManager *SessionManager
var once sync.Once

func newSessionManager() *SessionManager {
	return &SessionManager{
		sessions: sync.Map{},
	}
}

func GetSessionManager() *SessionManager {
	once.Do(func() {
		sessionManager = newSessionManager()
	})
	return sessionManager
}

func (sm *SessionManager) Set(userID int64, key string, value any) {
	k := fmt.Sprintf("%d::%s", userID, key)
	log.Debug().Interface(key, value).Int64("userID", userID).Msg("session set")
	sm.sessions.Store(k, value)
}

func (sm *SessionManager) Get(userID int64, key string) (value any, ok bool) {
	k := fmt.Sprintf("%d::%s", userID, key)
	if valueInner, ok := sm.sessions.Load(k); ok {
		log.Debug().Interface(key, valueInner).Int64("userID", userID).Msg("session get")
	}
	return sm.sessions.Load(k)
}

func (sm *SessionManager) Delete(userID int64, key string) {
	k := fmt.Sprintf("%d::%s", userID, key)
	log.Debug().Str("delete", key).Int64("userID", userID).Msg("session delete")
	sm.sessions.Delete(k)
}

type fallbackFunc func()

func Callback(userID int64, key string, fn fallbackFunc) {
	sm := GetSessionManager()
	if _, exists := sm.Get(userID, key); exists {
		fn()
		sm.Delete(userID, key)
	}
}
