package cache_test

import (
	"context"
	"testing"

	"github.com/taufMFI4430d/async-job-poc/internal/platform/cache"
)

func TestOpenRedisRejectsInvalidOptions(t *testing.T) {
	tests := []struct {
		name    string
		options cache.RedisOptions
	}{
		{
			name: "missing host",
			options: cache.RedisOptions{
				Port: 6379,
				DB:   0,
			},
		},
		{
			name: "invalid port",
			options: cache.RedisOptions{
				Host: "localhost",
				Port: 70000,
				DB:   0,
			},
		},
		{
			name: "negative database",
			options: cache.RedisOptions{
				Host: "localhost",
				Port: 6379,
				DB:   -1,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := cache.OpenRedis(
				context.Background(),
				test.options,
			)
			if err == nil {
				t.Fatal("expected an error for invalid Redis options")
			}
		})
	}
}
