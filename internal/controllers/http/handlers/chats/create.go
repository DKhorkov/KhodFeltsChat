package chats

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/controllers/http/mappers"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
)

// swagger:route POST /chats chats CreateChat
//
// CreateChat
//
// CreateChat creates new chat with provided info.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	201: Chat
//	400: BadRequest
//	401: Unauthorized
//	500: InternalServerError

// CreateChatHandler creates new Chat.
func CreateChatHandler(u interfaces.ChatsUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			middlewares.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var chat domains.Chat
		if err = json.Unmarshal(data, &chat); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		// Добавляем текущего пользователя в участники чата:
		chat.Members = append(chat.Members, domains.User{ID: userID})

		createdChat, err := u.CreateChat(r.Context(), chat)

		switch {
		case errors.Is(err, customerrors.ErrInvalidChat):
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		// Устанавливаем статус код и заголовок Content-Type
		w.Header().Set(common.ContentTypeHeaderName, common.ApplicationJSONContentType)

		// HTTP статус код должен устанавливаться перед записью тела ответа
		w.WriteHeader(http.StatusCreated)

		if err = json.NewEncoder(w).Encode(mappers.MapChat(*createdChat)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
