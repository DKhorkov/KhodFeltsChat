package web_push_subscriptions

import (
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
)

func MapCreateResponse(subscription domains.WebPushSubscription) schemas.CreateWebPushSubscriptionResponse {
	return schemas.CreateWebPushSubscriptionResponse{
		ID: subscription.ID,
	}
}
