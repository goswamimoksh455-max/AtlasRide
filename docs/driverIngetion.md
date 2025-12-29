# Day 3 — Driver Ingestion & Repository Design (System Design Perspective)

## Purpose of Day 3

Day 3 focuses on building a **correct and scalable driver location ingestion pipeline**, which is one of the most critical components in any real-time mobility system (Uber, Ola, Lyft, Rapido).

The goal is **not feature richness**, but **correctness under real-world conditions** such as:

* Out-of-order network packets
* Client–server clock skew
* Driver app crashes or disconnects
* High-frequency updates

This day establishes the **data correctness foundation** on which matching, sharding, and scaling will later rely.

---

## System Design Context

In a real ride-hailing system:

* Drivers send GPS updates every 1–3 seconds
* Updates may arrive late or out of order
* Some drivers silently disappear (network loss, app killed)

If ingestion is incorrect:

* Ghost drivers appear
* Stale locations overwrite fresh ones
* Matching becomes incorrect

Day 3 solves these problems at the **ingestion and storage boundary**.

---

## High-Level Architecture (Day 3 Scope)

```
Driver App
   ↓
Driver Ingestion Service
   ↓
Driver Repository (authoritative)
   ↓
In-Memory Store
   ↓
Spatial Index (mirror)
```

Key architectural decision:

* **Repository is the single source of truth for correctness**
* Services are orchestration-only

---

## Domain Model

### Driver

```go
type Driver struct {
    ID        string
    Location  Location
    Status    DriverStatus
    UpdatedAt time.Time
}
```

Design notes:

* Represents a long-lived entity
* `UpdatedAt` is **event time** (when the driver sent the update)
* No persistence or spatial logic inside the domain

---

### Location

```go
type Location struct {
    Lat float64
    Lng float64
}
```

Design notes:

* Immutable value object
* No distance or spatial computation here

---

### DriverStatus

```go
const (
    DriverIdle DriverStatus = iota
    DriverMatching
    DriverBusy
)
```

Design notes:

* Enum prevents illegal states
* Enables deterministic transitions later

---

## Repository Layer (Core of Day 3)

### Why the Repository Is Critical

In distributed systems:

* Network delivery is not ordered
* Client clocks are unreliable
* Multiple ingestion paths may exist

Therefore:

> The repository must be the final authority for accepting or rejecting updates.

---

### Internal Storage Model

```go
type driverEntry struct {
    driver     domain.Driver
    lastSeenAt time.Time
}
```

Two timestamps are deliberately separated:

| Timestamp          | Purpose                       |
| ------------------ | ----------------------------- |
| `driver.UpdatedAt` | Event ordering (monotonicity) |
| `lastSeenAt`       | TTL eviction (server time)    |

This separation avoids subtle distributed systems bugs.

---

### Upsert Logic (Correctness Guarantees)

Responsibilities of `Upsert`:

1. **Late packet rejection**
   Older updates must not overwrite newer state.

2. **Clock skew protection**
   Updates from the future (client clock ahead of server) are rejected with a small tolerance.

3. **TTL refresh**
   Server-side timestamp is updated only for accepted packets.

Design rationale:

* Centralizes all ordering guarantees
* Ensures correctness even with multiple services

---

### Why Services Do Not Enforce Ordering

If ordering logic is spread across services:

* Behavior diverges during scaling
* Bugs appear under concurrency

Centralizing this logic in the repository ensures:

* Deterministic behavior
* Easier migration to Redis or other stores

---

## Driver Ingestion Service

### Responsibility

```go
type DriverIngestionService struct {
    repo  DriverRepository
    index SpatialIndex
}
```

Service responsibilities:

* Accept incoming intent
* Construct domain objects
* Delegate correctness to repository
* Mirror accepted state to spatial index

Explicit non-responsibilities:

* Ordering logic
* TTL logic
* Concurrency control

This keeps services stateless and scalable.

---

### UpdateLocation Flow

Steps performed:

1. Build `Driver` object
2. Assign event timestamp
3. Call `repo.Upsert`
4. Update spatial index

The service does not know whether the update was accepted or rejected.

---

## TTL Eviction (Prepared, Not Automatic)

### Problem

Drivers may silently disconnect, causing stale data to persist.

### Design

* Repository exposes `Expired(ttl)`
* A separate service periodically removes expired drivers
* Repository remains passive and deterministic

This design allows easy migration to Redis TTL later.

---

## Failure Modes Addressed on Day 3

* Out-of-order packet arrival
* Late GPS updates
* Client clock skew
* Ghost drivers (prepared via TTL)

---

## What Is Intentionally Out of Scope

* Driver–rider matching
* Spatial search optimization
* Sharding
* Redis
* Microservices

These build on top of Day 3 guarantees.



## Interview Explanation (Concise)

> “We designed the driver ingestion pipeline so that the repository enforces monotonic updates and clock-skew protection. Services remain stateless orchestrators, which allows correctness to hold even as the system scales.”


## Status

Day 3 establishes a correctness-first ingestion layer.
All subsequent system components assume these guarantees.

Next step: **Day 4 — Matching Engine Design**
