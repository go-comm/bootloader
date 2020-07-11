package bootloader

import (
	"log"
)

type Logger interface {
	Println(v ...interface{})
	Printf(format string, v ...interface{})
	Show(show bool)
}

type logger struct {
	show bool
}

func (logger *logger) Show(show bool) {
	logger.show = show
}

func (logger *logger) Println(v ...interface{}) {
	if logger.show {
		log.Println(v...)
	}
}

func (logger *logger) Printf(format string, v ...interface{}) {
	if logger.show {
		log.Printf(format, v...)
	}
}
