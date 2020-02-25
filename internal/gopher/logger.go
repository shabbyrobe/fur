package gopher

import "log"

type Logger interface {
	Printf(format string, v ...interface{})
}

type logger struct {
	printf func(format string, v ...interface{})
}

func (l logger) Printf(format string, v ...interface{}) {
	l.printf(format, v...)
}

var stdLogger Logger = &logger{
	printf: log.Printf,
}
