package converter

import (
	"backend_task/internal/config"
	"backend_task/internal/domain/models"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
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
	totalBytesProcessed := 0

	log.Println("Starting WebSocket connection handling")

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
		}

		// Adjust buffer size dynamically
		bytesPerSample := int(wavHeader.BitsPerSample / 8)
		channelCount := int(wavHeader.NumChannels)
		dynamicBufferSize := ac.bufferSize

		// Ensure buffer size is a multiple of sample block size
		sampleBlockSize := bytesPerSample * channelCount
		if dynamicBufferSize%sampleBlockSize != 0 {
			dynamicBufferSize = (dynamicBufferSize / sampleBlockSize) * sampleBlockSize
		}

		log.Printf("Dynamic buffer size: %d", dynamicBufferSize)
		log.Printf("Sample block size: %d", sampleBlockSize)

		// Audio data processing
		if isHeaderRead && buffer.Len() >= dynamicBufferSize {
			log.Printf("Preparing to process audio data")
			log.Printf("Buffer size: %d, Required buffer size: %d", buffer.Len(), dynamicBufferSize)

			audioData := make([]byte, dynamicBufferSize)
			bytesRead, err := buffer.Read(audioData)
			if err != nil {
				log.Printf("Error reading audio data: %v", err)
				return err
			}

			totalBytesProcessed += bytesRead
			log.Printf("Read %d bytes of audio data", bytesRead)
			log.Printf("Total bytes processed: %d", totalBytesProcessed)

			flacData, err := ac.convertToFLAC(audioData, wavHeader)
			if err != nil {
				log.Printf("Error converting to FLAC: %v", err)
				continue
			}

			log.Printf("Converted to FLAC. FLAC data length: %d bytes", len(flacData))

			err = conn.WriteMessage(websocket.BinaryMessage, flacData)
			if err != nil {
				log.Printf("Error writing FLAC data: %v", err)
				return err
			}

			log.Println("Successfully wrote FLAC data to WebSocket")
		} else {
			log.Printf("Not enough data to process. Header read: %v, Buffer size: %d, Required: %d",
				isHeaderRead, buffer.Len(), dynamicBufferSize)
		}

		// handle remaining data or end of stream
		if totalBytesProcessed > 1024*1024*5 { // Limit to 1MB for testing
			log.Println("Reached maximum processing limit")
			break
		}
	}

	return nil
}
func (ac *Converter) convertToFLAC(wavData []byte, header models.WAVHeader) ([]byte, error) {
	log.Printf("Converting WAV to FLAC")
	log.Printf("WAV Data length: %d bytes", len(wavData))
	log.Printf("Channels: %d, Bits per sample: %d", header.NumChannels, header.BitsPerSample)

	output := bytes.NewBuffer(nil)

	// Detailed logging for sample calculations
	bytesPerSample := int(header.BitsPerSample / 8)
	totalSamples := len(wavData) / bytesPerSample
	samplesPerChannel := totalSamples / int(header.NumChannels)

	log.Printf("Bytes per sample: %d", bytesPerSample)
	log.Printf("Total samples: %d", totalSamples)
	log.Printf("Samples per channel: %d", samplesPerChannel)

	streamInfo := meta.StreamInfo{
		BlockSizeMin:  uint16(samplesPerChannel),
		BlockSizeMax:  uint16(samplesPerChannel),
		FrameSizeMin:  0,
		FrameSizeMax:  0,
		SampleRate:    header.SampleRate,
		NChannels:     uint8(header.NumChannels),
		BitsPerSample: uint8(header.BitsPerSample),
	}

	enc, err := flac.NewEncoder(output, &streamInfo)
	if err != nil {
		log.Printf("Error creating FLAC encoder: %v", err)
		return nil, fmt.Errorf("error creating FLAC encoder: %v", err)
	}
	defer enc.Close()

	// Samples conversion with extensive logging
	samples := make([]int32, totalSamples)
	if header.BitsPerSample == 16 {
		for i := 0; i < totalSamples; i++ {
			start := i * 2
			end := start + 2
			if end > len(wavData) {
				log.Printf("Warning: Incomplete sample at index %d", i)
				break
			}
			samples[i] = int32(int16(binary.LittleEndian.Uint16(wavData[start:end])))
		}
	} else {
		log.Printf("Unsupported bits per sample: %d", header.BitsPerSample)
		return nil, fmt.Errorf("unsupported bits per sample: %d", header.BitsPerSample)
	}

	// Frame and subframe creation with logging
	frameHeader := frame.Header{
		BlockSize:     uint16(samplesPerChannel),
		SampleRate:    header.SampleRate,
		Channels:      frame.Channels(header.NumChannels),
		BitsPerSample: uint8(header.BitsPerSample),
	}

	subframes := make([]*frame.Subframe, header.NumChannels)
	for ch := 0; ch < int(header.NumChannels); ch++ {
		channelSamples := make([]int32, samplesPerChannel)
		for i := 0; i < samplesPerChannel; i++ {
			channelSamples[i] = samples[i*int(header.NumChannels)+ch]
		}

		log.Printf("Channel %d samples: %d", ch, len(channelSamples))

		subframes[ch] = &frame.Subframe{
			SubHeader: frame.SubHeader{
				Pred:   frame.PredVerbatim,
				Wasted: 0,
				Order:  0,
			},
			Samples: channelSamples,
		}
	}

	fr := frame.Frame{
		Header:    frameHeader,
		Subframes: subframes,
	}

	if err := enc.WriteFrame(&fr); err != nil {
		log.Printf("Error writing FLAC frame: %v", err)
		return nil, fmt.Errorf("error writing FLAC frame: %v", err)
	}

	log.Printf("Successfully converted to FLAC. Output size: %d bytes", output.Len())
	return output.Bytes(), nil
}
