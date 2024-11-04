package interfaces

import (
	"github.com/gorilla/websocket"
)

type AudioConverter interface {
	HandleConnection(conn *websocket.Conn) error
}
