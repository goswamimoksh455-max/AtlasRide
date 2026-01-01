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