package auth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

const (
	ForgetPasswordTokenRouteKey = "forgetPasswordToken"
)

// swagger:route POST /users/password/forget/{forgetPasswordToken} users ForgetPassword
//
// ForgetPassword
//
// Changes forgotten password to new password for user with provided forgetPasswordToken.
//
// Responses:
//	204: NoContent
//	400: BadRequest
//	401: Unauthorized
//	404: NotFound
//	500: InternalServerError

// ForgetPasswordHandler changes forgotten password to new password of current user.
func ForgetPasswordHandler(u interfaces.AuthUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		forgetPasswordToken := mux.Vars(r)[ForgetPasswordTokenRouteKey]

		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var input schemas.ForgetPasswordInput
		if err = json.Unmarshal(data, &input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		err = u.ForgetPassword(r.Context(), forgetPasswordToken, input.Body.NewPassword)

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

		w.WriteHeader(http.StatusNoContent)
	}
}
