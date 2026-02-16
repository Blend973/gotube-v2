package playlist

import "testing"

func TestPaging(t *testing.T) {
	w := NewWindow(30)
	w.Next()
	if w.Start != 31 || w.End != 60 {
		t.Fatalf("next failed: %+v", w)
	}
	w.Previous()
	if w.Start != 1 || w.End != 30 {
		t.Fatalf("prev failed: %+v", w)
	}
	w.Previous()
	if w.Start != 1 || w.End != 30 {
		t.Fatalf("underflow bounds failed: %+v", w)
	}
}
