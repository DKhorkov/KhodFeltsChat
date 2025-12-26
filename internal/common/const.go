package common

import (
	"time"
	_ "time/tzdata" // нужно для подгрузки таймзон в докер контейнере
)

const (
	LoggingTraceSkipLevel                  = 1
	DateFormat                             = "02.01.2006"
	GroupTitleMaxLength                    = 50
	PlantTitleMaxLength                    = 50
	GroupsPerUserLimitWithoutSubscription  = 3
	GroupsPerUserLimitWithSubscription     = 20
	PlantsPerGroupLimitWithoutSubscription = 10
	PlantsPerGroupLimitWithSubscription    = 50

	ContextDataSeparator = ";"

	FeedbacksLimit = 5
	FeedbacksTTL   = time.Hour * 24
)

var Timezone *time.Location

func init() {
	var err error

	Timezone, err = time.LoadLocation("Europe/Moscow")
	if err != nil {
		panic(err)
	}
}
