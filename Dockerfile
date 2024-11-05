# Stage 1: Build the Go app
FROM golang:1.23.1-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ffmpeg

# Set the working directory
WORKDIR /app

# Copy the Go modules files
COPY go.mod go.sum ./

# Download the Go module dependencies
RUN go mod tidy

# Copy the source code into the container
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -o wav-to-flac-converter ./cmd/main.go

# Stage 2: Build a minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ffmpeg

# Copy the binary from the builder stage
COPY --from=builder /app/wav-to-flac-converter /usr/local/bin/

# Expose port
EXPOSE 8080

# Set environment variables with defaults
ENV PORT=8080
ENV FFMPEG_PATH=/usr/bin/ffmpeg
ENV BUFFER_SIZE=1048576

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s \
  CMD wget --spider http://localhost:8080/health || exit 1

# Command to run the executable
CMD ["/usr/local/bin/wav-to-flac-converter"]