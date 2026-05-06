# Ride-Sharing Microservices — Action Plan

Timeline: 2 months (8 weeks)

---

## Phase 1 — Foundation & Security (Weeks 1–2)

1. **Auth Service** — Go service with JWT access/refresh tokens, Google/GitHub SSO (OAuth2), bcrypt passwords, role-based (rider/driver/admin). Store users in PostgreSQL.
2. **API Gateway Auth Middleware** — Validate JWT on all routes, extract userID from token (replace current spoofable query param approach).
3. **PostgreSQL + Migrations** — Add Postgres for users, payments, ratings. Use golang-migrate for schema versioning. Keep MongoDB for trips.
4. **Secrets Management** — Move secrets.yaml to .gitignore. Use Kubernetes Sealed Secrets or AWS Secrets Manager.
5. **TLS for gRPC** — Encrypt inter-service traffic with mTLS.

---

## Phase 2 — Core Features (Weeks 3–4)

6. **Trip Completion Flow** — Driver marks trip complete → payment verification → update trip status → notify rider. Handle cancellation, timeout, disputes.
7. **OTP Verification** — Rider gets OTP, shares with driver to start trip. Publish via RabbitMQ, send via SMS (Twilio) or in-app.
8. **Real-time Driver Location** — Redis Streams for location updates, publish to rider via WebSocket. Redis GEO for proximity queries.
9. **Notification Aggregation** — Queue notifications for offline drivers, deliver on reconnect. Redis sorted sets with TTL.
10. **Payment Enhancements** — Handle checkout.session.expired, refund flow, payment history in Postgres.

---

## Phase 3 — Resilience & Scalability (Week 5)

### Circuit Breakers
11. **gRPC Circuit Breakers** — Use `go-grpc-middleware` or Sony's `gobreaker` on API Gateway → Trip/Driver gRPC calls. States: closed → open (after 5 consecutive failures) → half-open (probe). Return fallback responses when circuits open.
12. **RabbitMQ Consumer Circuit Breakers** — Wrap message processing with circuit breaker. On repeated failures, pause consumption and let messages accumulate (backpressure) instead of flooding DLQ.
13. **External Service Breakers** — OSRM routing calls, Stripe API, MongoDB connections. Each gets its own breaker with independent thresholds.

### Rate Limiting
14. **API Gateway Rate Limiting** — Token bucket algorithm per user (authenticated) or per IP (unauthenticated). Use Redis for distributed rate limit counters.
    - Trip creation: 5 req/min per user
    - Trip preview: 30 req/min per user
    - WebSocket connections: 3 per user
    - Stripe webhooks: IP whitelist (Stripe IPs only)
15. **gRPC Rate Limiting** — Server-side interceptor with per-service quotas. Prevents one service from overwhelming another.
16. **RabbitMQ QoS** — Set prefetch count per consumer to control throughput. Already partially done, tune values.

### Caching
17. **Redis Caching Layer** — Cache trip previews (route + fare) with TTL (5 min), driver availability zones, user sessions. Reduces gRPC/DB load.
18. **gRPC Response Caching** — Cache FindAvailableDrivers responses for same geohash (short TTL ~10s) during high demand.

### Autoscaling
19. **Horizontal Pod Autoscaler (HPA)** — Scale API Gateway and Trip Service on CPU/memory and custom metrics (request rate, queue depth).
20. **KEDA (Kubernetes Event-Driven Autoscaling)** — Scale DLQ Worker and consumers based on RabbitMQ queue length. Zero-to-N scaling.
21. **Cluster Autoscaler** — EKS node group auto-scaling based on pending pod pressure.

### Other Scalability Patterns
22. **Connection Pooling** — MongoDB, PostgreSQL, Redis, gRPC connection pools with proper limits and health checks.
23. **Graceful Shutdown** — Replace all log.Fatal() with signal handlers. Drain in-flight requests, close DB connections, stop consumers before exit.
24. **Bulkhead Pattern** — Isolate resources per service (separate Redis DBs, separate RabbitMQ vhosts) so failures don't cascade.
25. **Retry with Exponential Backoff** — Already have retry.go, extend to all external calls (DB, Stripe, OSRM) with jitter.
26. **Request Timeouts** — Set gRPC deadlines, HTTP client timeouts, and context cancellation propagation across the call chain.

---

## Phase 4 — Observability (Week 6)

27. **OpenTelemetry Collector** — Deploy as DaemonSet. Route: traces → Jaeger, metrics → Prometheus, logs → Loki.
28. **Structured Logging** — Replace log.Printf with slog across all services. Correlation IDs from trace context.
29. **Prometheus Metrics** — Request latency (p50/p95/p99), RabbitMQ queue depth, gRPC error rates, active WebSocket connections, circuit breaker state, rate limit rejections, cache hit/miss ratio.
30. **Grafana Dashboards** — Service health, trip funnel, payment success rate, circuit breaker status, rate limit analytics.
31. **Alerting** — Slack/PagerDuty for: queue backlog, error rate spike, circuit breaker open, service down, high latency.

---

## Phase 5 — Helm, Multi-Environment & CI/CD (Week 7)

### Helm Charts
32. **Helm Chart per Service** — Templatize all K8s manifests. Each service gets its own chart with configurable:
    - Replica count, resource limits, HPA settings
    - Environment-specific values (dev/staging/prod)
    - Image tag, pull policy
    - Service ports, health check paths
    - Circuit breaker thresholds, rate limit configs
33. **Umbrella Helm Chart** — Parent chart that deploys entire stack with dependencies (all services + infra). One command to spin up full environment.
34. **Helm Values per Environment**:
    - `values-dev.yaml` — 1 replica, debug logging, relaxed limits, local secrets
    - `values-staging.yaml` — 2 replicas, mirrors prod config, staging DB/MQ
    - `values-prod.yaml` — 3+ replicas, strict limits, production secrets refs
35. **Helm Hooks** — Pre-install: run DB migrations. Post-install: smoke tests. Pre-delete: drain connections.

### Multi-Environment Deployment
36. **Environment Strategy**:
    - **dev** — Tilt + Minikube (local), hot reload, debug logging, seeded fake data
    - **staging** — EKS namespace `staging`, mirrors prod infra, used for load testing and QA
    - **prod** — EKS namespace `prod`, HA replicas, strict resource limits, real Stripe keys
37. **Namespace Isolation** — Each env gets its own K8s namespace with dedicated secrets, configmaps, and network policies.
38. **Environment Promotion Flow**: dev → staging (auto on merge to main) → prod (manual approval or release tag).
39. **Separate Databases per Environment** — Staging gets its own MongoDB + Postgres instances (not shared with prod). Seed staging with fake data on deploy.
40. **Feature Flags (optional)** — Use environment variables or a simple config to toggle features per env (e.g., skip real Stripe in staging, use mock).

### CI/CD
41. **GitHub Actions Pipeline**:
    - PR: lint (golangci-lint) → test → build → security scan (trivy)
    - Merge to main: build images → push to ECR → `helm upgrade` to staging → run smoke tests
    - Release tag: promote staging → prod (manual approval gate)
42. **ArgoCD** — GitOps deployment. Watch Helm chart repo, auto-sync on values change. App-of-apps pattern for multi-service. Separate ArgoCD apps per environment.
43. **Terraform** — EKS cluster, RDS (Postgres), ElastiCache (Redis), Amazon MQ (RabbitMQ), VPC, IAM, ECR repos, Route53, ACM certs.

### Networking
44. **Ingress Controller** — AWS ALB Ingress or NGINX Ingress with:
    - TLS termination (ACM certs)
    - Path-based routing (/api/*, /ws/*, /webhook/*)
    - Rate limiting annotations
    - WAF integration
45. **Service Mesh (optional)** — Istio or Linkerd for mTLS, traffic management, canary deployments, and observability without code changes.
46. **Network Policies** — Restrict pod-to-pod traffic. Only API Gateway can reach gRPC services. Only services can reach RabbitMQ/Redis/DB.

---

## Phase 6 — Seed Data & Load Testing (Week 7–8)

### Fake Data / Seed Tool
47. **Seed CLI Tool** (`tools/seeder/`) — Go CLI that populates the system with realistic fake data:
    - **Drivers** (500+): name, phone, email, vehicle (make/model/plate), rating, status (online/offline)
    - **Riders** (1000+): name, phone, email, payment method, trip history
    - **Locations**: Real coordinates across a city grid (e.g., Delhi NCR, Bangalore, or NYC)
      - Driver locations: randomized within city bounds with geohash clustering (busy areas get more drivers)
      - Pickup/dropoff points: popular landmarks, metro stations, airports, malls
    - **Trips** (2000+): historical trips with realistic routes, fares, timestamps, statuses (completed, cancelled, in-progress)
    - **Payments**: Linked to trips with Stripe test payment IDs
48. **Data Generation Libraries** — Use `brianvoe/gofakeit` for names/phones/emails, custom geo functions for locations within bounding boxes.
49. **Seed Commands**:
    ```
    go run tools/seeder/main.go --env=dev --drivers=100 --riders=200 --trips=500
    go run tools/seeder/main.go --env=staging --drivers=500 --riders=1000 --trips=5000
    ```
50. **Location Datasets** — Predefined coordinate sets for major Indian cities:
    - Delhi NCR: 28.4–28.8°N, 76.8–77.4°E (Connaught Place, IGI Airport, Gurgaon, Noida)
    - Bangalore: 12.8–13.1°N, 77.5–77.7°E (MG Road, Electronic City, Airport)
    - Hotspots with higher driver density (airports, railway stations, IT parks)
51. **Driver Simulator** — Script that simulates N drivers sending location updates via WebSocket at configurable intervals (every 5s). Useful for testing real-time location + Redis Streams.
52. **Helm Hook for Seeding** — Post-install Helm hook runs seeder job in dev/staging environments. Skipped in prod.

### Load Testing
53. **k6 Load Tests** (`tests/load/`) — JavaScript-based load test scenarios:
    - **Trip Preview** — 500 concurrent users requesting route previews
    - **Trip Creation** — 200 concurrent trip requests, verify RabbitMQ fan-out
    - **WebSocket Connections** — 1000 simultaneous WS connections, verify message delivery latency
    - **Driver Matching** — Burst of 100 trip requests in same geohash zone, verify driver assignment fairness
    - **Payment Flow** — Simulated Stripe webhook callbacks under load
    - **Mixed Workload** — Realistic traffic pattern: 60% previews, 25% trips, 10% WS, 5% payments
54. **k6 Thresholds** — Pass/fail criteria per scenario:
    - p95 latency < 500ms for trip preview
    - p95 latency < 1s for trip creation
    - Error rate < 1%
    - WebSocket message delivery < 200ms
55. **Load Test in CI** — Run k6 against staging after each deploy. Fail pipeline if thresholds breached.
56. **Grafana k6 Dashboard** — Visualize load test results alongside service metrics. Correlate request rate with pod CPU, queue depth, circuit breaker trips.
57. **Soak Test** — 4-hour sustained load at 50% peak to detect memory leaks, connection exhaustion, and GC pauses.
58. **Spike Test** — Ramp from 0 → 1000 users in 30s, verify autoscaling kicks in and no requests are dropped.

---

## Phase 7 — Code Quality & Frontend (Week 8)

59. **Tests** — Unit tests for service logic, integration tests with testcontainers (gRPC + RabbitMQ + MongoDB). Target 70%+ on business logic.
60. **Frontend Pages** — Login/register, trip history, driver dashboard (earnings, requests), rider profile, admin panel.
61. **API Documentation** — OpenAPI spec from gateway routes, or grpc-gateway for REST + Swagger.
62. **Code Cleanup** — Consistent error handling, CORS whitelist, WebSocket message validation, graceful shutdown.

---

## Phase 8 — Advanced (Post-MVP)

63. **LLM Integration** — ETA prediction, smart driver matching, fare surge pricing.
64. **Analytics Pipeline** — RabbitMQ → Kafka → data warehouse. Dashboards for trip patterns, revenue, driver utilization.
65. **Rating System** — Post-trip ratings, driver score calculation, minimum thresholds.
66. **Canary Deployments** — Argo Rollouts with progressive traffic shifting and automated rollback on error rate.
67. **Chaos Engineering** — Litmus or Chaos Mesh to test circuit breakers, failover, and recovery in staging.

---

## Quick Wins (Do First)

- [ ] Add secrets.yaml to .gitignore
- [ ] Replace log.Fatal with graceful shutdown
- [ ] Add /healthz and /readyz endpoints to all services
- [ ] Restrict CORS to specific origins
- [ ] Add input validation on WebSocket messages
- [ ] Set gRPC deadlines on all client calls
- [ ] Add Helm .helmignore and chart boilerplate