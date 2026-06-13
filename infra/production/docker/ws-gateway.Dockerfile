FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/ws-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o ws-gateway ./main.go

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/ws-gateway/ws-gateway .
CMD ["./ws-gateway"]
