package main

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type Repository interface {
	Incr(string) error
	Get(string) (int, error)
}

type RedisRepository struct {
	rdb *redis.Client
}

func NewRedisRepository(addr string, password string, db int) (*RedisRepository, error) {

	repo := &RedisRepository{}

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

func (repo RedisRepository) Incr(ctx context.Context, key string) (int, error) {

	pipe := repo.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Minute) // this will expire from redis 1 min after the last "increment"

	_, err := pipe.Exec(ctx)
	return int(incr.Val()), err
}

func (repo RedisRepository) Get(key string) (int, error) {
	return repo.rdb.Get(context.Background(), key).Int()
}
