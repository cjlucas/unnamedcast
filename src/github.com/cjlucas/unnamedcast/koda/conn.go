package koda

import "time"

type ZRangeByScoreOpts struct {
	Min          float64
	Max          float64
	MinInclusive bool
	MaxInclusive bool
	Offset       int64
	Count        int64
}

type Conn interface {
	Incr(key string) (int, error)
	HIncr(key, field string) (int, error)
	HGet(key, field string) (string, error)
	HGetAll(key string) ([]string, error)
	HSet(key, field, value string) (bool, error)
	RPush(key string, value ...string) (int, error)
	LPush(key string, value ...string) (int, error)
	BRPop(timeout time.Duration, keys ...string) ([]string, error)
	ZAddNX(key string, score float64, member string) (int, error)
	ZRem(key string, members ...string) (int, error)
	ZRangeByScore(key string, opt *ZRangeByScoreOpts) ([]string, error)
	Close() error
}
