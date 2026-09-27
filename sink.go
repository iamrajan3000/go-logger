package logger

type Sink interface {
	Write(msg Message) error
}
