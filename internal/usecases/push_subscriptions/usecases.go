package push_subscriptions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
)

type UseCases struct {
	pushSubscriptionsService interfaces.PushSubscriptionsService
	webPushConfig            config.WebPushConfig
	logger                   logging.Logger
}

func New(
	pushSubscriptionsService interfaces.PushSubscriptionsService,
	webPushConfig config.WebPushConfig,
	logger logging.Logger,
) *UseCases {
	return &UseCases{
		pushSubscriptionsService: pushSubscriptionsService,
		webPushConfig:            webPushConfig,
		logger:                   logger,
	}
}

func (u *UseCases) CreatePushSubscription(
	ctx context.Context,
	subscription domains.PushSubscription,
) (*domains.PushSubscription, error) {
	return u.pushSubscriptionsService.CreatePushSubscription(ctx, subscription)
}

func (u *UseCases) GetPushSubscriptionsByUserID(
	ctx context.Context,
	userID uint64,
) ([]domains.PushSubscription, error) {
	return u.pushSubscriptionsService.GetPushSubscriptionsByUserID(ctx, userID)
}

func (u *UseCases) DeletePushSubscription(ctx context.Context, id uint64) error {
	return u.pushSubscriptionsService.DeletePushSubscription(ctx, id)
}

func (u *UseCases) SendPushNotification(
	ctx context.Context,
	subscription domains.PushSubscription,
	message domains.Message,
) error {
	payload, err := json.Marshal(map[string]interface{}{
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

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
		u.logger.Info(
			fmt.Sprintf("push subscription %d is no longer valid (status %d), deleting", subscription.ID, resp.StatusCode),
		)

		return u.pushSubscriptionsService.DeletePushSubscription(ctx, subscription.ID)
	}

	return nil
}
