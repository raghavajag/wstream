/*
package handlers

import (

	"backend_task/internal/config"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

)

	type WebSocketHandler struct {
		upgrader websocket.Upgrader
		config   *config.Config
	}

	func NewWebSocketHandler(cfg *config.Config) *WebSocketHandler {
		// Set maximum CPU usage
		runtime.GOMAXPROCS(runtime.NumCPU())

		return &WebSocketHandler{
			upgrader: websocket.Upgrader{
				ReadBufferSize:  1024 * 1024 * 2, // 2MB buffer
				WriteBufferSize: 1024 * 1024 * 2, // 2MB buffer
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

		// Increased channel buffer sizes
		inputChan := make(chan []byte, 2048)
		errorChan := make(chan error, 10)
		doneChan := make(chan struct{})

		// Optimized FFmpeg command for performance
		ffmpegCmd := exec.Command(h.config.Audio.FFmpegPath,
			"-f", "wav",
			"-i", "pipe:0",
			"-c:a", "aac",
			"-b:a", "256k", // Increased bitrate
			"-profile:a", "aac_low", // Use low-complexity profile
			"-movflags", "faststart+frag_keyframe+empty_moov+default_base_moof",
			"-frag_duration", "500", // Reduced fragment duration
			"-threads", fmt.Sprintf("%d", runtime.NumCPU()), // Use all CPU cores
			"-f", "mp4",
			"pipe:1",
		)

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

		if err := ffmpegCmd.Start(); err != nil {
			log.Printf("Failed to start FFmpeg: %v", err)
			return
		}

		// Optimized output handling
		go func() {
			defer close(doneChan)
			buffer := make([]byte, 1024*1024) // 1MB buffer

			for {
				n, err := stdout.Read(buffer)
				if err != nil {
					if err != io.EOF {
						select {
						case errorChan <- fmt.Errorf("FFmpeg output error: %v", err):
						default:
						}
					}
					return
				}

				// Non-blocking write to WebSocket
				if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
					continue
				}

				if err := conn.WriteMessage(websocket.BinaryMessage, buffer[:n]); err != nil {
					select {
					case errorChan <- fmt.Errorf("WebSocket write error: %v", err):
					default:
					}
					return
				}
			}
		}()

		// Optimized input handling
		go func() {
			defer stdin.Close()

			buffer := make([]byte, 0, 1024*1024) // Pre-allocate buffer

			for data := range inputChan {
				buffer = append(buffer[:0], data...)
				if _, err := stdin.Write(buffer); err != nil {
					select {
					case errorChan <- fmt.Errorf("FFmpeg input error: %v", err):
					default:
					}
					return
				}
			}
		}()

		// Set read deadline for better resource management
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		// Main message handling loop with optimized select
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

				// Non-blocking send with timeout
				select {
				case inputChan <- data:
				case <-time.After(100 * time.Millisecond):
					log.Println("Input buffer full, dropping frame")
				}
			}
		}
	}
*/
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
	inputChan := make(chan []byte, 2048) // Increased buffer size
	errorChan := make(chan error, 1)
	doneChan := make(chan struct{})

	// Create FFmpeg command with optimized parameters
	ffmpegCmd := exec.Command(h.config.Audio.FFmpegPath,
		"-f", "wav",
		"-i", "pipe:0",
		"-c:a", "aac",
		"-b:a", "192k",
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"-frag_duration", "500", // Reduced fragment duration for lower latency
		"-muxdelay", "0.001", // Minimal muxing delay
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

	// Capture FFmpeg stderr for debugging
	stderrPipe, err := ffmpegCmd.StderrPipe()
	if err != nil {
		log.Printf("Failed to create stderr pipe: %v", err)
		return
	}

	if err := ffmpegCmd.Start(); err != nil {
		log.Printf("Failed to start FFmpeg: %v", err)
		return
	}

	// Goroutine to capture FFmpeg logs
	go func() {
		scanner := io.TeeReader(stderrPipe, log.Writer())
		io.Copy(io.Discard, scanner) // Discarding FFmpeg stderr after logging
	}()

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
