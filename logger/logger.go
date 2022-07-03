package log

import (
	"log"
)

var Debug bool

func Printf(format string, v ...interface{}) {
	if Debug {
		log.Printf(format, v...)
	}
}

func Infof(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func Errorf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}
