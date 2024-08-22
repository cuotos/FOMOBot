package database

import (
	"context"
	"fmt"
	"log"
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

	connectionEstablished := false

	redisOptions.OnConnect = func(ctx context.Context, cn *redis.Conn) error {
		log.Print("[INFO] redis connection established")
		connectionEstablished = true
		return nil
	}

	var redisClient *redis.Client
	redisMaxConnctionAttempts := 10

	redisConnectAttempt := 0
	for {
		redisConnectAttempt++
		if redisConnectAttempt >= redisMaxConnctionAttempts {
			return nil, fmt.Errorf("[ERROR] failed to connect to redis after %d attempts", redisMaxConnctionAttempts)
		}

		// establishing a connection triggers the OnConnect callback which sets the "connectionEstablished" var to true
		redisClient = redis.NewClient(redisOptions)
		redisClient.Ping(context.Background()).Err()

		if connectionEstablished {
			break
		}

		log.Print("[TRACE] redis connection not established, trying again...")
		log.Print("[WARN] failed to connect to redis, will try again in 2 seconds")
		time.Sleep(time.Second * 2)

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
