package yajq_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cjlucas/unnamedcast/yajq"
)

type blah struct {
	I int
}

func TestSomething(t *testing.T) {
	q := yajq.New(&yajq.QueueInfo{
		Name: "test",
	})
	j, err := q.SubmitDelayed(100, 5, time.Now())
	fmt.Println(j, err)

	j, err = q.Wait()
	fmt.Println(j, err)

	j.Logf("omgtest: %s", "hithere")
	j.UpdateProgress(50)
}
