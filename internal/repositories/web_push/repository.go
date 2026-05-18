package web_push

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/libs/logging"
	"github.com/SherClockHolmes/webpush-go"
)

type Repository struct {
	webPushConfig config.WebPushConfig
	logger        logging.Logger
}

func New(
	webPushConfig config.WebPushConfig,
	logger logging.Logger,
) *Repository {
	return &Repository{
		webPushConfig: webPushConfig,
		logger:        logger,
	}
}

func (repo *Repository) SendNotification(
	_ context.Context,
	subscription domains.WebPushSubscription,
	message domains.Message,
) error {
	payload, err := json.Marshal(map[string]any{
		"title":     message.Sender.Username,
		"body":      message.Text,
		"chatId":    message.ChatID,
		"timestamp": message.CreatedAt.UnixMilli(),
	})
	if err != nil {
		return fmt.Errorf("marshal web push notification payload: %w", err)
	}

	resp, err := webpush.SendNotification(
		payload,
		&webpush.Subscription{
			Endpoint: subscription.Endpoint,
			Keys: webpush.Keys{
				P256dh: subscription.EncryptionKey,
				Auth:   subscription.Auth,
			},
		},
		&webpush.Options{ //nolint:exhaustruct // остальные поля библиотеки имеют разумные значения по умолчанию
			VAPIDPublicKey:  repo.webPushConfig.VAPIDPublicKey,
			VAPIDPrivateKey: repo.webPushConfig.VAPIDPrivateKey,
			Subscriber:      repo.webPushConfig.VAPIDContact,
			TTL:             repo.webPushConfig.TTL,
			Urgency:         webpush.UrgencyHigh, // Пробуждает устройство на iOS
		},
	)
	if err != nil {
		return fmt.Errorf("send web push notification: %w", err)
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			repo.logger.Error("failed to close web push notification response body", err)
		}
	}()

	if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf(
			"%w: id=%d, status=%d",
			customerrors.ErrWebPushSubscriptionExpired,
			subscription.ID,
			resp.StatusCode,
		)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"web push notification rejected: id=%d, status=%d, endpoint=%s, body=%s",
			subscription.ID,
			resp.StatusCode,
			subscription.Endpoint,
			string(body),
		)
	}

	return nil
}
