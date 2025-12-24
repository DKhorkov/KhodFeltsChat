package auth

import (
	"errors"
	"net/http"

	"github.com/DKhorkov/libs/cookies"

	"github.com/DKhorkov/kfc/internal/config"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

func RefreshTokensHandler(u interfaces.AuthUseCases, cookiesConfig config.CookiesConfig) http.HandlerFunc {
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

		w.WriteHeader(http.StatusOK)
	}
}
