package get

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	mappers "github.com/DKhorkov/kfc/internal/controllers/http/mappers/settings"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
)

// swagger:route GET /api/users/me/settings users GetSettings
//
// GetSettings
//
// Provides settings of the current authorized User.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	200: Settings
//	401: Unauthorized
//	404: NotFound
//	500: InternalServerError

// Handler provides settings of the current authorized User.
func Handler(s interfaces.SettingsUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		settings, err := s.GetSettingsByUserID(r.Context(), userID)

		switch {
		case errors.Is(err, customerrors.ErrSettingsNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set(common.ContentTypeHeaderName, common.ApplicationJSONContentType)
		w.WriteHeader(http.StatusOK)

		if err = json.NewEncoder(w).Encode(mappers.MapSettings(*settings)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
