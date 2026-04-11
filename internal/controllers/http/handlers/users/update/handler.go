package update

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	mappers "github.com/DKhorkov/kfc/internal/controllers/http/mappers/users"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
)

// swagger:route PUT /users/me users UpdateCurrentUser
//
// UpdateCurrentUser
//
// Updates information about User with specified ID.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	200: User
//	400: BadRequest
//	401: Unauthorized
//	404: NotFound
//	500: InternalServerError

// Handler updates current User.
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

		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var dto domains.UpdateUserDTO
		if err = json.Unmarshal(data, &dto); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		dto.ID = userID

		user, err := u.UpdateUser(r.Context(), dto)

		switch {
		case errors.Is(err, customerrors.ErrValidationFailed):
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
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
