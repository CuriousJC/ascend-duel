package ui

// **The prose: turning an event the engine has already decided into a sentence.**
//
// `logRows` is the walk — one line per thing that happened, in the order the resolver produced
// them — and everything under it is the vocabulary that walk draws on: the verb an attack
// takes, what a card does said in words, what a status is called while it is ticking.
//
// **It lives here and not in internal/combat** on purpose: the rules package names actions, it
// does not describe them. Everything here is presentation over a log that is already finished,
// which is what makes it impossible for a panel to disagree with the round it reports. It
// computes nothing.
//
// **It is not the fight log's, either.** The log is one caller. A shop describing what a relic
// does, and a room choice describing what an affix does, want the same vocabulary — which is
// why this is its own file rather than a section of combat_log.go.
//
// Split out of combat_panes.go on 2026-08-21, which held the prose and the pane widget together.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/carddesc"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"

	"image/color"
)

// handSwatch marks a line that is not one side acting but something the round did — a hand
// forming. **It is the yellow the enemy used to be**, freed when the opponent went gray on
// 2026-08-07: a hand is the loudest thing that can happen in a round and had been sharing a
// hue with every enemy action on screen.
//
// Darker than a screen yellow because it sits on a light pane now — the same figure that read
// as amber on plum reads as washed-out pale on off-white.
var handSwatch = color.RGBA{R: 198, G: 142, B: 16, A: 255}

// The two sides' colors: **green is you, gray is them.**
//
// The opponent was yellow until 2026-08-07 and went gray to give the yellow to `handSwatch` —
// a hand is the loudest thing that can happen in a round and was sharing a hue with every
// enemy action on screen. Gray is also the right *rank* for the opponent: their rows are
// context for yours, and a saturated color was claiming more attention than they earn.
//
// **It settles a collision recorded as open in `MECHANICS.md`**, where lightning's yellow card
// surface ran into `enemySwatch`. The player's green still collides with earth, which went green
// on 2026-08-14, so the element scheme is only half-untangled — and a card border and a row
// swatch are never seen side by side, which is why that half has been allowed to stand.
var (
	PlayerSwatch = color.RGBA{R: 46, G: 150, B: 70, A: 255}
	enemySwatch  = color.RGBA{R: 108, G: 110, B: 122, A: 255}
)

// PaneRowsFor draws worded lines as pane rows: the voice becomes a swatch, and each span's
// ink name becomes a color.
//
// **The colors are decided here and never stored**, which is what lets a saved run be re-colored
// by a change to this file rather than carrying a palette in its history. See session.LedgerLine.
func PaneRowsFor(lines []session.LedgerLine) []PaneRow {
	rows := make([]PaneRow, 0, len(lines))
	for _, l := range lines {
		spans := make([]paneSpan, 0, len(l.Spans))
		for _, r := range l.Spans {
			spans = append(spans, paneSpan{Text: r.Text, Ink: inkNamed(r.Ink), Mark: r.Mark})
		}
		rows = append(rows, PaneRow{
			Spans:  spans,
			Swatch: swatchForVoice(l.Voice),
			Indent: indentForVoice(l.Voice),
		})
	}
	return rows
}

// inkNamed is the color behind an ink's name. **Zero alpha is "the panel's own ink"**, which is
// what an unnamed span and an unrecognized name both get — a ledger written by another build must
// draw as words rather than refuse to draw.
//
// **Every color here is the one the combat screen uses for the same thing**, which is the point:
// the account should look like what it is an account of. The elements come through cards.BorderOf,
// which is the live table, so recoloring an element recolors its figures in the ledger too.
func inkNamed(name string) color.RGBA {
	switch name {
	case "":
		return color.RGBA{}
	case session.InkAttack:
		return VerbInkFor(combat.CategoryAttack)
	case session.InkDefend:
		return VerbInkFor(combat.CategoryDefend)
	case session.InkHand:
		// **No color**: the panel's own ink, and the run's Mark is what says it is the hand. See
		// session.InkHand, and handNameInk, which is the same decision on the combat screen.
		return color.RGBA{}
	case session.InkRelic:
		return BoostInk
	case session.InkTotal:
		return VerbInkFor(combat.CategoryAttack)
	}
	if e, ok := combat.ParseElement(name); ok {
		return cards.BorderOf(ArtFor(e))
	}
	return color.RGBA{}
}

// swatchForVoice is the square a line is drawn beside. **A zero-alpha swatch is a line with no
// swatch**, which drawPane centers — so headings read as blocks rather than as more of the list.
func swatchForVoice(voice string) color.RGBA {
	switch voice {
	case session.VoiceYou:
		return PlayerSwatch
	case session.VoiceFoe:
		return enemySwatch
	case session.VoiceHand:
		return handSwatch
	default:
		return color.RGBA{}
	}
}

// indentForVoice is how far a line is set in. Only the arithmetic's own terms are, which is also
// what keeps them left-aligned rather than centered — see paneRow.indent.
func indentForVoice(voice string) int {
	if voice == session.VoiceTerm {
		return termIndent
	}
	return 0
}

// CardWeight is what an attack card multiplies its owner's DMG by, in brackets: ` (1.5x)`.
//
// **It is on the line rather than on a line of its own** *(owner's call, 2026-09-02)*, because it
// is a fact about the card that was just named and not a thing that happened. It exists for the
// opponent's turn: a Giant Rat's gnaw and its maul are two sentences that read identically and land
// wildly different figures, and the only account of why was the number at the end.
//
// **Attacks only, and never the identity.** A defense multiplies nothing, and `(1x)` on every
// ordinary swing is a bracket that says nothing on most lines in the game.
func CardWeight(c combat.Card) string {
	if c.Category() != combat.CategoryAttack || c.Amount() == 100 {
		return ""
	}
	return " (" + multiplierText(c.Amount()) + ")"
}

// CategoryInk is the ink a category's verb is written in, as the ledger names it.
func CategoryInk(c combat.Category) string {
	if c == combat.CategoryDefend {
		return session.InkDefend
	}
	return session.InkAttack
}

// **The Resolution pane writes sentences, and this is where the English lives.**
//
// A line is `<who> <verb> <phrase>`: "Duelist attacks with a heavy strike". The verb comes
// from the action's category and the phrase from the card, which is why the two are separate
// tables rather than one string per card — the verb has to be its own span so it can be drawn
// on a colored background, and it would otherwise have to be sliced back out of a sentence.
//
// **The prose is here and not in `internal/combat`.** The rules package names actions; it does
// not describe them. A card renamed changes `String()`; a card that reads badly in a sentence
// changes only this file.
// **Every card's prose is generated from its verb, not written down** *(2026-08-16)*. There were
// two hand-maintained tables here, one string per concept, which worked while there were fourteen
// concepts. There are hundreds now — every enemy carries its own cards — so a table would be a
// list nobody could keep complete, and a card with no entry read as though nothing had happened.
//
// **The wording still lives here and not in `internal/combat`.** The rules package names cards; it
// does not describe them. What changed is that the description is now a function of the rule
// rather than a lookup beside it, which is the only version that can cover a deck written in JSON.

// AttackVerb is what a form is called on a card face. **The player's three forms are told apart by
// it** — nine attack cards on one ladder, and a card naming no form would leave the corner mark
// carrying the distinction alone. An enemy card belongs to no form and simply hits.
//
// **It is the form's name rather than a verb, in capitals** *(owner's call, 2026-09-07)*. It read
// "Slashes for 1x DMG" until then, wrapped by the measurer into three ragged lines — the word, the
// stray "for", then the figure — which spent the whole text column on a sentence to say two facts.
// The card now writes the two facts as two lines and nothing else, and the form is written the way
// the ladder and the hand names write it: as a label, not as something the card is doing.
func AttackVerb(f combat.Form) string {
	switch f {
	case combat.FormStab:
		return "STAB"
	case combat.FormSlash:
		return "SLASH"
	case combat.FormCrush:
		return "CRUSH"
	default:
		return "HITS"
	}
}

// multiplierText writes a damage multiplier the way a card says it: 0.5x, 1x, 1.5x, 2x.
//
// **A multiplier rather than a word** — "0.5x" instead of "half" — because a multiplier is what
// the rule actually is, and because the column is about a dozen characters wide.
func multiplierText(amount int) string {
	whole, frac := amount/100, amount%100
	if frac == 0 {
		return strconv.Itoa(whole) + "x"
	}
	if frac%10 == 0 {
		return fmt.Sprintf("%d.%dx", whole, frac/10)
	}
	return fmt.Sprintf("%d.%02dx", whole, frac)
}

// riderText is the lines a card's upgrade adds under its own, one authored line each.
//
// **The face has to say what a rune did to a card.** CLAUDE.md's rule about an altered card
// printing what it actually does is the whole reason effect text reads the card rather than the
// concept, and a rider is the largest thing a card can carry that the concept knows nothing about.
// An extra line is the cheapest honest answer: the band holds seven lines at this pitch, the card's
// own verb takes two, and no rider writes more than three.
//
// **The wording is `carddesc.FaceLines` and not this function's** *(2026-09-09)*. It said four of
// the ten riders and was silent about the other six — a card the player had spent a rune on
// that carried a wash and no words — and `tools/upgradesheet` kept a hand-written snapshot of it
// because `internal/screens` links Ebitengine. Moving it down to the windowless package fixes both:
// the face is total over `combat.RiderKinds()`, and the sheet prints the game's own strings rather
// than a copy that can drift.
//
// **It is not written in the relic pink.** That color means "a relic did this" everywhere else on
// screen, and a rune is not a relic; borrowing it would say something untrue about where the
// figure came from.
func riderText(card combat.Card) string {
	out := ""
	for _, line := range carddesc.FaceLines(card) {
		out += "\n" + line
	}
	return out
}

// ElementSpans cuts a clause into spans so the word naming an element is written in that element's
// color — "attacks with a fire cut", with `fire` in the fire orange.
//
// **The ledger's ink vocabulary already had the elements in it**, because a term in the arithmetic
// wears its own card's color; see inkNamed, which resolves an element's name through the same
// `cards.BorderOf` a card's border comes from. So this is the third reader of one table rather than
// a color decided here.
//
// **The cut is `cards.SplitSpans`**, the same one the card face uses, so where a word begins and ends
// is answered once — see internal/cards/render.go on why a second implementation would be two sets
// of answers to where BURN ends inside BURNING.
func ElementSpans(clause string) []session.LedgerSpan {
	found := cards.ElementSpans(clause)
	if len(found) == 0 {
		return []session.LedgerSpan{{Text: clause}}
	}

	var out []session.LedgerSpan
	for _, seg := range cards.SplitSpans(clause, found) {
		out = append(out, session.LedgerSpan{Text: seg.Text, Ink: elementInkNames[strings.ToLower(seg.Text)]})
	}
	return out
}

// elementInkNames is which of the ledger's ink names each colored word takes.
//
// **A span is named rather than colored**, because a ledger line is written once and read back three
// fights later — see session.LedgerSpan.Ink. A color stored in a line would be the color the build
// that wrote it happened to use, and the account would then disagree with the game it is an account
// of the first time the palette moved.
//
// Built once, off the same `statuses.json` the vocabulary itself is built from, so a status arriving
// later cannot be colored on a card and plain in the account.
var elementInkNames = buildElementInkNames()

func buildElementInkNames() map[string]string {
	out := map[string]string{}
	for _, e := range []combat.Element{combat.Fire, combat.Ice, combat.Lightning, combat.Earth, combat.Arcane} {
		out[e.String()] = e.String()
	}
	for _, st := range data.LoadStatuses() {
		if e, ok := combat.ParseElement(st.Element); ok {
			out[strings.ToLower(st.Name)] = e.String()
			out[strings.ToLower(st.Verb)] = e.String()
		}
	}
	return out
}

// StatusPhraseByKey is what a landed status says it did, as an outcome attached to the attacker's line.
// Each names the *effect* rather than the status, because "chills them" says what happens next and
// "applies chilled" says only that a rule fired.
//
// **Keyed by record rather than by element** *(2026-08-17)*, since a status is no longer a color:
// two relics can put two different statuses on the same fire card, and one phrase per color could
// not tell them apart. The fallback is what a status with no sentence of its own narrates as — its
// own name, which is at least true — so authoring a status in the file does not need a Go change to
// read properly.
// **It is keyed by the status's key rather than by its ordinal**, because a ledger record holds
// the key: a record outlives the build that wrote it, and a StatusID is an index into an array. A
// key this build no longer has falls back to the key itself, which is honest where a blank is a
// line the player would read as a bug.
func StatusPhraseByKey(key string) string {
	switch key {
	case "burning":
		return "sets them burning"
	case "chilled":
		return "chills them"
	case "shocked":
		return "shocks them"
	case "weighted":
		return "weighs them down"
	}
	return "leaves them " + lower(statusName(key))
}

// TickVerbByKey is how a damage-over-time status reads when it bites at the end of a round: "Goblin burns
// for 2". A status with no verb of its own falls back to its name, which is true rather than
// graceful — and is what stops a second such status narrating as a burn.
func TickVerbByKey(key string) string {
	if key == "burning" {
		return "burns for"
	}
	return "takes " + lower(statusName(key)) + " damage:"
}

// statusName is what a status is called, from its key, falling back to the key itself for one this
// build no longer carries. **A key is named rather than hidden**, for relicName's reason: a line in
// a saved account has to read as something.
func statusName(key string) string {
	if id, ok := combat.StatusByKey(key); ok {
		return combat.StatusOf(id).Name
	}
	return key
}

// VerbFor is the verb a category is spoken with.
//
// **"defends" covers a guard and a shield alike**, which is a small stretch on the second and the
// right one: the word is a *scanning* aid saying which half of the turn a line belongs to, not a
// description of the card. A third verb would be a third color on a pane that is read by color
// before it is read at all.
func VerbFor(c combat.Category) string {
	if c == combat.CategoryDefend {
		return "defends"
	}
	return "attacks"
}

// The color the verb is *written* in. **Red for attack, blue for defend** — the category made loud
// enough to scan a round by, without reading it.
//
// **The verb was a filled chip until 2026-08-08 and is now the word itself**, colored, bolded
// and underlined. The chip was a saturated block in a pane that already carries a swatch and a
// sentence, and it drew the eye to a rectangle rather than to the word inside it. Marking the
// word spends the same signal on the thing being read, which is the reasoning that already
// retired the full-width highlight bar a day earlier — this is the same mistake one scale
// smaller.
//
// **The defend phase keeps the blue** *(2026-08-15)*. With two categories the second color is the
// whole distinction, and a category rendered in the row's own ink would leave "attacks" as the
// only marked verb — which is a highlight, not a scheme.
func VerbInkFor(c combat.Category) color.RGBA {
	if c == combat.CategoryDefend {
		return color.RGBA{R: 52, G: 104, B: 196, A: 255}
	}
	return color.RGBA{R: 186, G: 52, B: 52, A: 255}
}

// lower is strings.ToLower under a shorter name, used only to drop a card name into the middle
// of a sentence.
func lower(s string) string { return strings.ToLower(s) }

// DuelistName is the fallback for a duelist record that names nobody. The record is still
// keyed `Fighter1` in duelists.json — a key is not a label, and renaming it would mean
// renaming it in the balance tool and the tests for no gain — but the record now carries a
// Name and that is what is normally shown.
const DuelistName = "DUELIST"

// PlayerRecord is the key the playable duelist is filed under in duelists.json. **Two screens
// hydrate the player now** — the combat screen for the fight and the reward screen for the card it
// puts up beside the relics — so the key is written once rather than in each of them.
const PlayerRecord = "Fighter1"

// HandName is what the attack phase formed, said in words: "Two Pair", "Four of a Kind".
//
// **A hand carries its whole name** *(2026-08-17)*. The name used to be assembled here from two
// parts — the element makeup in front of the hand, "Duo Bash Flurry" — and both of those axes
// are gone: color buys statuses rather than a multiplier, and a hand is named for its shape
// rather than for the card that formed it. A blow that formed no hand at all is named only as an
// attack; the pane does not announce those, but the trace does.
//
// The name comes from the catalog rather than being written here, so a hand renamed in
// `data/hands.json` is renamed once.
func HandName(e combat.Event) string {
	hand, ok := combat.HandByID(e.Hand)
	if !ok {
		return "attack"
	}
	return hand.Name
}

// HandTitle is the hand as the ledger names it: `Three of a Kind (Form)`.
//
// **The axis goes to the back and into brackets** *(owner's call, 2026-09-02)*. `hands.json` writes
// it in front — "Form Three of a Kind" — which puts the least interesting word first on the loudest
// line of the round and reads as a hand called "Form Three" to anybody skimming. What the rung is
// comes first; which axis counted it is the qualifier.
//
// **The bracket is the word the catalog used**, stripped off the front rather than looked up, so
// a hand renamed is renamed once — in the data. A name with no axis word in front of it is left
// exactly as it is, which is what keeps "No Hand" from becoming "No Hand (Card)".
func HandTitle(e combat.Event) string { return axisToBack(HandName(e)) }

// axisToBack does the moving, split out so it can be tested without an event.
func axisToBack(name string) string {
	for _, axis := range []string{"Card ", "Form ", "Elemental "} {
		if rest, ok := strings.CutPrefix(name, axis); ok {
			return rest + " (" + strings.TrimSpace(axis) + ")"
		}
	}
	return name
}

// HandMultiplierText writes a *hand's* percentage multiplier the way the design does: 350 as
// `3.5`, 200 as `2`, 1000 as `10`. Trailing zeros are dropped rather than padded to two places,
// because `x 10.00` reads as a precision the game does not have.
//
// It is not `multiplierText`, which writes a *card's* multiplier and keeps its `x`. The two read
// almost the same and are printed in different sentences: this one lands inside the arithmetic
// line, where the `x` is already there as an operator.
func HandMultiplierText(pct int) string {
	return strconv.FormatFloat(float64(pct)/100, 'f', -1, 64)
}

// ShieldCount is "1 shield" or "3 shields", and it is the one place the noun is pluralised.
//
// **The card face, the feed and the tooltip all read it**, because a face saying "1 shields" is
// the kind of thing that survives a review by being in three files at once.
func ShieldCount(n int) string {
	if n == 1 {
		return "1 shield"
	}
	return strconv.Itoa(n) + " shields"
}
