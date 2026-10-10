package ui

// **Say: a sentence written whole, with its colored parts named in place.**
//
// A line that mixes words and figures used to be assembled in code a piece at a time — "You have ",
// then the figure in crimson, then "." — so the sentence existed nowhere as a sentence. A template
// writes it once, in data/wording.json, and the code only hands over what goes in the holes:
//
//	"You have {total}."                          // {total} is a slot the caller fills
//	"{vitae:Vitae} proliferates for each {per}"  // {vitae:Vitae} is the word Vitae in the vitae ink
//	"{mark:took} {subject}"                      // {mark:took} is the word took, bold and underlined
//
// **A slot carries its own ink.** What color a figure is written in is a fact about the figure, so
// the caller decides it and the template only says where the figure sits — which is what lets the
// sentence be reworded, or reordered, without touching the code that fills it.
//
// **A slot the caller did not fill is drawn as its own braces**, `{total}`, rather than as nothing:
// a hole in a sentence reads as a word the author meant to leave out, and a visible name reads as
// the mistake it is.

import (
	"slices"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/session"
)

// Slots is what fills a template's holes, by name. A value is spans rather than a string so a slot
// can carry an ink, a mark, or a clause already cut by ElementSpans.
type Slots map[string][]session.LedgerSpan

// inkMark is the one name a literal may take that is not an ink: `{mark:word}` marks the word.
const inkMark = "mark"

// Say fills a template. Literal text between the holes is written in the panel's own ink.
func Say(tmpl string, slots Slots) []session.LedgerSpan {
	var out []session.LedgerSpan
	for tmpl != "" {
		open := strings.IndexByte(tmpl, '{')
		rel := -1
		if open >= 0 {
			rel = strings.IndexByte(tmpl[open+1:], '}')
		}
		if rel < 0 {
			out = appendSpan(out, session.LedgerSpan{Text: tmpl})
			break
		}
		shut := open + 1 + rel
		if open > 0 {
			out = appendSpan(out, session.LedgerSpan{Text: tmpl[:open]})
		}
		hole := tmpl[open+1 : shut]
		tmpl = tmpl[shut+1:]

		if ink, word, ok := strings.Cut(hole, ":"); ok {
			if ink == inkMark {
				out = appendSpan(out, session.LedgerSpan{Text: word, Mark: true})
			} else {
				out = appendSpan(out, session.LedgerSpan{Text: word, Ink: ink})
			}
			continue
		}
		fill, ok := slots[hole]
		if !ok {
			out = appendSpan(out, session.LedgerSpan{Text: "{" + hole + "}"})
			continue
		}
		for _, s := range fill {
			out = appendSpan(out, s)
		}
	}
	return out
}

// templateHoles is the names of a template's holes, sorted and without repeats — the words in
// `{ink:word}` and `{mark:word}` are not holes. It reads the template exactly as Say does.
func templateHoles(tmpl string) []string {
	seen := map[string]bool{}
	for {
		open := strings.IndexByte(tmpl, '{')
		if open < 0 {
			break
		}
		rel := strings.IndexByte(tmpl[open+1:], '}')
		if rel < 0 {
			break
		}
		hole := tmpl[open+1 : open+1+rel]
		tmpl = tmpl[open+2+rel:]
		if !strings.Contains(hole, ":") {
			seen[hole] = true
		}
	}
	out := make([]string, 0, len(seen))
	for h := range seen {
		out = append(out, h)
	}
	slices.Sort(out)
	return out
}

// SayText is a template filled and flattened, for a line that is drawn in one ink.
func SayText(tmpl string, slots Slots) string {
	return session.LedgerLine{Spans: Say(tmpl, slots)}.Text()
}

// appendSpan adds a span, joining it to the last one when the two are written alike, so a sentence
// filled from plain slots is one run rather than several the drawing has to measure separately.
func appendSpan(out []session.LedgerSpan, s session.LedgerSpan) []session.LedgerSpan {
	if s.Text == "" {
		return out
	}
	if n := len(out); n > 0 && out[n-1].Ink == s.Ink && out[n-1].Mark == s.Mark {
		out[n-1].Text += s.Text
		return out
	}
	return append(out, s)
}

// Plain is a slot written in the panel's own ink.
func Plain(text string) []session.LedgerSpan {
	return []session.LedgerSpan{{Text: text}}
}

// Inked is a slot written in a named ink.
func Inked(text, ink string) []session.LedgerSpan {
	return []session.LedgerSpan{{Text: text, Ink: ink}}
}

// Marked is a slot written in a named ink, bold and underlined — an action's verb.
func Marked(text, ink string) []session.LedgerSpan {
	return []session.LedgerSpan{{Text: text, Ink: ink, Mark: true}}
}
