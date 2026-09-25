package tutorial

import (
	"os"
	"testing"
)

// parkTutorial skips a tutorial test unless TUTORIAL_TESTS is set *(owner's call)*. The lesson is
// being left behind by the mechanics on purpose while they move, and is to be rebuilt later rather
// than kept green through every change; `TUTORIAL_TESTS=1 go test ./...` runs them anyway.
func parkTutorial(t *testing.T) {
	t.Helper()
	if os.Getenv("TUTORIAL_TESTS") == "" {
		t.Skip("tutorial tests are parked; set TUTORIAL_TESTS=1 to run them")
	}
}
