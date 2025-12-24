package auth

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

const (
	VerifyEmailTokenRouteKey = "verify_email_token"
)

func VerifyEmailHandler(u interfaces.AuthUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		verifyEmailToken := mux.Vars(r)[VerifyEmailTokenRouteKey]

		err := u.VerifyEmail(r.Context(), verifyEmailToken)
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

		w.WriteHeader(http.StatusOK)
	}
}
