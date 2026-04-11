package user_by_id

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/controllers/http/mappers"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/gorilla/mux"
)

// swagger:route GET /users/{id} users GetUserByID
//
// GetUserByID
//
// Provides User with specified ID.
//
// Responses:
//	200: User
//	400: BadRequest
//	404: NotFound
//	500: InternalServerError

// Handler provides information User with provided ID.
func Handler(u interfaces.UsersUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDStr := mux.Vars(r)[common.IDRouteKey]

		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

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
