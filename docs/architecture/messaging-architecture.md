# Messaging & Real-Time Communication Architecture

## Table of Contents

1. [System Overview](#system-overview)
2. [Service Catalogue](#service-catalogue)
   - [api-gateway](#api-gateway)
   - [ws-gateway](#ws-gateway)
   - [trip-service](#trip-service)
   - [driver-service](#driver-service)
   - [payment-service](#payment-service)
   - [chat-service](#chat-service)
   - [dlq-worker](#dlq-worker)
3. [Transport Layers](#transport-layers)
4. [WebSocket Layer](#websocket-layer)
5. [RabbitMQ Layer](#rabbitmq-layer)
6. [Redis Layer](#redis-layer)
  - [Redis Streams (Definition & Usage)](#redis-streams-definition--usage)
7. [RedisConnectionManager (RCM)](#redisconnectionmanager-rcm)
8. [Complete Data Flows](#complete-data-flows)
   - [Trip Creation](#flow-1-trip-creation)
   - [Driver Assignment](#flow-2-driver-assignment)
   - [Driver Decline / Retry](#flow-3-driver-decline--retry)
   - [Payment Session](#flow-4-payment-session)
   - [Real-Time Driver Location](#flow-5-real-time-driver-location)
   - [In-Trip Chat](#flow-6-in-trip-chat)
   - [Trip Cancellation](#flow-7-trip-cancellation)
7. [WS Topic Multiplexing](#ws-topic-multiplexing)
8. [Message Envelopes & Payloads](#message-envelopes--payloads)
9. [Exchange & Queue Bindings](#exchange--queue-bindings)
10. [Dead-Letter Queue](#dead-letter-queue)
11. [Cross-Pod Delivery via Redis Pub/Sub](#cross-pod-delivery-via-redis-pubsub)
12. [Scenario Walkthroughs — What Happens When…](#scenario-walkthroughs--what-happens-when)

---

## System Overview

```
                           ┌──────────────┐
              REST          │              │
   Rider ────────────────▶ │  API Gateway │◀────────── Driver
   Rider ◀──────── WS ───▶ │              │ ◀─ WS ───▶ Driver
                           └──────┬───────┘
                                  │ AMQP (topic exchange "trip")
                     ┌────────────┼────────────┬───────────────┐
                     ▼            ▼            ▼               ▼
              ┌─────────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐
              │ trip-service│  │ driver-  │  │ payment- │  │dlq-     │
              │             │  │ service  │  │ service  │  │worker   │
              └──────┬──────┘  └────┬─────┘  └────┬─────┘  └─────────┘
                     │  MongoDB     │  Redis       │  Stripe
                     │  (trips)     │  (geo, KV)   │
```

Every service publishes to and consumes from the **`trip` topic exchange** in RabbitMQ.  
The **ws-gateway** is the only service that holds WebSocket connections to the frontend.  
The **api-gateway** handles all REST ingress and forwards commands onto the AMQP exchange.

---

## Service Catalogue

### api-gateway

| | |
|---|---|
| **Port** | 8081 |
| **Protocols** | HTTP/REST, gRPC (outbound) |
| **Dependencies** | RabbitMQ, driver-service (gRPC) |

The REST entry-point for the frontend. Responsibilities:
- Validates JWT on every incoming request.
- Exposes `/trip/preview` (calls OSRM for routing) and `/trip/start` (writes to trip-service via gRPC or direct call).
- Translates driver WS commands (`trip_accept`, `trip_decline`, `location`) received from ws-gateway into AMQP messages on the `trip` exchange.
- Performs gRPC calls to driver-service for driver registration, unregistration, and geo-radius lookups.

---

### ws-gateway

| | |
|---|---|
| **Port** | 8082 |
| **Protocols** | WebSocket (upgrade from HTTP), AMQP (consumer), Redis Pub/Sub |
| **Dependencies** | RabbitMQ, Redis, driver-service (gRPC) |
| **Replicas** | 2 (dev) |

The sole WebSocket server. All real-time communication between the frontend and the backend flows through here.

**Key responsibilities:**
- Upgrades HTTP connections on `/ws/riders` and `/ws/drivers` after JWT validation.
- Runs one **global RabbitMQ consumer per notification queue** at startup — not per connection. Each consumer reads events and routes them to the correct socket via the [RedisConnectionManager](#redisconnectionmanager-rcm).
- Relays in-trip chat frames between rider and driver sockets using `ResolveTripChatPeer` (Redis KV lookup) and `connManager.BroadcastToRoom` on `trip:<tripID>:chat`.
- Forwards driver commands (`trip_accept`, `trip_decline`, `location`) as AMQP messages to the `trip` exchange.
- Enforces a per-user connection limit (max 3 simultaneous WebSocket connections) via a Redis-backed counter with `WsConnectionGate`.
- Maintains connection liveness with server-side Ping/Pong (`wsPingPeriod=20s`, `wsPongWait=45s`) to proactively clean up dead sockets.
- Uses a reconnect grace period (`driverReconnectGracePeriod=2s`) before unregistering a disconnected driver, preventing refresh races.
- Supports the **room model** for chat (`ws.room.join` / `ws.room.leave`) and the legacy **topic model** (`ws.topic.subscribe` / `ws.topic.unsubscribe`) for backwards compatibility.

**Global consumers started at boot:**

| Queue | Delivered to |
|---|---|
| `notify_trip_created` | Rider WS |
| `notify_driver_no_drivers_found` | Rider WS |
| `notify_driver_assign` | Rider WS |
| `notify_payment_session_created` | Rider WS |
| `notify_rider_driver_location` | Rider WS |
| `notify_trip_cancelled` | Rider WS + Driver WS |
| `chat.event.delivered` | Rider or Driver WS |
| `driver_cmd_trip_request` | Driver WS |

**Rate limiting:**

| Mechanism | Limit | Window |
|---|---|---|
| HTTP request rate limiter | configurable | rolling |
| WS connection gate | 3 simultaneous connections per userID | session |

---

### trip-service

| | |
|---|---|
| **Storage** | MongoDB (`trips` collection) |
| **Protocols** | AMQP (consumer + publisher) |

Owns the `Trip` aggregate. Handles trip lifecycle transitions:
- `trip.event.created` → persists trip, initiates driver search
- `driver.cmd.trip_accept` → marks trip `assigned`, attaches driver, triggers payment session
- `driver.cmd.trip_decline` → re-queues for next available driver

---

### driver-service

| | |
|---|---|
| **Storage** | Redis (GEO index, KV) |
| **Protocols** | AMQP (consumer + publisher), gRPC (server) |

Owns the real-time driver registry:
- Maintains a Redis GEO index for radius-based driver lookup.
- Tracks `active_rider` per driver for location relay.
- Exposes gRPC: `RegisterDriver`, `UnregisterDriver`, `GetNearbyDrivers`.

---

### payment-service

| | |
|---|---|
| **External** | Stripe Checkout API |
| **Protocols** | AMQP (consumer + publisher) |

Listens on `payment_trip_response`. On receipt, creates a Stripe Checkout Session and publishes `payment.event.session_created` → rider WS.

---

### chat-service

| | |
|---|---|
| **Storage** | MongoDB (`messages` collection) |
| **Protocols** | AMQP (consumer + publisher) |

Decoupled persistence layer for in-trip chat. Does **not** relay messages in real-time — that is handled directly by ws-gateway via Redis `SendMessage`. The chat-service exists to ensure durability and delivery receipts.
Decoupled persistence layer for in-trip chat. Does **not** relay messages in real-time — that is handled directly by ws-gateway via room broadcasts (`trip:<tripID>:chat`). The chat-service exists to ensure durability and delivery receipts.

**Architecture (hexagonal / ports-and-adapters):**

```
┌─────────────────────────────────────────────────┐
│                  chat-service                   │
│                                                 │
│  ┌──────────────┐      ┌────────────────────┐  │
│  │  domain      │      │  infrastructure    │  │
│  │              │      │                    │  │
│  │  Message     │      │  MongoRepository   │  │
│  │  MessageRepo │◀─────│  (implements       │  │
│  │  (interface) │      │   MessageRepo)     │  │
│  │              │      │                    │  │
│  │  MessagePub  │◀─────│  AmqpPublisher     │  │
│  │  (interface) │      │  (publishes        │  │
│  └──────┬───────┘      │  chat.event.       │  │
│         │              │  delivered)        │  │
│  ┌──────▼───────┐      └────────────────────┘  │
│  │  service     │                              │
│  │              │                              │
│  │  HandleIncoming(msg):                       │
│  │    1. repo.Save(msg)                        │
│  │    2. publisher.PublishDelivered(id, tripID)│
│  │                                             │
│  │  GetHistory(tripID, limit):                 │
│  │    repo.GetByTripID(...)                    │
│  └──────────────┘                              │
└─────────────────────────────────────────────────┘
```

**Message flow:**

```
ws-gateway relays chat frame to peer
        │
        └─ also publishes AMQP:
           routing_key: chat.cmd.send
           data: { tripID, senderID, text, sentAt, messageID }
                │
         ChatCmdSendQueue → chat-service consumer
                │
         ChatService.HandleIncoming():
           1. MongoRepository.Save()          ← persist
           2. AmqpPublisher.PublishDelivered() ← routing_key: chat.event.delivered
                │
         ChatEventDeliveredQueue → ws-gateway QueueConsumer
                │
         WS frame to rider + driver:
         { type: "chat.event.delivered",
           topic: "trip:<id>",
           data: { messageID, tripID } }
```

---

### dlq-worker

Standalone worker that consumes from `dead_letter_queue`. Implements progressive retry with up to 5 broker-level re-publishes back to the original exchange/routing key. Messages that exceed 5 retries are logged and acknowledged (dropped). See [Dead-Letter Queue](#dead-letter-queue).

---

## Transport Layers

| Layer | Technology | Direction | Purpose |
|---|---|---|---|
| REST | HTTP/JSON | Frontend → API Gateway | Trip preview, trip start |
| WebSocket | gorilla/websocket | Frontend ↔ API Gateway | All real-time events & commands |
| AMQP | RabbitMQ topic exchange | Inter-service | Async event-driven communication |
| gRPC | protobuf | API Gateway → Driver Service | Driver registration, unregistration, geo lookup |
| Redis KV | go-redis | Intra-service | Driver geo index, active rider mapping, chat pair keys |
| Redis Pub/Sub | go-redis | API Gateway pods | Cross-pod WS message delivery |

---

## WebSocket Layer

All WebSocket traffic is handled by **ws-gateway** (port 8082). The api-gateway (port 8081) handles REST only.

### Endpoints

| Service | Path | Consumer | Auth |
|---|---|---|---|
| ws-gateway | `GET /ws/riders?token=<jwt>` | Riders | JWT in query param |
| ws-gateway | `GET /ws/drivers?token=<jwt>&packageSlug=<slug>` | Drivers | JWT + package slug |

### Message Envelope

Every WebSocket frame (both directions) is a JSON object:

```json
{
  "type": "<TripEvents constant>",
  "topic": "trip:<tripID>",
  "data": { ... }
}
```

- `type` — identifies the event/command (see `TripEvents` enum in `web/src/contracts.ts`)
- `topic` — optional scope tag (e.g. `trip:abc123`). Absent on system/broadcast messages.
- `data` — event-specific payload

### Connection Health, Refresh, and Dead Socket Handling

The gateway treats every browser refresh as a short reconnect race and handles it in three layers:

1. **Socket liveness (Ping/Pong):**
   - Server sets `read deadline = now + 45s`.
   - Server sends Ping every `20s`.
   - Client Pong extends read deadline.
   - If no Pong arrives before deadline, read loop exits and socket is removed.

2. **Connection gate heartbeat (Redis):**
   - `WsConnectionGate` limits each user to `3` concurrent sockets.
   - Key TTL is refreshed every `20s` while socket is active.
   - On disconnect, gate counter is decremented.

3. **Driver refresh race protection:**
   - On driver socket close, gateway removes the socket first.
   - If another socket for same user already exists, unregister is skipped.
   - If none exists, gateway waits `2s` grace and checks again.
   - Only then it calls `UnregisterDriver`.

This prevents an old socket from unregistering a driver during a normal page refresh when the new socket arrives milliseconds later.

**Observed refresh timeline example (driver):**

```text
12:20:48 Driver WS read error: websocket: close 1001 (going away)
12:20:48 Removed socket <old-socket> (user <driverID>)
12:20:49 Added socket <new-socket> for user <driverID>
12:20:49 Skipping driver unregister for <driverID>: reconnected within grace period
```

### Subscriptions, Rooms, Channels, and Streams

The system separates **delivery scope** from **transport channel**:

| Layer | Identifier | Purpose |
|---|---|---|
| WS scope tag | `topic: trip:<tripID>` | Client-side trip correlation for owner-directed events |
| WS room | `roomID: trip:<tripID>:chat` | Real-time chat broadcast to all sockets in room |
| Redis user channel | `user:<userID>:events` | Cross-pod fan-out for user-targeted WS messages |
| Redis room channel | `room:<roomID>` | Cross-pod fan-out for room broadcasts |
| Redis stream | `user:<userID>:stream` | Offline fallback when no active user subscriber exists |

**Control frame examples:**

```json
{ "type": "ws.topic.subscribe", "data": { "topic": "trip:trip_123" } }
{ "type": "ws.topic.unsubscribe", "data": { "topic": "trip:trip_123:chat" } }
{ "type": "ws.room.join", "data": { "roomID": "trip:trip_123:chat" } }
```

**Chat receive frame example (room broadcast):**

```json
{
  "type": "chat.message.received",
  "roomID": "trip:trip_123:chat",
  "data": {
    "tripID": "trip_123",
    "roomID": "trip:trip_123:chat",
    "senderID": "user_456",
    "text": "I am 2 minutes away",
    "sentAt": 1717580400,
    "messageID": "a3b1-..."
  }
}
```

### Auto-Join/Leave Behavior (Current Implementation)

- Rider joins trip + chat scopes when receiving:
  - `trip.event.created`
  - `trip.event.driver_assigned`
  - `payment.event.session_created`
- Driver joins chat room server-side immediately on `driver.cmd.trip_accept`.
- On `trip.event.cancelled`, both sides unsubscribe trip + chat scopes on frontend, and gateway also performs server-side room cleanup on cancellation teardown.

### Client → Server message types

| `type` | Sender | `data` payload |
|---|---|---|
| `driver.cmd.location` | Driver | `{ location: { latitude, longitude } }` |
| `driver.cmd.trip_accept` | Driver | `{ tripID: string, riderID: string }` |
| `driver.cmd.trip_decline` | Driver | `{ tripID: string, riderID: string }` |
| `driver.cmd.register` | Driver | (handled internally, not sent from frontend) |
| `chat.message.send` | Rider or Driver | `{ tripID: string, text: string, messageID?: string }` |
| `ws.topic.subscribe` | Rider or Driver | `{ topic: string }` (legacy) |
| `ws.topic.unsubscribe` | Rider or Driver | `{ topic: string }` (legacy) |
| `ws.room.join` | Rider or Driver | `{ roomID: string }` |
| `ws.room.leave` | Rider or Driver | `{ roomID: string }` |

### Server → Client message types

| `type` | Recipient | `data` payload | `topic` |
|---|---|---|---|
| `trip.event.created` | Rider | `Trip` (proto JSON) | — |
| `trip.event.no_drivers_found` | Rider | `{ trip: Trip }` | `trip:<id>` |
| `trip.event.driver_not_interested` | Rider | `{ trip: Trip }` | `trip:<id>` |
| `trip.event.driver_assigned` | Rider | `Trip` (proto JSON with `driver`) | `trip:<id>` |
| `payment.event.session_created` | Rider | `{ tripID, sessionID, amount, currency }` | `trip:<id>` |
| `driver.event.location` | Rider | `{ latitude, longitude }` | — |
| `driver.cmd.trip_request` | Driver | `{ trip: Trip, pickupLat, pickupLng }` | — |
| `trip.event.cancelled` | Rider and Driver | `{ tripID: string }` | `trip:<id>` |
| `chat.message.received` | Rider or Driver | `{ tripID, roomID, senderID, text, sentAt, messageID? }` | room broadcast on `trip:<id>:chat` |

---

## RabbitMQ Layer

### Exchange

All messages flow through a single **topic exchange** named `trip`.  
Every published message carries a `routing key` that determines which queues receive it.

```
Producer ──→ [Exchange: "trip"]  ──binding──→  Queue  ──→ Consumer
```

### AMQP Envelope

Every message body (regardless of routing key) is a JSON-serialised `AmqpMessage`:

```json
{
  "ownerId": "<userID of the message target>",
  "data": <raw JSON bytes of the event-specific payload>
}
```

`ownerId` is the user whose WebSocket connection should ultimately receive the event (rider or driver).

---

## Redis Layer

### Keys

| Key pattern | Type | TTL | Purpose |
|---|---|---|---|
| `user:<userID>:events` | Pub/Sub channel | — | Cross-pod WS delivery |
| `user:<userID>:stream` | Redis Stream | 2 h | Offline event inbox for user-directed WS messages |
| `trip:<tripID>:chat:rider` | String | 2 h | Rider userID for chat authorisation |
| `trip:<tripID>:chat:driver` | String | 2 h | Driver userID for chat authorisation |
| `driver:<driverID>:active_rider` | String | — | Active rider linked to a driver (used for location relay) |

### Per-Connection Topic Registry (in-memory, `RedisConnectionManager`)

```
socketRooms: map[socketID] → set[roomID]
roomSockets: map[roomID] → set[socketID]
```

Stored in `RedisConnectionManager.socketRooms` and `RedisConnectionManager.roomSockets` (both guarded by a mutex).  
Populated via `JoinRoom(socketID, roomID)` called from:

- Frontend `ws.room.join` frame (preferred)
- Frontend `ws.topic.subscribe` frame (legacy alias that maps to `JoinRoom`)

### Redis Streams (Definition & Usage)

**Definition:** Redis Streams is an append-only log data structure (`XADD`, `XRANGE`, `XDEL`) that can persist ordered events for later replay.

In this system, Streams are used as a **fallback inbox** for user-directed WS events when no socket subscriber is currently online:

1. `connManager.SendMessage(userID, msg)` first tries local socket delivery.
2. If not local, it publishes to `user:<userID>:events` (Redis Pub/Sub).
3. If Pub/Sub has zero subscribers, ws-gateway appends the event to `user:<userID>:stream`.
4. On the next socket connect (`rcm.Add`), ws-gateway replays pending stream entries to that user and deletes acknowledged entries.

This prevents message loss during short disconnects, pod restarts, or reconnect races.

---

## Complete Data Flows

### Flow 1: Trip Creation

```
Frontend (Rider)
  │
  ├─ POST /trip/preview  →  API Gateway  →  OSRM  →  returns { route, rideFares }
  │
  └─ POST /trip/start  →  API Gateway  →  trip-service (gRPC or direct)
                                          │
                                          └─ Publishes AMQP:
                                             routing_key: trip.event.created
                                             ownerId:     <riderID>
                                             data: {
                                               trip: { id, userID, status, selectedFare, route, ... },
                                               pickupLat: float,
                                               pickupLng: float
                                             }
                                          │
           ┌───────────────────────────────┤
           │                               │
     notify_trip_created             find_available_drivers
     Queue (API Gateway consumer)    Queue (driver-service consumer)
           │                               │
           ▼                               ▼
   WS frame to rider:              driver-service.handleAndNotifyDrivers()
   { type: "trip.event.created",
     data: Trip }
           │
           ▼
   Rider frontend:
   - dispatch(setTripStatus("trip.event.created"))
   - sends ws.topic.subscribe { topic: "trip:<id>" }
```

**Rider REST payload for /trip/start:**
```json
{
  "rideFareID": "<uuid>",
  "userID": "<riderID>"
}
```

---

### Flow 2: Driver Assignment

```
driver-service  (FindAvailableDrivers → GEO RADIUS search in Redis)
  │
  └─ Publishes AMQP:
     routing_key: driver.cmd.trip_request
     ownerId:     <driverID>
     data: {
       trip: { id, userID, status, selectedFare, route, ... },
       pickupLat: float,
       pickupLng: float
     }
                │
     driver_cmd_trip_request
     Queue (API Gateway consumer)
                │
                ▼
     WS frame to driver:        (no topic — initial dispatch, driver not yet subscribed)
     { type: "driver.cmd.trip_request",
       data: { trip: Trip, pickupLat, pickupLng } }
                │
                ▼
     Driver frontend:
     - dispatch(setRequestedTrip(trip))
     - dispatch(setTripStatus("driver.cmd.trip_request"))
    - sends ws.topic.subscribe { topic: "trip:<id>" }   ← optional legacy room join
                │
     Driver clicks Accept
                │
     WS frame from driver:
     { type: "driver.cmd.trip_accept",
       data: { tripID: "<id>", riderID: "<id>" } }
                │
     API Gateway:
     1. connManager.SetTripChatPair(tripID, riderID, driverID, 2h)   ← Redis KV
    2. connManager.JoinRoom(socketID, "trip:<id>:chat")             ← driver joins chat room
    3. Publishes AMQP:
        routing_key: driver.cmd.trip_accept
        ownerId:     <driverID>
        data: {
          tripID, riderID, driverID, driverName, packageSlug
        }
                │
       driver_trip_response
       Queue (trip-service driverConsumer)
                │
                ▼
     trip-service.handleTripAccepted():
     1. Fetch trip from MongoDB
     2. UpdateTrip(status=assigned, driver={id, name})
     3. Publishes AMQP:
        routing_key: trip.event.driver_assigned
        ownerId:     <riderID>
        data: Trip (proto JSON with driver field)
     4. Publishes AMQP:
        routing_key: payment.cmd.create_session
        ownerId:     <riderID>
        data: { tripID, userID, driverID, amount, currency: "USD" }
                │
          ┌─────┴──────────────────────┐
          │                            │
  notify_driver_assign          payment_trip_response
  Queue (API Gateway)           Queue (payment-service)
  driver_trip_assigned
  Queue (driver-service)
          │
          ▼
  WS frame to rider:
  { type: "trip.event.driver_assigned",
    topic: "trip:<id>",
    data: Trip }                      ← owner-delivered; `topic` is a scope label for clients

  driver-service tripAssignedConsumer:
  - SetActiveRider(driverID, riderID)  ← Redis KV for location relay
```

---

### Flow 3: Driver Decline / Retry

```
Driver clicks Decline
  │
  WS frame from driver:
  { type: "driver.cmd.trip_decline",
    data: { tripID, riderID } }
  │
  API Gateway publishes AMQP:
  routing_key: driver.cmd.trip_decline
  ownerId:     <driverID>
  data: { tripID, riderID, driverID, driverName, packageSlug }
  │
  driver_trip_response Queue → trip-service.handleTripDeclined()
  │
  Publishes AMQP:
  routing_key: trip.event.driver_not_interested
  ownerId:     <riderID>
  data: { trip: Trip }
  │
  ┌─────────────────────────────┐
  │                             │
  find_available_drivers     notify_driver_no_drivers_found (if no more drivers)
  Queue (driver-service)     Queue (API Gateway)
  │
  ▼  (same as Flow 2 — pick next driver)

  If no drivers at all:
  WS frame to rider:
  { type: "trip.event.no_drivers_found",
    topic: "trip:<id>",
    data: { trip: Trip } }
```

---

### Flow 4: Payment Session

```
trip-service publishes payment.cmd.create_session  (end of Flow 2)
  │
  payment_trip_response Queue → payment-service.handleTripAccepted()
  │
  1. Calls Stripe API → creates Checkout Session
  2. Publishes AMQP:
     routing_key: payment.event.session_created
     ownerId:     <riderID>
     data: {
       tripID:    "<id>",
       sessionID: "<stripe_session_id>",
       amount:    <cents as float>,
       currency:  "USD"
     }
  │
  notify_payment_session_created Queue → API Gateway
  │
  WS frame to rider:
  { type: "payment.event.session_created",
    topic: "trip:<id>",
    data: { tripID, sessionID, amount, currency } }
```

---

### Flow 5: Real-Time Driver Location

```
Driver frontend (watchPosition callback fires)
  │
  WS frame from driver:
  { type: "driver.cmd.location",
    data: { location: { latitude, longitude } } }
  │
  API Gateway publishes AMQP:
  routing_key: driver.cmd.location
  ownerId:     <driverID>
  data: { packageSlug, latitude, longitude }
  │
  driver_location_update Queue → driver-service locationConsumer
  │
  1. UpdateDriverLocation(driverID, packageSlug, lat, lng)  ← Redis GEO index
  2. GetActiveRider(driverID)                               ← Redis KV
     if no active rider → stop
  3. Publishes AMQP:
     routing_key: driver.event.location
     ownerId:     <riderID>
     data: { latitude, longitude }
  │
  notify_rider_driver_location Queue → API Gateway
  │
  WS frame to rider:
  { type: "driver.event.location",
    data: { latitude, longitude } }    (no topic — always delivered)
```

---

### Flow 7: Trip Cancellation

```
Frontend (Rider)
  │
  └─ POST /trip/cancel  { tripID }  →  API Gateway
                                        │
                                        └─ trip-service (gRPC CancelTrip)
                                             │
                                             1. Fetch trip from MongoDB
                                             2. Validate requester == trip.UserID
                                             3. UpdateTrip(status=cancelled)
                                             4. Publishes AMQP:
                                                routing_key: trip.event.cancelled
                                                ownerId:     <riderID>
                                                data: {
                                                  tripID,
                                                  riderID,
                                                  driverID,          (empty string if no driver was assigned)
                                                  driverAccepted     (true when cancelled after assignment)
                                                }
                                        │
                               notify_trip_cancelled
                               Queue (ws-gateway cancelConsumer)
                                        │
                                        ▼
                               cancelConsumer:
                               1. Resolve effective teardown condition:
                                    shouldTearDown = driverAccepted || driverID != "" || GetActiveDriver(riderID) != ""
                               2. If shouldTearDown: ClearTripChatPair(tripID)
                                    │ deletes trip:<tripID>:chat:rider
                                    │ deletes trip:<tripID>:chat:driver
                                    │ deletes rider:<riderID>:active_driver
                                    │ deletes driver:<driverID>:active_rider
                               3. Force room leave for rider + driver on:
                                    trip:<tripID>
                                    trip:<tripID>:chat
                               4. SendMessage(riderID, wsMsg)
                               5. if driverID != "": SendMessage(driverID, wsMsg)
                                        │
                               WS frame to rider + driver:
                               { type: "trip.event.cancelled",
                                 topic: "trip:<id>",
                                 data: { tripID } }
                                        │
                    ┌───────────────────┴───────────────────────┐
                    ▼                                           ▼
           Rider frontend:                             Driver frontend:
           case TripEvents.Cancelled:                  case TripEvents.Cancelled:
           - sends ws.topic.unsubscribe trip:<id>      - sends ws.topic.unsubscribe trip:<id>
           - sends ws.topic.unsubscribe trip:<id>:chat - sends ws.topic.unsubscribe trip:<id>:chat
           - dispatch(resetTrip())                     - dispatch(resetTrip())
           - dispatch(setTripStatus(Cancelled))        - dispatch(setTripStatus(Cancelled))
```

**Idempotency:** If the trip is already `cancelled`, trip-service returns the existing trip and cancellation remains a no-op on persistence.

**Driver location relay:** Once `driver:<driverID>:active_rider` is deleted by `ClearTripChatPair`, the driver-service `locationConsumer` finds no active rider and stops forwarding location updates — the relay self-terminates on the next GPS frame.

---

### Flow 6: In-Trip Chat

Pre-condition: `trip:<tripID>:chat:rider` and `trip:<tripID>:chat:driver` keys exist in Redis  
(set on `driver.cmd.trip_accept`, TTL 2 hours).

```
Rider sends:
  WS frame: { type: "chat.message.send",
              data: { tripID, text, messageID? } }
  │
  API Gateway relayTripChatMessage():
  1. ResolveTripChatPeer(tripID, riderID)  → driverID  (or error if pair missing)
  2. Build received frame:
     { type: "chat.message.received",
      roomID: "trip:<id>:chat",
      data: { tripID, roomID, senderID: <riderID>, text, sentAt, messageID? } }
    3. BroadcastToRoom("trip:<id>:chat", frame)
  │
    All sockets in room "trip:<id>:chat" (rider + driver, cross-pod) receive the message

Driver sends: identical flow, senderID = driverID
```

---

## WS Topic Multiplexing

A single WebSocket connection carries multiple logical channels by tagging each frame with a `topic` field.

### Subscription Lifecycle

```
Client connects
      │
      │  (initial dispatch messages have no topic — always delivered)
      │
      ├─ Rider receives trip.event.created ──▶ sends { type: "ws.topic.subscribe", data: { topic: "trip:X" } }
      │
      ├─ Driver receives driver.cmd.trip_request ──▶ sends { type: "ws.topic.subscribe", data: { topic: "trip:X" } }
      │
      ├─ Driver accepts ──▶ Gateway joins chat room `trip:X:chat` for that socket
      │
      │  Owner-directed frames are delivered by userID.
      │  `topic` is attached as a scope tag so clients can correlate events to a trip.
      │  Chat uses room broadcast (`roomID="trip:X:chat"`).
      │
    Client disconnects ──▶ socket is removed from all joined rooms
```

### Control Frames

| `type` | Direction | `data` |
|---|---|---|
| `ws.topic.subscribe` | Client → Server | `{ topic: "trip:<tripID>" }` |
| `ws.topic.unsubscribe` | Client → Server | `{ topic: "trip:<tripID>" }` |

### Messages NOT tagged (always delivered)

- `trip.event.created` — first rider notification
- `driver.cmd.trip_request` — first driver dispatch notification
- `driver.event.location` — continuous GPS stream, not scoped to a topic

### Messages tagged with `trip:<tripID>`

- `trip.event.no_drivers_found`
- `trip.event.driver_not_interested`
- `trip.event.driver_assigned`
- `payment.event.session_created`

### Chat room messages

- `chat.message.received` is broadcast in room `trip:<tripID>:chat` (not tagged with `topic=trip:<tripID>`)

---

## Message Envelopes & Payloads

### AMQP Outer Envelope (all queues)

```json
{
  "ownerId": "<userID>",
  "data": <raw JSON bytes>
}
```

### WS Outer Envelope (all frames)

```json
{
  "type": "<routing key string>",
  "topic": "trip:<id>",
  "data": { ... }
}
```

### Key Payload Shapes

**TripEventData** (published by trip-service on creation)
```json
{
  "trip": {
    "id": "...",
    "userID": "...",
    "status": "created",
    "selectedFare": { "id", "packageSlug", "totalPriceInCents", ... },
    "route": { "geometry": { "coordinates": [[lng,lat], ...] }, "distance", "duration" },
    "driver": null
  },
  "pickupLat": 28.6139,
  "pickupLng": 77.2090
}
```

**DriverTripResponseData** (driver accept/decline enriched by gateway)
```json
{
  "tripID": "...",
  "riderID": "...",
  "driverID": "...",
  "driverName": "John Doe",
  "packageSlug": "economy"
}
```

**PaymentTripResponseData** (trip-service → payment-service)
```json
{
  "tripID": "...",
  "userID": "...",
  "driverID": "...",
  "amount": 1250.0,
  "currency": "USD"
}
```

**PaymentEventSessionCreatedData** (payment-service → rider WS)
```json
{
  "tripID": "...",
  "sessionID": "cs_test_...",
  "amount": 1250.0,
  "currency": "USD"
}
```

**DriverLocationUpdateData** (driver WS → driver-service)
```json
{
  "packageSlug": "economy",
  "latitude": 28.6139,
  "longitude": 77.2090
}
```

**DriverLocationEventData** (driver-service → rider WS)
```json
{
  "latitude": 28.6139,
  "longitude": 77.2090
}
```

**ChatMessageData** (gateway relay, both directions)
```json
{
  "tripID": "...",
  "senderID": "<userID>",
  "text": "I'm 2 minutes away",
  "sentAt": 1716470400,
  "messageID": "<uuid optional>"
}
```

---

## Exchange & Queue Bindings

Exchange name: `trip` (type: `topic`)

| Routing Key | Queues bound |
|---|---|
| `trip.event.created` | `notify_trip_created`, `find_available_drivers` |
| `trip.event.driver_not_interested` | `find_available_drivers`, `notify_driver_no_drivers_found` |
| `trip.event.no_drivers_found` | `notify_driver_no_drivers_found` |
| `trip.event.driver_assigned` | `notify_driver_assign`, `driver_trip_assigned` |
| `driver.cmd.trip_request` | `driver_cmd_trip_request` |
| `driver.cmd.trip_accept` | `driver_trip_response` |
| `driver.cmd.trip_decline` | `driver_trip_response` |
| `driver.cmd.location` | `driver_location_update` |
| `payment.cmd.create_session` | `payment_trip_response` |
| `payment.event.session_created` | `notify_payment_session_created` |
| `driver.event.location` | `notify_rider_driver_location` |
| `trip.event.cancelled` | `notify_trip_cancelled` |
| `chat.cmd.send` | `chat_cmd_send` |
| `chat.event.delivered` | `chat_event_delivered` |

### Queue → Consumer Mapping

| Queue | Consumer service |
|---|---|
| `notify_trip_created` | ws-gateway (QueueConsumer → rider WS) |
| `find_available_drivers` | driver-service (`tripConsumer`) |
| `driver_cmd_trip_request` | ws-gateway (QueueConsumer → driver WS) |
| `driver_trip_response` | trip-service (`driverConsumer`) |
| `notify_driver_no_drivers_found` | ws-gateway (QueueConsumer → rider WS) |
| `notify_driver_assign` | ws-gateway (QueueConsumer → rider WS) |
| `driver_trip_assigned` | driver-service (`tripAssignedConsumer`) |
| `driver_location_update` | driver-service (`locationConsumer`) |
| `notify_rider_driver_location` | ws-gateway (QueueConsumer → rider WS) |
| `payment_trip_response` | payment-service (`TripConsumer`) |
| `notify_payment_session_created` | ws-gateway (QueueConsumer → rider WS) |
| `notify_trip_cancelled` | ws-gateway (`cancelConsumer` → rider WS + driver WS) |
| `chat_cmd_send` | chat-service (AMQP consumer) |
| `chat_event_delivered` | ws-gateway (QueueConsumer → rider + driver WS) |
| `dead_letter_queue` | dlq-worker |

---

## Dead-Letter Queue

Any message that fails processing after `MaxRetries` (default 3 with exponential backoff) is:

1. Rejected with `requeue=false`
2. Routed to exchange `dlx` → queue `dead_letter_queue`
3. Enriched with headers: `x-death-reason`, `x-origin-exchange`, `x-original-routing-key`, `x-retry-count`
4. Consumed by the `dlq-worker` service for alerting / manual replay

---

## Cross-Pod Delivery via Redis Pub/Sub

When the API Gateway runs as multiple replicas, a rider's WS connection exists on only one pod.  
Events published by backend services target a `userID`, not a pod.

```
Any pod:  connManager.SendMessage(userID, msg)
            │
            ├─ try localCM.SendMessage(userID, msg)
            │    └─ success? return
            │
       └─ not local → rdb.Publish("user:<userID>:events", json(msg))
                                │
                         Redis Pub/Sub
                                │
                         Pod that has the WS connection
          subscribed goroutine receives msg
            │
                         localCM.SendMessage(userID, msg) → WS frame
```

Each `Add(userID, conn)` call subscribes that pod's goroutine to `user:<userID>:events`.  
Each socket `Remove(socketID)` unsubscribes user/room channels only when no local sockets remain.

---

## RedisConnectionManager (RCM)

`RedisConnectionManager` (`shared/messaging/rcm.go`) is the central nervous system of the ws-gateway. It bridges in-process WebSocket connections with cross-pod Redis Pub/Sub so that any backend service can deliver a message to any user regardless of which gateway pod holds their socket.

### Internal Structure

```
RedisConnectionManager
│
├── localCM *ConnectionManager          ← in-memory socket registry
│     ├── bySocket  map[socketID]*connRecord    (socketID → conn + userID)
│     └── byUser    map[userID]set[socketID]    (userID  → all sockets on this pod)
│
├── userSubs  map[userID]*redis.PubSub   ← one Redis sub per user, shared across their sockets
├── roomSubs  map[roomID]*redis.PubSub   ← one Redis sub per room
│
├── socketRooms map[socketID]set[roomID] ← which rooms a socket has joined
└── roomSockets map[roomID]set[socketID] ← which local sockets are in a room
```

One mutex (`rcm.mu`) guards the Redis subscription maps.  
`localCM` has its own separate `sync.RWMutex` for the socket maps.

---

### Delivery Models

The RCM supports two distinct delivery models:

#### 1. User-Direct (system & trip events)

Each user gets a dedicated Redis channel `user:<userID>:events`.  
Any pod can publish to this channel; the pod that holds the user's socket delivers it locally.

```
Any pod
  │
  connManager.SendMessage(userID, msg)
  │
  ├─ localCM.SendMessage(userID, msg)
  │    └─ found locally → write to all sockets of this user on this pod → done
  │
  └─ not found locally
       │
       rdb.Publish("user:<userID>:events", json(msg))
              │
           Redis Pub/Sub
              │
       Pod holding the connection
       runUserSubscription goroutine receives msg
              │
       localCM.SendMessage(userID, msg) → WS frame sent
```

#### 2. Room Broadcast (in-trip chat)

Chat uses named rooms (`trip:<tripID>:chat`). All sockets in the room — potentially on different pods — receive the message.

```
connManager.BroadcastToRoom(roomID, msg)
  │
  ├─ fan-out to all local sockets in the room
  │
  └─ rdb.Publish("room:<roomID>", json(msg))
              │
           Redis Pub/Sub
              │
       Every other pod subscribed to this room
       runRoomSubscription goroutine receives msg
              │
       fan-out to that pod's local sockets in the room
```

#### 3. User Stream Replay (offline fallback)

When no pod has an active subscriber for `user:<userID>:events`, ws-gateway stores the message in `user:<userID>:stream` for replay.

```
connManager.SendMessage(userID, msg)
  │
  ├─ localCM.SendMessage(userID, msg)
  │    └─ not found locally
  │
  ├─ rdb.Publish("user:<userID>:events", msg)
  │    └─ subscribers == 0
  │
  └─ XADD user:<userID>:stream payload=msg   (TTL + max length)

Later, on Add(userID, socketID, conn):
  │
  ├─ XRANGE user:<userID>:stream - +
  ├─ localCM.SendMessage(userID, msg) for each entry
  └─ XDEL acknowledged entry IDs
```

Operational limits currently used:

- Stream key TTL: 2 hours
- Approximate max length: 500 entries per user stream

---

### Socket Lifecycle

```
Client connects (HTTP Upgrade)
        │
        ws-gateway generates socketID = uuid.New()
        │
        rcm.Add(userID, socketID, conn)
              │
              ├─ localCM.Add(socketID, userID, conn)     ← register socket
              │
              └─ if first socket for this user on this pod:
                   rdb.Subscribe("user:<userID>:events") ← subscribe Redis channel
                   go runUserSubscription(userID, pubsub) ← delivery goroutine starts
                    go replayUserStream(userID)            ← flush offline stream backlog


Client joins room (ws.room.join or ws.topic.subscribe)
        │
        rcm.JoinRoom(socketID, roomID)
              │
              ├─ socketRooms[socketID].add(roomID)
              ├─ roomSockets[roomID].add(socketID)
              │
              └─ if first local socket in this room:
                   rdb.Subscribe("room:<roomID>")        ← subscribe room channel
                   go runRoomSubscription(roomID, pubsub) ← delivery goroutine starts


Client disconnects
        │
        rcm.Remove(socketID)
              │
              ├─ leave all rooms this socket was in
              │    └─ if room has no more local sockets:
              │         pubsub.Close()                   ← unsubscribe room channel
              │         delete roomSubs[roomID]
              │
              ├─ localCM.Remove(socketID)                ← deregister socket
              │
              └─ if user has no more sockets on this pod:
                   pubsub.Close()                        ← unsubscribe user channel
                   delete userSubs[userID]
```

---

### Why One Subscription Per User (Not Per Socket)

A single user may have multiple WebSocket connections (e.g. two browser tabs). The RCM subscribes to `user:<userID>:events` **once** per pod, regardless of how many sockets that user has. When a message arrives on the channel, `localCM.SendMessage` fans it out to **all** of that user's sockets on this pod.

This avoids duplicate Redis subscriptions and duplicate message delivery for multi-tab users.

### User ↔ Socket Mapping

`ConnectionManager` keeps a bi-directional in-memory index:

```
bySocket: map[socketID] -> { conn, userID }
byUser:   map[userID]   -> set[socketID]
```

Implications:

1. One user can have multiple active sockets (multi-tab / multi-device).
2. User-directed messages fan out to all sockets in `byUser[userID]`.
3. On socket close, only that `socketID` is removed; user subscription remains until the last socket is gone.
4. Because this map is process memory, pod restarts clear it; Redis Pub/Sub + Streams bridge that gap across reconnect.

---

### Topic / Room Duality

The ws-gateway supports two naming conventions that map to the same room model internally:

| Client frame | Maps to |
|---|---|
| `{ type: "ws.room.join", data: { roomID: "trip:abc:chat" } }` | `rcm.JoinRoom(socketID, "trip:abc:chat")` |
| `{ type: "ws.topic.subscribe", data: { topic: "trip:abc" } }` | `rcm.JoinRoom(socketID, "trip:abc")` (legacy alias) |

Both control frames result in an identical Redis room subscription. The legacy topic API is maintained for frontend backward compatibility.

---

## Scenario Walkthroughs — What Happens When…

End-to-end numbered steps for every user-facing event. Each step names the actor, the transport used, and the state change produced.

---

### 1. Rider Opens the App

1. Frontend calls `GET /auth/google` → login-service → Google OAuth → returns JWT pair.
2. Frontend opens WebSocket: `GET ws://ws-gateway:8082/ws/riders?token=<jwt>`.
3. ws-gateway validates the JWT (extracts `userID` from claims).
4. ws-gateway calls `rl.WsConnectionGate(userID, 3)` → increments Redis counter; rejects if already 3 open connections.
5. ws-gateway generates `socketID = uuid.New()` and calls `connManager.Add(userID, socketID, conn)`.
6. RCM registers the socket locally and subscribes this pod to Redis channel `user:<userID>:events` (once per pod per user).
7. Connection is now live. No messages are sent until the rider starts a trip.

---

### 2. Driver Opens the App

Same steps as Rider with endpoint `/ws/drivers?token=<jwt>&packageSlug=<slug>`.

Additionally, ws-gateway reads `packageSlug` from the query param and calls driver-service gRPC `RegisterDriver(driverID, packageSlug, lat, lng)` which writes the driver into the Redis GEO index so they are discoverable for trip matching.

---

### 3. Rider Requests a Trip

| Step | Actor | Transport | Action |
|---|---|---|---|
| 1 | Rider frontend | REST POST `/trip/preview` | api-gateway calls OSRM, returns route + fare options |
| 2 | Rider selects fare, taps "Book" | REST POST `/trip/start` `{ rideFareID, userID }` | api-gateway → trip-service |
| 3 | trip-service | MongoDB | Persists `Trip` document with `status=created` |
| 4 | trip-service | AMQP publish | `trip.event.created` → binds to `notify_trip_created` + `find_available_drivers` |
| 5 | ws-gateway QueueConsumer | RCM `SendMessage(riderID)` | WS frame `{ type: "trip.event.created", data: Trip }` delivered to rider |
| 6 | Rider frontend | WS send | Sends `{ type: "ws.topic.subscribe", data: { topic: "trip:<id>" } }` → pod calls `JoinRoom(socketID, "trip:<id>")` |
| 7 | driver-service tripConsumer | Redis GEO | `GEORADIUS` search for nearby drivers by `packageSlug` |
| 8 | driver-service | AMQP publish | `driver.cmd.trip_request` → `driver_cmd_trip_request` queue |
| 9 | ws-gateway QueueConsumer | RCM `SendMessage(driverID)` | WS frame `{ type: "driver.cmd.trip_request", data: { trip, pickupLat, pickupLng } }` delivered to driver |

---

### 4. Driver Accepts the Trip

| Step | Actor | Transport | Action |
|---|---|---|---|
| 1 | Driver taps Accept | WS send | `{ type: "driver.cmd.trip_accept", data: { tripID, riderID } }` |
| 2 | ws-gateway | Redis KV (SET) | `trip:<tripID>:chat:rider = riderID`, `trip:<tripID>:chat:driver = driverID`, TTL 2h — authorises the chat pair |
| 3 | ws-gateway | RCM `JoinRoom` | Auto-subscribes driver server-side to `trip:<id>` room (no round-trip needed) |
| 4 | ws-gateway | AMQP publish | `driver.cmd.trip_accept` → `driver_trip_response` queue |
| 5 | trip-service driverConsumer | MongoDB | `UpdateTrip(status=assigned, driver={id,name})` |
| 6 | trip-service | AMQP publish (×2) | `trip.event.driver_assigned` → `notify_driver_assign` + `driver_trip_assigned` |
| | | | `payment.cmd.create_session` → `payment_trip_response` |
| 7 | ws-gateway QueueConsumer | RCM `SendMessage(riderID)` | `{ type: "trip.event.driver_assigned", topic: "trip:<id>", data: Trip }` → rider WS |
| 8 | driver-service tripAssignedConsumer | Redis KV (SET) | `driver:<driverID>:active_rider = riderID` — enables location relay |
| 9 | payment-service TripConsumer | Stripe API | Creates Checkout Session → gets `sessionID` |
| 10 | payment-service | AMQP publish | `payment.event.session_created` → `notify_payment_session_created` |
| 11 | ws-gateway QueueConsumer | RCM `SendMessage(riderID)` | `{ type: "payment.event.session_created", topic: "trip:<id>", data: { sessionID, amount, currency } }` → rider WS |

---

### 5. Driver Declines the Trip

| Step | Actor | Transport | Action |
|---|---|---|---|
| 1 | Driver taps Decline | WS send | `{ type: "driver.cmd.trip_decline", data: { tripID, riderID } }` |
| 2 | ws-gateway | AMQP publish | `driver.cmd.trip_decline` → `driver_trip_response` queue |
| 3 | trip-service driverConsumer | — | Marks driver as not interested for this trip |
| 4 | trip-service | AMQP publish | `trip.event.driver_not_interested` → `find_available_drivers` + `notify_driver_no_drivers_found` |
| 5a | driver-service tripConsumer | Redis GEO | Searches for next available driver (excluding declined drivers) → repeat from Step 8 of Flow 3 |
| 5b (if none left) | ws-gateway QueueConsumer | RCM `SendMessage(riderID)` | `{ type: "trip.event.no_drivers_found", topic: "trip:<id>", data: { trip } }` → rider WS |

---

### 6. Driver Sends Live Location

| Step | Actor | Transport | Action |
|---|---|---|---|
| 1 | Driver frontend (GPS callback) | WS send | `{ type: "driver.cmd.location", data: { location: { latitude, longitude } } }` |
| 2 | ws-gateway | AMQP publish | `driver.cmd.location` → `driver_location_update` queue |
| 3 | driver-service locationConsumer | Redis GEO | `GEOADD driver:<packageSlug>:locations driverID lat lng` — updates geo index |
| 4 | driver-service | Redis KV GET | `driver:<driverID>:active_rider` — finds linked rider |
| 5 (if rider active) | driver-service | AMQP publish | `driver.event.location` → `notify_rider_driver_location` queue |
| 6 | ws-gateway QueueConsumer | RCM `SendMessage(riderID)` | `{ type: "driver.event.location", data: { latitude, longitude } }` — no topic, always delivered |
| 7 | Rider frontend | Redux dispatch | `setDriverLocation({ lat, lng })` → map marker moves |

---

### 7. Rider or Driver Sends a Chat Message

Pre-condition: both parties must have joined room `trip:<tripID>:chat` (via `ws.room.join` or `ws.topic.subscribe`).

| Step | Actor | Transport | Action |
|---|---|---|---|
| 1 | Sender (rider or driver) | WS send | `{ type: "chat.message.send", data: { tripID, text, messageID? } }` |
| 2 | ws-gateway `relayTripChatMessage` | Redis KV GET | `ResolveTripChatPeer(tripID, senderID)` — looks up `trip:<tripID>:chat:rider/driver` to verify both parties exist and find the peer |
| 3 | ws-gateway | — | Assigns `messageID = uuid` if not provided; records `sentAt = now` |
| 4 | ws-gateway | RCM `BroadcastToRoom` | Delivers `{ type: "chat.message.received", roomID: "trip:<tripID>:chat", data: { tripID, senderID, text, sentAt, messageID } }` to all sockets in the room — including the sender (echo) and the peer on any pod |
| 5 | ws-gateway (fire-and-forget) | AMQP publish | `chat.cmd.send` → `chat_cmd_send` queue |
| 6 | chat-service consumer | MongoDB | `MessageRepository.Save(msg)` — durable persistence |
| 7 | chat-service | AMQP publish | `chat.event.delivered` → `chat_event_delivered` queue |
| 8 | ws-gateway QueueConsumer | RCM `BroadcastToRoom` | `{ type: "chat.event.delivered", data: { messageID, tripID } }` → both sockets in the room |
| 9 | Frontend | Redux dispatch | Marks message as delivered (tick indicator) |

> **Latency note:** Steps 1–4 happen synchronously on the same ws-gateway pod in memory. The peer sees the message before Step 5 even completes. Steps 5–9 are the async durability path.

---

### 8. Cross-Pod Message Delivery (ws-gateway scaled to 2 replicas)

When pod A needs to deliver a message to a user whose socket is on pod B:

```
Pod A — ws-gateway
  │  connManager.SendMessage(userID, msg)
  │    └─ localCM.SendMessage(userID) → not found (user not on this pod)
  │
  │  rdb.Publish("user:<userID>:events", json(msg))
  │            │
  │         Redis Pub/Sub
  │            │
Pod B — ws-gateway
  │  runUserSubscription goroutine unblocks
  │  localCM.SendMessage(userID, msg) → socket found → WS frame written
```

For room broadcasts (chat), the same pattern applies but the channel is `room:<roomID>` and all local sockets in that room on pod B receive it.

---

### 9. WebSocket Connection Drops / Rider Reconnects

1. TCP connection closes or browser refresh occurs (`1000`/`1001`) → read loop exits.
2. The deferred `connManager.Remove(socketID)` fires:
   - Leaves all rooms the socket was in.
   - If it was the user's last socket on this pod, closes and deletes the `user:<userID>:events` Redis subscription.
3. Rider frontend `ws.onclose` fires:
   - Code 1000/1001 → clean close, no error shown.
   - Code 1006 → checks JWT expiry:
     - Expired → `dispatch(logout())`.
     - Not expired → `dispatch(addError({ message: "Lost connection to the server" }))`.
   - Other codes → `dispatch(addError({ message: e.reason || "Connection closed unexpectedly (code X)" }))`.
4. On reconnect, gateway `Add(userID, socketID, conn)`:
   - re-subscribes to `user:<userID>:events` if needed,
   - replays `user:<userID>:stream` backlog when present.
5. Frontend auto-rejoins trip scopes when trip lifecycle events arrive (`created`, `driver_assigned`, `payment_session_created`).

**Data-path example when user is offline during event:**

```text
QueueConsumer -> connMgr.SendMessage(userID,msg)
  -> Publish user:<userID>:events
  -> receivedBy = 0
  -> XADD user:<userID>:stream payload=msg

Next connect:
  Add(userID,socketID)
  -> replayUserStream(userID)
  -> Send replayed messages to local sockets
  -> XDEL acknowledged entries
```

---

### 10. AMQP Message Processing Failure (DLQ path)

1. Consumer goroutine calls handler; handler returns an error.
2. `shared/messaging/rabbitmq.go ConsumeMessages`: calls `msg.Reject(requeue=false)`.
3. RabbitMQ routes the rejected message to exchange `dlx` → queue `dead_letter_queue` (bound at startup via `setupExchangesAndQueues`).
4. dlq-worker consumer reads the message:
   - Reads `x-retry-count` header (default 0).
   - If `retry-count < 5`: increments header, re-publishes to original exchange + routing key with an exponential delay, then Ack.
   - If `retry-count >= 5`: logs full message details and Ack (dropped).
5. The re-published message re-enters the normal consumer path from Step 1.

---

### 12. Rider Cancels a Trip

| Step | Actor | Transport | Action |
|---|---|---|---|
| 1 | Rider taps Cancel | REST `POST /trip/cancel` `{ tripID }` | api-gateway receives request; extracts `userID` from JWT claims |
| 2 | api-gateway | gRPC `CancelTrip(tripID, userID)` | trip-service validates requester == trip owner |
| 3 | trip-service | MongoDB | `UpdateTrip(status=cancelled)` — persists the state change |
| 4 | trip-service | AMQP publish | `trip.event.cancelled` → `notify_trip_cancelled` queue; payload `{ tripID, riderID, driverID }` |
| 5 | ws-gateway cancelConsumer | Redis DEL (×4) | `ClearTripChatPair(tripID)` — removes `trip:<id>:chat:rider`, `trip:<id>:chat:driver`, `rider:<riderID>:active_driver`, `driver:<driverID>:active_rider` |
| 6 | ws-gateway cancelConsumer | RCM `SendMessage(riderID)` | `{ type: "trip.event.cancelled", topic: "trip:<id>", data: { tripID } }` → rider WS |
| 7 | ws-gateway cancelConsumer | RCM `SendMessage(driverID)` | Same frame → driver WS (skipped if no driver was assigned) |
| 8 | Rider frontend | WS send (×2) | `ws.topic.unsubscribe trip:<id>` and `ws.topic.unsubscribe trip:<id>:chat` — leaves in-memory room maps on gateway |
| 9 | Rider frontend | Redux dispatch | `resetTrip()` + `setTripStatus("trip.event.cancelled")` — clears UI state |
| 10 | Driver frontend | WS send (×2) | `ws.topic.unsubscribe trip:<id>` and `ws.topic.unsubscribe trip:<id>:chat` |
| 11 | Driver frontend | Redux dispatch | `resetTrip()` + `setTripStatus("trip.event.cancelled")` — hides active trip UI; driver returns to available state |
| 12 | driver-service locationConsumer | — | On next GPS frame, `GetActiveRider(driverID)` returns empty → location relay stops automatically |

**Note:** Steps 6–7 use `connManager.SendMessage` which fans out cross-pod via Redis Pub/Sub if the target socket is on a different ws-gateway replica.

---

### 11. JWT Expiry During an Active Session

1. Rider's access token expires mid-session.
2. Frontend Axios interceptor receives a `401` response on any REST call.
3. Interceptor pauses the request queue, calls `POST /auth/refresh` with the refresh token.
4. On success: updates Redux `authSlice` with new tokens, replays queued requests.
5. On failure (refresh token also expired): `dispatch(logout())` → clears Redux state → redirect to login.
6. For the WebSocket: the next WS connection attempt (after a disconnect) uses the new token from Redux state. A mid-session WS connection that drops with code 1006 triggers the `isTokenExpired` pre-flight check; if expired, `dispatch(logout())` fires before any reconnect is attempted.
