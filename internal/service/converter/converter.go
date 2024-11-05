package converter

import (
	"backend_task/internal/config"
	"backend_task/internal/domain/models"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"

	"github.com/gorilla/websocket"
)

type Converter struct {
	config     *config.Config
	bufferSize int
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	mutex      sync.Mutex
}

func NewConverter(cfg *config.Config) *Converter {
	return &Converter{
		config:     cfg,
		bufferSize: cfg.Audio.BufferSize,
	}
}

func (ac *Converter) StartFFmpeg(wavHeader models.WAVHeader) error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	// Initialize ffmpeg command with WAV header parameters
	ac.cmd = exec.Command(
		"ffmpeg",
		"-f", "wav",
		"-i", "pipe:0", // Input from stdin
		"-f", "flac",
		"-compression_level", "8",
		"pipe:1", // Output to stdout
	)

	// Set up pipes
	var err error
	ac.stdin, err = ac.cmd.StdinPipe()
	if err != nil {
		return err
	}

	ac.stdout, err = ac.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	// Optionally capture stderr for debugging
	stderr, err := ac.cmd.StderrPipe()
	if err != nil {
		return err
	}

	// Start the ffmpeg process
	if err := ac.cmd.Start(); err != nil {
		return err
	}

	// Log ffmpeg stderr
	go func() {
		buf := new(bytes.Buffer)
		io.Copy(buf, stderr)
		if buf.Len() > 0 {
			log.Printf("ffmpeg stderr: %s", buf.String())
		}
	}()

	return nil
}

func (ac *Converter) HandleConnection(conn *websocket.Conn) error {
	buffer := bytes.NewBuffer(nil)
	isHeaderRead := false
	var wavHeader models.WAVHeader
	totalBytesProcessed := 0

	log.Println("Starting WebSocket connection handling")

	// Start a goroutine to read FLAC data from ffmpeg and send to WebSocket
	flacSender := make(chan []byte)
	done := make(chan struct{})

	go func() {
		defer close(flacSender)
		flacBuffer := make([]byte, 4096)
		for {
			n, err := ac.stdout.Read(flacBuffer)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from ffmpeg stdout: %v", err)
				}
				break
			}
			flacData := make([]byte, n)
			copy(flacData, flacBuffer[:n])
			flacSender <- flacData
		}
		close(done)
	}()

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			return err
		}

		log.Printf("Received message - Type: %d, Length: %d bytes", messageType, len(data))

		if messageType != websocket.BinaryMessage {
			log.Println("Received non-binary message, skipping")
			continue
		}

		buffer.Write(data)
		log.Printf("Current buffer size: %d bytes", buffer.Len())

		// WAV Header parsing
		if !isHeaderRead && buffer.Len() >= binary.Size(wavHeader) {
			err := binary.Read(bytes.NewReader(buffer.Bytes()), binary.LittleEndian, &wavHeader)
			if err != nil {
				log.Printf("Error reading WAV header: %v", err)
				return err
			}

			log.Printf("WAV Header Details:")
			log.Printf("  ChunkID: %s", string(wavHeader.ChunkID[:]))
			log.Printf("  Format: %s", string(wavHeader.Format[:]))
			log.Printf("  NumChannels: %d", wavHeader.NumChannels)
			log.Printf("  SampleRate: %d", wavHeader.SampleRate)
			log.Printf("  BitsPerSample: %d", wavHeader.BitsPerSample)

			if string(wavHeader.ChunkID[:]) != "RIFF" || string(wavHeader.Format[:]) != "WAVE" {
				log.Printf("Invalid WAV format")
				return fmt.Errorf("invalid WAV format")
			}

			isHeaderRead = true
			buffer.Next(binary.Size(wavHeader))

			// Start ffmpeg after reading header
			if err := ac.StartFFmpeg(wavHeader); err != nil {
				log.Printf("Error starting ffmpeg: %v", err)
				return err
			}
		}

		// Send WAV data to ffmpeg's stdin
		if isHeaderRead {
			_, err := ac.stdin.Write(data)
			if err != nil {
				log.Printf("Error writing to ffmpeg stdin: %v", err)
				return err
			}
			totalBytesProcessed += len(data)
			log.Printf("Total bytes processed: %d", totalBytesProcessed)
		}

		// Send any available FLAC data to WebSocket
		select {
		case flacData, ok := <-flacSender:
			if !ok {
				// FLAC sender closed
				return nil
			}
			if len(flacData) > 0 {
				err := conn.WriteMessage(websocket.BinaryMessage, flacData)
				if err != nil {
					log.Printf("Error writing FLAC data to WebSocket: %v", err)
					return err
				}
				log.Printf("Sent FLAC chunk of %d bytes to WebSocket", len(flacData))
			}
		default:
			// No FLAC data available
		}

		if totalBytesProcessed > 1024*1024*50 { // 50MB
			log.Println("Reached maximum processing limit")
			break
		}
	}

	return nil
}
