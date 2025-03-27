package rpc

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

func SOL_GetSignatureStatuses(tx string) (string, error) {
	var result string
	reqBody := fmt.Sprintf(`{ 
        "jsonrpc": "2.0", 
        "id": 1, 
        "method": "getSignatureStatuses", 
        "params": [ 
            ["%s"], 
            {"searchTransactionHistory": true} 
        ]
    }`, tx)

	resp, err := http.Post(sol_rpc,
		"application/json",
		strings.NewReader(reqBody))
	if err != nil {
		return result, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("failed to read response: %w", err)
	}

	if errMsg := gjson.GetBytes(data, "error.message").String(); errMsg != "" {
		return result, fmt.Errorf("rpc error: %s", errMsg)
	}

	valueArray := gjson.GetBytes(data, "result.value").Array()
	if len(valueArray) == 0 {
		return result, errors.New("empty response")
	}

	firstItem := valueArray[0]
	if firstItem.Type == gjson.Null {
		return result, errors.New("not found")
	}

	if firstItem.Get("err").Type == gjson.Null {
		result = tx
		return result, nil
	}

	return "", nil
}

func SOL_PollTransactionStatus(tx string) error {
	const (
		maxRetries    = 10
		retryInterval = 2 * time.Second
	)

	for i := 0; i < maxRetries; i++ {
		result, err := SOL_GetSignatureStatuses(tx)
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
