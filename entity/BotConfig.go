package entity

import (
	"context"

	"github.com/go-telegram/bot"
)

type BotConfig struct {
	UserId  int64              `json:"userId"`
	BotName string             `json:"botName"`
	ApiKey  string             `json:"apiKey"`
	Type    int8               `json:"type"`
	Id      int64              `json:"id"`
	Cancel  context.CancelFunc `json:"-"`
}

var BotConfigs []BotConfig

var BotMap = make(map[int64]*bot.Bot)
var BotConfigMap = make(map[int64]BotConfig)
var UserBotConfigMap = make(map[int64][]BotConfig)
