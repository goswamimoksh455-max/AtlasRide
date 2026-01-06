# 📍 AtlasRide: Execution Plan

## 1. Core State Definitions
Standardize driver states to ensure the Matching Engine and Ingestion Service speak the same language.

| Status     | Description |
| :---       | :--- |
| **IDLE** | Available for ride requests. Tracked in Geo-Index. |
| **MATCHING** | Temporarily locked. Currently being offered a ride. |
| **BUSY** | Actively on a trip. Unavailable for new requests. |

---

## 2. In-Memory Repository (Thread-Safe)
The "Source of Truth" for driver locations and availability.

### Key Responsibilities:
* **`Upsert(driver)`**: Handles high-frequency location updates.
* **`UpdateStatus(id, old, new)`**: Implements **Atomic Compare-And-Swap (CAS)** logic.
    * *Example:* Only move to `MATCHING` if current state is exactly `IDLE`.
* **`TTL Eviction Loop`**: Background goroutine that removes `IDLE` drivers who haven't pinged in >N seconds.

---

## 3. Service Boundaries

### A. Driver Ingestion Service (The Hot Path)
* **Goal**: Low-latency, high-throughput location processing.
* **Logic**:
    1. Validate incoming packet (Sanity Check).
    2. Check Monotonicity (Packet time vs. Memory time).
    3. Update location in Repository.
    4. **Constraint**: Never modifies `Status`.

### B. Matching Engine (Business Logic)
* **Goal**: Efficiently pair riders with available drivers.
* **Logic**:
    1. Query Repository for nearby `IDLE` drivers.
    2. Atomically transition selected driver: `IDLE` → `MATCHING`.
    3. If transition fails (race condition), skip to next driver.
    4. Dispatch notification to Driver App.

### C. Ride Lifecycle Handler
* **Acceptance**: Driver clicks accept → Transition `MATCHING` → `BUSY`.
* **Rejection/Timeout**: Transition `MATCHING` → `IDLE` (returns driver to the pool).
* **Completion**: Trip ends → Transition `BUSY` → `IDLE`.

---

## 4. Technical Requirements for 9.5/10 Score
* [ ] **Concurrency**: Use `sync.RWMutex` or `Sharded Maps` to prevent race conditions during updates.
* [ ] **Monotonicity**: Ensure late-arriving network packets don't overwrite newer location data.
* [ ] **Atomic Transitions**: Status changes must be atomic to prevent "Double Booking" a driver.
* [ ] **Observability**: Expose metrics for `active_drivers` and `eviction_count`.
* [ ] **Clean Shutdown**: Ensure the Eviction Loop stops gracefully when the server terminates.

---

## 5. Future Scalability (V2)
* Replace flat-map iteration with **Geospatial Indexing** (Uber H3 or S2).
* Introduce **Redis** as a persistent backup for the in-memory state.





##### BUGS NEED TO BE SOLVED:

# 1 - handled in spatial - Update()
- The "Busy Driver" Eviction Bug
This is the most common mistake in ride-sharing systems.

-- The Scenario: A driver is BUSY on a 20-minute ride. Because they are driving, the app might stop sending "Available" heartbeats, or the server logic ignores them because the driver isn't "Idle."

-- The Error: The TTL Loop sees no heartbeats for 60 seconds and deletes the driver from memory.

-- The Consequence: The system "forgets" the driver is on a ride. When the ride ends, the driver can't "Complete" the trip because their record is gone.

-- 9.5/10 Fix: Your Eviction Loop must check the status. Never evict a driver if their status is BUSY or MATCHING, even if their heartbeat is old.

# 2 
- Memory Fragmentation (Churn)
If you have 10,000 drivers constantly connecting and disconnecting:

-- Churn: High frequency of make() and delete() on your Go map.

-- TTL: Constantly clearing out old entries.

-- The Risk: In Go, deleting from a map doesn't always shrink the memory immediately. If you have massive "Churn + TTL," your RAM usage might keep climbing even though the number of drivers stays the same.


# 3
Redis backed OfferGroupStore ( distributed Coordination)
+ Redis is our distributed lock + TTL coordination
+ Setting the expirey rathor TTL swepper

### feature 
10/10 System: Take those top 3 and call an external Routing API (like OSRM or Google Maps) to find the "Road Distance" (accounting for one-way streets and traffic). Haversine is your fast filter before doing expensive routing calls.

- In the h3Index : need to add the HeatMap Worker : that periodicaly computes density, stores Optimal Resolution in Redis
res9 -> res8 -> res7 (for sparse areas)

++> Fairness shuffling : rand.Shuffle(len(results), ...)

++> Metrics : avg_rings_used, p99_match_latency

++> ETA

