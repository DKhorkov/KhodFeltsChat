package logout

import (
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/login"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	"github.com/DKhorkov/libs/cookies"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
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

// Handler logouts User.
func Handler(u interfaces.AuthUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		if err = u.LogoutUser(r.Context(), userID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		// Deleting cookies:
		cookies.Set(w, login.AccessTokenCookieName, "", cookies.Config{MaxAge: -1})
		cookies.Set(w, login.RefreshTokenCookieName, "", cookies.Config{MaxAge: -1})

		w.WriteHeader(http.StatusNoContent)
	}
}
