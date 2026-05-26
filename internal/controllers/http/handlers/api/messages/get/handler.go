package get_message

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/controllers/http/mappers/messages"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/gorilla/mux"
)

// swagger:route GET /api/messages/{id} messages GetMessage
//
// GetMessage
//
// Returns a message by ID.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	200: Message
//	400: BadRequest
//	401: Unauthorized
//	404: NotFound
//	500: InternalServerError

// Handler returns a message by ID.
func Handler(u interfaces.MessagesUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		messageIDStr := mux.Vars(r)[common.IDRouteKey]

		messageID, err := strconv.ParseUint(messageIDStr, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		message, err := u.GetMessageByID(r.Context(), userID, messageID)

		switch {
		case errors.Is(err, customerrors.ErrMessageNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err = json.NewEncoder(w).Encode(messages.MapMessage(*message)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
