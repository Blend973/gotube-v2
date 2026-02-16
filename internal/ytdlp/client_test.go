package ytdlp

import "testing"

func TestParseResultRecovery(t *testing.T) {
	in := "noise before json\n{\"entries\":[{\"id\":\"abc\",\"url\":\"u\",\"title\":\"t\"}]}"
	r, err := ParseResult(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Entries) != 1 || r.Entries[0].ID != "abc" {
		t.Fatalf("unexpected parse result: %#v", r)
	}
}
