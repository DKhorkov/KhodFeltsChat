package logout

import (
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/login"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	"github.com/DKhorkov/libs/cookies"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
)

// swagger:route DELETE /api/sessions sessions Logout
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
func Handler(u interfaces.AuthUseCases, cookiesConfig config.CookiesConfig) http.HandlerFunc {
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
		cookiesConfig.AccessToken.MaxAge = -1
		cookiesConfig.RefreshToken.MaxAge = -1
		cookies.Set(w, login.AccessTokenCookieName, "", cookiesConfig.AccessToken)
		cookies.Set(w, login.RefreshTokenCookieName, "", cookiesConfig.RefreshToken)

		w.WriteHeader(http.StatusNoContent)
	}
}
