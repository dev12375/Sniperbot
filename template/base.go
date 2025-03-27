package template

import (
	"errors"

	"github.com/flosch/pongo2/v6"
	"github.com/hellodex/tradingbot/api"
	"github.com/hellodex/tradingbot/util"
)

// formatTime filter
var _ = func() interface{} {
	pongo2.RegisterFilter("formatTime", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		return pongo2.AsValue(util.FormatTime(in.String())), nil
	})
	return nil
}()

// formatNumber filter
var _ = func() interface{} {
	pongo2.RegisterFilter("formatNumber", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		return pongo2.AsValue(util.FormatNumber(in.String())), nil
	})
	return nil
}()

// getChainName filter
var _ = func() interface{} {
	pongo2.RegisterFilter("getChainName", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		return pongo2.AsValue(api.GetChainNameFallbackCode(in.String())), nil
	})
	return nil
}()

var _ = func() interface{} {
	pongo2.RegisterFilter("formatPer", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		return pongo2.AsValue(util.FormatPercentage(in.String())), nil
	})
	return nil
}()

var ErrRander = errors.New("出错了，请联系客服！")
