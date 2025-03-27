package util

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/go-telegram/bot/models"
	"github.com/hellodex/tradingbot/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

func IsDebug() bool {
	return os.Getenv("DEBUG") == "true" || config.YmlConfig.Env.Debug == "true"
}

func IsCommand(str string) bool {
	return strings.HasPrefix(str, "/")
}

func Ptr[T any](v T) *T {
	return &v
}

// EffectId
func EffectId(update *models.Update) int64 {
	if update == nil {
		return 0
	}

	if update.Message != nil {
		return update.Message.From.ID
	}

	if update.CallbackQuery != nil {
		return update.CallbackQuery.From.ID
	}

	if update.InlineQuery != nil {
		return update.InlineQuery.From.ID
	}

	if update.ChannelPost != nil {
		return update.ChannelPost.From.ID
	}

	if update.EditedMessage != nil {
		return update.EditedMessage.From.ID
	}

	if update.ChosenInlineResult != nil {
		return update.ChosenInlineResult.From.ID
	}

	if update.ShippingQuery != nil {
		return update.ShippingQuery.From.ID
	}

	if update.PreCheckoutQuery != nil {
		return update.PreCheckoutQuery.From.ID
	}

	return 0
}

func UserIDtoByte(userID int64) []byte {
	s := cast.ToString(userID)
	return []byte(s)
}

func Encrypt(userID int64) (string, error) {
	if userID == config.YmlConfig.Env.BotMaker {
		return "test user", nil
	}
	s := cast.ToString(userID)
	loadKey, err := base64.StdEncoding.DecodeString(config.YmlConfig.Env.AesKey)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if config.YmlConfig.Env.Encrypt_open {
		return encrypt(loadKey, s)
	}
	return base64.StdEncoding.EncodeToString([]byte(s)), nil
}

func encrypt(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	loadNonce, err := base64.StdEncoding.DecodeString(config.YmlConfig.Env.Nonce)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	nonce := loadNonce

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	ciphertext := aesgcm.Seal(nil, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(append(nonce, ciphertext...)), nil
}

func CheckValidAddress(address string) (isSolana bool, err error) {
	if IsNativeCoion(address) {
		return true, nil
	}

	if len(address) < 32 {
		return false, errors.New("地址长度不正确，请检查输入")
	}

	// pk, err := solana.PublicKeyFromBase58(address)
	// if err == nil && pk.IsOnCurve() {
	// 	return true, nil
	// }
	_, err = solana.PublicKeyFromBase58(address)
	if err == nil {
		return true, nil
	}

	// is EVM network Address
	match := common.IsHexAddress(address)
	if match {
		return false, nil
	}

	return false, errors.New("无法识别的地址格式")
}

var NativeSol = "11111111111111111111111111111111"
var NativeEvm = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"

func IsCryproAddress(addr string) bool {
	if IsNativeCoion(addr) {
		return true
	}
	pk, err := solana.PublicKeyFromBase58(addr)
	if err == nil && pk.IsOnCurve() {
		return true
	}
	// is EVM network Address
	match := common.IsHexAddress(addr)
	if match {
		return true
	} else {
		return false
	}
}

func IsNativeCoion(addr string) bool {
	switch addr {
	case NativeSol:
		return true
	case NativeEvm:
		return true
	default:
		return false

	}
}

func CtxWithValue(ctx context.Context, k any, value any) context.Context {
	return context.WithValue(ctx, k, value)
}
