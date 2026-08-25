package search

import "testing"

func TestEncodePath(t *testing.T) {
	if got := encodePath("dir name/a#b.go"); got != "dir%20name/a%23b.go" {
		t.Fatalf("unexpected path: %s", got)
	}
}

func TestMatchLineNumberUsesKeywordPositionInSnippet(t *testing.T) {
	snippet := "self.sentry_url = 'https://example.com'\nself.data = {\n    'username': 'target-key',\n}"
	if got := matchLineNumber(7, snippet, "target-key"); got != 9 {
		t.Fatalf("got line %d, want 9", got)
	}
}

func TestMatchLineNumberFallsBackToSnippetStart(t *testing.T) {
	if got := matchLineNumber(7, "no match", "target"); got != 7 {
		t.Fatalf("got line %d, want 7", got)
	}
}
