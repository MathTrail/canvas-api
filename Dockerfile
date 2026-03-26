FROM golang:1.26.0-alpine AS builder
RUN apk add --no-cache git
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/canvas-api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 appuser
COPY --from=builder /app /app
USER 10001
EXPOSE 8080
ENTRYPOINT ["/app"]
