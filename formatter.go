package logger

import "fmt"

type Formatter struct {
	TimeLayout string
}

func NewFormatter(timeLayout string) *Formatter {
	return &Formatter{TimeLayout: timeLayout}
}

func (f *Formatter) Format(msg Message) string {
	return fmt.Sprintf("%s [%s] %s", msg.Level, msg.Timestamp.Format(f.TimeLayout), msg.Content)
}
