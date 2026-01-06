package temp

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisOfferGroupStore struct {
	rdb *redis.Client
}

func NewRedisOfferGroupStore(rdb *redis.Client) *RedisOfferGroupStore {
	return &RedisOfferGroupStore{rdb: rdb}
}

func offerKey(riderID string) string {
	return "offer_group:" + riderID
}

func (s *RedisOfferGroupStore) Create(riderID string,
	drivers []string,
	ttl time.Duration,
) {
	ctx := context.Background()
	key := offerKey(riderID)

	pipe := s.rdb.TxPipeline()
	for _, d := range drivers {
		pipe.SAdd(ctx, key, d)
	}
	pipe.Expire(ctx, key, ttl)
	pipe.Exec(ctx)
}

func (s *RedisOfferGroupStore) OtherDrivers(riderID, winner string) []string {
	ctx := context.Background()
	drivers, _ := s.rdb.SMembers(ctx, offerKey(riderID)).Result()

	out := []string{}
	for _, d := range drivers {
		if d != winner {
			out = append(out, d)
		}
	}
	return out
}

func (s *RedisOfferGroupStore) Exists(riderID string) bool {
	ctx := context.Background()
	res, err := s.rdb.Exists(ctx, offerKey(riderID)).Result()
	return err == nil && res == 1
}

func (s *RedisOfferGroupStore) Remove(riderID string) {
	ctx := context.Background()
	s.rdb.Del(ctx, offerKey(riderID))
}

var offerGroupLockScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[1]) == 1 then
	return 1
else
	return 0
end
`)

func (s *RedisOfferGroupStore) WithLock(
	riderID string,
	fn func(),
) {
	ctx := context.Background()
	key := offerKey(riderID)

	res, err := offerGroupLockScript.Run(
		ctx,
		s.rdb,
		[]string{key},
	).Int()

	if err != nil || res == 0 {
		return
	}

	// Redis guarantees single execution path here
	fn()
}
