package auth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

// swagger:route POST /users users RegisterUser
//
// RegisterUser
//
// Registers new User with provided info.
//
// Responses:
//	201: User
//	400: BadRequest
//	409: Conflict
//	500: InternalServerError

// RegisterHandler creates new User.
func RegisterHandler(u interfaces.AuthUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var input schemas.RegisterInput
		if err = json.Unmarshal(data, &input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		dto := domains.RegisterDTO{
			Username: input.Body.Username,
			Email:    input.Body.Email,
			Password: input.Body.Password,
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

		w.WriteHeader(http.StatusCreated)
	}
}
