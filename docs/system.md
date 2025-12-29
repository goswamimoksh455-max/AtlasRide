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
