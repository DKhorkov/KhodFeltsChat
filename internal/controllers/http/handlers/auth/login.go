package auth

import (
	"encoding/json"
	"errors"
	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/cookies"
	"github.com/DKhorkov/libs/validation"
	"io"
	"net/http"
)

const (
	AccessTokenCookieName  = "accessToken"
	RefreshTokenCookieName = "refreshToken"
)

func LoginHandler(u interfaces.AuthUseCases, cookiesConfig config.CookiesConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var dto domains.LoginDTO
		if err = json.Unmarshal(data, &dto); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		tokens, err := u.LoginUser(r.Context(), dto)
		switch {
		case errors.As(err, &customerrors.UserNotFoundError{}):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case errors.As(err, &customerrors.EmailIsNotConfirmedError{}):
			http.Error(w, err.Error(), http.StatusForbidden)

			return
		case errors.As(err, &customerrors.WrongPasswordError{}):
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		case errors.As(err, &validation.Error{}):
			http.Error(w, err.Error(), http.StatusBadRequest)

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
