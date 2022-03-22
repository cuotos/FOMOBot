package main

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// TODO we are not actually using this interface anywhere!
type Repository interface {
	Incr(context.Context, string) (int, error)
	Get(context.Context, string) (int, error)
	Set(context.Context, string, interface{}, time.Duration) error
}

type RedisRepository struct {
	rdb                         *redis.Client
	notificationThresholdPeriod time.Duration
}

func NewRedisRepository(addr string, password string, db int, notificationThreshold time.Duration) (Repository, error) {

	repo := &RedisRepository{
		notificationThresholdPeriod: notificationThreshold,
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	repo.rdb = redisClient

	return repo, nil
}

func (repo *RedisRepository) Incr(ctx context.Context, key string) (int, error) {

	pipe := repo.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	// This is will set TTL back to threshold on every reaction, its only wanted on the first one...
	// or is this ok?? "the counter is reset on every call... if there are none for <timeout> seconds, the redis item disapears..."
	pipe.Expire(ctx, key, repo.notificationThresholdPeriod)

	_, err := pipe.Exec(ctx)
	return int(incr.Val()), err
}

func (repo *RedisRepository) Get(ctx context.Context, key string) (int, error) {
	return repo.rdb.Get(context.Background(), key).Int()
}

func (repo *RedisRepository) Set(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	return repo.rdb.Set(ctx, key, value, exp).Err()
}
