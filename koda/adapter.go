package koda

import (
	"time"

	"gopkg.in/redis.v3"
)

type Conn interface {
	Incr(key string) (int, error)
	HIncr(key, field string) (int, error)
	HGet(key, field string) (string, error)
	HSet(key, field, value string) (bool, error)
	RPush(key string, value ...string) (int, error)
	LPush(key string, value ...string) (int, error)
	BRPop(timeout time.Duration, keys ...string) ([]string, error)
	ZAddNX(key string, score float64, member string) (int, error)
	Close() error
}

type GoRedisAdapter struct {
	R *redis.Client
}

func (r *GoRedisAdapter) Incr(key string) (int, error) {
	cmd := r.R.Incr(key)
	return int(cmd.Val()), cmd.Err()
}

func (r *GoRedisAdapter) HIncr(key, field string) (int, error) {
	cmd := r.R.HIncrBy(key, field, 1)
	return int(cmd.Val()), cmd.Err()
}

func (r *GoRedisAdapter) HGet(key, field string) (string, error) {
	return r.R.HGet(key, field).Result()
}

func (r *GoRedisAdapter) HSet(key, field, value string) (bool, error) {
	return r.R.HSet(key, field, value).Result()
}

func (r *GoRedisAdapter) RPush(key string, value ...string) (int, error) {
	cmd := r.R.RPush(key, value...)
	return int(cmd.Val()), cmd.Err()
}

func (r *GoRedisAdapter) LPush(key string, value ...string) (int, error) {
	cmd := r.R.LPush(key, value...)
	return int(cmd.Val()), cmd.Err()
}

func (r *GoRedisAdapter) BRPop(timeout time.Duration, keys ...string) ([]string, error) {
	return r.R.BRPop(timeout, keys...).Result()
}

func (r *GoRedisAdapter) ZAddNX(key string, score float64, member string) (int, error) {
	cmd := r.R.ZAddNX(key, redis.Z{
		Score:  score,
		Member: member,
	})

	return int(cmd.Val()), cmd.Err()
}

func (r *GoRedisAdapter) Close() error {
	return r.R.Close()
}
