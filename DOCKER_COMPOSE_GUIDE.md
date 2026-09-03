# Docker Compose Test Environment

## Services Available

| Service | Port | Credentials | Status |
|---------|------|-------------|--------|
| **Redis** | 6379 | No auth | Ready |
| **MongoDB** | 27017 | root:testpassword | Ready |
| **PostgreSQL** | 5432 | testuser:testpassword | Ready |

## Quick Commands

### Start All Services
```bash
docker-compose -f docker-compose.test.yml up -d
```

### Check Service Status
```bash
docker-compose -f docker-compose.test.yml ps
```

### View Logs
```bash
# All services
docker-compose -f docker-compose.test.yml logs -f

# Specific service
docker-compose -f docker-compose.test.yml logs -f redis
docker-compose -f docker-compose.test.yml logs -f mongodb
docker-compose -f docker-compose.test.yml logs -f postgres
```

### Stop All Services
```bash
docker-compose -f docker-compose.test.yml down
```

### Restart a Service
```bash
docker-compose -f docker-compose.test.yml restart redis
```

### Remove Everything (Including Data)
```bash
docker-compose -f docker-compose.test.yml down -v
```

## Testing Workflow

### Step 1: Start Services
```bash
docker-compose -f docker-compose.test.yml up -d
```

### Step 2: Wait for Health Checks
```bash
# Wait until all services show "healthy"
docker-compose -f docker-compose.test.yml ps
```

### Step 3: Run Tests
```bash
# Unit tests
go test -v -short ./...

# Integration tests (requires services running)
go test -v ./...
```

### Step 4: Stop Services
```bash
docker-compose -f docker-compose.test.yml down
```

## Connection Strings for Tests

### Redis
```go
redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})
```

### MongoDB
```go
mongo.Connect(ctx, options.Client().
    ApplyURI("mongodb://root:testpassword@localhost:27017"))
```

### PostgreSQL
```go
sql.Open("postgres", 
    "host=localhost port=5432 user=testuser password=testpassword dbname=ride-sharing sslmode=disable")
```

## Health Checks

All services include health checks that verify the service is ready:

- **Redis**: Checks if it responds to PING
- **MongoDB**: Checks if it responds to ping command
- **PostgreSQL**: Checks if pg_isready succeeds

The health checks run every 5 seconds with a 3-second timeout, allowing up to 5 retries.

## Volumes

Data is persisted in named volumes:
- `redis-data` - Redis persistence
- `mongodb-data` - MongoDB data
- `mongodb-config` - MongoDB configuration
- `postgres-data` - PostgreSQL data

Volumes are created automatically and cleaned up when using `docker-compose down`.

## Network

All services communicate on the `test-network` bridge network, allowing services to reach each other by container name (e.g., `redis:6379`, `mongodb:27017`).

## Integration with Tests

The test files use testcontainers-go to spin up their own containers, but you can also use these services for local testing:

```go
// Option 1: Use testcontainers (automatic cleanup)
container, _ := testcontainers.GenericContainer(ctx, ...)

// Option 2: Use docker-compose services (manual management)
redis.NewClient(&redis.Options{Addr: "localhost:6379"})
```

## Troubleshooting

### Services Won't Start
```bash
# Check Docker daemon is running
docker ps

# Check for port conflicts
lsof -i :6379
lsof -i :27017
lsof -i :5432
```

### Health Checks Failing
```bash
# View detailed logs
docker-compose -f docker-compose.test.yml logs redis

# Check service is running
docker ps | grep test-
```

### Can't Connect to Services
```bash
# Verify services are healthy
docker-compose -f docker-compose.test.yml ps

# Test connectivity
docker exec test-redis redis-cli ping
docker exec test-mongodb mongosh --eval "db.adminCommand('ping')"
docker exec test-postgres pg_isready
```

## Performance Tips

1. Pre-pull images to avoid download delays:
   ```bash
   docker pull redis:7-alpine
   docker pull mongo:6
   docker pull postgres:15-alpine
   ```

2. Keep services running during development:
   ```bash
   docker-compose -f docker-compose.test.yml up -d
   # Keep running, run tests multiple times
   go test -v ./...
   go test -v ./...
   ```

3. Use `-d` (detach) flag to keep services in background

## CI/CD Integration

For GitHub Actions, services can be configured in the workflow file instead:

```yaml
services:
  redis:
    image: redis:7-alpine
    options: >-
      --health-cmd "redis-cli ping"
      --health-interval 10s
      --health-timeout 5s
      --health-retries 5
    ports:
      - 6379:6379
```

See `TESTING_DEPENDENCIES.md` for full CI/CD setup examples.
