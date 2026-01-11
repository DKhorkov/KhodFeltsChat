package messages

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/controllers/http/mappers"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
	"github.com/gorilla/mux"
)

// swagger:route GET /chats/{id}/messages chats GetChatMessages
//
// GetChatMessages
//
// Provides list of messages, which were sent to the chat with specified id, with pagination.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	200: []Message
//	400: BadRequest
//	401: Unauthorized
//	403: Forbidden
//	404: NotFound
//	500: InternalServerError

// GetChatMessagesHandler provides chat messages with pagination.
func GetChatMessagesHandler(u interfaces.MessagesUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			middlewares.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		chatIDStr := mux.Vars(r)[common.IDRouteKey]

		chatID, err := strconv.ParseUint(chatIDStr, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		pagination := common.GetPaginationFromRequest(r)

		messages, err := u.GetChatMessages(r.Context(), userID, chatID, pagination)

		switch {
		case errors.Is(err, customerrors.ErrUserIsNotChatMember):
			http.Error(w, err.Error(), http.StatusForbidden)

			return
		case errors.Is(err, customerrors.ErrChatNotFound),
			errors.Is(err, customerrors.ErrUserNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		if err = json.NewEncoder(w).Encode(mappers.MapMessages(messages)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
