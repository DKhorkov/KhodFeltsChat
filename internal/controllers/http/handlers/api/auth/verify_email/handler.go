package verify_email

import (
	"errors"
	"net/http"
	"strconv"

	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/gorilla/mux"
)

const (
	TokenRouteKey = "verifyEmailToken"
)

// swagger:route GET /api/users/email/verify/{verifyEmailToken} users VerifyEmail
//
// VerifyEmail
//
// Verifies email for user with provided verifyEmailToken.
//
// Responses:
//	204: NoContent
//	401: Unauthorized
//	404: NotFound
//	409: Conflict
//	500: InternalServerError

// Handler changes forgotten password to new password of current user.
func Handler(u interfaces.AuthUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code, err := strconv.ParseUint(mux.Vars(r)[TokenRouteKey], 10, 64)
		if err != nil {
			http.Error(w, customerrors.ErrInvalidJWT.Error(), http.StatusUnauthorized)

			return
		}

		err = u.VerifyEmail(r.Context(), code)

		switch {
		case errors.Is(err, customerrors.ErrInvalidJWT):
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		case errors.Is(err, customerrors.ErrUserNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case errors.Is(err, customerrors.ErrEmailAlreadyConfirmed):
			http.Error(w, err.Error(), http.StatusConflict)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
