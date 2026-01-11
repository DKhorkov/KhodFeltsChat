package chats

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/mappers"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
)

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

		if err = json.NewEncoder(w).Encode(mappers.MapChat(*createdChat)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
