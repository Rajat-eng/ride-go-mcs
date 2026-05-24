# Messaging & Real-Time Communication Architecture

## Table of Contents

1. [System Overview](#system-overview)
2. [Transport Layers](#transport-layers)
3. [WebSocket Layer](#websocket-layer)
4. [RabbitMQ Layer](#rabbitmq-layer)
5. [Redis Layer](#redis-layer)
6. [Complete Data Flows](#complete-data-flows)
   - [Trip Creation](#flow-1-trip-creation)
   - [Driver Assignment](#flow-2-driver-assignment)
   - [Driver Decline / Retry](#flow-3-driver-decline--retry)
   - [Payment Session](#flow-4-payment-session)
   - [Real-Time Driver Location](#flow-5-real-time-driver-location)
   - [In-Trip Chat](#flow-6-in-trip-chat)
7. [WS Topic Multiplexing](#ws-topic-multiplexing)
8. [Message Envelopes & Payloads](#message-envelopes--payloads)
9. [Exchange & Queue Bindings](#exchange--queue-bindings)
10. [Dead-Letter Queue](#dead-letter-queue)
11. [Cross-Pod Delivery via Redis Pub/Sub](#cross-pod-delivery-via-redis-pubsub)

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
The API Gateway is the only service that has a WebSocket connection to the frontend.

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

### Endpoints

| Path | Consumer | Auth |
|---|---|---|
| `GET /riders?token=<jwt>` | Riders | JWT in query param |
| `GET /drivers?token=<jwt>&packageSlug=<slug>` | Drivers | JWT + package slug |

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

### Client → Server message types

| `type` | Sender | `data` payload |
|---|---|---|
| `driver.cmd.location` | Driver | `{ location: { latitude, longitude } }` |
| `driver.cmd.trip_accept` | Driver | `{ tripID: string, riderID: string }` |
| `driver.cmd.trip_decline` | Driver | `{ tripID: string, riderID: string }` |
| `driver.cmd.register` | Driver | (handled internally, not sent from frontend) |
| `chat.message.send` | Rider or Driver | `{ tripID: string, text: string, messageID?: string }` |
| `ws.topic.subscribe` | Rider or Driver | `{ topic: string }` |
| `ws.topic.unsubscribe` | Rider or Driver | `{ topic: string }` |

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
| `chat.message.received` | Rider or Driver | `{ tripID, senderID, text, sentAt, messageID? }` | `trip:<id>` |

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
| `driver:<userID>:events` | Pub/Sub channel | — | Cross-pod WS delivery |
| `trip:<tripID>:chat:rider` | String | 2 h | Rider userID for chat authorisation |
| `trip:<tripID>:chat:driver` | String | 2 h | Driver userID for chat authorisation |
| `driver:<driverID>:active_rider` | String | — | Active rider linked to a driver (used for location relay) |

### Per-Connection Topic Registry (in-memory, `RedisConnectionManager`)

```
topicsByUser: map[userID] → set[topic]
```

Stored in `RedisConnectionManager.topicsByUser` (Go `map[string]map[string]struct{}` with a mutex).  
Populated via `SubscribeTopic(userID, topic)` called from:

- Frontend `ws.topic.subscribe` frame processed by the gateway read loop
- Auto-subscription on `DriverCmdTripAccept` (server-side, no round-trip required for driver)

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
     - sends ws.topic.subscribe { topic: "trip:<id>" }   ← subscribe for future scoped events
                │
     Driver clicks Accept
                │
     WS frame from driver:
     { type: "driver.cmd.trip_accept",
       data: { tripID: "<id>", riderID: "<id>" } }
                │
     API Gateway:
     1. connManager.SetTripChatPair(tripID, riderID, driverID, 2h)   ← Redis KV
     2. connManager.SubscribeTopic(driverID, "trip:<id>")             ← auto-subscribe server-side
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
    data: Trip }                      ← delivered only if rider subscribed to topic

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
       topic: "trip:<id>",
       data: { tripID, senderID: <riderID>, text, sentAt, messageID? } }
  3. SendMessage(riderID, frame)   ← echo to sender
  4. SendMessage(driverID, frame)  ← relay to peer
  │
  Both recipients see the message if they are subscribed to topic "trip:<id>"

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
      ├─ Driver accepts ──▶ Gateway auto-subscribes driver server-side (no round-trip needed)
      │
      │  All subsequent topic-tagged frames are filtered:
      │    topic="" → always delivered (system messages)
      │    topic="trip:X" → delivered only if subscribed to "trip:X"
      │
Client disconnects ──▶ topic registry cleared from memory
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
- `chat.message.received`

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

### Queue → Consumer Mapping

| Queue | Consumer service |
|---|---|
| `notify_trip_created` | API Gateway (QueueConsumer → rider WS) |
| `find_available_drivers` | driver-service (`tripConsumer`) |
| `driver_cmd_trip_request` | API Gateway (QueueConsumer → driver WS) |
| `driver_trip_response` | trip-service (`driverConsumer`) |
| `notify_driver_no_drivers_found` | API Gateway (QueueConsumer → rider WS) |
| `notify_driver_assign` | API Gateway (QueueConsumer → rider WS) |
| `driver_trip_assigned` | driver-service (`tripAssignedConsumer`) |
| `driver_location_update` | driver-service (`locationConsumer`) |
| `notify_rider_driver_location` | API Gateway (QueueConsumer → rider WS) |
| `payment_trip_response` | payment-service (`TripConsumer`) |
| `notify_payment_session_created` | API Gateway (QueueConsumer → rider WS) |
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
            └─ not local → rdb.Publish("driver:<userID>:events", json(msg))
                                │
                         Redis Pub/Sub
                                │
                         Pod that has the WS connection
                         subscribed goroutine receives msg
                                │
                         topic filter check (isTopicAllowed)
                                │
                         localCM.SendMessage(userID, msg) → WS frame
```

Each `Add(userID, conn)` call subscribes that pod's goroutine to `driver:<userID>:events`.  
Each `Remove(userID)` unsubscribes and clears the topic registry for that user.
