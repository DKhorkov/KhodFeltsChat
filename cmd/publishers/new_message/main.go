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

	payload, err := json.Marshal(domains.NewMessagePayload{
		MessageID: 1,
	})
	if err != nil {
		panic(err)
	}

	emailDTO := domains.EmailNotificationDTO{
		Type:    domains.EmailTypeNewMessage,
		UserID:  1,
		Payload: payload,
	}

	content, err := json.Marshal(emailDTO)
	if err != nil {
		panic(err)
	}

	err = natsPublisher.Publish(settings.NATS.Subjects.EmailNotification, content)
	if err != nil {
		panic(err)
	}

	webPushDTO := domains.WebPushNotificationDTO{
		Type:    domains.WebPushTypeNewMessage,
		UserID:  1,
		Payload: payload,
	}

	content, err = json.Marshal(webPushDTO)
	if err != nil {
		panic(err)
	}

	err = natsPublisher.Publish(settings.NATS.Subjects.WebPushNotification, content)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 2)
}
