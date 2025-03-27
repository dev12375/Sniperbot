package queue

import (
	"fmt"
)

var workerNum = 5

type Event int

const (
	Success    Event = iota // 成功
	Failed                  // 失败
	Processing              // 处理中
)

var ErrQueueFull = fmt.Errorf("出错了，请联系客服")
