Key : Offer_gorup:{riderID}
value : anything like e.g, 1
TTL : 5 seconds


## We need:

Atomic check
Exclusive execution
No race between pods
Solution :
👉 Redis Lua Script

We rely on:

Redis single-threaded execution
TTL guarantees auto-unlock


building central main memory store (in-memory store) so that in future multiple node can interact with it
