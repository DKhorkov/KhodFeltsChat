package interfaces

import (
	"net/http"

	"github.com/gorilla/websocket"
)

//go:generate mockgen -source=upgrader.go -destination=../../mocks/upgrader/upgrader.go -package=mockupgrader -exclude_interfaces=
type Upgrader interface {
	Upgrade(
		w http.ResponseWriter,
		r *http.Request,
		responseHeader http.Header,
	) (*websocket.Conn, error)
}
