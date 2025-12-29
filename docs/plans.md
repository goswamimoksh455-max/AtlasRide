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