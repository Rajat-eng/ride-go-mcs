FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/driver-service
RUN CGO_ENABLED=0 GOOS=linux go build -o driver-service .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/driver-service/driver-service .
CMD ["./driver-service"]
