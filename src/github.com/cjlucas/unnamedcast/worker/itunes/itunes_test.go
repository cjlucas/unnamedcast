package itunes_test

import (
	"testing"

	"github.com/cjlucas/unnamedcast/worker/itunes"
)

func TestFetchReviewStats(t *testing.T) {
	t.Skip("Skipping until a more reliable test can be made")

	_, err := itunes.FetchReviewStats(528458508)
	if err != nil {
		t.Error(err)
	}
}
