package auth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

// swagger:route POST /users/password/change users ChangePassword
//
// ChangePassword
//
// Changes old password to new password of current user.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	204: NoContent
//	400: BadRequest
//	401: Unauthorized
//	404: NotFound
//	500: InternalServerError

// ChangePasswordHandler changes old password to new password of current user.
func ChangePasswordHandler(u interfaces.AuthUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessTokenCookie, err := r.Cookie(AccessTokenCookieName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var dto domains.ChangePasswordDTO
		if err = json.Unmarshal(data, &dto); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		err = u.ChangePassword(
			r.Context(),
			accessTokenCookie.Value,
			dto.OldPassword,
			dto.NewPassword,
		)

		switch {
		case errors.Is(err, customerrors.ErrValidationFailed),
			errors.Is(err, customerrors.ErrWrongPassword):
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

		w.WriteHeader(http.StatusNoContent)
	}
}
