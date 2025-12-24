package auth

import (
	"errors"
	"net/http"

	"github.com/DKhorkov/libs/cookies"

	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

// swagger:route DELETE /sessions sessions Logout
//
// Logout
//
// Logout User and deletes access and refresh tokens.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	204: NoContent
//	401: Unauthorized
//	500: InternalServerError

// LogoutHandler logouts User.
func LogoutHandler(u interfaces.AuthUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessTokenCookie, err := r.Cookie(AccessTokenCookieName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		err = u.LogoutUser(r.Context(), accessTokenCookie.Value)

		switch {
		case errors.Is(err, customerrors.ErrInvalidJWT):
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		// Deleting cookies:
		cookies.Set(w, AccessTokenCookieName, "", cookies.Config{MaxAge: -1})
		cookies.Set(w, RefreshTokenCookieName, "", cookies.Config{MaxAge: -1})

		w.WriteHeader(http.StatusNoContent)
	}
}
