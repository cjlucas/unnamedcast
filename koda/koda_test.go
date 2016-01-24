package koda_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cjlucas/unnamedcast/koda"
)

type blah struct {
	I int
}

func TestSomething(t *testing.T) {
}

func TestAllJobs(t *testing.T) {
	q := koda.GetQueue("test")

	for i := 0; i < 100000; i++ {
		q.Submit(0, nil)
	}

	now := time.Now()
	jobs, err := q.AllJobs()
	d := time.Now().Sub(now)
	fmt.Println(d, len(jobs), err)
}
