package koda

import (
	"fmt"
	"strconv"
	"time"

	"gopkg.in/redis.v3"
)

var NilError = redis.Nil

// GoRedisAdapter is an adapter for the redis.v3 library
type GoRedisAdapter struct {
	R *redis.Client
}

func (r *GoRedisAdapter) Incr(key string) (int, error) {
	cmd := r.R.Incr(key)
	return int(cmd.Val()), cmd.Err()
}

func (r *GoRedisAdapter) HGetAll(key string) ([]string, error) {
	return r.R.HGetAll(key).Result()
}

func (r *GoRedisAdapter) HSetAll(key string, fields map[string]string) error {
	_, err := r.R.HMSetMap(key, fields).Result()
	return err
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

func (r *GoRedisAdapter) ZRem(key string, members ...string) (int, error) {
	cmd := r.R.ZRem(key, members...)
	return int(cmd.Val()), cmd.Err()
}

func (r *GoRedisAdapter) ZRangeByScore(key string, opt ZRangeByScoreOpts) ([]string, error) {
	var rangeStr [2]string
	ranges := []float64{opt.Min, opt.Max}
	inclusive := []bool{opt.MinInclusive, opt.MaxInclusive}

	for i := range ranges {
		rangeStr[i] = strconv.FormatFloat(ranges[i], 'E', -1, 64)
		if !inclusive[i] {
			rangeStr[i] = fmt.Sprintf("(%s", rangeStr[i])
		}
	}

	cmd := r.R.ZRangeByScore(key, redis.ZRangeByScore{
		Min:    rangeStr[0],
		Max:    rangeStr[1],
		Offset: int64(opt.Offset),
		Count:  int64(opt.Count),
	})

	return cmd.Val(), cmd.Err()
}

func (r *GoRedisAdapter) Scan(cursor int, match string, count int) (int, []string, error) {
	cmd := r.R.Scan(int64(cursor), match, int64(count))
	offset, results := cmd.Val()
	return int(offset), results, cmd.Err()
}

func (r *GoRedisAdapter) Close() error {
	return r.R.Close()
}
