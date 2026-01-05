package ws

import (
	"errors"
	"net/http"
	"slices"
	"sync"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	"github.com/DKhorkov/libs/logging"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
	"github.com/gorilla/websocket"
)

type Handler struct {
	upgrader    interfaces.Upgrader
	users       interfaces.UsersUseCases
	chats       interfaces.ChatsUseCases
	logger      logging.Logger
	connections *sync.Map
}

func New(
	upgrader interfaces.Upgrader,
	usersUseCases interfaces.UsersUseCases,
	chatsUseCases interfaces.ChatsUseCases,
	logger logging.Logger,
) Handler {
	return Handler{
		upgrader:    upgrader,
		users:       usersUseCases,
		chats:       chatsUseCases,
		logger:      logger,
		connections: new(sync.Map),
	}
}

// swagger:route GET /ws websockets CreateWebsocket
//
// CreateWebsocket
//
// Creates websocket connection for current user.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	101: SwitchingProtocols
//	401: Unauthorized
//	404: NotFound
//	500: InternalServerError

// Handle creates websocket connection for current user.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, err := contextlib.ValueFromContext[uint64](
		r.Context(),
		middlewares.UserIDContextKey,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)

		return
	}

	user, err := h.users.GetUserByID(r.Context(), userID)

	switch {
	case errors.Is(err, customerrors.ErrUserNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)

		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.LogError(
			h.logger,
			"Failed to upgrade connection",
			err,
		)

		return
	}

	defer func() {
		if err = conn.Close(); err != nil {
			logging.LogError(
				h.logger,
				"Failed to close connection",
				err,
			)
		}
	}()

	h.connections.Store(user.ID, conn)

	h.listen(conn, user)

	h.connections.Delete(user.ID)
}

func (h *Handler) listen(conn *websocket.Conn, user *domains.User) {
	for {
		message := domains.Message{
			Sender: &domains.Sender{
				UserID:   user.ID,
				Username: user.Username,
			},
		}

		if err := conn.ReadJSON(&message); err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				logging.LogError(
					h.logger,
					"Failed to read message",
					err,
					"Sender", message.Sender,
				)
			}

			return
		}

		chatMembers, err := h.chats.GetChatMembers(message.ChatID)
		if err != nil {
			logging.LogError(
				h.logger,
				"Failed to get chat members",
				err,
				"Message", message,
			)

			return
		}

		if !h.senderIsChatMember(*message.Sender, chatMembers) {
			logging.LogInfo(
				h.logger,
				"Sender is not a chat member",
				"Message", message,
			)

			return
		}

		for _, member := range chatMembers {
			// Не отправляем обратно отправителю:
			if member.ID == user.ID {
				continue
			}

			value, exists := h.connections.Load(member.ID)
			if !exists {
				continue
			}

			connection, ok := value.(*websocket.Conn)
			if !ok {
				logging.LogInfo(
					h.logger,
					"Failed to parse connection from sync.Map value",
					"Message", message,
					"ChatMember", member,
				)

				h.connections.Delete(member.ID)

				continue
			}

			if err = connection.WriteJSON(message); err != nil {
				logging.LogError(
					h.logger,
					"Failed to write message",
					err,
					"Message", message,
					"ChatMember", member,
				)

				if err = connection.Close(); err != nil {
					logging.LogError(
						h.logger,
						"Failed to close connection",
						err,
					)
				}

				h.connections.Delete(member.ID)
			}
		}
	}
}

func (h *Handler) senderIsChatMember(sender domains.Sender, chatMembers []domains.User) bool {
	return slices.ContainsFunc(
		chatMembers,
		func(member domains.User) bool {
			return member.ID == sender.UserID
		},
	)
}
