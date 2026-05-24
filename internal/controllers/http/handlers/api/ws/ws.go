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

// userConnections хранит все WebSocket-соединения одного пользователя (мультисессия).
type userConnections struct {
	mu          sync.Mutex
	connections []*websocket.Conn
}

type Handler struct {
	upgrader         interfaces.Upgrader
	usersUseCases    interfaces.UsersUseCases
	chatsUseCases    interfaces.ChatsUseCases
	messagesUseCases interfaces.MessagesUseCases
	logger           logging.Logger
	connections      *sync.Map // map[uint64]*userConnections
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
) *Handler {
	return &Handler{
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

	h.addConnection(user.ID, conn)

	h.listen(conn, user)

	h.removeConnection(user.ID, conn)
}

// BroadcastMessageDeleted sends a message_deleted event to all chat members.
func (h *Handler) BroadcastMessageDeleted(
	ctx context.Context,
	chatID uint64,
	messageID uint64,
) {
	chatMembers, err := h.chatsUseCases.GetChatMembers(ctx, chatID)
	if err != nil {
		logging.LogErrorContext(
			ctx,
			h.logger,
			"Failed to get chat members for message deleted broadcast",
			err,
			"ChatID", chatID,
			"MessageID", messageID,
		)

		return
	}

	event := domains.WSEvent{
		Type: domains.WSEventMessageDeleted,
		Payload: domains.MessageDeletedPayload{
			MessageID: messageID,
			ChatID:    chatID,
		},
	}

	for _, member := range chatMembers {
		h.sendToUser(ctx, member.ID, event)
	}
}

func (h *Handler) addConnection(userID uint64, conn *websocket.Conn) {
	val, _ := h.connections.LoadOrStore(userID, &userConnections{})

	uc, ok := val.(*userConnections)
	if !ok {
		h.connections.Delete(userID)

		return
	}

	uc.mu.Lock()
	uc.connections = append(uc.connections, conn)
	uc.mu.Unlock()
}

func (h *Handler) removeConnection(userID uint64, conn *websocket.Conn) {
	val, ok := h.connections.Load(userID)
	if !ok {
		return
	}

	uc, ok := val.(*userConnections)
	if !ok {
		h.connections.Delete(userID)

		return
	}

	uc.mu.Lock()

	uc.connections = slices.DeleteFunc(uc.connections, func(c *websocket.Conn) bool {
		return c == conn
	})

	remaining := len(uc.connections)
	uc.mu.Unlock()

	if remaining == 0 {
		h.connections.Delete(userID)
	}
}

// sendToUser отправляет событие во все соединения пользователя. Битые соединения закрываются и удаляются.
func (h *Handler) sendToUser(
	ctx context.Context,
	userID uint64,
	event domains.WSEvent,
) {
	val, ok := h.connections.Load(userID)
	if !ok {
		return
	}

	uc, ok := val.(*userConnections)
	if !ok {
		h.connections.Delete(userID)

		return
	}

	uc.mu.Lock()
	defer uc.mu.Unlock()

	var broken []*websocket.Conn

	for _, conn := range uc.connections {
		if err := conn.WriteJSON(event); err != nil {
			logging.LogErrorContext(
				ctx,
				h.logger,
				"Failed to write event to connection",
				err,
				"UserID", userID,
				"EventType", event.Type,
			)

			broken = append(broken, conn)
		}
	}

	for _, conn := range broken {
		if err := conn.Close(); err != nil {
			logging.LogErrorContext(
				ctx,
				h.logger,
				"Failed to close broken connection",
				err,
			)
		}

		uc.connections = slices.DeleteFunc(uc.connections, func(c *websocket.Conn) bool {
			return c == conn
		})
	}

	if len(uc.connections) == 0 {
		h.connections.Delete(userID)
	}
}

// hasConnections проверяет, есть ли у пользователя активные соединения.
func (h *Handler) hasConnections(userID uint64) bool {
	_, ok := h.connections.Load(userID)

	return ok
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

		messageToSend.IsRead = false // Сообщение не прочитано для всех получателей, так как является новым

		event := domains.WSEvent{
			Type:    domains.WSEventNewMessage,
			Payload: messageToSend,
		}

		for _, member := range chatMembers {
			if !h.hasConnections(member.ID) {
				// Отправляем уведомления только офлайн-пользователям
				if member.ID != user.ID {
					h.publishNewMessageNotifications(
						ctx,
						member.ID,
						savedMessage.ID,
					)
				}

				continue
			}

			h.sendToUser(ctx, member.ID, event)
		}
	}
}

func (h *Handler) publishNewMessageNotifications(
	ctx context.Context,
	userID, messageID uint64,
) {
	payload, err := json.Marshal(domains.NewMessagePayload{
		MessageID: messageID,
	})
	if err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to marshal notification payload", err)

		return
	}

	// Web push notification
	webPushDTO := domains.WebPushNotificationDTO{
		Type:    domains.WebPushTypeNewMessage,
		UserID:  userID,
		Payload: payload,
	}

	content, err := json.Marshal(webPushDTO)
	if err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to marshal web-push DTO", err)

		return
	}

	if err = h.natsPublisher.Publish(
		h.natsConfig.Subjects.WebPushNotification,
		content,
	); err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to publish web-push notification", err)
	}

	// Email notification
	emailDTO := domains.EmailNotificationDTO{
		Type:    domains.EmailTypeNewMessage,
		UserID:  userID,
		Payload: payload,
	}

	content, err = json.Marshal(emailDTO)
	if err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to marshal email DTO", err)

		return
	}

	if err = h.natsPublisher.Publish(
		h.natsConfig.Subjects.EmailNotification,
		content,
	); err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to publish email notification", err)
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
