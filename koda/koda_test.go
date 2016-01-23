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

	var data struct {
		I int
	}

	data.I = 9223372036854775807

	j, err := q.SubmitDelayed(2*time.Second, data)
	fmt.Println("--before-- Got ID:", j.ID, err)

	data.I = 0

	for {
		j, err = q.Wait()
		if err != nil {
			panic(err)
		}
		fmt.Println("--after--- Got ID:", j.ID, err)
		fmt.Println(j, err)
		if err != nil {
			panic(err)
		}

		if err := j.UnmarshalPayload(&data); err != nil {
			panic(err)
		}
		fmt.Println(data)

		// time.Sleep(time.Second * 2)

		j.Retry(2 * time.Second)

		j.Logf("omgtest: %s", "hithere")
		j.Kill()
		// j.Done()
	}
}
