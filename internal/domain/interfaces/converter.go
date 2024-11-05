package interfaces

import (
	"backend_task/internal/domain/models"

	"github.com/gorilla/websocket"
)

type AudioConverter interface {
	HandleConnection(conn *websocket.Conn) error
	ProcessAudioStream(stream *models.AudioStream) ([]byte, error)
}
