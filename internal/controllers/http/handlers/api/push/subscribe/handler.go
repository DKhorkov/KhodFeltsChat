package subscribe

import (
	"encoding/json"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	mappers "github.com/DKhorkov/kfc/internal/controllers/http/mappers/push_subscriptions"
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
)

// swagger:route POST /api/push/subscribe web-pushes CreatePushSubscription
//
// CreatePushSubscription
//
// Creates a new push subscription for the current authorized User.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	201: CreatePushSubscriptionResponse
//	400: BadRequest
//	401: Unauthorized
//	500: InternalServerError

// Handler creates a new push subscription for the current authorized User.
func Handler(u interfaces.PushSubscriptionsUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		var body schemas.CreatePushSubscriptionRequest
		if err = json.NewDecoder(r.Body).Decode(&body.Body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		subscription, err := u.CreatePushSubscription(
			r.Context(),
			domains.PushSubscription{
				UserID:        userID,
				Endpoint:      body.Body.Endpoint,
				EncryptionKey: body.Body.Keys.EncryptionKey,
				Auth:          body.Body.Keys.Auth,
			},
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set(common.ContentTypeHeaderName, common.ApplicationJSONContentType)
		w.WriteHeader(http.StatusCreated)

		if err = json.NewEncoder(w).Encode(mappers.MapCreateResponse(*subscription)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
