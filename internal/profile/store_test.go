package profile

// **The two doors a name from further up goes through, and the one write that is not a whole
// document.**

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestAppendLineAddsOneLineAtATime holds the property the journal is built on: what is already in
// the file is not rewritten, so the line before a panic is already on disk.
func TestAppendLineAddsOneLineAtATime(t *testing.T) {
	dir := t.TempDir()
	s := At(dir)

	for i := 0; i < 3; i++ {
		if err := s.AppendLine("journal.jsonl", map[string]any{"n": i}); err != nil {
			t.Fatalf("appending %d: %v", i, err)
		}
	}

	raw, err := os.ReadFile(filepath.Join(dir, "journal.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	// Compact, unlike every other write here: one record per line is what makes the file
	// appendable without parsing what is above it.
	want := "{\"n\":0}\n{\"n\":1}\n{\"n\":2}\n"
	if string(raw) != want {
		t.Fatalf("the file holds %q, want %q", raw, want)
	}
}

// TestNeitherDoorTakesANameThatIsNotOne holds the rule every name from further up is under: both
// of these are reachable from a click, and one of them writes wherever it is told.
func TestNeitherDoorTakesAName(t *testing.T) {
	s := At(t.TempDir())

	for _, bad := range []string{"", "..", "../escape.json", `..\escape.json`, "sub/dir.json"} {
		if err := s.AppendLine(bad, map[string]any{}); err == nil {
			t.Fatalf("AppendLine(%q) was allowed", bad)
		}
		if _, err := s.CopyFile(bad, "fine.json"); err == nil {
			t.Fatalf("CopyFile(%q, …) was allowed", bad)
		}
		if _, err := s.CopyFile("fine.json", bad); err == nil {
			t.Fatalf("CopyFile(…, %q) was allowed", bad)
		}
	}
}

// TestCopyingWhatIsNotThereIsNotAFailure. A crash takes a copy of the journal, and a run that
// crashed before writing one still gets its report.
func TestCopyingWhatIsNotThereIsNotAFailure(t *testing.T) {
	s := At(t.TempDir())

	copied, err := s.CopyFile("journal.jsonl", "crash.jsonl")
	if err != nil {
		t.Fatalf("copying a file that is not there: %v", err)
	}
	if copied {
		t.Fatal("reported a copy of a file that does not exist")
	}
}

// TestWriteBytesLandsExactlyWhatItWasGiven is the door a crash screenshot goes through. Every other
// whole-document write here marshals a value; this one is handed bytes that are already a file, so
// what it owes is that they arrive unchanged and under a checked name.
func TestWriteBytesLandsExactlyWhatItWasGiven(t *testing.T) {
	dir := t.TempDir()
	s := At(dir)

	want := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0xFF}
	path, err := s.WriteBytes("crash-20260923T000000Z-AAAAAA.png", want)
	if err != nil {
		t.Fatalf("WriteBytes: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading it back: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %v, want the bytes that went in", got)
	}

	// **The same name check WriteExport is under**, and it is not optional: this is a second door
	// out of the store taking a name from further up.
	if _, err := s.WriteBytes("../escape.png", want); err == nil {
		t.Fatal("WriteBytes wrote outside the store")
	}

	// An inert store is the unwritable machine, which reports rather than panics.
	if _, err := (Store{}).WriteBytes("x.png", want); err == nil {
		t.Fatal("an inert store reported success with nowhere to write")
	}
}
