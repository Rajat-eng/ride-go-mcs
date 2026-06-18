FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/dlq-worker
RUN CGO_ENABLED=0 GOOS=linux go build -o dlq-worker .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/dlq-worker/dlq-worker .
CMD ["./dlq-worker"]
