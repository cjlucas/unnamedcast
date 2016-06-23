package koda

import (
	"strconv"
	"time"
)

type mockConn struct {
	keys   map[string]string
	lists  map[string][]string
	hashes map[string]map[string]string
	sets   map[string]map[string]float64 // map[member]score
}

func newConn() Conn {
	return &mockConn{
		keys:   make(map[string]string),
		hashes: make(map[string]map[string]string),
	}
}

func (c mockConn) incrVal(val string) (int, error) {
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}

	return n + 1, nil
}

func (c mockConn) Incr(key string) (int, error) {
	n, err := c.incrVal(c.keys[key])
	if err != nil {
		return 0, err
	}
	c.keys[key] = string(n)
	return n, nil
}

func (c mockConn) HGetAll(key string) ([]string, error) {
	hash, ok := c.hashes[key]
	if !ok {
		return nil, nil
	}

	pairs := make([]string, len(hash)*2)
	for k, v := range hash {
		pairs = append(pairs, k)
		pairs = append(pairs, v)
	}

	return pairs, nil
}

func (c mockConn) HSetAll(key string, fields map[string]string) error {
	if _, ok := c.hashes[key]; !ok {
		c.hashes[key] = make(map[string]string)
	}

	for k, v := range fields {
		c.hashes[key][k] = v
	}

	return nil
}

func (c mockConn) LPush(key string, value ...string) (int, error) {
	c.lists[key] = append(c.lists[key], value...)
	return len(c.lists[key]), nil
}

func (c mockConn) BRPop(timeout time.Duration, keys ...string) ([]string, error) {
	// TODO: add support for timeouts?
	for _, key := range keys {
		if len(c.lists[key]) > 0 {
			v := c.lists[key][0]
			c.lists[key] = c.lists[key][1:]
			return []string{key, v}, nil
		}
	}

	return nil, nil
}

func (c mockConn) ZAddNX(key string, score float64, member string) (int, error) {
	// handle the NX part
	if set, ok := c.sets[key]; ok {
		if _, ok := set[member]; ok {
			return 0, nil
		}
	}

	if _, ok := c.sets[key]; !ok {
		c.sets[key] = make(map[string]float64)
	}

	c.sets[key][member] = score
	return 1, nil
}
func (c mockConn) ZRem(key string, members ...string) (int, error) {
	if _, ok := c.sets[key]; !ok {
		return 0, nil
	}

	nRemoved := 0
	for _, m := range members {
		if _, ok := c.sets[key][m]; ok {
			delete(c.sets[key], m)
			nRemoved++
		}
	}

	return nRemoved, nil
}
func (c mockConn) ZRangeByScore(key string, opt ZRangeByScoreOpts) ([]string, error) {
	if _, ok := c.sets[key]; !ok {
		c.sets[key] = make(map[string]float64)
	}

	var members []string
	for m, s := range c.sets[key] {
		if (opt.MinInclusive && s < opt.Min) ||
			(!opt.MinInclusive && s <= opt.Min) ||
			(opt.MaxInclusive && s > opt.Max) ||
			(!opt.MaxInclusive && s >= opt.Min) {
			continue
		}

		members = append(members, m)
	}

	lo := opt.Offset
	if len(members) < opt.Offset {
		lo = len(members)
	}
	hi := lo + opt.Count
	if hi > len(members) {
		hi = len(members)
	}

	return members[lo:hi], nil
}

func (c mockConn) Close() error { return nil }
