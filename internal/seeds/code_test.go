package seeds

import "testing"

// The codec's whole job is that a code written down names the same run when it comes back, so
// these tests are about round-tripping and about the two ways a code reaches the game wrong:
// typed in the other case, and typed with a character the alphabet has not got.

func TestEverySeedInTheSpaceRoundTrips(t *testing.T) {
	// A sample rather than all 2.1 billion, taken with a stride that is coprime with the base
	// so it does not only ever land on codes ending in the same digit.
	for n := int64(0); n < Space; n += 7_919_003 {
		code := Code(n)
		if len(code) != CodeLen {
			t.Fatalf("Code(%d) = %q, want %d characters", n, code, CodeLen)
		}
		back, err := Parse(code)
		if err != nil {
			t.Fatalf("Parse(Code(%d)) = %v", n, err)
		}
		if back != n {
			t.Fatalf("round trip of %d came back as %d via %q", n, back, code)
		}
	}
}

func TestTheEndsOfTheSpaceAreCodes(t *testing.T) {
	if got := Code(0); got != "000000" {
		t.Fatalf("Code(0) = %q, want 000000", got)
	}
	if got := Code(Space - 1); got != "ZZZZZZ" {
		t.Fatalf("Code(Space-1) = %q, want ZZZZZZ", got)
	}
}

// Case is not information: a code is read off a screen by someone with no reason to know whether
// the letter was capital.
func TestCaseIsNotInformation(t *testing.T) {
	for _, code := range []string{"abcd12", "ABCD12", "AbCd12", "  abcd12  "} {
		got, err := Parse(code)
		if err != nil {
			t.Fatalf("Parse(%q) = %v", code, err)
		}
		want, _ := Parse("ABCD12")
		if got != want {
			t.Fatalf("Parse(%q) = %d, want %d", code, got, want)
		}
	}
}

func TestACodeIsAlwaysUpperCase(t *testing.T) {
	for n := int64(0); n < 5000; n++ {
		for _, c := range Code(n) {
			if c >= 'a' && c <= 'z' {
				t.Fatalf("Code(%d) = %q has a lower-case letter", n, Code(n))
			}
		}
	}
}

func TestWhatIsNotACodeIsRefused(t *testing.T) {
	// `U` is in the list on purpose: it is excluded from the alphabet and, unlike I/L/O, does
	// not fold, because it is not a mistake for anything.
	for _, bad := range []string{"", "ABC", "ABCDEFG", "ABCD-2", "ABCD 2", "ABÇD12", "!@#$%^", "ABCDU2", "UUUUUU"} {
		if _, err := Parse(bad); err == nil {
			t.Fatalf("Parse(%q) was accepted", bad)
		}
	}
}

// Normalize is what lets the clock name a run that can be written down, and it has to land in
// range for a negative input as well as a huge one.
func TestNormalizeLandsInTheSpace(t *testing.T) {
	for _, n := range []int64{0, -1, 1, Space, Space - 1, -Space - 1, 1 << 62, -(1 << 62)} {
		got := Normalize(n)
		if got < 0 || got >= Space {
			t.Fatalf("Normalize(%d) = %d, outside [0,%d)", n, got, Space)
		}
		// The proof that it is usable: it can be rendered without panicking.
		if len(Code(got)) != CodeLen {
			t.Fatalf("Normalize(%d) did not produce a codeable seed", n)
		}
	}
}

// A seed outside the space is a programmer error and must be loud, not folded — quietly
// rendering a number that will not round-trip is how a shared code comes to name a different run.
func TestCodeRefusesASeedOutsideTheSpace(t *testing.T) {
	for _, n := range []int64{-1, Space, Space + 1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("Code(%d) did not panic", n)
				}
			}()
			Code(n)
		}()
	}
}

// The four dropped characters are the whole point of the reduced alphabet, so both halves are
// pinned: Code never emits one, and Parse still understands the three that are mistakes.

func TestNoCodeContainsAConfusableCharacter(t *testing.T) {
	for n := int64(0); n < Space; n += 7_919_003 {
		code := Code(n)
		for _, c := range code {
			switch c {
			case 'I', 'L', 'O', 'U':
				t.Fatalf("Code(%d) = %q contains %q", n, code, c)
			}
		}
	}
}

func TestTheConfusableLettersFoldToTheDigitTheyLookLike(t *testing.T) {
	want, err := Parse("011234")
	if err != nil {
		t.Fatalf("Parse(011234) = %v", err)
	}
	for _, typed := range []string{"OII234", "oil234", "OLI234", "0Il234"} {
		got, err := Parse(typed)
		if err != nil {
			t.Fatalf("Parse(%q) = %v, want it folded", typed, err)
		}
		if got != want {
			t.Fatalf("Parse(%q) = %d, want %d", typed, got, want)
		}
	}
}
