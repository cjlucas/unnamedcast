package utctime

import (
	"testing"
	"time"
)

func TestBefore(t *testing.T) {
	cases := []struct {
		lhs time.Time
		rhs time.Time
	}{
		{time.Time{}, time.Time{}},
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
