package vapid_key

import (
	"encoding/json"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
)

// swagger:route GET /api/web-push/vapid-key web-pushes GetVAPIDKey
//
// GetVAPIDKey
//
// Returns the VAPID public key for push subscription registration.
//
// Responses:
//	200: VAPIDKeyResponse
//	500: InternalServerError

// Handler returns the VAPID public key.
func Handler(vapidPublicKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(common.ContentTypeHeaderName, common.ApplicationJSONContentType)
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(schemas.VAPIDKeyResponse{PublicKey: vapidPublicKey}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
