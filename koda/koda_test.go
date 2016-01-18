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
	q := koda.New(&koda.QueueInfo{
		Name: "test",
	})
	j, err := q.SubmitDelayed(100, 5, time.Now())
	fmt.Println("--before-- Got ID:", j.ID, err)

	for {
		j, err = q.Wait()
		fmt.Println("--after--- Got ID:", j.ID, err)
		fmt.Println(j, err)

		time.Sleep(time.Second * 2)

		q.Logf(j, "omgtest: %s", "hithere")
		q.UpdateProgress(j, 50)
		q.Done(j)
	}
}
