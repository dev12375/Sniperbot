package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Channel struct {
		Code    string `yaml:"code"`
		Key     string `yaml:"key"`
		Version string `yaml:"version"`
	} `yaml:"channel"`

	App struct {
		Ver    string `yaml:"ver"`
		Appid  string `yaml:"appid"`
		Appkey string `yaml:"appkey"`
	} `yaml:"app"`

	Env struct {
		ApiEndpoint  string `yaml:"api_endpoint"`
		SolRpc       string `yaml:"sol_rpc"`
		BscRpc       string `yaml:"bsc_rpc"`
		Debug        string `yaml:"debug"`
		BotName      string `yaml:"bot_name"`
		BotApiKey    string `yaml:"bot_api_key"`
		BotMaker     int64  `yaml:"bot_maker"`
		AesKey       string `yaml:"aes_key"`
		Nonce        string `yaml:"nonce"`
		Encrypt_open bool   `yaml:"encrypt_open"`
		KchartUrl    string `yaml:"kchart_url"`
		TgHook       string `yaml:"tg_hook"`
		WebHookOpen  bool   `yaml:"web_hook_open"`
		TgHookToken  string `yaml:"tg_hook_token"`
		LocalHost    string `yaml:"local_host"`
	} `yaml:"env"`

	Redis struct {
		Ip       string `yaml:"ip"`
		Port     int    `yaml:"port"`
		Db       int    `yaml:"db"`
		Username string `yaml:"username"`
		Passwd   string `yaml:"passwd"`
	} `yaml:"redis"`

	RedisPush struct {
		Ip        string `yaml:"ip"`
		Port      int    `yaml:"port"`
		Db        int    `yaml:"db"`
		Username  string `yaml:"username"`
		Passwd    string `yaml:"passwd"`
		MessageCh string `yaml:"message_channel"`
	} `yaml:"redis_push"`
}

var YmlConfig *Config

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
func init() {
	var confFilePath string
	if configFilePathFromEnv := os.Getenv("TGBOT_APP_ENV"); configFilePathFromEnv != "" {
		confFilePath = configFilePathFromEnv
	} else {
		confFilePath = "./prod.yml"
	}
	cfg, err := LoadConfig(confFilePath)
	if err != nil {
		panic(err)
	}
	YmlConfig = cfg
}
