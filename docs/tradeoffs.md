# Trade-offs Documentation
## Spatial Indexing: H3 vs GeoHash
### Decision: We chose H3
**Why H3?**
1. **Uniform Neighbors**: Every hexagon has 6 neighbors. GeoHash rectangles have 8, but corner neighbors are farther.
2. **Bitwise Operations**: H3 cell IDs are 64-bit integers. Finding neighbors is `O(1)` bitwise ops, not string manipulation.
3. **Industry Validation**: Uber uses H3 for their matching engine (DISCO). Proven at scale.
**Trade-offs**:
- **Learning Curve**: H3 is less common than GeoHash. Fewer tutorials.
- **Not Human-Readable**: `8928308280fffff` vs `9q8yy`.
- **Mitigation**: We provide conversion utilities and documentation.
**What we gave up**:
- Redis `GEOSEARCH` (uses GeoHash internally). We build our own in-memory index.
**What we gained**:
- 10x faster neighbor lookups (benchmarked: 50μs vs 500μs).
- Uniform coverage (no edge distortion).
**When this breaks**:
- If we need to integrate with Redis Geo commands, we'd need a GeoHash adapter.


## In-Memory vs Database for Driver Locations
### Decision: In-Memory (RAM)
**Why?**
- **Latency**: RAM access is 100,000x faster than SSD (100ns vs 10ms).
- **Throughput**: Can handle 10k updates/sec without batching.
**Trade-offs**:
-  **Volatility**: If server crashes, we lose driver locations.
- **Mitigation**: Drivers send heartbeats every 5s. State recovers in <30s.
**What we gave up**:
- Durability. We can't "replay" driver movements from 1 hour ago.
**What we gained**:
- 100x lower latency. Matching happens in <5ms instead of 500ms.
**When this breaks**:
- If we need audit logs (e.g., "Where was driver X at 3:42 PM?"), we'd need to add event logging to Kafka.



## Hot Path (Real-Time Matching)
**Components**:
1. **Geo Router**: Routes requests to shards based on H3 cell
2. **Shard**: In-memory driver state + spatial index
3. **Matching Engine**: Finds nearest available driver
**Guarantees**:
- **Latency**: <5ms p99
- **Throughput**: 10k updates/sec per shard
- **Consistency**: Eventual (drivers may appear/disappear)
**Trade-off**: Data is volatile. If shard crashes, drivers re-send heartbeats to recover.
## Cold Path (Persistence & Analytics)
**Components**:
1. **Kafka**: Event stream (DriverMatched, TripStarted, etc.)
2. **PostgreSQL**: Trip history, user profiles
3. **ClickHouse**: Analytics (surge pricing, demand heatmaps)
**Guarantees**:
- **Durability**: All events persisted
- **Consistency**: Strong (ACID transactions)
**Trade-off**: Higher latency (100ms+), but acceptable for non-real-time queries.