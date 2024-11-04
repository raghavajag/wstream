package converter

import (
	"log"

	"github.com/gorilla/websocket"
)

type Converter struct{}

func New() *Converter {
	return &Converter{}
}

func (c *Converter) HandleConnection(conn *websocket.Conn) error {
	defer conn.Close()

	// Send a welcome message
	if err := conn.WriteMessage(websocket.TextMessage, []byte("Connected to WebSocket server")); err != nil {
		log.Printf("Error sending welcome message: %v", err)
		return err
	}

	for {
		// Read message from the client
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
			}
			break
		}

		// Log the received message
		log.Printf("Received message: %s", message)

		// Echo the message back to the client
		if err := conn.WriteMessage(messageType, message); err != nil {
			log.Printf("Error writing message: %v", err)
			return err
		}
	}

	return nil
}
