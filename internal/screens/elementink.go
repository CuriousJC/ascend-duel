package screens

// **Colouring the element words in screen text**, which is the `text/v2` half of what
// `cards.ElementHighlights` does on a card face.
//
// Two rasterisers draw this game's words: `internal/cards` sets a card's own text with
// golang.org/x/image, and everything else on screen goes through Ebitengine's `text/v2`. They share
// no code and cannot — see internal/cards/text.go for why that package must not make an
// `*ebiten.Image`. So the *vocabulary* is shared instead: `cards.ElementRuns` is the one table, and
// this file is the second reader of it.
//
// **The colouring happens here rather than in `internal/systems`** because that package cannot see
// `internal/cards` — `cards` imports `systems`, so the arrow only goes one way. The widget draws
// runs and never learns why one is coloured, exactly as it never learns why a line breaks where it
// does.

import (
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/models"
)

// tipLines is a set of authored lines with their element words picked out — what every caller hands
// `models.Tooltip.Point`.
//
// **One door, so a tooltip cannot be built uncoloured.** Every scene that points a tooltip goes
// through this, which is what stops a new one shipping as the only panel in the game whose ring
// text is grey.
func tipLines(lines []string) []models.TipLine {
	if len(lines) == 0 {
		return nil
	}
	out := make([]models.TipLine, 0, len(lines))
	for _, line := range lines {
		out = append(out, tipLine(line))
	}
	return out
}

// tipLine cuts one line into the runs it is drawn as.
//
// **A line with nothing to colour comes back as one run**, which is what most of them are and is
// drawn exactly as it was before runs existed.
func tipLine(line string) models.TipLine {
	runs := cards.ElementRuns(line)
	if len(runs) == 0 {
		return models.TipLine{{Text: line}}
	}

	var out models.TipLine
	for _, seg := range cards.SplitRuns(line, runs) {
		out = append(out, models.TextRun{Text: seg.Text, Ink: seg.Ink})
	}
	return out
}
