package main

import (
	"encoding/json"
	"time"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	customnats "github.com/DKhorkov/libs/nats"
	"github.com/nats-io/nats.go"
)

func main() {
	settings := config.New()

	natsPublisher, err := customnats.NewPublisher(
		settings.NATS.ClientURL,
		nats.Name("kfc-test"),
	)
	if err != nil {
		panic(err)
	}

	defer natsPublisher.Close()

	ticketUpdatedDTO := domains.VerifyEmailNotificationDTO{
		UserID: 1,
	}

	content, err := json.Marshal(ticketUpdatedDTO)
	if err != nil {
		panic(err)
	}

	err = natsPublisher.Publish(settings.NATS.Subjects.VerifyEmail, content)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 2)
}
