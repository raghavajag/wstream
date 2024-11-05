package converter

import (
	"backend_task/internal/config"
	"backend_task/internal/domain/models"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

type Converter struct {
	config     *config.Config
	bufferSize int
}

func NewConverter(cfg *config.Config) *Converter {
	return &Converter{
		config:     cfg,
		bufferSize: cfg.Audio.BufferSize,
	}
}

func (ac *Converter) HandleConnection(conn *websocket.Conn) error {
	buffer := bytes.NewBuffer(nil)
	isHeaderRead := false
	var wavHeader models.WAVHeader

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			return err
		}

		if messageType != websocket.BinaryMessage {
			log.Default().Println("Not a binary message")
			continue
		}
		log.Default().Println("Received binary message")
		buffer.Write(data)

		if !isHeaderRead && buffer.Len() >= binary.Size(wavHeader) {
			err := binary.Read(bytes.NewReader(buffer.Bytes()), binary.LittleEndian, &wavHeader)
			fmt.Printf("WAV Header: %v\n", wavHeader)

			if err != nil {
				log.Printf("Error reading WAV header: %v", err)
				return err
			}
			isHeaderRead = true

			if string(wavHeader.ChunkID[:]) != "RIFF" || string(wavHeader.Format[:]) != "WAVE" {
				log.Printf("Invalid WAV format")
				return err
			}

			buffer.Next(binary.Size(wavHeader))
		}

	}
}
