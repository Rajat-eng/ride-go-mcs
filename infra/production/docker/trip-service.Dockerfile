FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/trip-service
RUN CGO_ENABLED=0 GOOS=linux go build -o trip-service ./cmd/main.go

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/trip-service/trip-service .
CMD ["./trip-service"]