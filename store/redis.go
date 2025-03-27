package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hellodex/tradingbot/config"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var redisClient *redis.Client

func InitRedis() {
	rdb := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", config.YmlConfig.Redis.Ip, config.YmlConfig.Redis.Port),
		Username:        config.YmlConfig.Redis.Username,
		Password:        config.YmlConfig.Redis.Passwd,
		DB:              config.YmlConfig.Redis.Db,
		PoolSize:        10,
		MinIdleConns:    5,
		MaxIdleConns:    10,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	})

	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	log.Debug().Msgf("connecting redis [%s:%d]", config.YmlConfig.Redis.Ip, config.YmlConfig.Redis.Port)
	if err != nil {
		log.Fatal().Err(err).Msg("redis init error")
	}

	redisClient = rdb
}

func NewRedisClient(ip string, port int, userName string, passwd string, db int) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", ip, port),
		Username:        userName,
		Password:        passwd,
		DB:              db,
		PoolSize:        10,
		MinIdleConns:    5,
		MaxIdleConns:    10,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	})

	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	log.Debug().Msgf("connecting redis [%s:%d]", ip, port)
	if err != nil {
		log.Fatal().Err(err).Msg("redis init error")
	}
	return rdb
}

func checkRedis() {
	if redisClient == nil {
		log.Fatal().Msg("redisClient is not set")
	}
}

func profileKey(userID int64) string {
	return fmt.Sprintf("user:profile:%d", userID)
}

func RedisSetUserProfile(userID int64, profileDataStruct any) bool {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	profileJSON, err := json.Marshal(profileDataStruct)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to marshal profile for user %d: %v", userID, err)
		return false
	}
	key := profileKey(userID)

	err = redisClient.Set(ctx, key, profileJSON, 24*time.Hour).Err()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to set profile for user %d: %v", userID, err)
		return false
	}

	return true
}

func RedisGetUserProfile(userID int64) ([]byte, bool) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := profileKey(userID)

	// 从 Redis 获取数据
	data, err := redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// key 不存在
			log.Error().Err(err).Msgf("Profile not found for user %d", userID)
		} else {
			// 其他错误
			log.Error().Err(err).Msgf("Failed to get profile for user %d: %v", userID, err)
		}
		return nil, false
	}

	return data, true
}

func RedisDeleteUserProfile(userID int64) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := profileKey(userID)

	// 删除 key
	_, err := redisClient.Del(ctx, key).Result()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to delete profile for user %d: %v", userID, err)
	}
}

func RedisUserSubscribeInfo(userID int64, dataByteJson any) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := fmt.Sprintf("userSubscribeInfo:%d", userID)

	err := redisClient.Set(ctx, key, dataByteJson, -1).Err() // -1 也表示永不过期
	if err != nil {
		log.Error().Err(err).Msgf("Failed to set UserInBot for user %d: %v", userID, err)
	}
}

func SubChannel(channelName string) (<-chan *redis.Message, error) {
	checkRedis()
	log.Debug().Str("sub channel", channelName).Send()

	ctx := context.Background()
	redisC := NewRedisClient(
		config.YmlConfig.RedisPush.Ip,
		config.YmlConfig.RedisPush.Port,
		config.YmlConfig.RedisPush.Username,
		config.YmlConfig.RedisPush.Passwd,
		config.YmlConfig.RedisPush.Db,
	)

	ch := make(chan *redis.Message, 1000)
	go func() {
		defer redisC.Close()

		pubsub := redisC.Subscribe(ctx, channelName)
		defer pubsub.Close()

		_, err := pubsub.Receive(ctx)
		if err != nil {
			log.Error().Err(err).Msg("can't receive message")
			close(ch)
			return
		}

		msgCh := pubsub.Channel()

		for msg := range msgCh {
			ch <- msg
		}

		close(ch)
	}()

	return ch, nil
}

func UserSetState(userId int64, state string) error {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stateKey := fmt.Sprintf("userInState:%d", userId)

	if err := redisClient.Set(ctx, stateKey, state, 5*time.Minute).Err(); err != nil {
		log.Debug().Err(err).Send()
		return err
	}
	return nil
}

func UserInState(userId int64, state string) bool {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stateKey := fmt.Sprintf("userInState:%d", userId)

	if nowState := redisClient.Get(ctx, stateKey).String(); nowState != "" && nowState == state {
		return true
	}
	return false
}

func UserSetWallet(userId int64, value any) error {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setKey := fmt.Sprintf("%s:%d", "wallet", userId)

	if err := redisClient.Set(ctx, setKey, value, -1).Err(); err != nil {
		log.Debug().Err(err).Send()
		return err
	}
	return nil
}

func UserGetWallet(userId int64) []byte {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setKey := fmt.Sprintf("%s:%d", "wallet", userId)

	if b, err := redisClient.Get(ctx, setKey).Bytes(); err != nil {
		log.Debug().Err(err).Send()
		return b
	}
	return nil
}

func UserSetAiMonitorInfo(userId int64, AISubscribeReqData any) error {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setKey := fmt.Sprintf("%s:%d", "ai_monitor", userId)

	if err := redisClient.Set(ctx, setKey, AISubscribeReqData, -1).Err(); err != nil {
		log.Debug().Err(err).Send()
		return err
	}
	return nil
}

func UserGetAiMonitorInfo(userId int64) ([]byte, bool) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setKey := fmt.Sprintf("%s:%d", "ai_monitor", userId)
	data, err := redisClient.Get(ctx, setKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			log.Debug().Msg("Key not found in Redis")
			return nil, false
		}
		log.Debug().Err(err).Send()
		return nil, false
	}
	return data, true
}

func UserDeleteAiMonitorInfo(userId int64) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	key := fmt.Sprintf("%s:%d", "ai_monitor", userId)
	_, err := redisClient.Del(ctx, key).Result()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to delete profile for user %d: %v", userId, err)
	}
}

func UserSetBot(userId int64, uuid string) error {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setKey := fmt.Sprintf("%s::%s", "userInBot", uuid)

	id := GetEnv(BOT_ID)
	log.Debug().Str("get bot id", id).Send()

	value := fmt.Sprintf("%s::%d", id, userId)

	if err := redisClient.Set(ctx, setKey, value, -1).Err(); err != nil {
		log.Debug().Err(err).Send()
		return err
	}
	return nil
}

func UserInBot(uuid string) (string, error) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setKey := fmt.Sprintf("%s::%s", "userInBot", uuid)
	str, err := redisClient.Get(ctx, setKey).Result()
	log.Debug().Str("result", str).Send()
	return str, err
}

func NewPusherMessage(chatId int64, msg_id, message string) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	id := GetEnv(BOT_ID)
	setKey := fmt.Sprintf("bot_%s_%s::%d", id, "userInBot", chatId)
	log.Debug().Str("get bot id", id).Send()

	redisClient.HSet(ctx, setKey, map[string]string{
		msg_id: message,
	})
}

func GetMessageByMsgId(chatId int64, msgId string) (string, error) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	id := GetEnv(BOT_ID)
	hashKey := fmt.Sprintf("bot_%s_%s::%d", id, "userInBot", chatId)

	message, err := redisClient.HGet(ctx, hashKey, msgId).Result()

	if err == redis.Nil {
		return "", fmt.Errorf("message with ID %s not found", msgId)
	} else if err != nil {
		return "", fmt.Errorf("failed to get message: %w", err)
	}

	log.Debug().Msg(message)

	return message, nil
}

func UserSetCommissionInfo(chatId int64, body []byte) error {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := fmt.Sprintf("%s:%d", "cmInfo", chatId)

	if err := redisClient.Set(ctx, key, body, 0).Err(); err != nil {
		return err
	}

	return nil
}

func UserGetCommissionInfo(chatId int64) ([]byte, bool) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := fmt.Sprintf("%s:%d", "cmInfo", chatId)
	data, err := redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	return data, true
}

func UserDeleteCommissionInfo(chatId int64) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	key := fmt.Sprintf("%s:%d", "cmInfo", chatId)
	redisClient.Del(ctx, key)
}

func UserSetOrderHistory(chatId int64, data []byte) error {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := fmt.Sprintf("%s:%d", "orderHistory", chatId)

	if err := redisClient.Set(ctx, key, data, 0).Err(); err != nil {
		return err
	}

	return nil
}

func UserGetOrderHistory(chatId int64) ([]byte, bool) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := fmt.Sprintf("%s:%d", "orderHistory", chatId)
	data, err := redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	return data, true
}

func RedisSetState(chatId int64, stateName string, state string, expir time.Duration) error {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cacheKey := fmt.Sprintf("%s:%d", stateName, chatId)

	if err := redisClient.Set(ctx, cacheKey, state, expir).Err(); err != nil {
		return err
	}

	return nil
}

func RedisGetState(chatId int64, stateName string) (string, bool) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cacheKey := fmt.Sprintf("%s:%d", stateName, chatId)

	state, err := redisClient.Get(ctx, cacheKey).Result()
	if err != nil {
		return "", false
	}
	return state, true
}

func RedisDeleteState(chatId int64, stateName string) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cacheKey := fmt.Sprintf("%s:%d", stateName, chatId)
	redisClient.Del(ctx, cacheKey)
}

func RedisSetSendMessageParams(chatId int64, messageId int, msgParams []byte) error {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cacheKey := fmt.Sprintf("message:%d:%d", chatId, messageId)
	if err := redisClient.Set(ctx, cacheKey, msgParams, 24*time.Hour).Err(); err != nil {
		return err
	}
	return nil
}

func RedisGetSendMessageParams(chatId int64, messageId int) ([]byte, error) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cacheKey := fmt.Sprintf("message:%d:%d", chatId, messageId)

	return redisClient.Get(ctx, cacheKey).Bytes()
}

func RedisSetUserAiList(chatId int64, listRaw string) error {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cacheKey := fmt.Sprintf("%s:%d", "ai_list", chatId)

	if err := redisClient.Set(ctx, cacheKey, listRaw, 0).Err(); err != nil {
		return err
	}

	return nil
}

func RedisGetUserAiList(chatId int64) ([]byte, error) {
	checkRedis()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cacheKey := fmt.Sprintf("%s:%d", "ai_list", chatId)

	return redisClient.Get(ctx, cacheKey).Bytes()
}

const messageCountKey = "bot:message:counter"

func BotMessageAdd() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipe := redisClient.Pipeline()

	botId := os.Getenv("BOT_ID")
	botUserName := os.Getenv("BOT_USERNAME")
	cacheKey := fmt.Sprintf("%s:%s:%s", messageCountKey, botId, botUserName)
	incrCmd := pipe.Incr(ctx, cacheKey)

	pipe.Expire(ctx, cacheKey, 3*time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	count := incrCmd.Val()
	return count, nil
}

func BotMessageCount() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	botId := os.Getenv("BOT_ID")
	botUserName := os.Getenv("BOT_USERNAME")
	cacheKey := fmt.Sprintf("%s:%s:%s", messageCountKey, botId, botUserName)
	val, err := redisClient.Get(ctx, cacheKey).Int64()
	if err == redis.Nil {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return val, nil
}

func GetBotsStatus() (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keys, cursor, err := redisClient.Scan(ctx, 0, messageCountKey+"*", 100).Result()
	_ = cursor
	if err != nil {
		return nil, fmt.Errorf("扫描键错误: %w", err)
	}

	if len(keys) == 0 {
		return map[string]string{}, nil
	}

	pipe := redisClient.Pipeline()

	cmdMap := make(map[string]*redis.StringCmd)
	for _, key := range keys {
		cmdMap[key] = pipe.Get(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("执行管道命令错误: %w", err)
	}

	result := make(map[string]string)
	for key, cmd := range cmdMap {
		val, err := cmd.Result()
		if err == redis.Nil || err != nil {
			continue
		}

		parts := strings.Split(key, ":")
		if len(parts) >= 5 {
			botUsername := parts[len(parts)-1]
			result[botUsername] = val
		}
	}

	return result, nil
}
