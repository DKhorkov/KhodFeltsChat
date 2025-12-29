package users

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth"
	"github.com/DKhorkov/kfc/internal/controllers/http/mappers"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
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

// UpdateCurrentUserHandler updates current User.
func UpdateCurrentUserHandler(u interfaces.UsersUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var dto domains.RawUpdateUserDTO
		if err = json.Unmarshal(data, &dto); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		accessTokenCookie, err := r.Cookie(auth.AccessTokenCookieName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		dto.AccessToken = accessTokenCookie.Value

		user, err := u.UpdateUser(r.Context(), dto)

		switch {
		case errors.Is(err, customerrors.ErrValidationFailed):
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		case errors.Is(err, customerrors.ErrInvalidJWT):
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		case errors.Is(err, customerrors.ErrUserNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		if err = json.NewEncoder(w).Encode(mappers.MapUser(*user)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
