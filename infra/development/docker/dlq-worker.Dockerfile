FROM alpine:3.20
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    && update-ca-certificates

WORKDIR /app

# Add necessary files
COPY build/dlq-worker /app/build/
COPY shared /app/shared/

# Make binary executable
RUN chmod +x /app/build/dlq-worker

# Create non-root user
RUN adduser -D appuser
USER appuser

ENTRYPOINT ["/app/build/dlq-worker"]