package delete_avatar

import (
	"errors"
	"net/http"

	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
)

// swagger:route DELETE /api/users/me/avatar users DeleteAvatar
//
// DeleteAvatar
//
// Deletes the avatar of the current user.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	204: NoContent
//	401: Unauthorized
//	404: NotFound
//	500: InternalServerError

// Handler deletes the current user's avatar.
func Handler(u interfaces.UsersUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		err = u.DeleteAvatar(r.Context(), userID)

		switch {
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
