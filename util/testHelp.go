package util

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
)

func TestHttpRespPrint(data []byte) {
	var r map[string]any
	err := json.Unmarshal(data, &r)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	jd, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	fmt.Println(string(jd))
}

func TestPrintStruct(s any) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Error().Err(err).Send()
	}

	fmt.Println(string(data))
}
