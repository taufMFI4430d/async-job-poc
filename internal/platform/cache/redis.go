package cache

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redisDialTimeout  = 5 * time.Second
	redisReadTimeout  = 3 * time.Second
	redisWriteTimeout = 3 * time.Second
	redisPoolSize     = 20
	redisMinIdleConns = 2
)

type RedisOptions struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type Redis struct {
	client *redis.Client
}

func OpenRedis(
	ctx context.Context,
	options RedisOptions,
) (*Redis, error) {
	if err := validateRedisOptions(options); err != nil {
		return nil, err
	}

	client := redis.NewClient(&redis.Options{
		Addr: net.JoinHostPort(
			options.Host,
			strconv.Itoa(options.Port),
		),
		Password:     options.Password,
		DB:           options.DB,
		DialTimeout:  redisDialTimeout,
		ReadTimeout:  redisReadTimeout,
		WriteTimeout: redisWriteTimeout,
		PoolSize:     redisPoolSize,
		MinIdleConns: redisMinIdleConns,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()

		return nil, fmt.Errorf(
			"ping Redis at %s: %w",
			client.Options().Addr,
			err,
		)
	}

	return &Redis{client: client}, nil
}

// Client is intended only for outer infrastructure adapters.
func (redisConnection *Redis) Client() *redis.Client {
	if redisConnection == nil {
		return nil
	}

	return redisConnection.client
}

func (redisConnection *Redis) Ping(ctx context.Context) error {
	if redisConnection == nil || redisConnection.client == nil {
		return errors.New("Redis connection is not initialized")
	}

	if err := redisConnection.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping Redis: %w", err)
	}

	return nil
}

func (redisConnection *Redis) Close() error {
	if redisConnection == nil || redisConnection.client == nil {
		return nil
	}

	if err := redisConnection.client.Close(); err != nil {
		return fmt.Errorf("close Redis connection: %w", err)
	}

	return nil
}

func validateRedisOptions(options RedisOptions) error {
	if strings.TrimSpace(options.Host) == "" {
		return errors.New("Redis host is required")
	}

	if options.Port < 1 || options.Port > 65535 {
		return errors.New("Redis port must be between 1 and 65535")
	}

	if options.DB < 0 {
		return errors.New("Redis DB must be zero or greater")
	}

	return nil
}
