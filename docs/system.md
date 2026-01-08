#
- Repository is the source of truth.
- Index is a cache that mirrors accepted state.
- Never update index unless repo accepts the change.

#
- Domain has zero dependencies

- Application layer orchestrates logic

- Repositories hide DB / Redis / Memory

## Matching Service - Mental Model

            Rider Request
                    ↓
            H3 Cell for Rider
                    ↓
            K-Ring Expansion
                    ↓
            Candidate Drivers (Snapshot)
                    ↓
            Filter (IDLE, Distance)
                    ↓       
            Score (Distance)
                    ↓
            Pick Best
                    ↓
            Mark BUSY (Atomic intent)


## Transition:
-      OFFLINE → IDLE    (driver comes online)
        IDLE → MATCHING      (candidate selected)
        MATCHING → BUSY      (ride accepted)
        MATCHING → IDLE      (ride rejected / timeout)
        BUSY → IDLE          (ride completed)
        IDLE → OFFLINE       (driver goes offline)

##
main.go
 ├─ domain/
 ├─ repository/        ← driverRepo (source of truth)
 ├─ spatial/           ← H3Index
 ├─ matching/          ← Matching Service


## 
bussy is too late ,because of two rider founds Same rider Idle , it is possible that both try to lock them, and evantual one of them get success , but both already sent pings
Soln : before Matching , Introduce a soft lock
               IDLE
                ↓
        INTENT_LOCKED (TTL based)
                ↓
             MATCHING
                ↓
               BUSY
Properties :
Auto-expires, Non-Blocking, Idempotent,Safer under crash


little changes for Async accept/reject:
IDLE
 └─ TryLockIntent(rider)
     └─ MATCHING (offer sent)
         ├─ Accept → BUSY
         ├─ Reject → IDLE
         └─ Timeout → Recovery → IDLE


Redis IS:
A single-threaded, in-memory, atomic state machine with TTL

##

Requirement	            Why
Bounded concurrency	prevent overload
Backpressure	        system stability
Retry with limits	transient failure handling
Decoupling	        matching ≠ delivery
Observability	        logs & metrics
Failure isolation	matching must not crash

+------------------+
| Matching Service |
+--------+---------+
         |
         | OfferEvent
         v
+------------------+
|  Event Queue     |  (bounded channel)
+--------+---------+
         |
         v
+------------------+
| Worker Pool (N)  |
+--------+---------+
         |
         v
+------------------+
| Push / Kafka /   |
| Notification     |
+------------------+


## notes
Only the sender OR the owner closes a channel — never both


Redis solves “WHO wins the rider”
Intent locks solve “WHO owns the driver”


##
┌────────────────────────┐
│ Match()                │
│                        │
│ 1. Find candidates     │
│ 2. TryLockIntent (DRV) │ ◀─ driver-side gate
│ 3. Create OfferGroup   │ ◀─ rider-side gate
│ 4. Emit offers         │
└────────────────────────┘


##
┌────────────────────────┐
│ Driver Accept          │
│                        │
│ 1. Redis.Accept()     │ ◀─ rider arbitration
│ 2. FSM transition     │ ◀─ driver commit
└────────────────────────┘

##
┌──────────┐
│  main.go │  ← composition root
└────┬─────┘
     │
     ▼
┌────────────┐     enqueue      ┌──────────────┐
│  Service   │ ───────────────▶ │  Dispatcher  │
│ (business) │                  │  (infra)     │
└────┬───────┘                  └─────┬────────┘
     ▲                                │
     │        callback                ▼
┌──────────────┐ ◀───────────── ┌──────────────┐
│ OfferSender  │                │ Worker Queue │
│ (transport)  │                └──────────────┘
└──────────────┘



##
Test creates Service
    ↓
Test creates InMemoryOfferSender(service) ← passes Service as ResponseHandler
    ↓
Match() → locks 3 drivers → creates offer_group → enqueues offers
    ↓
Dispatcher → calls SendDriverOffer()
    ↓
SendDriverOffer() → calls service.HandleDriverResponse() ✅ (was broken before)
    ↓
HandleDriverResponse() → calls Redis Accept() → first one wins
    ↓
Winner: TransitionStatus(BUSY)
Losers: TransitionStatus(IDLE)