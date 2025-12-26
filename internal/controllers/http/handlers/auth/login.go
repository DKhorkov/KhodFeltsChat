package auth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/cookies"
)

const (
	AccessTokenCookieName  = "accessToken"
	RefreshTokenCookieName = "refreshToken"
)

// swagger:route POST /sessions sessions Login
//
// Login
//
// Logins User and provides access and refresh tokens.
//
// Responses:
//	204: NoContent
//	400: BadRequest
//	401: Unauthorized
//	403: Forbidden
//	404: NotFound
//	500: InternalServerError

// LoginHandler logins User.
func LoginHandler(u interfaces.AuthUseCases, cookiesConfig config.CookiesConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var input schemas.LoginInput
		if err = json.Unmarshal(data, &input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		dto := domains.LoginDTO{
			Email:    input.Body.Email,
			Password: input.Body.Password,
		}

		tokens, err := u.LoginUser(r.Context(), dto)

		switch {
		case errors.Is(err, customerrors.ErrUserNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case errors.Is(err, customerrors.ErrEmailNotConfirmed):
			http.Error(w, err.Error(), http.StatusForbidden)

			return
		case errors.Is(err, customerrors.ErrWrongPassword):
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		case errors.Is(err, customerrors.ErrValidationFailed):
			http.Error(w, err.Error(), http.StatusBadRequest)

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
