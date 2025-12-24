package auth

import (
	"encoding/json"
	"errors"
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"io"
	"net/http"

	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

func SendForgetPasswordMessageHandler(u interfaces.AuthUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var dto schemas.SendForgetPasswordDTO
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

		w.WriteHeader(http.StatusOK)
	}
}
