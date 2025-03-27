package store

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
)

var CacheStore = cache.New(5*time.Minute, 30*time.Minute)

var TradeSession string = "tradeSession"
var WaitCleanMessage string = "waitCleanMessage"

func cacheKey(userID int64, key string) string {
	return fmt.Sprintf("%d_%s", userID, key)
}

func AppendBy_(userID int64, key, value string) {
	if existingValue, exists := CacheStore.Get(key); exists {
		newValue := fmt.Sprintf("%s_%s", value, existingValue)
		CacheStore.Set(key, newValue, 25*time.Hour)
		return
	}
	CacheStore.Set(cacheKey(userID, key), value, 25*time.Hour)
}

func Set(userID int64, key string, value interface{}, duration time.Duration) {
	CacheStore.Set(cacheKey(userID, key), value, duration)
}

func Get(userID int64, key string) (interface{}, bool) {
	return CacheStore.Get(cacheKey(userID, key))
}

func Delete(userID int64, key string) {
	CacheStore.Delete(cacheKey(userID, key))
}
