package database

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisDatabase struct {
	rdb                         *redis.Client
	notificationThresholdPeriod time.Duration
}

func NewRedisDatabase(addr string, password string, dbID int, notificationThreshold time.Duration) (Database, error) {

	db := &RedisDatabase{
		notificationThresholdPeriod: notificationThreshold,
	}

	redisOptions := &redis.Options{
		Addr: addr,
		DB:   dbID,
	}

	if password != "" {
		redisOptions.Password = password
	}

	redisClient := redis.NewClient(redisOptions)

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	db.rdb = redisClient

	return db, nil
}

func (db *RedisDatabase) Incr(ctx context.Context, key string) (int, error) {

	pipe := db.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	// This is will set TTL back to threshold on every reaction, its only wanted on the first one...
	// or is this ok?? "the counter is reset on every call... if there are none for <timeout> seconds, the redis item disapears..."
	pipe.Expire(ctx, key, db.notificationThresholdPeriod)

	_, err := pipe.Exec(ctx)
	return int(incr.Val()), err
}

func (db *RedisDatabase) Healthy(ctx context.Context) error {
	err := db.rdb.Ping(context.Background()).Err()
	return err

}
