package auth

import (
	"encoding/json"
	"errors"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"io"
	"net/http"
)

func RegisterHandler(u interfaces.AuthUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var dto domains.RegisterDTO
		if err = json.Unmarshal(data, &dto); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		user, err := u.RegisterUser(r.Context(), dto)
		switch {
		case errors.Is(err, customerrors.ErrUserAlreadyExists):
			http.Error(w, err.Error(), http.StatusConflict)

			return
		case errors.Is(err, customerrors.ErrValidationFailed):
			http.Error(w, err.Error(), http.StatusBadRequest)

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
