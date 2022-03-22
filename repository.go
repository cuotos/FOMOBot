package main

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/slack-go/slack/slackevents"
)

type Repository interface {
	Incr(string) error
	Get(string) (int, error)
}

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository() (*RedisRepository, error) {
	repo := &RedisRepository{}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	repo.client = redisClient

	return repo, nil
}

func (repo RedisRepository) Incr(key string) error {
	return repo.client.Incr(context.Background(), key).Err()
}

func (repo RedisRepository) Get(key string) (int, error) {
	return repo.client.Get(context.Background(), key).Int()
}

func GenerateKey(event *slackevents.ReactionAddedEvent) string {
	return fmt.Sprintf("%s_%s", event.Item.Channel, event.Item.Timestamp)
}
