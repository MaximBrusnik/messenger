// Package redis provides a shared Redis client with helpers for online
// tracking and caching used by multiple MessangerMax services.
package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// Client wraps *redis.Client with domain helpers.
type Client struct {
	*redis.Client
}

// ServerOnlineTTL is how long an online heartbeat survives without renewal.
const ServerOnlineTTL = 90 * time.Second

// New creates a Redis client with the given address and password.
func New(addr, password string) *Client {
	return &Client{redis.NewClient(&redis.Options{Addr: addr, Password: password})}
}

// SetOnline marks a user as online with a TTL.
func (c *Client) SetOnline(ctx context.Context, userID uint) error {
	key := onlineKey(userID)
	return c.Client.Set(ctx, key, time.Now().Unix(), ServerOnlineTTL).Err()
}

// IsOnline reports whether the user's heartbeat is fresh.
func (c *Client) IsOnline(ctx context.Context, userID uint) bool {
	_, err := c.Client.Get(ctx, onlineKey(userID)).Result()
	return err == nil
}

// SetOffline removes the user's online marker.
func (c *Client) SetOffline(ctx context.Context, userID uint) error {
	return c.Client.Del(ctx, onlineKey(userID)).Err()
}

// CacheProfile stores a serialized JSON profile for a user.
func (c *Client) CacheProfile(ctx context.Context, userID uint, json string, ttl time.Duration) error {
	return c.Client.Set(ctx, profileKey(userID), json, ttl).Err()
}

// GetCachedProfile returns a cached profile JSON, if any.
func (c *Client) GetCachedProfile(ctx context.Context, userID uint) (string, error) {
	return c.Client.Get(ctx, profileKey(userID)).Result()
}

// DeleteCachedProfile invalidates a cached profile.
func (c *Client) DeleteCachedProfile(ctx context.Context, userID uint) error {
	return c.Client.Del(ctx, profileKey(userID)).Err()
}

func onlineKey(userID uint) string {
	return "mm:online:" + itoa(userID)
}

func profileKey(userID uint) string {
	return "mm:profile:" + itoa(userID)
}

func itoa(n uint) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
