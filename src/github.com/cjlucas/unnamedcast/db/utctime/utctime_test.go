package utctime

import (
	"testing"
	"time"
)

func TestBefore(t *testing.T) {
	ti := time.Now()

	cases := []struct {
		lhs time.Time
		rhs time.Time
	}{
		{ti, ti},
		{ti.Add(-1 * time.Second), ti.Add(1 * time.Second)},
		{ti.Add(1 * time.Second), ti.Add(-1 * time.Second)},
	}

	for _, c := range cases {
		expected := c.lhs.Before(c.rhs)

		lhs := Time{c.lhs}
		rhs := Time{c.rhs}
		actual := lhs.Before(rhs)
		if actual != expected {
			t.Fatalf("Expected %t, got %t", expected, actual)
		}
	}
}
