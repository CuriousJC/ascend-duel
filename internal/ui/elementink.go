package ui

// **Coloring the element words in screen text**, which is the `text/v2` half of what
// `cards.ElementHighlights` does on a card face.
//
// Two rasterizers draw this game's words: `internal/cards` sets a card's own text with
// golang.org/x/image, and everything else on screen goes through Ebitengine's `text/v2`. They share
// no code and cannot — see internal/cards/text.go for why that package must not make an
// `*ebiten.Image`. So the *vocabulary* is shared instead: `cards.ElementSpans` is the one table, and
// this file is the second reader of it.
//
// **The coloring happens here rather than in `internal/systems`** because that package cannot see
// `internal/cards` — `cards` imports `systems`, so the arrow only goes one way. The widget draws
// spans and never learns why one is colored, exactly as it never learns why a line breaks where it
// does.

import (
	"github.com/curiousjc/ascend-duel/internal/carddesc"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// TipLines is a set of authored lines with their element words picked out — what every caller hands
// `models.Tooltip.Point`.
//
// **One door, so a tooltip cannot be built uncolored.** Every scene that points a tooltip goes
// through this, which is what stops a new one shipping as the only panel in the game whose relic
// text is gray.
func TipLines(lines []string) []models.TipLine {
	if len(lines) == 0 {
		return nil
	}
	out := make([]models.TipLine, 0, len(lines))
	for _, line := range lines {
		out = append(out, TipLine(line))
	}
	return out
}

// TipLine cuts one line into the spans it is drawn as.
//
// **A line with nothing to color comes back as one span**, which is what most of them are and is
// drawn exactly as it was before spans existed.
func TipLine(line string) models.TipLine {
	var out models.TipLine
	for _, seg := range chromatic(line) {
		out = append(out, models.TextSpan{Text: seg.Text, Ink: seg.Ink, Texture: seg.Texture})
	}
	return out
}

// chromatic is one line cut into its colored segments — the element words first, then CHROMATIC
// into the wildcard's own wash.
//
// **The two passes are in that order and cannot be swapped.** `SplitWash` only touches a segment
// nothing has colored, so running the element vocabulary first is what lets a word be an element
// or the wildcard and never both.
//
// **The wash cut is `internal/cards`' and not this file's**, because `tools/upgradesheet` prints
// this same title and cannot import a package that links Ebitengine — the argument `ElementSpans`
// already exists for, and the one that put the tooltip's wording in `internal/carddesc`.
func chromatic(line string) []cards.Segment {
	var out []cards.Segment
	for _, seg := range cards.SplitSpans(line, cards.ElementSpans(line)) {
		out = append(out, cards.SplitWash(seg, carddesc.Chromatic, systems.UpgradeWild)...)
	}
	// **The forms are the last pass and that is not arbitrary.** Every pass before it claims words
	// out of a vocabulary of its own, and each one only looks at a segment nothing has colored — so
	// the order is what guarantees a word is one thing. The forms go last because theirs is the
	// newest and least settled vocabulary: a collision with an element or a metal should resolve in
	// favor of the older one rather than against it.
	return cards.SplitRarities(cards.SplitForms(cards.SplitMetals(out)))
}
