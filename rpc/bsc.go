package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

func BSC_GetSignatureStatuses(tx string) (string, error) {
	data := map[string]interface{}{
		"method":  "eth_getTransactionReceipt",
		"params":  []interface{}{tx},
		"id":      1,
		"jsonrpc": "2.0",
	}

	reqData, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).Send()
		return "", err
	}

	resp, err := http.Post(bsc_rpc, "application/json", bytes.NewBuffer(reqData))
	if err != nil {
		log.Error().Err(err).Send()
		return "", err
	}
	defer resp.Body.Close()

	jsonByte, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Send()
		return "", err
	}

	resultData := gjson.GetBytes(jsonByte, "result")
	if resultData.Type == gjson.Null {
		return "", errors.New("not found")
	}

	if status := gjson.GetBytes(jsonByte, "result.status").String(); status == "0x1" {
		return tx, nil
	} else {
		return "", errors.New("status != 1")
	}

}

func BSC_PollTransactionStatus(tx string) error {
	const (
		maxRetries    = 10
		retryInterval = 2 * time.Second
	)

	for i := 0; i < maxRetries; i++ {
		result, err := BSC_GetSignatureStatuses(tx)
		if err != nil {
			if err.Error() == "not found" {
				log.Debug().
					Str("tx", tx).
					Int("attempt", i+1).
					Msg("transaction not found, retrying...")
				time.Sleep(retryInterval)
				continue
			}
			return err
		}

		if result != "" {
			log.Info().Str("tx", tx).Msg("transaction processed")
			return nil
		}

		log.Debug().Str("tx", tx).Msg("transaction not processed, retrying...")
		time.Sleep(retryInterval)
	}
	log.Debug().Msgf("transaction %s not processed after %d attempts", tx, maxRetries)

	return ErrPollTxMaxRetry
}
