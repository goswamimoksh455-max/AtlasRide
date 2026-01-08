package offer

import "context"

type OfferGroupStore interface {
	Create(ctx context.Context, riderID string, ttlSeconds int) error
	Accept(ctx context.Context, riderID, driverID string) (bool, error)
	GetWinner(ctx context.Context, riderID string) (string, bool, error)
	Remove(ctx context.Context, riderID string) error
}

//abstraction
//for redis we are using the redis+Lua script

// with lua :
//Single - threaded execution inside Redis
//Atomic Decision
//Distributed mutex replacement
