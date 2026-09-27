package logger

import "time"

type Message struct {
	Content   string
	Level     Level
	Namespace string
	Timestamp time.Time
}
