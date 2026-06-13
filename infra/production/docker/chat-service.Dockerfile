FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/chat-service
RUN CGO_ENABLED=0 GOOS=linux go build -o chat-service ./cmd/main.go

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/chat-service/chat-service .
CMD ["./chat-service"]
