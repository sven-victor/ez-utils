package redis

import "github.com/go-redis/redis"

// Session embeds a go-redis Client for use as a request-scoped handle.
type Session struct {
	*redis.Client
}
