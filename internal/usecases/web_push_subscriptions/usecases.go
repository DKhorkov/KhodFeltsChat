package web_push_subscriptions

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

type UseCases struct {
	webPushSubscriptionsService interfaces.WebPushSubscriptionsService
}

func New(
	webPushSubscriptionsService interfaces.WebPushSubscriptionsService,
) *UseCases {
	return &UseCases{
		webPushSubscriptionsService: webPushSubscriptionsService,
	}
}

func (u *UseCases) CreateWebPushSubscription(
	ctx context.Context,
	subscription domains.WebPushSubscription,
) (*domains.WebPushSubscription, error) {
	return u.webPushSubscriptionsService.CreateWebPushSubscription(ctx, subscription)
}

func (u *UseCases) DeleteWebPushSubscription(ctx context.Context, id uint64) error {
	return u.webPushSubscriptionsService.DeleteWebPushSubscription(ctx, id)
}
