# WAV to FLAC Streaming Converter Service

A real-time audio streaming service that converts WAV files to FLAC format using WebSocket communication. Built with Go, featuring an embedded web UI and containerized deployment support.
# Access the web interface at [link](http://4.240.97.177/static/)
## Features

- Real-time WAV to FLAC streaming conversion
- FFmpeg integration for reliable audio conversion
- WebSocket-based communication
- Embedded web interface
- Docker and Kubernetes support
- Efficient memory usage through streaming

## Local Development Setup using Docker, Kubernetes
```bash
git clone https://github.com/raghavajag/backend-task
cd wav-to-flac-converter
docker build -t wav-to-flac-converter:latest .
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```
## without docker/k8s

1. Clone the repository:
```bash
git clone https://github.com/raghavajag/backend-task
cd wav-to-flac-converter
go mod tidy
```
2. Install FFmpeg
```bash
Ubuntu/Debian
sudo apt-get update && sudo apt-get install -y ffmpeg

macOS
brew install ffmpeg

Windows
choco install ffmpeg
```

3. Run the service
```bash
go run cmd/main.go
```
