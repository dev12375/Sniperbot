package rpc

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/hellodex/tradingbot/config"
	"github.com/hellodex/tradingbot/logger"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var ErrPollTxMaxRetry = errors.New("链上确认中，请在交易历史中确认交易结果")

func pollTxStatusLog() *zerolog.Event {
	return log.Debug().Func(logger.WithCategory("PollTransactionStatus"))
}

var sol_rpc = func() string {
	return config.YmlConfig.Env.SolRpc
}()
var bsc_rpc = func() string {
	return config.YmlConfig.Env.BscRpc
}()

type ChainChecker struct{}

func (c *ChainChecker) CheckSOLANATxStatus(tx string) error {
	pollTxStatusLog().Str("call CheckSOLANATxStatus", tx).Send()
	return SOL_PollTransactionStatus(tx)
}

func (c *ChainChecker) CheckBSCTxStatus(tx string) error {
	pollTxStatusLog().Str("call CheckBSCTxStatus", tx).Send()
	return BSC_PollTransactionStatus(tx)
}

func PollTransactionStatus(chainCode string, tx string) error {
	pollTxStatusLog().Str("chainCode", chainCode).Str("tx", tx).Send()

	checker := &ChainChecker{}

	funcName := "Check" + chainCode + "TxStatus"
	method := reflect.ValueOf(checker).MethodByName(funcName)
	pollTxStatusLog().Str("make funcName", funcName).Send()
	if !method.IsValid() {
		pollTxStatusLog().Msgf("method not found %s", method)
		return fmt.Errorf("unsupported chain: %s", chainCode)
	}

	results := method.Call([]reflect.Value{reflect.ValueOf(tx)})
	if len(results) > 0 && !results[0].IsNil() {
		pollTxStatusLog().Msg("call but no return")
		return results[0].Interface().(error)
	}

	return nil
}
