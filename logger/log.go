package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
)

const (
	CategoryField = "category"
)

const (
	CategoryTx      = "tx"
	CategoryToken   = "token"
	CategoryUser    = "user"
	CategoryNetwork = "network"
	CategorySwap    = "swap"
)

// 添加分类的辅助函数
func WithCategory(category string) func(e *zerolog.Event) {
	return func(e *zerolog.Event) {
		e.Str(CategoryField, category)
	}
}

func WithTxCategory(e *zerolog.Event) *zerolog.Event {
	return e.Str(CategoryField, CategoryTx)
}

func WithTokenCategory(e *zerolog.Event) *zerolog.Event {
	return e.Str(CategoryField, CategoryToken)
}

// var Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
func StdLogger() *zerolog.Logger {
	outPut := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    false,
		TimeFormat: time.DateTime,
		FormatFieldName: func(i interface{}) string {
			return fmt.Sprintf("%s: ", i)
		},
		FieldsOrder: []string{"接口", "参数", "返回"},
	}
	log := zerolog.New(outPut).With().Timestamp().Logger()

	return &log
}

func NewStdLog(endpoint string, req []byte, result []byte) {
	log := StdLogger()
	log.Info().Str("接口", endpoint).RawJSON("参数", req).RawJSON("返回", result).Send()
}
