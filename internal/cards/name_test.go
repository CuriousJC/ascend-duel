package cards

import "testing"

// **A hand card's name that would run into the badge breaks onto two lines**, at the space that
// balances them, and each line then fits the band. A name that fits stays one line, and a single
// word stays whole however long it is, because there is nowhere to break it.
func TestALongNameBreaksOntoTwoLines(t *testing.T) {
	f := faces(t)
	for _, tc := range []struct {
		name string
		want []string
	}{
		{"Strike", nil},
		{"Toxin Flood", []string{"Toxin", "Flood"}},
		{"Needle Rain", []string{"Needle", "Rain"}},
	} {
		lines, size, ok, err := nameLines(f, tc.name, Hand)
		if err != nil {
			t.Fatal(err)
		}
		if tc.want == nil {
			if ok {
				t.Errorf("%q fits one line and was broken into %q", tc.name, lines)
			}
			continue
		}
		if !ok || len(lines) != 2 || lines[0] != tc.want[0] || lines[1] != tc.want[1] {
			t.Errorf("%q broke into %q, want %q", tc.name, lines, tc.want)
			continue
		}
		for _, line := range lines {
			w, _, err := f.Measure(size, line)
			if err != nil {
				t.Fatal(err)
			}
			if w > Hand.NameWrapWidth {
				t.Errorf("%q: the line %q is %dpx where the band holds %d", tc.name, line, w, Hand.NameWrapWidth)
			}
		}
	}

	if _, _, ok, _ := nameLines(f, "Unbreakablyverylongname", Hand); ok {
		t.Error("a single word was broken, and there is no space to break it at")
	}
}
