package util

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func UnsafeGetUpdateChan(b *bot.Bot) (chan *models.Update, error) {
	if b == nil {
		return nil, errors.New("bot is nil")
	}

	v := reflect.ValueOf(b).Elem()
	updateChan := v.FieldByName("updates")

	if !updateChan.IsValid() {
		return nil, errors.New("updateChan is not valid")
	}

	if updateChan.Kind() != reflect.Chan {
		return nil, errors.New("field is not a channel")
	}

	ch := reflect.NewAt(updateChan.Type(), unsafe.Pointer(updateChan.UnsafeAddr())).Elem()

	if ch.IsNil() {
		return nil, errors.New("channel is nil")
	}

	c, ok := ch.Interface().(chan *models.Update)
	if !ok {
		return nil, fmt.Errorf("failed to convert to channel type: %v", ch.Type())
	}

	return c, nil
}
