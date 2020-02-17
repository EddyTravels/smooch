package storage

import (
	"github.com/gomodule/redigo/redis"
)

// RedisStorage defines struct property for redis storage
type RedisStorage struct {
	pool   *redis.Pool
	jwtKey string
}

// NewRedisStorage initializes new instance of redis storage
func NewRedisStorage(p *redis.Pool) *RedisStorage {
	return &RedisStorage{
		pool:   p,
		jwtKey: "smooch-jwt-token",
	}
}

// SaveTokenToRedis will save jwt token to redis
func (rs *RedisStorage) SaveTokenToRedis(token string, ttl int64) error {
	conn := rs.pool.Get()
	defer conn.Close()

	_, err := conn.Do("SETEX", rs.jwtKey, ttl, token)
	return err
}

// GetTokenFromRedis will retrieve jwt token from redis
func (rs *RedisStorage) GetTokenFromRedis() (string, error) {
	conn := rs.pool.Get()
	defer conn.Close()

	val, err := redis.String(conn.Do("GET", rs.jwtKey))
	if err != nil {
		return "", err
	}
	return val, nil
}
