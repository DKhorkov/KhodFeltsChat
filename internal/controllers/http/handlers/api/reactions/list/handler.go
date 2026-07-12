package list

import (
	"encoding/json"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	reactionsmapper "github.com/DKhorkov/kfc/internal/controllers/http/mappers/reactions"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

// swagger:route GET /api/reactions reactions ListReactions
//
// ListReactions
//
// Provides the dictionary of available emoji reactions for the reactions picker.
//
// Responses:
//	200: []Reaction
//	401: Unauthorized
//	500: InternalServerError

// Handler returns the dictionary of available reactions.
func Handler(u interfaces.ReactionsUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reactions, err := u.ListReactions(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set(common.ContentTypeHeaderName, common.ApplicationJSONContentType)
		w.WriteHeader(http.StatusOK)

		if err = json.NewEncoder(w).Encode(reactionsmapper.MapReactions(reactions)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
