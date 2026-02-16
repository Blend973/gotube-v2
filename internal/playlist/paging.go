package playlist

type Window struct {
	Start int
	End   int
	Size  int
}

func NewWindow(size int) Window {
	if size <= 0 {
		size = 30
	}
	return Window{Start: 1, End: size, Size: size}
}

func (w *Window) Next() {
	w.Start += w.Size
	w.End += w.Size
}

func (w *Window) Previous() {
	w.Start -= w.Size
	if w.Start <= 0 {
		w.Start = 1
	}
	w.End -= w.Size
	if w.End < w.Size {
		w.End = w.Size
	}
}

func (w *Window) Reset() {
	w.Start = 1
	w.End = w.Size
}
