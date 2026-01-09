# AtlasRider System Architecture

> **Last Updated**: January 8, 2026
> **Author**: Moksh Goswami
> **Status**: Phase 1 - Foundation & Geo-Sharding

---

## 1. Overview

AtlasRider is a **geo-sharded ride-matching engine** designed to handle 10,000+ requests/second with sub-5ms latency.

**Core Problem**: Match riders with nearby available drivers in real-time at global scale.

**Key Constraints**:
- **Latency**: <5ms for matching (p99)
- **Throughput**: 10k location updates/sec per shard
- **Correctness**: No double-booking (atomic assignment)
- **Scale**: Support 1M+ drivers globally

---

## 2. High-Level Architecture

```
┌─────────────────────────────┐
│   Driver / Rider Clients    │
│   (Mobile Apps)             │
└──────────────┬──────────────┘
               │ GPS Updates (1-5s interval)
               │ Match Requests
               ▼
┌─────────────────────────────┐
│   API Gateway (Go)          │
│   - JWT Authentication      │
│   - Rate Limiting           │
│   - Request Validation      │
└──────────────┬──────────────┘
               │
               ▼
┌─────────────────────────────┐
│   Geo Router                │
│   (H3 Cell ID → Shard)      │
│   Resolution 5 (~8km)       │
└──────────────┬──────────────┘
               │
        ┌──────┴──────┐
        ▼             ▼
┌─────────────┐ ┌─────────────┐
│ Shard: NYC  │ │ Shard: SF   │
│ (Hot Path)  │ │ (Hot Path)  │
│             │ │             │
│ ┌─────────┐ │ │ ┌─────────┐ │
│ │Drivers  │ │ │ │Drivers  │ │
│ │(In-Mem) │ │ │ │(In-Mem) │ │
│ └─────────┘ │ │ └─────────┘ │
│ ┌─────────┐ │ │ ┌─────────┐ │
│ │H3 Index │ │ │ │H3 Index │ │
│ └─────────┘ │ │ └─────────┘ │
│ ┌─────────┐ │ │ ┌─────────┐ │
│ │Matching │ │ │ │Matching │ │
│ │Engine   │ │ │ │Engine   │ │
│ └─────────┘ │ │ └─────────┘ │
└──────┬──────┘ └──────┬──────┘
       │               │
       └───────┬───────┘
               ▼
       ┌───────────────┐
       │  Kafka Events │
       │  (Async)      │
       └───────┬───────┘
               ▼
       ┌───────────────┐
       │  Cold Path    │
       │               │
       │ - PostgreSQL  │
       │ - ClickHouse  │
       │ - Analytics   │
       └───────────────┘
```

---

## 3. Component Details

### 3.1 Geo Router

**Purpose**: Route requests to the correct geographic shard based on location.

**Algorithm**:
1. Extract lat/lng from request
2. Convert to H3 cell (Resolution 5 = city-level, ~8km edge)
3. Map cell → shard
4. Forward request

**Example**:
```go
// San Francisco: lat=37.7749, lng=-122.4194
cell := h3.LatLngToCell(h3.LatLng{Lat: 37.7749, Lng: -122.4194}, 5)
// cell = 85283473fffffff
shard := router.GetShard(cell) // → Shard: SF
```

**Edge Cases**:
- **Cross-shard queries**: Rider on border needs drivers from 2 shards
- **Solution**: Query neighbor shards using k-ring expansion

**Performance**:
- Routing decision: <100μs
- Stateless (can scale horizontally)

---

### 3.2 Shard (Hot Path)

**Purpose**: Maintain real-time driver state for a geographic region.

**Components**:
1. **Driver Map**: `map[driverID]Driver` (in-memory)
2. **H3 Spatial Index**: `map[h3Cell][]driverID` (Resolution 9 = ~600m)
3. **Matching Engine**: Finds nearest available driver
4. **TTL Eviction**: Removes silent drivers after 30s

**Data Flow**:
```
Driver Update → Shard → Update Map → Update H3 Index
Match Request → Shard → Query H3 Index → Score → Return Driver
```

**Guarantees**:
- **Latency**: <5ms (all in RAM)
- **Consistency**: Eventual (drivers may lag by heartbeat interval)
- **Throughput**: 10k updates/sec per shard

**Limitations**:
- **Volatile**: Data lost on crash (mitigated by heartbeats)
- **Memory**: ~1KB per driver = 100MB for 100k drivers

---

### 3.3 H3 Spatial Index

**Purpose**: Fast nearest-neighbor queries using Uber's H3 hexagonal indexing.

**Data Structure**:
```go
type H3Index struct {
    cells      map[h3.Cell][]string // cell → driver IDs
    resolution int                  // 9 = ~600m hexagons
    mu         sync.RWMutex
}
```

**Algorithm** (Adaptive K-Ring Search):
```
1. Convert rider location to H3 cell (Resolution 9)
2. Start with k=1 (7 cells: center + 6 neighbors)
3. Get all drivers in those cells
4. If < 10 drivers found, expand to k=2 (19 cells)
5. Repeat until k=5 or enough drivers found
6. Return candidates
```

**Complexity**:
- **Best Case**: O(1) - drivers in immediate cell
- **Average Case**: O(k²) where k=1-2 (7-19 cells)
- **Worst Case**: O(k²) where k=5 (91 cells)

**Benchmark**: 50μs for k=2 with 100 drivers

**Why H3 over GeoHash?**
- Uniform neighbors (6 vs 8 for rectangles)
- Bitwise operations (faster than string manipulation)
- No edge discontinuities

---

### 3.4 Matching Engine

**Purpose**: Select the best driver for a rider using multi-factor scoring.

**Algorithm**:
1. Get candidate drivers from H3 Index (adaptive search)
2. Filter: Only `AVAILABLE` status, no active intent
3. Score each driver:
   - **Distance** (40%): Exponential decay `exp(-dist/2000m)`
   - **ETA** (30%): From OSRM or Haversine fallback
   - **Rating** (15%): Normalized 0-5 → 0-1
   - **Acceptance Rate** (15%): Historical acceptance %
4. Sort by total score (descending)
5. Atomic assignment (lock driver with intent)

**Correctness** (Phase 1):
- **Problem**: Two riders request same driver simultaneously
- **Current**: Mutex-based locking (single-node)
- **Phase 2**: Redis Lua scripts (distributed locking)

**Performance**:
- Scoring 10 drivers: <1ms
- Total match time: <5ms (p99)

---

## 4. Data Flow Examples

### 4.1 Driver Location Update

```
1. Driver app sends GPS: POST /api/v1/drivers/{id}/location
   Body: {"lat": 37.7749, "lng": -122.4194, "status": "AVAILABLE"}

2. API Gateway:
   - Validates JWT token
   - Checks rate limit (max 1 update/sec per driver)

3. Geo Router:
   - Calculates H3 cell (Res 5): 85283473fffffff
   - Routes to SF Shard

4. SF Shard:
   - Updates driver map: drivers[id].location = (37.7749, -122.4194)
   - Updates driver map: drivers[id].updatedAt = now()
   - Updates H3 Index: h3Index.Update(id, lat, lng)
   - Resets TTL: drivers[id].lastHeartbeat = now()

5. Response: 200 OK (total: 3ms)
```

### 4.2 Match Request

```
1. Rider app requests: POST /api/v1/match
   Body: {
     "riderId": "r123",
     "pickup": {"lat": 37.7749, "lng": -122.4194},
     "dropoff": {"lat": 37.8044, "lng": -122.2712}
   }

2. API Gateway validates

3. Geo Router → SF Shard

4. SF Shard:
   a. h3Index.AdaptiveSearch(37.7749, -122.4194, 10)
      → Returns 10 candidate drivers
   
   b. matchingEngine.Filter(candidates)
      → Removes BUSY drivers, drivers with active intents
   
   c. matchingEngine.Score(filtered)
      → Calculates multi-factor scores
   
   d. matchingEngine.Assign(bestDriver)
      → Sets driver.Intent = {riderId: "r123", expiresAt: now()+30s}

5. Emit event (async): 
   DriverMatchedEvent{
     riderId: "r123",
     driverId: "d456",
     matchedAt: now()
   }

6. Response: 200 OK
   Body: {
     "driverId": "d456",
     "eta": "3 min",
     "distance": "1.2 km"
   }
   (total: 4ms)
```

---

## 5. Design Decisions & Trade-offs

### 5.1 In-Memory vs Database

**Decision**: Store driver locations in RAM, not database.

**Rationale**:
- RAM access: 100ns
- SSD access: 10ms
- **100,000x faster**

**Trade-off**: Volatility (data lost on crash)

**Mitigation**: 
- Drivers send heartbeats every 5s
- State recovers in <30s after crash
- Critical data (trips, payments) goes to PostgreSQL via Kafka

**When this breaks**:
- If we need audit logs ("Where was driver X at 3:42 PM?"), we add event logging to Kafka

---

### 5.2 H3 vs GeoHash

**Decision**: Use H3 hexagonal indexing.

**Rationale**:
- **Uniform neighbors**: 6 vs 8 for rectangles
- **Bitwise operations**: Faster than string manipulation
- **Uber-proven**: Used in production at global scale

**Trade-off**: 
- Less common (steeper learning curve)
- Not human-readable (`8928308280fffff` vs `9q8yy`)

**Benchmark**: 10x faster neighbor lookups (50μs vs 500μs)

---

### 5.3 Geo-Sharding vs Random Sharding

**Decision**: Shard by geography (H3 cells), not random hash.

**Rationale**:
- **Locality**: NYC drivers only match NYC riders (no cross-region queries)
- **Scale**: Add shards for new cities independently
- **Failure isolation**: SF shard crash doesn't affect NYC

**Trade-off**: 
- Cross-shard queries for riders on borders
- Uneven load (NYC has more drivers than rural areas)

**Mitigation**:
- Neighbor shard expansion for border cases
- Sub-sharding for hot cities (split NYC into 4 shards)

---

### 5.4 Hot Path vs Cold Path

**Decision**: Separate real-time matching from persistence/analytics.

**Hot Path** (Matching):
- In-memory, <5ms latency
- Volatile, eventual consistency

**Cold Path** (Persistence):
- PostgreSQL, 100ms+ latency
- Durable, strong consistency

**Bridge**: Kafka event stream (async)

**Why?**
- Hot path can't wait for DB writes (too slow)
- Cold path doesn't need sub-5ms latency

---

## 6. Scaling Strategy

### Current (Phase 1): Single Region
- **1 Geo Router** (stateless, can add more)
- **2-5 Shards** (NYC, SF, LA, Chicago, Boston)
- **10k drivers per shard** = 50k total
- **1k matches/sec** total throughput

### Phase 2: Multi-Region
- **Geo Router per region** (US-East, US-West, EU)
- **10-20 shards per region**
- **100k drivers per region** = 300k total
- **10k matches/sec** total throughput

### Phase 3: Global
- **Global load balancer** (GeoDNS)
- **100+ shards** across 10 regions
- **1M+ drivers**
- **100k matches/sec**

**Bottlenecks**:
- **Geo Router**: Can handle 100k req/sec (not a bottleneck yet)
- **Shard**: Limited to 10k updates/sec → Split into sub-shards
- **Network**: Cross-region latency → Deploy regionally

---

## 7. Failure Modes & Recovery

### 7.1 Shard Crash

**Impact**: Drivers in that shard disappear from matching.

**Detection**: Health check fails, load balancer removes shard.

**Recovery**:
1. Drivers detect connection loss (heartbeat timeout)
2. Reconnect to Geo Router
3. Router assigns to new/recovered shard
4. State recovers in <30s (drivers re-send locations)

**Data Loss**: Last 5-30s of location updates (acceptable)

---

### 7.2 Geo Router Crash

**Impact**: No new requests routed.

**Detection**: Load balancer health check.

**Recovery**:
- Deploy multiple Geo Routers (stateless, easy to replicate)
- Load balancer fails over to healthy router
- **Downtime**: <1s

---

### 7.3 Network Partition

**Impact**: Some drivers can't reach their shard.

**Detection**: TTL eviction (drivers silent for 30s).

**Recovery**:
- Silent drivers auto-removed from matching
- Riders see "No drivers available" in that area
- When network recovers, drivers reconnect

**User Impact**: Temporary service degradation in affected area

---

### 7.4 Database (Cold Path) Failure

**Impact**: Can't persist trip history, can't query analytics.

**Hot Path Impact**: **NONE** (matching continues normally)

**Recovery**:
- Kafka retains events for 7 days
- Replay events to rebuild database
- **Data Loss**: None (events are durable)

---

## 8. Monitoring & Observability

### Metrics (Prometheus)

**Hot Path**:
- `driver_updates_per_sec` (by shard)
- `match_requests_per_sec` (by shard)
- `match_latency_seconds` (p50, p95, p99)
- `active_drivers_count` (by shard, by status)
- `match_success_rate` (%)

**System**:
- `shard_cpu_usage` (%)
- `shard_memory_usage` (MB)
- `geo_router_requests_per_sec`

### Alerts

**Critical**:
- p99 latency > 50ms for 5 minutes
- Match success rate < 80% for 5 minutes
- Shard CPU > 90% for 2 minutes

**Warning**:
- Active drivers dropped by 20% in 1 minute
- TTL evictions > 100/min (network issues?)

### Dashboards (Grafana)

1. **System Overview**: Active drivers, match rate, latency
2. **Shard Health**: Per-shard CPU, memory, throughput
3. **Geo Distribution**: Heatmap of drivers by H3 cell

---

## 9. Security Considerations

### Phase 1 (Current)
- ❌ No authentication (TODO: Phase 2)
- ❌ No rate limiting (TODO: Phase 2)
- ✅ Input validation (lat/lng bounds)

### Phase 2 (Planned)
- ✅ JWT authentication
- ✅ Rate limiting (1 update/sec per driver)
- ✅ HTTPS only
- ✅ API key for mobile apps

### Phase 3 (Future)
- ✅ OAuth 2.0
- ✅ Encryption at rest
- ✅ Audit logs

---

## 10. Future Enhancements

### Phase 2: Distributed Systems
- [ ] Redis Lua scripts for distributed locking
- [ ] Saga pattern for multi-step workflows
- [ ] Circuit breakers for fault tolerance

### Phase 3: Event-Driven
- [ ] Kafka event sourcing
- [ ] CQRS (separate read/write models)
- [ ] Event replay for debugging

### Phase 4: Production
- [ ] Kubernetes deployment
- [ ] Chaos engineering tests
- [ ] Multi-region failover

### Advanced Features
- [ ] ML-based demand prediction
- [ ] Dynamic surge pricing
- [ ] Ride pooling optimization

---

## 11. References

- [Uber H3 Documentation](https://h3geo.org/)
- [Uber Engineering Blog: H3](https://eng.uber.com/h3/)
- [Designing Data-Intensive Applications](https://dataintensive.net/) (Chapter 6: Partitioning)
- [Google SRE Book](https://sre.google/sre-book/table-of-contents/)

---

**This is a living document. Update as the system evolves.**
