package template

import (
	"encoding/json"
	"fmt"

	"github.com/flosch/pongo2/v6"
	"github.com/rs/zerolog/log"
)

type SubInfo struct {
	BaseAddress    string `json:"baseAddress"`
	ChainCode      string `json:"chainCode"`
	Data           string `json:"data"`
	TargetPrice    string `json:"targetPrice"`
	StartPrice     string `json:"startPrice"`
	Symbol         string `json:"symbol"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	NoticeType     int64  `json:"noticeType"`
	LastNoticeTime int64  `json:"lastNoticeTime"`
	Status         int64  `json:"status"`
	Price          string `json:"price"`
	Chg            string `json:"chg"`
	PairAddress    string `json:"pairAddress"`
}

var listAimonitor = `
订阅列表:
{% for sub in subList | slice:slice_range %}
{{ sub.ChainCode | getChainName }}
📊 监听信息:
|--Token: <b>{{ sub.BaseAddress }}</b>
|--Symbol: <b>{{ sub.Symbol }}</b>
|--价格: <b>${{ sub.Price|formatNumber }}</b>
|--目标价格: <b>${{ sub.TargetPrice|formatNumber }}</b>
|--开始价格: <b>${{ sub.StartPrice|formatNumber }}</b>
|--类型: <b>{{ sub.Type }}</b>
{% endfor %}
`

func RanderListAimonitor(data []byte) (string, error) {
	// Compile the template first (i. e. creating the AST)
	tpl, err := pongo2.FromString(listAimonitor)
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	page := 1
	pageSize := 6
	start := (page - 1) * pageSize
	end := start + pageSize
	sliceRange := fmt.Sprintf("%d:%d", start, end)

	var subList []SubInfo
	err = json.Unmarshal(data, &subList)
	if err != nil {
		return "", ErrRander
	}

	out, err := tpl.Execute(pongo2.Context{
		"subList":     subList,
		"slice_range": sliceRange,
		"size":        pageSize,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return "", ErrRander
	}

	return out, nil
}
