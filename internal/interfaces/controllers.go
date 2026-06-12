package interfaces

import "context"

//go:generate mockgen -source=controllers.go -destination=../../mocks/controllers/controller.go -package=mockcontrollers -exclude_interfaces=WSBroadcaster
type Controller interface {
	Run()
	Stop()
}

//go:generate mockgen -source=controllers.go -destination=../../mocks/controllers/ws_broadcaster.go -package=mockcontrollers -exclude_interfaces=Controller
type WSBroadcaster interface {
	BroadcastMessageDeleted(ctx context.Context, chatID, messageID, userID uint64)
	SendMessageDeletedToUser(ctx context.Context, chatID, messageID, userID uint64)
	BroadcastMessageEdited(ctx context.Context, chatID, messageID, userID uint64)
}
