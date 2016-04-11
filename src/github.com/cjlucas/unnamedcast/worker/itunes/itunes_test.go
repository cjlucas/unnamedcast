package itunes_test

import (
	"testing"

	"github.com/cjlucas/unnamedcast/worker/itunes"
)

func TestFetchReviewStats(t *testing.T) {
	_, err := itunes.FetchReviewStats(528458508)
	if err != nil {
		t.Error(err)
	}
}
