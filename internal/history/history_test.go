package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBangSelection(t *testing.T) {
	lines := []string{"one", "two", "three"}
	if got := ResolveBangSelection("!1", lines); got != "three" {
		t.Fatalf("got %q", got)
	}
	if got := ResolveBangSelection("!3", lines); got != "one" {
		t.Fatalf("got %q", got)
	}
}

func TestAppendUnique(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "history.txt")
	if err := AppendUnique(f, "cats"); err != nil {
		t.Fatal(err)
	}
	if err := AppendUnique(f, "dogs"); err != nil {
		t.Fatal(err)
	}
	if err := AppendUnique(f, "cats"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "dogs\ncats\n" {
		t.Fatalf("unexpected history: %q", string(data))
	}
}
