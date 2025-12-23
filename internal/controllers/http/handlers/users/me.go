package users

import (
	"encoding/json"
	"errors"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"net/http"
)

func GetMeHandler(u interfaces.UsersUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessTokenCookie, err := r.Cookie(auth.AccessTokenCookieName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		user, err := u.GetMe(r.Context(), accessTokenCookie.Value)
		switch {
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

		if err = json.NewEncoder(w).Encode(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
