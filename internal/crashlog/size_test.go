package crashlog

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// fatFight is one ledger fight big enough to be worth shedding.
func fatFight(n int) session.LedgerFight {
	return session.LedgerFight{Number: n, Enemy: strings.Repeat("x", 4096)}
}

// TestShedGoesBottomUp is the order the whole ceiling rests on: the screen's own account first, the
// problem ring next, the ledger a fight at a time after that, and the identity never.
func TestShedGoesBottomUp(t *testing.T) {
	quiet(t)
	r := Report{
		Version:  "v1.2.3",
		Scene:    map[string]any{"round": 3},
		Problems: []Problem{{Tick: 1, What: "something"}},
		Ledger:   []session.LedgerFight{fatFight(1), fatFight(2)},
	}

	want := []string{shedScene, shedProblems, shedLedger, shedLedger}
	for i, name := range want {
		if !shed(&r) {
			t.Fatalf("shed %d: nothing left to drop, want %s", i, name)
		}
	}
	if shed(&r) {
		t.Fatal("shed kept going with every sheddable tier already gone")
	}

	if r.Scene != nil || r.Problems != nil || len(r.Ledger) != 0 {
		t.Fatalf("a tier survived being shed: %+v", r)
	}
	if r.Version != "v1.2.3" {
		t.Fatal("the identity was shed, and it is the one tier that may never be")
	}

	// **One ledger note however many fights went**, or the shedding would take up the room it was
	// making.
	if got := strings.Join(r.Shed, ","); got != shedScene+","+shedProblems+","+shedLedger {
		t.Fatalf("Shed = %q, want one entry per tier", got)
	}
}

// TestAFatReportIsBroughtUnderTheCeiling is the whole point of Encode: a report too big to send is a
// report nobody reads.
func TestAFatReportIsBroughtUnderTheCeiling(t *testing.T) {
	quiet(t)
	r := Report{
		Version: "v1.2.3",
		Panic:   "boom",
		Scene:   map[string]any{"blob": strings.Repeat("x", maxReport)},
	}

	raw, err := Encode(r)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(raw) > maxReport {
		t.Fatalf("the report is %d bytes and the ceiling is %d", len(raw), maxReport)
	}

	var got Report
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("the shed report is not readable: %v", err)
	}
	if got.Panic != "boom" || got.Version != "v1.2.3" {
		t.Fatalf("shedding took the identity with it: %+v", got)
	}
	if len(got.Shed) == 0 {
		t.Fatal("a partial report does not say it is partial")
	}
}

// TestASmallReportShedsNothing holds the other half: the ordinary report is written whole, and a
// `shed` key on one would be a reader told to go looking for something that was never left out.
func TestASmallReportShedsNothing(t *testing.T) {
	quiet(t)
	raw, err := Encode(Report{Version: "v1", Scene: map[string]any{"round": 2}})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var got Report
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Shed != nil || got.Scene == nil {
		t.Fatalf("a report under the ceiling lost something: %+v", got)
	}
}

// TestAHugeScreenshotIsLeftOut is the picture's own ceiling. It is a separate number from the
// document's because it is a separate file, and this is the case that says so: a picture that will
// not fit costs the report nothing.
func TestAHugeScreenshotIsLeftOut(t *testing.T) {
	quiet(t)

	// Incompressible noise rather than a flat fill or a pattern. PNG squeezes either of those down
	// to nothing, so the test would be about the encoder rather than about the ceiling — a
	// gradient over a million pixels came back at 131 KB. The shift register is the one
	// internal/music's drum part uses, and for the same reason: no math/rand anywhere.
	img := image.NewRGBA(image.Rect(0, 0, 1024, 1024))
	state := uint32(0x9E3779B9)
	for i := range img.Pix {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		img.Pix[i] = uint8(state)
	}
	if shot := EncodeShot(img); shot != nil {
		t.Fatalf("a %d-byte screenshot was filed over a ceiling of %d", len(shot), maxShot)
	}

	if shot := EncodeShot(nil); shot != nil {
		t.Fatal("EncodeShot invented a picture out of nothing")
	}
}

// TestAScreenshotIsFiledBesideItsReport holds the naming rule that makes the two travel together —
// and, because Prune sweeps a report's siblings by base name, the one that keeps a picture from
// outliving the report it belongs to.
func TestAScreenshotIsFiledBesideItsReport(t *testing.T) {
	quiet(t)
	dir := t.TempDir()
	s := profile.At(dir)

	shot := EncodeShot(image.NewRGBA(image.Rect(0, 0, 16, 16)))
	if shot == nil {
		t.Fatal("a small picture would not encode")
	}

	path, err := Write(s, Build(State{RunSeed: 1}, "boom", nil), shot)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	base := strings.TrimSuffix(filepath.Base(path), fileSuffix)
	if _, err := os.Stat(filepath.Join(dir, base+shotSuffix)); err != nil {
		t.Fatalf("the screenshot is not beside its report: %v", err)
	}

	raw, _ := os.ReadFile(path)
	var got Report
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Screenshot != base+shotSuffix {
		t.Fatalf("Screenshot = %q, want the file that was written", got.Screenshot)
	}
}

// TestAReportWithNoPictureSaysNothingAboutOne is the absence rule: a panic in Update happens
// between two frames, so there is no screen to read and the field is simply not there.
func TestAReportWithNoPictureSaysNothingAboutOne(t *testing.T) {
	quiet(t)
	s := profile.At(t.TempDir())

	path, err := Write(s, Build(State{RunSeed: 1}, "boom", nil), nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "screenshot") {
		t.Fatal("a report with no picture claims one")
	}
}

// TestTheSceneTierRidesAlong holds the one wire between a screen and the file: what ui.Reporter
// hands over is what a reader gets.
func TestTheSceneTierRidesAlong(t *testing.T) {
	quiet(t)
	r := Build(State{RunSeed: 1, Scene: map[string]any{"round": 4, "cursor": 11}}, "boom", nil)
	raw, err := Encode(r)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var got Report
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Scene["round"] != float64(4) || got.Scene["cursor"] != float64(11) {
		t.Fatalf("scene = %v, want what the screen said", got.Scene)
	}
}
