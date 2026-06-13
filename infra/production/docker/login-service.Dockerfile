FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/login-service
RUN CGO_ENABLED=0 GOOS=linux go build -o login-service ./cmd/main.go

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/services/login-service/login-service .
COPY services/login-service/migrations /app/migrations
CMD ["./login-service"]
