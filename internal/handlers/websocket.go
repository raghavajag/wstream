package handlers

import (
	"backend_task/internal/config"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WebSocket upgrade failed"})
		return
	}
	defer conn.Close()

	// Create FFmpeg command for WAV to FLAC conversion
	ffmpegCmd := exec.Command(h.config.Audio.FFmpegPath,
		"-f", "wav", // Input format
		"-i", "pipe:0", // Read from stdin
		"-f", "flac", // Output format
		"-c:a", "flac", // FLAC codec
		"-compression_level", "5", // Compression level
		"pipe:1", // Write to stdout
	)

	// Create pipes for communication
	stdinPipe, err := ffmpegCmd.StdinPipe()
	if err != nil {
		log.Printf("Error creating stdin pipe: %v", err)
		return
	}

	stdoutPipe, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		log.Printf("Error creating stdout pipe: %v", err)
		return
	}

	if err := ffmpegCmd.Start(); err != nil {
		log.Printf("Failed to start FFmpeg: %v", err)
		return
	}

	// Goroutine to read FLAC output and send via WebSocket
	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		defer ffmpegCmd.Process.Kill()

		buffer := make([]byte, h.config.Audio.BufferSize)
		for {
			n, err := stdoutPipe.Read(buffer)
			if err != nil {
				log.Printf("Error reading FFmpeg output: %v", err)
				return
			}

			if err := conn.WriteMessage(websocket.BinaryMessage, buffer[:n]); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}()

	// Process incoming WAV chunks
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		// Write WAV chunk to FFmpeg stdin
		if _, err := stdinPipe.Write(data); err != nil {
			log.Printf("Error writing to FFmpeg stdin: %v", err)
			break
		}
	}

	// Wait for output processing to complete
	<-doneChan
}
