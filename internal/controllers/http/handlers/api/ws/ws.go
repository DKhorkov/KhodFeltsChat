package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"sync"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/mappers/messages"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	"github.com/DKhorkov/libs/logging"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	customnats "github.com/DKhorkov/libs/nats"
	"github.com/gorilla/websocket"
)

type Handler struct {
	upgrader         interfaces.Upgrader
	usersUseCases    interfaces.UsersUseCases
	chatsUseCases    interfaces.ChatsUseCases
	messagesUseCases interfaces.MessagesUseCases
	logger           logging.Logger
	connections      *sync.Map
	natsPublisher    customnats.Publisher
	natsConfig       config.NATSConfig
}

func New(
	upgrader interfaces.Upgrader,
	usersUseCases interfaces.UsersUseCases,
	chatsUseCases interfaces.ChatsUseCases,
	messagesUseCases interfaces.MessagesUseCases,
	logger logging.Logger,
	natsPublisher customnats.Publisher,
	natsConfig config.NATSConfig,
) Handler {
	return Handler{
		upgrader:         upgrader,
		usersUseCases:    usersUseCases,
		chatsUseCases:    chatsUseCases,
		messagesUseCases: messagesUseCases,
		logger:           logger,
		connections:      new(sync.Map),
		natsPublisher:    natsPublisher,
		natsConfig:       natsConfig,
	}
}

// swagger:route GET /api/ws websockets CreateWebsocket
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
		authmiddleware.UserIDContextKey,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)

		return
	}

	user, err := h.usersUseCases.GetUserByID(r.Context(), userID)

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
		ctx := context.Background()
		message := domains.NewMessage().From(*user).Received()

		if err := conn.ReadJSON(message); err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				logging.LogErrorContext(
					ctx,
					h.logger,
					"Failed to read message",
					err,
					"Sender", message.Sender,
				)
			}

			return
		}

		chatMembers, err := h.chatsUseCases.GetChatMembers(ctx, message.ChatID)
		if err != nil {
			logging.LogErrorContext(
				ctx,
				h.logger,
				"Failed to get chat members",
				err,
				"Message", message,
			)

			return
		}

		if !senderIsChatMember(message.Sender, chatMembers) {
			logging.LogInfoContext(
				ctx,
				h.logger,
				"Sender is not a chat member",
				"Message", message,
			)

			return
		}

		// Сохраняем сообщение в БД и возвращаем полное доменное сообщение:
		var savedMessage *domains.Message

		if savedMessage, err = h.messagesUseCases.SaveMessage(ctx, *message); err != nil {
			logging.LogErrorContext(
				ctx,
				h.logger,
				"Failed to save message",
				err,
				"Message", message,
			)

			return
		}

		messageToSend := messages.MapMessage(*savedMessage)

		messageToSend.IsRead = false // Соощбение не прочитано дял всех получателей, так как является новым

		for _, member := range chatMembers {
			// Не отправляем обратно отправителю:
			if member.ID == user.ID {
				continue
			}

			value, exists := h.connections.Load(member.ID)
			if !exists {
				h.publishWebPushNotification(ctx, member.ID, savedMessage.ID)

				continue
			}

			connection, ok := value.(*websocket.Conn)
			if !ok {
				logging.LogInfoContext(
					ctx,
					h.logger,
					"Failed to parse connection from sync.Map value",
					"Message", messageToSend,
					"ChatMember", member,
				)

				h.connections.Delete(member.ID)

				continue
			}

			if err = connection.WriteJSON(messageToSend); err != nil {
				logging.LogErrorContext(
					ctx,
					h.logger,
					"Failed to write message",
					err,
					"Message", messageToSend,
					"ChatMember", member,
				)

				if err = connection.Close(); err != nil {
					logging.LogErrorContext(
						ctx,
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

func (h *Handler) publishWebPushNotification(ctx context.Context, userID, messageID uint64) {
	pushDTO := domains.WebPushNotificationDTO{
		UserID:    userID,
		MessageID: messageID,
	}

	content, err := json.Marshal(pushDTO)
	if err != nil {
		logging.LogErrorContext(
			ctx,
			h.logger,
			"Failed to marshal push notification DTO",
			err,
		)

		return
	}

	if err = h.natsPublisher.Publish(
		h.natsConfig.Subjects.WebPushNotification,
		content,
	); err != nil {
		logging.LogErrorContext(
			ctx,
			h.logger,
			"Failed to publish push notification",
			err,
		)
	}
}

func senderIsChatMember(sender domains.User, chatMembers []domains.User) bool {
	return slices.ContainsFunc(
		chatMembers,
		func(member domains.User) bool {
			return member.ID == sender.ID
		},
	)
}
