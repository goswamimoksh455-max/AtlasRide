package offer

//our redis coordination engine, designed for correctness-first, proved deterministic behavior
import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisOfferGroupStore struct {
	rdb    *redis.Client
	accept *redis.Script
}

func NewRedisOfferGroupStore(rdb *redis.Client) *RedisOfferGroupStore {
	return &RedisOfferGroupStore{
		rdb:    rdb,
		accept: redis.NewScript(loadAcceptLua()),
	}
}

func (s *RedisOfferGroupStore) Create(
	ctx context.Context,
	riderID string,
	ttl int,
) error {

	key := "offer_group:" + riderID

	created, err := s.rdb.HSetNX(ctx, key, "status", "pending").Result()
	if err != nil {
		return err
	}

	if !created {
		return nil // idempotent create
	}

	_, err = s.rdb.HSet(ctx, key,
		"created_at", time.Now().Unix(),
	).Result()
	if err != nil {
		return err
	}

	_, err = s.rdb.Expire(ctx, key, time.Duration(ttl)*time.Second).Result()
	return err
}

func (s *RedisOfferGroupStore) Accept(ctx context.Context, riderID, driverID string) (bool, error) {
	key := "offer_group:" + riderID

	res, err := s.accept.Run(
		ctx,
		s.rdb,
		[]string{key},
		driverID,
	).Int()

	if err != nil {
		if err.Error() == "OFFER_GROUP_NOT_FOUND" {
			return false, errors.New("offer expired")
		}
		return false, err
	}

	return res == 1, nil
}

func (s *RedisOfferGroupStore) GetWinner(
	ctx context.Context,
	riderID string,
) (string, bool, error) {

	key := "offer_group:" + riderID

	res, err := s.rdb.HGet(ctx, key, "winner").Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	return res, true, nil
}

func (s *RedisOfferGroupStore) Remove(
	ctx context.Context,
	riderID string,
) error {

	key := "offer_group:" + riderID
	return s.rdb.Del(ctx, key).Err()
}
