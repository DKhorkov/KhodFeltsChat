package unsubscribe

import (
	"net/http"
	"strconv"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/gorilla/mux"
)

// swagger:route DELETE /api/push/subscriptions/{id} web-pushes DeletePushSubscription
//
// DeletePushSubscription
//
// Deletes a push subscription by ID for the current authorized User.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	204:
//	400: BadRequest
//	401: Unauthorized
//	500: InternalServerError

// Handler deletes a push subscription by ID.
func Handler(u interfaces.PushSubscriptionsUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		idStr := mux.Vars(r)[common.IDRouteKey]

		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		if err = u.DeletePushSubscription(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
