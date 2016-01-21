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
	q := koda.GetQueue("test")

	j, err := q.SubmitDelayed(nil, 5*time.Hour)
	fmt.Println("--before-- Got ID:", j.ID, err)

	for {
		j, err = q.Wait()
		fmt.Println("--after--- Got ID:", j.ID, err)
		fmt.Println(j, err)

		time.Sleep(time.Second * 2)

		j.Logf("omgtest: %s", "hithere")
		j.Done()
	}
}
