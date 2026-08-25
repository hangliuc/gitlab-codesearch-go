package gitlab

import "testing"

func TestNormalizeURL(t *testing.T) {
	got, err := normalizeURL("gitlab.example.com/")
	if err != nil || got != "https://gitlab.example.com" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := normalizeURL("ftp://gitlab.example.com"); err == nil {
		t.Fatal("expected invalid scheme to fail")
	}
}
