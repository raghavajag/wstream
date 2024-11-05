package handlers

import (
	"backend_task/internal/config"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	upgrader websocket.Upgrader
	config   *config.Config
}

func NewWebSocketHandler(cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  cfg.Audio.BufferSize,
			WriteBufferSize: cfg.Audio.BufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		config: cfg,
	}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Create buffered channels for data flow control
	inputChan := make(chan []byte, 1024)
	errorChan := make(chan error, 1)
	doneChan := make(chan struct{})

	// Create FFmpeg command with optimized parameters
	ffmpegCmd := exec.Command(h.config.Audio.FFmpegPath,
		"-f", "wav",
		"-i", "pipe:0",
		"-c:a", "aac", // Use AAC codec for better compatibility
		"-b:a", "192k",
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"-frag_duration", "1000", // Fragment duration in milliseconds
		"pipe:1",
	)

	// Set up pipes with error handling
	stdin, err := ffmpegCmd.StdinPipe()
	if err != nil {
		log.Printf("Failed to create stdin pipe: %v", err)
		return
	}

	stdout, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to create stdout pipe: %v", err)
		return
	}

	// Start FFmpeg with error handling
	if err := ffmpegCmd.Start(); err != nil {
		log.Printf("Failed to start FFmpeg: %v", err)
		return
	}

	// Goroutine for handling FFmpeg output
	go func() {
		defer close(doneChan)
		buffer := make([]byte, h.config.Audio.BufferSize)

		for {
			n, err := stdout.Read(buffer)
			if err != nil {
				if err != io.EOF {
					errorChan <- fmt.Errorf("FFmpeg output error: %v", err)
				}
				return
			}

			// Send converted data through WebSocket with error handling
			if err := conn.WriteMessage(websocket.BinaryMessage, buffer[:n]); err != nil {
				errorChan <- fmt.Errorf("WebSocket write error: %v", err)
				return
			}
		}
	}()

	// Goroutine for handling input data
	go func() {
		defer stdin.Close()

		for data := range inputChan {
			_, err := stdin.Write(data)
			if err != nil {
				errorChan <- fmt.Errorf("FFmpeg input error: %v", err)
				return
			}
		}
	}()

	// Main loop for handling WebSocket messages
	for {
		select {
		case err := <-errorChan:
			log.Printf("Error in processing: %v", err)
			return
		case <-doneChan:
			return
		default:
			messageType, data, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			if messageType != websocket.BinaryMessage {
				continue
			}

			// Non-blocking send to input channel
			select {
			case inputChan <- data:
			default:
				log.Println("Input buffer full, dropping frame")
			}
		}
	}
}
