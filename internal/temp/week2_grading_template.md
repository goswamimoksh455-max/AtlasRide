# 📝 Week 2 Grading Template
## Student Submission Review

**Student Name:** ______________________  
**Submission Date:** ______________________  
**Reviewer:** Sir (Mentor)

---

## 📋 Grading Breakdown (100 Points Total)

### Part 1: Redis Implementation (35 points)

#### RedisOfferGroupStore Implementation (20 pts)
- [ ] (5 pts) `CreateOfferGroup()` - Correct key format, NX flag, TTL
  - **Score:** ___ / 5
  - **Feedback:** 
  
- [ ] (5 pts) `GetOfferGroup()` - Proper error handling for missing keys
  - **Score:** ___ / 5
  - **Feedback:** 
  
- [ ] (10 pts) `LockDriver()` - Uses Lua script atomically
  - **Score:** ___ / 10
  - **Feedback:** 
    - Checks driver in offer list? ___
    - Checks existing lock? ___
    - Atomic SET operation? ___
    - Returns correct boolean? ___

**Subtotal Part 1:** ___ / 20

#### Integration with MatchingService (15 pts)
- [ ] (5 pts) Redis store injected correctly
  - **Score:** ___ / 5
  - **Feedback:** 

- [ ] (5 pts) `Match()` creates offer before transitioning drivers
  - **Score:** ___ / 5
  - **Feedback:** 

- [ ] (5 pts) `AcceptOffer()` uses LockDriver for atomicity
  - **Score:** ___ / 5
  - **Feedback:** 

**Subtotal Part 2:** ___ / 15

---

### Part 2: Multi-Factor Scoring (25 points)

#### Normalization Functions (10 pts)
- [ ] (3 pts) Distance normalization (0.0 = far, 1.0 = close)
  - **Score:** ___ / 3
  - **Test Result:** Distance 0m → ___,  Distance 5000m → ___
  
- [ ] (2 pts) Rating normalization (3.0 → 0.0, 5.0 → 1.0)
  - **Score:** ___ / 2
  
- [ ] (3 pts) Fairness (wait time) normalization
  - **Score:** ___ / 3
  
- [ ] (2 pts) Surge normalization
  - **Score:** ___ / 2

**Subtotal:** ___ / 10

#### Weighted Scoring (10 pts)
- [ ] (4 pts) Correct weights applied (0.4, 0.3, 0.2, 0.1)
  - **Score:** ___ / 4
  
- [ ] (3 pts) Final score in range [0.0, 1.0]
  - **Score:** ___ / 3
  
- [ ] (3 pts) Higher scores for better drivers
  - **Score:** ___ / 3

**Subtotal:** ___ / 10

#### Performance (5 pts)
- [ ] (5 pts) Benchmark shows \< 100ns per driver scoring
  - **Score:** ___ / 5
  - **Actual:** ___ ns/op
  - **Screenshot:** ___

**Subtotal:** ___ / 5

---

### Part 3: Fairness Implementation (15 points)

- [ ] (5 pts) `IdleSince` field added to Driver domain
  - **Score:** ___ / 5
  
- [ ] (5 pts) Wait time calculated and passed to scorer
  - **Score:** ___ / 5
  
- [ ] (5 pts) `shuffleTies()` randomizes drivers with similar scores
  - **Score:** ___ / 5
  - **Tolerance:** ___ (should be ~0.01)

**Subtotal Part 3:** ___ / 15

---

### Part 4: Testing (25 points)

#### Unit Tests (15 pts)
Run: `go test ./internal/offer ./internal/matching`

- [ ] (3 pts) Redis CreateOfferGroup tests
  - **Score:** ___ / 3
  - **Pass/Fail:** ___
  
- [ ] (3 pts) Redis LockDriver tests
  - **Score:** ___ / 3
  - **Pass/Fail:** ___
  
- [ ] (3 pts) Scoring algorithm tests
  - **Score:** ___ / 3
  - **Pass/Fail:** ___
  
- [ ] (3 pts) Fairness tests
  - **Score:** ___ / 3
  - **Pass/Fail:** ___
  
- [ ] (3 pts) Edge case coverage (nil checks, missing drivers, etc.)
  - **Score:** ___ / 3
  - **Pass/Fail:** ___

**Subtotal:** ___ / 15

#### Concurrency Tests (10 pts)
Run: `go test -race ./internal/matching -run TestConcurrent`

- [ ] (4 pts) Concurrent match same driver test
  - **Score:** ___ / 4
  - **Result:** Only ___ success(es) (should be 1)
  
- [ ] (3 pts) Accept/Reject race test
  - **Score:** ___ / 3
  - **Pass/Fail:** ___
  
- [ ] (3 pts) No race conditions detected
  - **Score:** ___ / 3
  - **Race warnings:** ___

**Subtotal:** ___ / 10

---

## 💯 Final Score: ___ / 100

### Letter Grade
- 90-100: A (FAANG-ready)
- 80-89: B (Strong, minor improvements)
- 70-79: C (Needs work on concurrency/testing)
- Below 70: Needs revision

**Grade:** ___

---

## 📝 Detailed Feedback

### ✅ Strengths
1. 
2. 
3. 

### 🔧 Areas for Improvement
1. 
2. 
3. 

### 🐛 Critical Bugs Found
1. 
2. 

### 💡 Interview Discussion Points
*What to highlight when discussing this project in interviews:*

1. **Redis Atomicity:** 
   
2. **Scoring Tradeoffs:** 
   
3. **Concurrency Handling:** 
   
4. **Testing Strategy:** 

---

## 🎯 Recommendations for Week 3

Based on your Week 2 performance:

- [ ] **If score \>= 90:** Ready for Week 3 (Kafka + Geo-Sharding)
- [ ] **If score 70-89:** Review concurrency concepts, then proceed
- [ ] **If score \< 70:** Revise Week 2 before moving forward

**Specific areas to review before Week 3:**
1. 
2. 
3. 

---

## 📚 Additional Resources Suggested

Based on gaps identified:
- 
- 
- 

---

**Reviewed by:** Sir (Mentor)  
**Review Date:** _______________  
**Signature:** _______________
