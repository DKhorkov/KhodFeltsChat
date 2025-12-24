package auth

import (
	"encoding/json"
	"errors"
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

const (
	ForgetPasswordTokenRouteKey = "forget_password_token"
)

func ForgetPasswordHandler(u interfaces.AuthUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		forgetPasswordToken := mux.Vars(r)[ForgetPasswordTokenRouteKey]

		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var dto schemas.ForgetPasswordDTO
		if err = json.Unmarshal(data, &dto); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		err = u.ForgetPassword(r.Context(), forgetPasswordToken, dto.NewPassword)

		switch {
		case errors.Is(err, customerrors.ErrValidationFailed):
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
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

		w.WriteHeader(http.StatusOK)
	}
}
