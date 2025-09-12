FROM alpine:3.20
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    && update-ca-certificates

WORKDIR /app

# Add necessary files
COPY build/trip-service /app/build/
COPY shared /app/shared/

# Make binary executable
RUN chmod +x /app/build/trip-service

# Create non-root user
RUN adduser -D appuser
USER appuser

ENTRYPOINT ["/app/build/trip-service"]