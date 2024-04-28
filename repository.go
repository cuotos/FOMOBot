package main

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// TODO we are not actually using this interface anywhere!
type Repository interface {
	Incr(context.Context, string) (int, error)
	Healthy(context.Context) error
}

type RedisRepository struct {
	rdb                         *redis.Client
	notificationThresholdPeriod time.Duration
}

func NewRedisRepository(addr string, password string, db int, notificationThreshold time.Duration) (Repository, error) {

	repo := &RedisRepository{
		notificationThresholdPeriod: notificationThreshold,
	}

	redisOptions := &redis.Options{
		Addr: addr,
		DB:   db,
	}

	if password != "" {
		redisOptions.Password = password
	}

	redisClient := redis.NewClient(redisOptions)

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

func (repo *RedisRepository) Healthy(ctx context.Context) error {
	err := repo.rdb.Ping(context.Background()).Err()
	return err

}
