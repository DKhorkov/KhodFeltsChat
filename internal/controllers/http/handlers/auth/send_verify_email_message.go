package auth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

// swagger:route POST /users/email/verify users SendVerifyEmailMessage
//
// SendVerifyEmailMessage
//
// Sends message with information how to verify email.
//
// Responses:
//	204: NoContent
//	400: BadRequest
//	404: NotFound
//	409: Conflict
//	500: InternalServerError

// SendVerifyEmailMessageHandler sends message with information how to verify.
func SendVerifyEmailMessageHandler(u interfaces.AuthUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var input schemas.SendVerifyEmailInput
		if err = json.Unmarshal(data, &input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		err = u.SendVerifyEmailMessage(r.Context(), input.Body.Email)

		switch {
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

		w.WriteHeader(http.StatusNoContent)
	}
}
