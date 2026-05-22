package logout

import (
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/login"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/cookies"
)

// swagger:route DELETE /api/sessions sessions Logout
//
// Logout
//
// Logout User from the current session and deletes access and refresh tokens.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	204: NoContent
//	401: Unauthorized
//	500: InternalServerError

// Handler logouts User from current session.
func Handler(u interfaces.AuthUseCases, cookiesConfig config.CookiesConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		refreshTokenCookie, err := r.Cookie(login.RefreshTokenCookieName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		if err = u.LogoutUser(r.Context(), refreshTokenCookie.Value); err != nil {
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
