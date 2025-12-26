package auth

import (
	"errors"
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/cookies"
)

// swagger:route PUT /sessions sessions RefreshTokens
//
// RefreshTokens
//
// Refreshes accessToken and refreshToken of current user.
//
// Responses:
//	204: NoContent
//	401: Unauthorized
//	500: InternalServerError

// RefreshTokensHandler refreshes accessToken and refreshToken of current user.
func RefreshTokensHandler(
	u interfaces.AuthUseCases,
	cookiesConfig config.CookiesConfig,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		refreshTokenCookie, err := r.Cookie(RefreshTokenCookieName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		tokens, err := u.RefreshTokens(r.Context(), refreshTokenCookie.Value)

		switch {
		case errors.Is(err, customerrors.ErrInvalidJWT),
			errors.Is(err, customerrors.ErrAccessTokenDoesNotBelongToRefreshToken):
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		cookies.Set(w, AccessTokenCookieName, tokens.AccessToken, cookiesConfig.AccessToken)
		cookies.Set(w, RefreshTokenCookieName, tokens.RefreshToken, cookiesConfig.RefreshToken)

		w.WriteHeader(http.StatusNoContent)
	}
}
