package systems

// **Wrapping a tooltip line, runs and all.**
//
// The panel used to be measured against lines its caller had already broken — a relic's authored
// `\n`, a hand-counted phrase in `ui.RoundTimerTip`. That put a *layout* decision in every file
// that writes a sentence, and it was a decision none of them could make: an author guessing where
// a line breaks is guessing at a font size, a panel width and a wearer's name length all at once,
// and is wrong the moment any of the three moves.
//
// **An authored break survives and can only ever add a line**, which is the rule the card faces
// are already under — a caller that splits on `\n` is handing over several lines, and each of them
// is wrapped here on its own. So a break still forces one and nothing forces a line to run wide.

import (
	"strings"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// tipMaxW is how wide the type may run before it wraps, in pixels, before the padding is added.
//
// **A measure rather than a character count.** Roughly sixty characters of Kubasta at the body
// size, which is about where a line stops being scannable at a glance — a tooltip is read while
// deciding, so it is a column of short lines rather than a paragraph.
const tipMaxW = 690

// WrapRuns breaks one line into as many as the measure needs, keeping every run's ink.
//
// **Exported because the tutorial bubble wraps too**, and had its own wrapper doing the same job
// over plain strings — which is why Bob was the one voice in the game that named the elements
// without coloring them. One wrapper over runs answers both.
//
// **A line with no spaces is never broken**, so a long single token runs wide rather than being cut
// mid-word. That is the card text band's posture: the strings are authored in this repo, so an
// overrun is an authoring mistake to see rather than a case to hide.
//
// **A blank line comes back as itself.** Callers use one as a spacer between a status and the next,
// and a wrapper that dropped it would close up the panel's only paragraph break.
func WrapRuns(line models.TipLine, face *text.GoTextFace, max float64) []models.TipLine {
	if len(line) == 0 {
		return []models.TipLine{line}
	}

	var out []models.TipLine
	var cur models.TipLine
	width := 0.0

	// flush ends the line being built. It is called between words, so a run split across a break
	// keeps its ink on both sides of it.
	flush := func() {
		out = append(out, cur)
		cur, width = nil, 0
	}

	for _, run := range line {
		if run.Text == "" {
			// An empty run carries no words but may be the whole of a spacer line.
			if len(cur) == 0 && len(out) == 0 {
				cur = append(cur, run)
			}
			continue
		}
		for _, word := range splitKeepingSpaces(run.Text) {
			w := measureText(word, face)
			// **Only a word with something in front of it may start a line**, so a leading space
			// never lands at the head of a wrapped line and a break never produces an empty one.
			if width+w > max && len(cur) > 0 && strings.TrimSpace(word) != "" {
				flush()
				word = strings.TrimLeft(word, " ")
				w = measureText(word, face)
			}
			cur = appendRun(cur, models.TextSpan{Text: word, Ink: run.Ink, Texture: run.Texture})
			width += w
		}
	}
	if len(cur) > 0 || len(out) == 0 {
		flush()
	}
	return out
}

// appendRun adds a word to the line, joining it to the run before it when the two share an ink.
//
// **Joined rather than one span per word**, because the panel measures and places run by run: a
// line of twelve one-word spans is eleven joins where kerning is lost, and it reads as type set
// slightly too loose. One span per stretch of one color is the same shape the coloring produced.
//
// **The material has to match as well as the ink**, because a textured run is rendered whole into
// its own image: joining it to the plain run beside it would put the grain through both.
func appendRun(line models.TipLine, run models.TextSpan) models.TipLine {
	if n := len(line); n > 0 && line[n-1].Ink == run.Ink && line[n-1].Texture == run.Texture {
		line[n-1].Text += run.Text
		return line
	}
	return append(line, run)
}

// splitKeepingSpaces cuts a run into words with their leading spaces still attached, so that
// re-joining the pieces reproduces the run exactly. A wrapper that split on spaces and rejoined
// with one would quietly normalize the two spaces after a full stop.
func splitKeepingSpaces(s string) []string {
	var out []string
	start := 0
	for i := 1; i < len(s); i++ {
		if s[i] == ' ' && s[i-1] != ' ' {
			out = append(out, s[start:i])
			start = i
		}
	}
	return append(out, s[start:])
}

// measureText is one string's advance in a face. Wrapped so the wrapper and the panel's own
// measurement cannot drift apart.
func measureText(s string, face *text.GoTextFace) float64 {
	w, _ := text.Measure(s, face, 0)
	return w
}
