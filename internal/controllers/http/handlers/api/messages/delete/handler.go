package delete_message

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

// swagger:route DELETE /api/messages/{id} messages DeleteMessage
//
// DeleteMessage
//
// Deletes a message by ID. If forAll is true, deletes for all chat members (author only).
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

// Handler deletes a message.
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

		var dto domains.DeleteMessageDTO
		if err = json.NewDecoder(r.Body).Decode(&dto); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		dto.MessageID = messageID
		dto.UserID = userID

		// Fetch message before deletion to get chatID for WS broadcast:
		var chatID uint64

		if dto.ForAll {
			message, err := u.GetMessageByID(r.Context(), userID, messageID)
			if err != nil {
				http.Error(w, customerrors.ErrMessageNotFound.Error(), http.StatusNotFound)

				return
			}

			chatID = message.ChatID
		}

		err = u.DeleteMessage(r.Context(), dto)

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

		if dto.ForAll {
			broadcaster.BroadcastMessageDeleted(r.Context(), chatID, messageID, userID)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
