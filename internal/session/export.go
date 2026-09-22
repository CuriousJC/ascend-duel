package session

// **The run's account, written out as a file somebody can read.**
//
// The ledger is already saved — `run.json` carries it so a resumed run keeps its record — but that
// snapshot is written for the game to read back: spans with ink names, folded into the run's own
// file, gone the moment the run is abandoned. This is the other direction. It is one file per
// export, named for the run and the moment, holding what happened in plain sentences, so a fight
// can be pasted into a bug report or read months later without the build that wrote it.
//
// **The words are the panel's own.** This package stores records rather than sentences — see
// record.go — and deciding what a record reads as needs the drawing layer, which sits above this
// one. So the caller hands the translator in and an export cannot disagree with the panel it is an
// export of: they are the same function.
//
// **Spans collapse to one string, deliberately.** The inks are a picture of the screen — which
// element a figure came off, which relic priced it — and a reader outside the game has no palette
// to map them onto. The voice survives, because who is speaking is the one thing a flat line loses
// that a reader actually needs.

import (
	"fmt"
	"time"
)

// LedgerExport is one run's account as a file: what run it was, when it was written, and every
// fight in it.
//
// **Every field is a name, a number or a sentence.** It is read by people and by whatever they
// paste it into, so nothing here may be an ordinal — the rule the saved snapshot is under, for a
// file that outlives its build even harder than a save does.
type LedgerExport struct {
	// Seed is the run code, so the export names the run it came from in the form a player can type
	// back in.
	Seed string `json:"seed"`

	// Exported is when the file was written, RFC 3339 with the machine's own offset.
	Exported string `json:"exported"`

	// Floor is how far the climb had got when the export was taken.
	Floor int `json:"floor"`

	Fights []ExportFight `json:"fights"`
}

// ExportFight is one duel.
type ExportFight struct {
	Number int    `json:"fight"`
	Floor  int    `json:"floor"`
	Enemy  string `json:"enemy"`

	// Outcome is "won", "lost", or "fighting" for the duel that was still being fought.
	Outcome string `json:"outcome"`

	// Dealt is what the player's blows came to across the fight, and Rounds how long it ran —
	// the two figures the panel's own heading prints.
	Dealt  int `json:"dealt"`
	Rounds int `json:"rounds"`

	Log []ExportRound `json:"log"`

	// After is what the player did between this fight and the next, in the ledger's wording.
	After []ExportLine `json:"after,omitempty"`
}

// ExportRound is one round of one fight.
type ExportRound struct {
	Number int          `json:"round"`
	Lines  []ExportLine `json:"lines"`
}

// ExportLine is one line of the account: who spoke and what was said.
type ExportLine struct {
	Voice string `json:"voice"`
	Text  string `json:"text"`
}

// ExportLedger is the run's account as a written record, taken at the moment given.
//
// **The clock is a parameter rather than a call to time.Now.** Nothing in this package reads a
// wall clock — see the determinism rules — and an export is not a rule, but a function that took
// one would be the first, and could not be tested to the second either.
func (s *Session) ExportLedger(code string, at time.Time, words Worder) LedgerExport {
	if words == nil {
		words = func([]LedgerRecord) []LedgerLine { return nil }
	}
	out := LedgerExport{
		Seed:     code,
		Exported: at.Format(time.RFC3339),
		Floor:    s.Floor(),
		Fights:   make([]ExportFight, 0, len(s.ledger.Fights)),
	}
	for _, f := range s.ledger.Fights {
		rec := ExportFight{
			Number:  f.Number,
			Floor:   f.Floor,
			Enemy:   f.Enemy,
			Outcome: exportOutcome(f.Outcome),
			Dealt:   f.Dealt(),
			Rounds:  f.RoundCount(),
			Log:     make([]ExportRound, 0, len(f.Rounds)),
			After:   exportLines(words(f.After)),
		}
		for _, r := range f.Rounds {
			rec.Log = append(rec.Log, ExportRound{
				Number: r.Number, Lines: exportLines(words(r.Records)),
			})
		}
		out.Fights = append(out.Fights, rec)
	}
	return out
}

// Worder turns a block of records into the lines a panel would draw.
//
// **A parameter rather than an import**, because the words live in `internal/ui` and this package
// sits below it. One function serves the panel and the file, which is what stops an export from
// reading differently to the account it was taken of.
type Worder func([]LedgerRecord) []LedgerLine

// LedgerExportName is what the file is called: the run it is of, and the moment it was taken.
//
// **Both, because either alone collides.** One run exported twice would overwrite itself with only
// the code, and two runs exported in the same second is not a thing a player can do — so the pair
// is the one name that never eats an earlier export.
func LedgerExportName(code string, at time.Time) string {
	return fmt.Sprintf("fightlog-%s-%s.json", code, at.Format("20060102-150405"))
}

// exportOutcome words an unfinished fight. **A word rather than an empty string**, because a
// reader outside the game has nothing to read an absence against.
func exportOutcome(outcome string) string {
	if outcome == "" {
		return "fighting"
	}
	return outcome
}

// exportLines flattens a block of ledger lines.
func exportLines(in []LedgerLine) []ExportLine {
	if len(in) == 0 {
		return nil
	}
	out := make([]ExportLine, 0, len(in))
	for _, l := range in {
		out = append(out, ExportLine{Voice: l.Voice, Text: l.Text()})
	}
	return out
}
