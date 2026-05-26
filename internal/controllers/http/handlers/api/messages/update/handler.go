package update_message

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/gorilla/mux"
)

// swagger:route PUT /api/messages/{id} messages UpdateMessage
//
// UpdateMessage
//
// Updates text of a message by ID. Only the author can edit.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	204: NoContent
//	400: BadRequest
//	401: Unauthorized
//	403: Forbidden
//	404: NotFound
//	500: InternalServerError

// Handler updates a message text.
func Handler(u interfaces.MessagesUseCases, broadcaster interfaces.WSBroadcaster) http.HandlerFunc {
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

		var dto domains.UpdateMessageDTO
		if err = json.NewDecoder(r.Body).Decode(&dto); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		dto.MessageID = messageID
		dto.UserID = userID

		if dto.Text == "" {
			http.Error(w, "text is required", http.StatusBadRequest)

			return
		}

		updatedMessage, err := u.UpdateMessage(r.Context(), dto)

		switch {
		case errors.Is(err, customerrors.ErrMessageNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case errors.Is(err, customerrors.ErrNotMessageAuthor):
			http.Error(w, err.Error(), http.StatusForbidden)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		broadcaster.BroadcastMessageEdited(r.Context(), updatedMessage.ChatID, messageID)

		w.WriteHeader(http.StatusNoContent)
	}
}
