# Domain Model

## Driver
- Represents a physical driver
- Exists independently of availability
- Can be IDLE, MATCHING, or BUSY

### DriverStatus (Enum)
- Prevents invalid transitions
- Makes illegal states unrepresentable
- Faster than string-based state

## Location
- Immutable value object
- Contains no spatial or distance logic

## Rider
- Request-scoped entity
- Stateless
- Short-lived compared to Driver

## Key Invariants
- A driver can never be matched twice
- Location updates must be monotonic in time
- Driver existence ≠ availability

## Repository Interface
- Decouples storage from business logic
- Enables in-memory, Redis, or SQL implementations
- Allows unit testing without infrastructure
- Upsert avoids race-prone branching
- ListByCell aligns with H3 without full scans

## Spatial Index
- Databases are inefficient for nearest-neighbor queries
- In-memory index enables deterministic hot-path performance
- CellID abstraction allows swapping GeoHash / H3 / QuadKey

## Matcher
- Encapsulates business decision logic
- Independent of transport and storage

## MatchingService
- Orchestrates components
- Defines use cases
- Acts as API boundary
- Services coordinate; they do not decide
