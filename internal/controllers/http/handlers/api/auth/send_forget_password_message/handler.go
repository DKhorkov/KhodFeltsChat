package send_forget_password_message

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/DKhorkov/kfc/internal/domains"
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

// Handler sends message with information how to forget old password.
func Handler(u interfaces.AuthUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var dto domains.SendForgetPasswordMessageDTO
		if err = json.Unmarshal(data, &dto); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		err = u.SendForgetPasswordMessage(r.Context(), dto.Email)

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
