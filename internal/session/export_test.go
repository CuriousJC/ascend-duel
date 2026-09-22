package session

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// theExportMoment is a fixed clock, so the name and the stamp in the file can be asserted at all.
var theExportMoment = time.Date(2026, 9, 16, 14, 12, 33, 0, time.UTC)

// noted is a record carrying a sentence, for the tests in this package.
//
// **The words a record reads as are decided above this package** — see internal/ui/ledger_prose.go
// — so a session test cannot reach the real translator without importing upward. What these tests
// hold is that a run stores what it is given and an export carries what it is handed, neither of
// which is a question about the wording, so a stub is the honest fixture rather than a shortcut.
func noted(voice, text string) LedgerRecord {
	return LedgerRecord{Kind: KindAct, Side: voice, Note: text}
}

// testWords is the Worder those records read back through.
func testWords(recs []LedgerRecord) []LedgerLine {
	out := make([]LedgerLine, 0, len(recs))
	for _, r := range recs {
		out = append(out, Line(r.Side, r.Note))
	}
	return out
}

// **An export says what the panel says, in the panel's own words.** Both read the same records
// through the same translator, so a test on the flattened line is a test that the export carries
// the words it was handed rather than paraphrasing what the run recorded.
func TestTheExportCarriesTheLedgersOwnWords(t *testing.T) {
	s := New(testDeck())
	s.BeginFight(1, "Giant Bat")
	s.RecordRound([]LedgerRecord{noted(VoiceYou, "Duelist attacks with a fire strike")}, 40)
	s.RecordAfter([]LedgerRecord{noted(VoicePlain, "Took Jab")})
	s.EndFight(OutcomeWon)

	out := s.ExportLedger("0009D4", theExportMoment, testWords)
	if out.Seed != "0009D4" {
		t.Errorf("the export names run %q", out.Seed)
	}
	if out.Exported != "2026-09-16T14:12:33Z" {
		t.Errorf("the export is stamped %q", out.Exported)
	}
	if len(out.Fights) != 1 {
		t.Fatalf("the export holds %d fights, want 1", len(out.Fights))
	}
	f := out.Fights[0]
	if f.Enemy != "Giant Bat" || f.Outcome != OutcomeWon || f.Dealt != 40 || f.Rounds != 1 {
		t.Errorf("the fight exported as %+v", f)
	}
	if len(f.Log) != 1 || len(f.Log[0].Lines) != 1 {
		t.Fatalf("the fight exported with %d rounds", len(f.Log))
	}
	line := f.Log[0].Lines[0]
	if line.Voice != VoiceYou || line.Text != "Duelist attacks with a fire strike" {
		t.Errorf("the line exported as %+v", line)
	}
	if len(f.After) != 1 || f.After[0].Text != "Took Jab" {
		t.Errorf("the aftermath exported as %+v", f.After)
	}
}

// **A fight still being fought says so in a word.** An empty outcome is an absence the panel reads
// against the run it is drawing; a file read months later has nothing to read it against.
func TestAnUnfinishedFightExportsAsFighting(t *testing.T) {
	s := New(testDeck())
	s.BeginFight(1, "Giant Bat")
	s.RecordRound([]LedgerRecord{noted(VoiceYou, "Duelist attacks")}, 10)

	out := s.ExportLedger("0009D4", theExportMoment, testWords)
	if len(out.Fights) != 1 || out.Fights[0].Outcome != "fighting" {
		t.Fatalf("the open fight exported as %+v", out.Fights)
	}
}

// **The name carries the run and the moment**, because either alone collides: one run exported
// twice would overwrite itself.
func TestTheExportIsNamedForTheRunAndTheMoment(t *testing.T) {
	name := LedgerExportName("0009D4", theExportMoment)
	if want := "fightlog-0009D4-20260916-141233.json"; name != want {
		t.Errorf("the export is called %q, want %q", name, want)
	}
}

// **It is JSON somebody can read.** The export exists to be opened outside the game, so the one
// thing it may not be is a structure that needs this build to make sense of.
func TestTheExportIsReadableJSON(t *testing.T) {
	s := New(testDeck())
	s.BeginFight(3, "Giant Bat")
	s.RecordRound([]LedgerRecord{noted(VoiceHand, "Elemental Four of a Kind, 69")}, 69)
	s.EndFight(OutcomeLost)

	raw, err := json.MarshalIndent(s.ExportLedger("0009D4", theExportMoment, testWords), "", "  ")
	if err != nil {
		t.Fatalf("the export must marshal: %v", err)
	}
	for _, want := range []string{`"seed": "0009D4"`, `"enemy": "Giant Bat"`, `"outcome": "lost"`,
		`"text": "Elemental Four of a Kind, 69"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("the file does not say %s", want)
		}
	}
}
