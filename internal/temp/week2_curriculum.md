# 🚀 Week 2: Advanced Matching & Redis Integration
## Phase 1 - FAANG-Ready AtlasRider Project

---

## 📋 Week 1 Recap - What You've Built

Excellent work on Week 1! You've implemented:
- ✅ **Domain Models**: Driver FSM, Rider, Location, OfferGroup
- ✅ **In-Memory Repository**: Thread-safe with CAS operations, TTL eviction
- ✅ **Basic Matching Engine**: Geo-spatial search using H3
- ✅ **Event System**: Async dispatcher for offer notifications
- ✅ **Concurrency Primitives**: Intent locking, state transitions

**Systems Concepts Demonstrated:**
- State machines (Idle → Matching → Busy)
- Race condition prevention (CAS, mutex)
- Event-time vs. server-time (monotonicity)
- TTL-based eviction

---

## 🎯 Week 2 Learning Objectives

This week focuses on **distributed systems fundamentals** and **production-grade matching**:

1. **Redis Integration** (Day 1-2)
   - Learn distributed state management
   - Implement Redis-backed OfferGroupStore
   - Master Lua scripting for atomic operations

2. **Advanced Scoring Algorithm** (Day 3-4)
   - Multi-factor scoring (distance, surge, driver rating)
   - Fairness mechanisms (prevent starvation)
   - Performance optimization (P99 latency \< 50ms)

3. **Concurrency Testing** (Day 5)
   - Race condition testing
   - Load testing (10K concurrent requests)
   - Chaos engineering basics

---

## 📚 Day-by-Day Breakdown

### **Day 1: Redis Fundamentals for Distributed Systems**

#### 🧠 Theory (30 minutes)
**Why Redis for AtlasRider?**

When multiple matching servers run simultaneously (horizontal scaling), they need **shared coordination**:

| Problem | In-Memory Solution | Redis Solution |
|:--------|:-------------------|:---------------|
| **Duplicate Offers** | Server A and B both offer same ride to Driver X | Redis SET with `NX` (set if not exists) |
| **Stale Data** | Server crashes, offers never expire | Redis TTL (automatic key expiration) |
| **Consistency** | No cross-server visibility | Single source of truth |

**Key Redis Concepts:**
1. **Atomic Operations**: `SET key value EX 30 NX` → Set with 30s expiry, only if key doesn't exist
2. **Lua Scripts**: Multi-command atomicity (like database transactions)
3. **PubSub**: Notify all servers when an offer is accepted

---

#### 💻 Assignment: Redis-Backed OfferGroupStore

**Your Task:** Implement `internal/offer/redis_offer_group_store.go`

**File Structure:**
```go
package offer

import (
    "context"
    "github.com/redis/go-redis/v9"
    "time"
)

type RedisOfferGroupStore struct {
    client *redis.Client
    ttl    time.Duration
}

func NewRedisOfferGroupStore(client *redis.Client, ttl time.Duration) *RedisOfferGroupStore {
    return &RedisOfferGroupStore{client: client, ttl: ttl}
}
```

**Methods to Implement:**

1. **`CreateOfferGroup(ctx context.Context, riderID string, driverIDs []string) error`**
   - Key format: `offer:rider:{riderID}`
   - Value: JSON-encoded `domain.OfferGroup`
   - Use `SET` with `NX` flag (create only if not exists)
   - Set TTL to 30 seconds
   - **Edge Case**: What if offer already exists? Return specific error

2. **`GetOfferGroup(ctx context.Context, riderID string) (*domain.OfferGroup, error)`**
   - Use `GET` to fetch offer
   - Parse JSON
   - **Edge Case**: Key doesn't exist → return `nil, ErrOfferNotFound`

3. **`LockDriver(ctx context.Context, riderID, driverID string) (bool, error)`**
   - **Critical**: Use Lua script for atomicity
   - Check if driver is still in the offer list
   - Check if no other driver is locked yet
   - Mark driver as locked
   - **Return**: `true` if lock acquired, `false` if another driver already locked

**Lua Script Example:**
```lua
-- LockDriver.lua
local offerKey = KEYS[1]  -- "offer:rider:R123"
local driverID = ARGV[1]  -- "D456"

local offer = redis.call('GET', offerKey)
if not offer then
    return {err = "offer not found"}
end

local data = cjson.decode(offer)

-- Check if driver is in offer list
local found = false
for _, id in ipairs(data.driver_ids) do
    if id == driverID then
        found = true
        break
    end
end

if not found then
    return {err = "driver not in offer"}
end

-- Check if already locked
if data.locked_driver_id then
    return 0  -- Lock failed
end

-- Acquire lock
data.locked_driver_id = driverID
redis.call('SET', offerKey, cjson.encode(data), 'KEEPTTL')
return 1  -- Lock acquired
```

**How to Load Lua Script in Go:**
```go
const lockDriverScript = `
    -- Your Lua script here
`

var lockDriverSha string

func (s *RedisOfferGroupStore) loadScripts(ctx context.Context) error {
    sha, err := s.client.ScriptLoad(ctx, lockDriverScript).Result()
    if err != nil {
        return err
    }
    lockDriverSha = sha
    return nil
}

func (s *RedisOfferGroupStore) LockDriver(ctx context.Context, riderID, driverID string) (bool, error) {
    result, err := s.client.EvalSha(ctx, lockDriverSha, 
        []string{fmt.Sprintf("offer:rider:%s", riderID)}, 
        driverID,
    ).Result()
    
    if err != nil {
        return false, err
    }
    
    locked, ok := result.(int64)
    if !ok {
        return false, errors.New("unexpected script result")
    }
    
    return locked == 1, nil
}
```

---

#### 🧪 Testing Checklist
- [ ] Test: Create offer → succeeds
- [ ] Test: Create duplicate offer → fails with `ErrOfferExists`
- [ ] Test: Get non-existent offer → returns `ErrOfferNotFound`
- [ ] Test: Lock driver in offer → succeeds
- [ ] Test: Lock driver not in offer → fails
- [ ] Test: Lock when another driver already locked → fails
- [ ] Test: Offer expires after TTL (use 1 second for test)

**Bonus:** Run `redis-cli MONITOR` to watch real-time Redis commands!

---

### **Day 2: Distributed Coordination Patterns**

#### 🧠 Theory (45 minutes)
**The Double-Booking Problem**

**Scenario:**
1. Server A finds Driver D is IDLE
2. Server B finds Driver D is IDLE (same millisecond)
3. Both servers transition D to MATCHING
4. Driver D gets 2 offers for different riders!

**Solutions:**

| Approach | How It Works | Tradeoff |
|:---------|:-------------|:---------|
| **Distributed Lock** | Redis `SET driver:D lock NX EX 10` | Extra network call |
| **Intent Field** | Store `intent` in driver record, CAS on it | Requires careful cleanup |
| **OfferGroup as Lock** | Only offer if `CreateOfferGroup` succeeds | Current approach! |

**Your Implementation Uses:** OfferGroup creation as implicit lock!

---

#### 💻 Assignment: Integrate Redis Store into Matching Service

**File:** `internal/matching/service.go`

**Current Flow (In-Memory):**
```
Match() → scorer.Score() → offerStore.Create() → dispatcher.Send()
```

**Updated Flow (Redis):**
```
Match() → scorer.Score() → redisStore.CreateOfferGroup() 
    → if success: repoTransitionStatus(MATCHING) 
    → dispatcher.Send()
    → if fail: skip driver (already in offer)
```

**Your Changes:**

1. **Add Redis Client to MatchingService:**
```go
type MatchingService struct {
    repo          repository.DriverRepository
    spatial       spatial.Index
    scorer        *Scorer
    offerStore    OfferGroupStore  // Change to interface
    redisStore    *RedisOfferGroupStore  // Add Redis store
    dispatcher    events.Dispatcher
}
```

2. **Modify `Match()` to Use Redis:**
```go
func (s *MatchingService) Match(ctx context.Context, rider domain.Rider) ([]string, error) {
    // ... existing code to find candidates ...
    
    topDriverIDs := extractIDs(scoredDrivers[:topN])
    
    // Try to create offer group (atomic lock)
    err := s.redisStore.CreateOfferGroup(ctx, rider.ID, topDriverIDs)
    if err == ErrOfferExists {
        return nil, errors.New("rider already has pending offer")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to create offer: %w", err)
    }
    
    // Only transition drivers AFTER successful offer creation
    for _, driverID := range topDriverIDs {
        err := s.repo.TransitionStatus(driverID, domain.DriverMatching)
        if err != nil {
            // Rollback: driver might have gone BUSY in parallel
            continue
        }
    }
    
    // Send offers
    offerEvent := events.OfferEvent{
        RiderID:   rider.ID,
        DriverIDs: topDriverIDs,
        Pickup:    rider.Pickup,
        Dropoff:   rider.Dropoff,
    }
    
    s.dispatcher.Dispatch(offerEvent)
    
    return topDriverIDs, nil
}
```

3. **Add Accept/Reject Handlers:**
```go
func (s *MatchingService) AcceptOffer(ctx context.Context, riderID, driverID string) error {
    // Try to lock driver (prevents double acceptance)
    locked, err := s.redisStore.LockDriver(ctx, riderID, driverID)
    if err != nil {
        return err
    }
    if !locked {
        return errors.New("another driver already accepted")
    }
    
    // Transition driver to BUSY
    err = s.repo.TransitionStatus(driverID, domain.DriverBusy)
    if err != nil {
        return err
    }
    
    // TODO: Transition other drivers back to IDLE
    // TODO: Delete offer from Redis
    
    return nil
}
```

---

#### 🧪 Testing Checklist
- [ ] Test: Concurrent matches for same driver → only 1 offer created
- [ ] Test: Driver accepts → transitions to BUSY
- [ ] Test: Offer expires → drivers return to IDLE (Day 3)
- [ ] Test: Driver accepts, then another driver tries → fails

---

### **Day 3: Advanced Scoring Algorithm**

#### 🧠 Theory (60 minutes)
**Beyond Distance: Multi-Factor Scoring**

**Current Scoring:** Only uses distance (Haversine)

**Production Scoring Factors:**

1. **Distance** (40% weight)
   - Haversine: Fast approximation
   - Road Distance: Accurate but expensive (Google Maps API)
   - **Hybrid**: Use Haversine for filtering, road distance for top 5

2. **Surge Pricing** (30% weight)
   - High demand area → prioritize nearby drivers
   - Low demand → allow farther drivers

3. **Driver Quality** (20% weight)
   - Rating (4.8 vs 4.2)
   - Acceptance rate
   - Cancellation rate

4. **Fairness** (10% weight)
   - Time since last ride (prevent starvation)
   - Randomize ties to avoid always picking same driver

**Scoring Formula:**
```
Score = (0.4 * DistanceScore) + (0.3 * SurgeScore) + 
        (0.2 * QualityScore) + (0.1 * FairnessScore)
```

**Normalization:** Each sub-score must be 0.0 to 1.0

---

#### 💻 Assignment: Implement Multi-Factor Scorer

**File:** `internal/matching/scorer.go`

**New Scorer Structure:**
```go
type Scorer struct {
    maxDistance float64  // e.g., 5000 meters
}

type ScoringFactors struct {
    Distance       float64  // meters
    DriverRating   float64  // 0.0 to 5.0
    WaitTime       time.Duration  // how long driver has been idle
    SurgeFactor    float64  // 1.0 (normal) to 3.0 (high demand)
}

func (s *Scorer) Score(rider domain.Rider, driver domain.Driver, factors ScoringFactors) float64 {
    // Implement multi-factor scoring
}
```

**Implementation Steps:**

1. **Distance Score (0.0 = far, 1.0 = very close):**
```go
func normalizeDistance(distance, maxDistance float64) float64 {
    if distance >= maxDistance {
        return 0.0
    }
    return 1.0 - (distance / maxDistance)
}
```

2. **Quality Score:**
```go
func normalizeRating(rating float64) float64 {
    // 5.0 → 1.0, 3.0 → 0.0
    return (rating - 3.0) / 2.0
}
```

3. **Fairness Score (longer wait = higher priority):**
```go
func normalizeFairness(waitTime time.Duration) float64 {
    maxWait := 30 * time.Minute
    if waitTime >= maxWait {
        return 1.0
    }
    return float64(waitTime) / float64(maxWait)
}
```

4. **Surge Score (inverse of surge - lower surge = higher score):**
```go
func normalizeSurge(surgeFactor float64) float64 {
    // 1.0x surge → 1.0 score, 3.0x surge → 0.0 score
    return 1.0 - ((surgeFactor - 1.0) / 2.0)
}
```

5. **Combine with Weights:**
```go
func (s *Scorer) Score(rider domain.Rider, driver domain.Driver, factors ScoringFactors) float64 {
    distScore := normalizeDistance(factors.Distance, s.maxDistance)
    qualityScore := normalizeRating(factors.DriverRating)
    fairnessScore := normalizeFairness(factors.WaitTime)
    surgeScore := normalizeSurge(factors.SurgeFactor)
    
    finalScore := (0.4 * distScore) + 
                  (0.3 * surgeScore) + 
                  (0.2 * qualityScore) + 
                  (0.1 * fairnessScore)
    
    return finalScore
}
```

---

#### 🧪 Testing Checklist
- [ ] Test: Close driver (100m) scores higher than far driver (2km)
- [ ] Test: High-rated driver (4.9) beats low-rated (4.2) at same distance
- [ ] Test: Driver waiting 20min beats driver waiting 5min (fairness)
- [ ] Test: Surge 1.0x scores higher than 3.0x
- [ ] Test: Edge case - all factors at max → score = 1.0
- [ ] Test: Edge case - all factors at min → score = 0.0

**Bonus:** Write a benchmark to ensure scoring \< 100ns per driver!

---

### **Day 4: Fairness & Performance Optimization**

#### 🧠 Theory (45 minutes)
**The Starvation Problem**

**Scenario:**
- Driver A is 100m from pickup, rating 4.9
- Driver B is 200m from pickup, rating 4.7
- Driver A **always** gets matched first
- Driver B waits 3 hours without a ride!

**Solutions:**

1. **Random Shuffling (Simple)**
```go
rand.Shuffle(len(drivers), func(i, j int) {
    drivers[i], drivers[j] = drivers[j], drivers[i]
})
```

2. **Wait-Time Boost (Production)**
   - Already implemented in fairness score!

3. **Round-Robin Zones (Advanced)**
   - Divide city into zones
   - Track last matched driver per zone
   - Rotate selection

---

#### 💻 Assignment: Add Fairness to Matching

**File:** `internal/matching/service.go`

**Changes:**

1. **Track Driver Idle Time:**
   - Add `IdleSince time.Time` to `domain.Driver`
   - Update when driver transitions to IDLE

2. **Inject Wait Time into Scoring:**
```go
func (s *MatchingService) Match(ctx context.Context, rider domain.Rider) ([]string, error) {
    candidates := s.spatial.QueryRadius(rider.Pickup, 5000)
    
    now := time.Now()
    var scored []ScoredDriver
    
    for _, d := range candidates {
        if d.Status != domain.DriverIdle {
            continue
        }
        
        distance := haversine(rider.Pickup, d.Location)
        waitTime := now.Sub(d.IdleSince)  // Fairness factor
        
        factors := ScoringFactors{
            Distance:     distance,
            DriverRating: d.Rating,
            WaitTime:     waitTime,
            SurgeFactor:  1.0,  // TODO: Implement surge
        }
        
        score := s.scorer.Score(rider, d, factors)
        scored = append(scored, ScoredDriver{Driver: d, Score: score})
    }
    
    // Sort by score
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].Score > scored[j].Score
    })
    
    // Add randomization for ties
    scored = s.shuffleTies(scored, 0.01)  // Shuffle if scores within 1%
    
    // ... rest of matching logic ...
}
```

3. **Shuffle Ties Function:**
```go
func (s *MatchingService) shuffleTies(drivers []ScoredDriver, tolerance float64) []ScoredDriver {
    if len(drivers) == 0 {
        return drivers
    }
    
    var groups [][]ScoredDriver
    currentGroup := []ScoredDriver{drivers[0]}
    
    for i := 1; i < len(drivers); i++ {
        if math.Abs(drivers[i].Score - currentGroup[0].Score) < tolerance {
            currentGroup = append(currentGroup, drivers[i])
        } else {
            groups = append(groups, currentGroup)
            currentGroup = []ScoredDriver{drivers[i]}
        }
    }
    groups = append(groups, currentGroup)
    
    // Shuffle each group
    var result []ScoredDriver
    for _, group := range groups {
        rand.Shuffle(len(group), func(i, j int) {
            group[i], group[j] = group[j], group[i]
        })
        result = append(result, group...)
    }
    
    return result
}
```

---

#### 🧪 Performance Testing
**Goal:** Process 10,000 matches in \< 5 seconds (P99 latency \< 50ms)

**Create:** `internal/matching/performance_test.go`

```go
func BenchmarkMatchingService(b *testing.B) {
    // Setup
    repo := repository.NewInMemoryDriverRepository()
    spatial := spatial.NewH3Index(repo)
    scorer := NewScorer(5000)
    offerStore := offer.NewInMemoryOfferGroupStore()
    dispatcher := events.NewAsyncDispatcher()
    
    service := NewMatchingService(repo, spatial, scorer, offerStore, dispatcher)
    
    // Seed with 10,000 drivers
    for i := 0; i < 10000; i++ {
        driver := domain.Driver{
            ID:        fmt.Sprintf("D%d", i),
            Location:  randomLocation(),
            Status:    domain.DriverIdle,
            Rating:    4.0 + rand.Float64(),
            IdleSince: time.Now().Add(-time.Duration(rand.Intn(3600)) * time.Second),
        }
        repo.Upsert(driver)
        spatial.Update(driver.ID, driver.Location, driver.Status)
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        rider := domain.Rider{
            ID:     fmt.Sprintf("R%d", i),
            Pickup: randomLocation(),
        }
        _, err := service.Match(context.Background(), rider)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

**Expected Results:**
- Throughput: \> 2000 matches/sec
- P99 Latency: \< 50ms
- Memory: \< 500MB for 10K drivers

---

### **Day 5: Concurrency & Chaos Testing**

#### 🧠 Theory (60 minutes)
**Real-World Concurrency Bugs**

**Bug 1: The Lost Update**
```
Thread A: Read driver D (status=IDLE)
Thread B: Read driver D (status=IDLE)
Thread A: Update driver D (status=MATCHING)
Thread B: Update driver D (status=MATCHING)  ← Overwrites A's change!
```

**Fix:** Use CAS (Compare-And-Swap) in repository

**Bug 2: The Deadlock**
```
Server A: Lock(Driver1) → Lock(Rider1)
Server B: Lock(Rider1) → Lock(Driver1)  ← Deadlock!
```

**Fix:** Always acquire locks in same order (e.g., alphabetically)

**Bug 3: The Dirty Read**
```
Thread A: TransitionStatus(D, MATCHING) → Dispatch()
Thread B: Reads driver D (still IDLE because mutex released)
```

**Fix:** Use RWMutex, hold lock until all updates complete

---

#### 💻 Assignment: Chaos Testing Suite

**File:** `internal/matching/chaos_test.go`

**Test 1: Race Condition Detection**
```go
func TestConcurrentMatchingSameDriver(t *testing.T) {
    // Setup
    repo := repository.NewInMemoryDriverRepository()
    spatial := spatial.NewH3Index(repo)
    
    driver := domain.Driver{
        ID:       "D1",
        Location: domain.Location{Lat: 37.7749, Lng: -122.4194},
        Status:   domain.DriverIdle,
    }
    repo.Upsert(driver)
    spatial.Update(driver.ID, driver.Location, driver.Status)
    
    service := NewMatchingService(repo, spatial, /*...*/)
    
    // 100 concurrent riders trying to match same driver
    results := make(chan error, 100)
    
    for i := 0; i < 100; i++ {
        go func(riderID string) {
            rider := domain.Rider{
                ID:     riderID,
                Pickup: driver.Location,  // Same location
            }
            _, err := service.Match(context.Background(), rider)
            results <- err
        }(fmt.Sprintf("R%d", i))
    }
    
    // Collect results
    successes := 0
    for i := 0; i < 100; i++ {
        err := <-results
        if err == nil {
            successes++
        }
    }
    
    // Only 1 should succeed (driver can only be in 1 offer)
    if successes != 1 {
        t.Fatalf("Expected 1 success, got %d", successes)
    }
    
    // Verify driver is in MATCHING state
    d, _ := repo.Get("D1")
    if d.Status != domain.DriverMatching {
        t.Fatalf("Expected MATCHING, got %s", d.Status)
    }
}
```

**Test 2: Redis Failure Handling**
```go
func TestRedisDownDuringMatch(t *testing.T) {
    // Setup with real Redis
    redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
    redisStore := offer.NewRedisOfferGroupStore(redisClient, 30*time.Second)
    
    service := NewMatchingService(repo, spatial, scorer, redisStore, dispatcher)
    
    // First match succeeds
    rider1 := domain.Rider{ID: "R1", Pickup: domain.Location{Lat: 37.7749, Lng: -122.4194}}
    _, err := service.Match(context.Background(), rider1)
    if err != nil {
        t.Fatal("First match should succeed")
    }
    
    // Simulate Redis crash (stop Redis in Docker)
    // docker stop redis
    
    // Second match should fail gracefully
    rider2 := domain.Rider{ID: "R2", Pickup: domain.Location{Lat: 37.7749, Lng: -122.4194}}
    _, err = service.Match(context.Background(), rider2)
    if err == nil {
        t.Fatal("Expected error when Redis is down")
    }
    
    // Verify error is informative
    if !strings.Contains(err.Error(), "redis") {
        t.Fatalf("Error should mention Redis: %v", err)
    }
}
```

**Test 3: Accept/Reject Race**
```go
func TestConcurrentAcceptReject(t *testing.T) {
    // Setup: Match creates offer with 3 drivers
    service := setupServiceWithOffer(t, "R1", []string{"D1", "D2", "D3"})
    
    // All 3 drivers accept simultaneously
    results := make(chan error, 3)
    
    go func() { results <- service.AcceptOffer(context.Background(), "R1", "D1") }()
    go func() { results <- service.AcceptOffer(context.Background(), "R1", "D2") }()
    go func() { results <- service.AcceptOffer(context.Background(), "R1", "D3") }()
    
    // Exactly 1 should succeed
    successes := 0
    for i := 0; i < 3; i++ {
        err := <-results
        if err == nil {
            successes++
        }
    }
    
    if successes != 1 {
        t.Fatalf("Expected 1 success, got %d", successes)
    }
}
```

---

#### 🧪 Load Testing with `go test`
```bash
# Run with race detector
go test -race ./internal/matching/...

# Run chaos tests 100 times
go test -count=100 ./internal/matching -run TestConcurrent

# Benchmark with profiling
go test -bench=. -cpuprofile=cpu.prof ./internal/matching
go tool pprof cpu.prof
```

---

## 📊 Week 2 Grading Rubric (100 Points)

### Implementation (60 points)
- [ ] **Redis OfferGroupStore** (20 pts)
  - CreateOfferGroup works correctly (5)
  - GetOfferGroup handles missing keys (5)
  - LockDriver uses Lua script atomically (10)

- [ ] **Multi-Factor Scoring** (20 pts)
  - Distance normalization (5)
  - Quality + Fairness scoring (10)
  - Performance \< 100ns per driver (5)

- [ ] **Concurrency Safety** (20 pts)
  - No race conditions in `go test -race` (10)
  - Proper mutex usage (5)
  - OfferGroup prevents double booking (5)

### Testing (30 points)
- [ ] **Unit Tests** (15 pts)
  - All Day 1-2 tests pass (5)
  - All Day 3-4 tests pass (5)
  - All Day 5 chaos tests pass (5)

- [ ] **Performance Tests** (15 pts)
  - Benchmark: \> 2000 matches/sec (5)
  - P99 latency \< 50ms (5)
  - Memory \< 500MB for 10K drivers (5)

### Code Quality (10 points)
- [ ] **Clean Architecture** (5 pts)
  - Interfaces properly defined
  - No circular dependencies
  - Clear separation of concerns

- [ ] **Error Handling** (5 pts)
  - All errors wrapped with context
  - No panics in production code
  - Graceful degradation (e.g., Redis down)

---

## 🎓 Learning Resources

### Redis
- [Redis University - RU101](https://university.redis.com/courses/ru101/)
- [Lua Scripting Guide](https://redis.io/docs/manual/programmability/eval-intro/)

### Concurrency
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [Uber's Concurrency Guide](https://github.com/uber-go/guide/blob/master/style.md#synchronization)

### Distributed Systems
- [Designing Data-Intensive Applications](https://dataintensive.net/) (Chapter 5: Replication)

---

## 🚀 Submission Instructions

When you're ready for grading:

1. **Push your code** (ensure all files committed)
2. **Run full test suite:**
   ```bash
   go test -race -cover ./internal/...
   ```
3. **Create a summary document:** `docs/week2_submission.md`
   - What you implemented
   - Test results (screenshots)
   - Challenges faced
   - Performance metrics

4. **Tag me**: Send a message "Week 2 complete, ready for grading!"

I'll review your code and provide:
- ✅ Line-by-line feedback
- 📊 Performance analysis
- 🎯 Suggestions for FAANG interview discussions
- ⭐ Grade (0-100)

---

## 💡 Pro Tips for FAANG Interviews

**When discussing this project:**

1. **Start with the problem:** "In a distributed ride-matching system, the same driver could be offered to multiple riders simultaneously..."

2. **Explain your solution:** "I used Redis SET with NX flag as a distributed lock, combined with Lua scripting to make the check-and-lock operation atomic..."

3. **Show tradeoffs:** "I chose Redis over Zookeeper because our consistency requirements are 'eventual' not 'strong,' and Redis offers better latency..."

4. **Mention testing:** "I wrote chaos tests that simulate concurrent accepts and verified no double-booking with `go test -race`..."

5. **Discuss scalability:** "This design supports horizontal scaling - multiple matching servers can coordinate through Redis, with each offer attempt being atomic..."

**Interviewers love candidates who:**
- ✅ Identify edge cases (race conditions)
- ✅ Make informed tradeoffs (latency vs consistency)
- ✅ Test rigorously (chaos engineering)
- ✅ Think about production (what if Redis crashes?)

---

Good luck with Week 2! Remember: **The goal isn't just working code - it's understanding WHY each design decision matters at scale.** 🚀

---

**Next Week Preview:** Week 3 will cover Kafka event streaming, geo-sharding, and real-time surge pricing!
