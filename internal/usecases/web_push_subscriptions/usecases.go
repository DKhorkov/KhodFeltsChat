package web_push_subscriptions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	"github.com/SherClockHolmes/webpush-go"
)

type UseCases struct {
	webPushSubscriptionsService interfaces.WebPushSubscriptionsService
	webPushConfig               config.WebPushConfig
	logger                      logging.Logger
}

func New(
	webPushSubscriptionsService interfaces.WebPushSubscriptionsService,
	webPushConfig config.WebPushConfig,
	logger logging.Logger,
) *UseCases {
	return &UseCases{
		webPushSubscriptionsService: webPushSubscriptionsService,
		webPushConfig:               webPushConfig,
		logger:                      logger,
	}
}

func (u *UseCases) CreateWebPushSubscription(
	ctx context.Context,
	subscription domains.WebPushSubscription,
) (*domains.WebPushSubscription, error) {
	return u.webPushSubscriptionsService.CreateWebPushSubscription(ctx, subscription)
}

func (u *UseCases) GetWebPushSubscriptionsByUserID(
	ctx context.Context,
	userID uint64,
) ([]domains.WebPushSubscription, error) {
	return u.webPushSubscriptionsService.GetWebPushSubscriptionsByUserID(ctx, userID)
}

func (u *UseCases) DeleteWebPushSubscription(ctx context.Context, id uint64) error {
	return u.webPushSubscriptionsService.DeleteWebPushSubscription(ctx, id)
}

func (u *UseCases) SendWebPushNotification(
	ctx context.Context,
	subscription domains.WebPushSubscription,
	message domains.Message,
) error {
	payload, err := json.Marshal(map[string]any{
		"title":  message.Sender.Username,
		"body":   message.Text,
		"chatId": message.ChatID,
	})
	if err != nil {
		return fmt.Errorf("marshal push notification payload: %w", err)
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
		&webpush.Options{
			VAPIDPublicKey:  u.webPushConfig.VAPIDPublicKey,
			VAPIDPrivateKey: u.webPushConfig.VAPIDPrivateKey,
			Subscriber:      u.webPushConfig.VAPIDContact,
		},
	)
	if err != nil {
		return fmt.Errorf("send push notification: %w", err)
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			u.logger.Error("failed to close push notification response body", err)
		}
	}()

	if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
		u.logger.Info(
			fmt.Sprintf(
				"push subscription %d is no longer valid (status %d), deleting",
				subscription.ID,
				resp.StatusCode,
			),
		)

		return u.webPushSubscriptionsService.DeleteWebPushSubscription(ctx, subscription.ID)
	}

	return nil
}
