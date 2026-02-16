package app

import "testing"

func TestParseSearchFilter(t *testing.T) {
	cases := []struct {
		in     string
		wantSP string
		wantQ  string
	}{
		{in: "cats", wantSP: "EgIQAQ%253D%253D", wantQ: "cats"},
		{in: ":hour cats", wantSP: "EgIIAQ%253D%253D", wantQ: "cats"},
		{in: ":today cats", wantSP: "EgIIAg%253D%253D", wantQ: "cats"},
		{in: ":week cats", wantSP: "EgIIAw%253D%253D", wantQ: "cats"},
		{in: ":month cats", wantSP: "EgIIBA%253D%253D", wantQ: "cats"},
		{in: ":year cats", wantSP: "EgIIBQ%253D%253D", wantQ: "cats"},
	}
	for _, tc := range cases {
		sp, q := ParseSearchFilter(tc.in)
		if sp != tc.wantSP || q != tc.wantQ {
			t.Fatalf("input=%q got=(%q,%q) want=(%q,%q)", tc.in, sp, q, tc.wantSP, tc.wantQ)
		}
	}
}
