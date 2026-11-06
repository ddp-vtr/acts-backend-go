# Multi-stage lightweight Dockerfile for Go backend
# Stage 1: Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY go.mod go.sum* ./
RUN go mod download || true

COPY . .

# Build statically linked standalone Linux binary stripped of debug symbols
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /fb_server ./cmd/fb_server

# Stage 2: Runtime stage using minimal Alpine Linux with non-root security
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -u 10001 -s /sbin/nologin appuser

WORKDIR /app

COPY --from=builder /fb_server /app/fb_server

USER appuser

# Default port fallback (Cloud injects $PORT)
ENV PORT=8081
EXPOSE 8081 10000

ENTRYPOINT ["/app/fb_server"]
