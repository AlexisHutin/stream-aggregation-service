# --- Build stage ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /build/stream-aggregation-service .

# --- Final stage ---
FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /build/stream-aggregation-service /stream-aggregation-service

ENV CONFIG_FILE=config.json

WORKDIR /

EXPOSE 8080

ENTRYPOINT ["/stream-aggregation-service"]