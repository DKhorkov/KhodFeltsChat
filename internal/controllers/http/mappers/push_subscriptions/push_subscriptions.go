package push_subscriptions

import (
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
)

func MapCreateResponse(subscription domains.PushSubscription) schemas.CreatePushSubscriptionResponse {
	return schemas.CreatePushSubscriptionResponse{
		ID: subscription.ID,
	}
}
