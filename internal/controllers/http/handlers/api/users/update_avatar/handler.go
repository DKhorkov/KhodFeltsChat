package update_avatar

import (
	"errors"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
)

const (
	formFileKey = "avatar"
	maxMemory   = 20 << 20 // 20 MB
)

// swagger:route PUT /api/users/me/avatar users UpdateAvatar
//
// UpdateAvatar
//
// Uploads or updates the avatar of the current user.
// Accepts multipart/form-data with an "avatar" file field.
// Image is resized to 256x256 JPEG.
//
// Security:
// - cookieAuth: []
//
// Consumes:
// - multipart/form-data
//
// Responses:
//	200: OK
//	400: BadRequest
//	401: Unauthorized
//	404: NotFound
//	413: RequestEntityTooLarge
//	500: InternalServerError

// Handler uploads or updates the current user's avatar.
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

		r.Body = http.MaxBytesReader(w, r.Body, maxMemory)

		//nolint:gosec // body is limited by MaxBytesReader above.
		if err = r.ParseMultipartForm(maxMemory); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		file, _, err := r.FormFile(formFileKey)
		if err != nil {
			http.Error(w, "avatar file is required", http.StatusBadRequest)

			return
		}

		defer func() {
			if err = file.Close(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}()

		avatarURL, err := u.UpdateAvatar(r.Context(), userID, file)

		switch {
		case errors.Is(err, customerrors.ErrInvalidImageFormat):
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		case errors.Is(err, customerrors.ErrFileTooLarge):
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)

			return
		case errors.Is(err, customerrors.ErrUserNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set(common.ContentTypeHeaderName, common.TextPlainContentType)
		w.WriteHeader(http.StatusOK)

		if _, err = w.Write([]byte(avatarURL)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
