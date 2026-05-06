FROM alpine:3.20
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    && update-ca-certificates

WORKDIR /app

# Add necessary files
COPY build/login-service /app/build/
COPY services/login-service/migrations /app/migrations/

# Make binary executable
RUN chmod +x /app/build/login-service

# Create non-root user
RUN adduser -D appuser
USER appuser

ENTRYPOINT ["/app/build/login-service"]
