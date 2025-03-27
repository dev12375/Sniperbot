package store

import "os"

const BOT_ID = "BOT_ID"
const BOT_USERNAME = "BOT_USERNAME"

func NewEnv(key, value string) error {
	return os.Setenv(key, value)
}
func GetEnv(key string) string {
	return os.Getenv(key)
}
