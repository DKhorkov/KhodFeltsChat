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

// swagger:route POST /users/password/forget users SendForgetPasswordMessage
//
// SendForgetPasswordMessage
//
// Sends message with information how to forget old password.
//
// Responses:
//	204: NoContent
//	400: BadRequest
//	403: Forbidden
//	404: NotFound
//	500: InternalServerError

// SendForgetPasswordMessageHandler sends message with information how to forget old password.
func SendForgetPasswordMessageHandler(u interfaces.AuthUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var input schemas.SendForgetPasswordInput
		if err = json.Unmarshal(data, &input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		err = u.SendForgetPasswordMessage(r.Context(), input.Body.Email)

		switch {
		case errors.Is(err, customerrors.ErrUserNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case errors.Is(err, customerrors.ErrEmailNotConfirmed):
			http.Error(w, err.Error(), http.StatusForbidden)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
