package search

import "testing"

func TestEncodePath(t *testing.T) {
	if got := encodePath("dir name/a#b.go"); got != "dir%20name/a%23b.go" {
		t.Fatalf("unexpected path: %s", got)
	}
}
