# Build Stage
FROM golang:1.26.11-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build (Pure Go, no CGO required thanks to WASM SQLite)
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o supernova ./cmd/server && CGO_ENABLED=0 GOOS=linux go build -o supernova-backup ./cmd/db-backup

# Run Stage
FROM alpine:latest

# Install ffmpeg for real-time audio transcoding
RUN apk add --no-cache ffmpeg

WORKDIR /root/

# Copy the statically compiled binary
COPY --from=builder /app/supernova .
COPY --from=builder /app/supernova-backup .

EXPOSE 8080

CMD ["./supernova"]
