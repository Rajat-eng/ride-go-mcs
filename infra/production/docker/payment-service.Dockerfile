FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/payment-service
RUN CGO_ENABLED=0 GOOS=linux go build -o payment-service ./cmd/main.go

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/payment-service/payment-service .
CMD ["./payment-service"]
