package me

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	mappers "github.com/DKhorkov/kfc/internal/controllers/http/mappers/users"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
)

// swagger:route GET /users/me users GetCurrentUser
//
// GetCurrentUser
//
// Provides current authorized User.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	200: User
//	401: Unauthorized
//	404: NotFound
//	500: InternalServerError

// Handler provides information about current authorized User.
func Handler(u interfaces.UsersUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			middlewares.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		user, err := u.GetUserByID(r.Context(), userID)

		switch {
		case errors.Is(err, customerrors.ErrUserNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		// Устанавливаем статус код и заголовок Content-Type
		w.Header().Set(common.ContentTypeHeaderName, common.ApplicationJSONContentType)

		// HTTP статус код должен устанавливаться перед записью тела ответа
		w.WriteHeader(http.StatusOK)

		if err = json.NewEncoder(w).Encode(mappers.MapUser(*user)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
