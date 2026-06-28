# Stage 1: Build the frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Serve frontend using a lightweight web server
FROM nginx:alpine AS frontend
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=frontend-builder /app/frontend/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]

# Stage 3: Build the Go backend
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ ./
ENV CGO_ENABLED=0
RUN go build -o supernova-server cmd/server/main.go

# Stage 4: Run the Go backend
FROM alpine:latest AS backend
# Install ffmpeg so our Go app can exec it
RUN apk add --no-cache ffmpeg
WORKDIR /app
COPY --from=backend-builder /app/backend/supernova-server .
RUN mkdir -p /app/data
EXPOSE 8080
CMD ["./supernova-server"]
