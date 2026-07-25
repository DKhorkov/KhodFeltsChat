package common

import (
	"time"
	_ "time/tzdata" // нужно для подгрузки таймзон в докер контейнере
)

const (
	LoggingTraceSkipLevel = 1
	DateFormat            = "02.01.2006"
)

var Timezone *time.Location

func init() {
	var err error

	Timezone, err = time.LoadLocation("Europe/Moscow")
	if err != nil {
		panic(err)
	}
}
