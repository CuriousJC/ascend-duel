package ui

// **The translator: a record read back as the line the panel draws.**
//
// `session.LedgerRecord` is what happened, as flat named fields. This is the one place those
// become English, which is what lets a wording change reach every account already on disk — see
// session/record.go, where the records are defined and where the argument for storing data rather
// than sentences lives.
//
// **It is total over the record kinds**, and a test holds it that way: a record with no case here
// would draw as a blank line, and a blank line and a record nobody wrote look identical on a panel.
//
// **It computes nothing.** Every figure comes off the record, which came off the resolver's own
// event, so a line cannot claim a sum the round did not use. What is decided here is only how a
// figure is spelled and which ink names it carries.
//
// **The attaching is here rather than in the record stream.** An outcome — damage, a status, a
// shield going up — is what became of a card that was played, and the sentence for it is the act
// above it. So the records stay one-per-thing-that-happened and the state machine that folds them
// into readable lines lives with the wording, where it can be changed without rewriting an account.

import (
	"fmt"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/session"
)

// LedgerLines is a block of records read back as lines: a round's, or the gap after a fight.
func LedgerLines(recs []session.LedgerRecord) []session.LedgerLine {
	w := lineWriter{}
	for _, r := range recs {
		w.write(r)
	}
	return w.rows
}

// lineWriter folds a record stream into lines.
//
// **cur is the line an outcome attaches to**, or -1 when the last thing written was an announcement
// with nothing to hang off. curSide is whose line it is, tracked rather than read back off the
// row's swatch: the blow wears the hand's amber and takes outcomes, so a damage record compared
// against that swatch would read every hit as belonging to the wrong duelist.
type lineWriter struct {
	rows     []session.LedgerLine
	cur      int
	curSide  string
	outcomes int
}

// attach adds an outcome to the tail of the open sentence, after the verb, so the colored verb
// never moves as a line grows.
func (w *lineWriter) attach(what string) {
	if w.cur < 0 {
		return
	}
	sep := " - "
	if w.outcomes > 0 {
		sep = ", "
	}
	w.rows[w.cur].Spans = append(w.rows[w.cur].Spans, session.LedgerSpan{Text: sep + what})
	w.outcomes++
}

// announce opens a line belonging to nobody's card, which nothing may then attach to.
func (w *lineWriter) announce(voice, text string) {
	w.rows = append(w.rows, session.Line(voice, text))
	w.cur = -1
}

// open starts a line outcomes may attach to.
func (w *lineWriter) open(line session.LedgerLine, side string) {
	w.rows = append(w.rows, line)
	w.cur, w.curSide, w.outcomes = len(w.rows)-1, side, 0
}

func (w *lineWriter) write(r session.LedgerRecord) {
	switch r.Kind {

	// ---- the lines something can attach to ----

	case session.KindAct:
		// "<who> <verb> <phrase>": "Duelist attacks with a fire strike". The verb is its own span
		// so it can carry its category's color.
		clause := " " + cardClause(r) + cardWeightText(r.Weight)
		w.open(session.LedgerLine{
			Voice: voiceForSide(r.Side),
			Spans: append([]session.LedgerSpan{
				{Text: r.Name + " "},
				{Text: verbWord(r.Verb), Ink: verbInkName(r.Verb), Mark: true},
			}, ElementSpans(clause)...),
		}, r.Side)

	case session.KindBlow:
		// **The hand's name is the whole line.** The amber swatch says a hand formed and in a
		// player's ledger the duelist is who forms them, so every word in front of the name was
		// already said by something on the row. **And it is not marked**: bold is the whole panel,
		// and an underline under a name alone on its line reads as a mistake rather than emphasis.
		w.open(session.LedgerLine{
			Voice: session.VoiceHand,
			Spans: []session.LedgerSpan{{Text: r.Hand, Ink: session.InkHand}},
		}, r.Side)

	// ---- the outcomes ----

	case session.KindMissed:
		// **Naming the shock is the whole point.** A blow that simply missed would look like a bug
		// in a game with no dice in it.
		w.attach("misses - shocked")

	case session.KindStatus:
		w.attach(StatusPhraseByKey(r.Status))

	case session.KindDrained:
		// **The relic names itself**, so a second drain relic cannot narrate identically to the
		// first — the argument a ticking status is already under.
		w.attach(fmt.Sprintf("%s drains %d", r.Relic, r.Amount))

	case session.KindRaised:
		// **The count that is standing, not the count this card added.** Two Guards in a turn is
		// one duelist behind six shields, and a line saying "+3" twice makes the reader do the
		// arithmetic the readout has already done.
		w.attach(ShieldCount(r.Amount) + " up")

	case session.KindHeld:
		w.attach(fmt.Sprintf("kept back for %d vitae", r.Amount))

	case session.KindSilver:
		// **Two riders pay vitae and they are different sentences.** A held card is paid for being
		// kept back; a played silver card gambled and came up.
		w.attach(fmt.Sprintf("silver pays %d vitae", r.Amount))

	case session.KindLapsed:
		// **Shields that were never spent are the player's own decision coming back**, and a
		// readout that simply went blank would read as a bug.
		w.attach(ShieldCount(r.Amount) + " lapse")

	case session.KindBlocked:
		// **The only record that the attack happened at all**, since it landed nothing and there
		// is no damage line coming.
		w.attach(fmt.Sprintf("blocked - %s left", ShieldCount(r.Amount)))

	case session.KindDamage:
		// **Damage whose side does not match the line it is attaching to is damage running the
		// other way**, which reads as something done back rather than as a hit of its own. Nothing
		// produces it today; it costs one branch and catches the case rather than mis-narrating it.
		if w.cur >= 0 && w.curSide != r.Side {
			w.attach(fmt.Sprintf("hits back for %d", r.Amount))
			return
		}
		w.attach(fmt.Sprintf("%d damage", r.Amount))

	// ---- the announcements ----

	case session.KindChilled:
		w.announce(voiceForSide(r.Side),
			fmt.Sprintf("%s is chilled - %v is lost", r.Name, r.Card))

	case session.KindRegenerated:
		// **A line of its own, where a drain attaches to one.** This happens at the top of a turn
		// with nothing before it, so there is nothing to attach to.
		w.announce(voiceForSide(r.Side),
			fmt.Sprintf("%s restores %d - %s", r.Name, r.Amount, r.Relic))

	case session.KindTicked:
		// A tick belongs to nobody's card, so it opens its own line, and it carries the victim's
		// swatch because it is a thing happening *to* them. **The status names itself**, so a
		// second damage-over-time status cannot narrate as a burn.
		w.announce(voiceForSide(r.Target),
			fmt.Sprintf("%s %s %d", r.Name, TickVerbByKey(r.Status), r.Amount))

	case session.KindTimeUp:
		// **A line of its own, and it opens one.** Nobody swung, so there is no attacker's sentence
		// for this to attach to — and the fall on the next record would otherwise be the only
		// account of the biggest thing that can happen in a fight.
		w.announce(voiceForSide(r.Target),
			fmt.Sprintf("%s is out of time - the duel takes %d", r.Name, r.Amount))

	case session.KindDefeated:
		w.announce(voiceForSide(r.Target), r.Name+" falls")

	// ---- the working under a blow ----

	case session.KindTerm:
		w.rows = append(w.rows, termLine(r))
		w.cur = -1

	// ---- the gap between two fights ----

	case session.KindChanged:
		w.rows = append(w.rows, afterLine(r.Kind, r.Subject+" into "+r.Into))
		w.cur = -1

	case session.KindRaisedRung:
		w.rows = append(w.rows, afterLine("raised",
			fmt.Sprintf("%s to +%d", r.Hand, r.Amount)))
		w.cur = -1

	case session.KindGained, session.KindPaid:
		verb := "gained"
		if r.Kind == session.KindPaid {
			verb = "spent"
		}
		w.rows = append(w.rows, afterLine(verb, fmt.Sprintf("%d vitae", r.Amount)))
		w.cur = -1

	case session.KindTook, session.KindCut, session.KindWore, session.KindSold, session.KindSpent:
		w.rows = append(w.rows, afterLine(r.Kind, r.Subject))
		w.cur = -1
	}
}

// termLine is one line of a blow's working, drawn by its role.
//
// **The name column is one width for every role**, so a card's figure, a flat term's and the
// relic's all land in the same place and the working reads as one column of figures.
func termLine(r session.LedgerRecord) session.LedgerLine {
	switch r.Role {

	case session.RoleDMG:
		// **It carries no figure in the sum's column**, because the bigger figure is already inside
		// every term below it. This says where it came from, which is the whole reason it is drawn:
		// a relic folded into a number the game already shows is a relic the player cannot see.
		return session.LedgerLine{Voice: session.VoiceTerm, Spans: []session.LedgerSpan{
			{Text: fmt.Sprintf("%-14s", r.Relic+" ("+r.Hand+")"), Ink: session.InkRelic},
			{Text: fmt.Sprintf("+%d DMG", r.Amount), Ink: session.InkRelic},
		}}

	case session.RoleFlat:
		// **The relic's name takes the relic ink and the figure does not**: a relic put the term in
		// the sum, but the term is the hand paying rather than a number a relic moved on a card.
		return session.LedgerLine{Voice: session.VoiceTerm, Spans: []session.LedgerSpan{
			{Text: fmt.Sprintf("%-14s", r.Relic+" ("+r.Note+")"), Ink: session.InkRelic},
			{Text: fmt.Sprintf("%4d", r.Amount)},
		}}

	case session.RoleSum:
		return session.LedgerLine{Voice: session.VoiceTerm, Spans: sumSpans(r)}

	default:
		// RoleCard: the card, what it was worth before its relics, and what each relic did to it.
		ink := r.Element
		spans := []session.LedgerSpan{
			{Text: fmt.Sprintf("%-14s", termCardName(r)), Ink: ink},
			{Text: fmt.Sprintf("%4d", r.Base), Ink: ink},
		}
		return session.LedgerLine{Voice: session.VoiceTerm, Spans: append(spans, factorNotes(r.Factors)...)}
	}
}

// termCardName is what a landing is called in a column of figures: the card, and its color when it
// has one. **Not a clause** — that is what an act line writes.
func termCardName(r session.LedgerRecord) string {
	if r.Element == "" {
		return r.Card
	}
	return r.Card + " (" + r.Element + ")"
}

// factorNotes is what the relics did to one term: the landings they bought, then the figures they
// priced it at, in the order they fired.
//
// **A landing and a multiplier are said differently because they are different things.** An echo
// relic buys a term and contributes no figure, so a note reading `x Echo 1x` would credit it with
// arithmetic it did not do.
//
// **A relic firing at the identity still fired** and is written: leaving it out is how a growing
// relic's climb off 1x becomes invisible.
func factorNotes(factors []session.LedgerFactor) []session.LedgerSpan {
	var out []session.LedgerSpan
	for _, f := range factors {
		if f.Landed {
			out = append(out, session.LedgerSpan{
				Text: "  + " + f.Relic + " lands it again", Ink: session.InkRelic,
			})
			continue
		}
		note := "  x " + f.Relic + " " + HandMultiplierText(f.Scale) + "x"

		// **What the relic stood at after this term**, written only where it moved — the one case
		// in which the same relic prices two terms of one blow differently.
		if f.Grown > 0 {
			note += fmt.Sprintf(" (grown %d)", f.Grown)
		}
		out = append(out, session.LedgerSpan{Text: note, Ink: session.InkRelic})
	}
	return out
}

// sumSpans is the blow written out as the sum it is: `(10 x 2 x 1.5) + 10 x 2.5 = 100`.
//
// **A relic's figure stays with the term it priced**, in brackets, rather than being folded into
// the term or hung on the end of the whole sum. Folding it in hides the relic; hanging it on the
// end reads as multiplying every term, which is not what happened and does not come to the total.
func sumSpans(r session.LedgerRecord) []session.LedgerSpan {
	var spans []session.LedgerSpan

	for _, t := range r.Terms {
		if len(spans) > 0 {
			spans = append(spans, session.LedgerSpan{Text: " + "})
		}

		// **The term is the product the game worked out, not its answer**: the DMG the hand was
		// swung at, times this card's own multiplier, times whatever a relic priced it at.
		if t.Split {
			spans = append(spans, session.LedgerSpan{Text: "(" + strconv.Itoa(t.DMG)})
			spans = append(spans, session.LedgerSpan{
				Text: " x " + HandMultiplierText(t.Weight), Ink: t.Element,
			})
			spans = append(spans, scaleSpans(t.Scales)...)
			spans = append(spans, session.LedgerSpan{Text: ")"})
			continue
		}

		// **The flat form is for the term the split cannot describe** — an echo's rounding, or a
		// card that hit the damage floor.
		if len(t.Scales) == 0 {
			spans = append(spans, session.LedgerSpan{Text: strconv.Itoa(t.Base), Ink: t.Element})
			continue
		}
		spans = append(spans, session.LedgerSpan{Text: "(" + strconv.Itoa(t.Base), Ink: t.Element})
		spans = append(spans, scaleSpans(t.Scales)...)
		spans = append(spans, session.LedgerSpan{Text: ")", Ink: t.Element})
	}

	// **The flat terms, after the cards and before the multiplier**, which is where the resolver
	// adds them in. Written in no ink at all: a relic put them in the sum and the term is still the
	// hand paying.
	for _, flat := range r.Flats {
		if len(spans) > 0 {
			spans = append(spans, session.LedgerSpan{Text: " + "})
		}
		spans = append(spans, session.LedgerSpan{Text: strconv.Itoa(flat)})
	}

	// A blow whose record carries no terms still has its two figures. Nothing produces one today;
	// saying the sum it did is better than a line reading `x 1.5 = 30` with nothing in front.
	if len(spans) == 0 {
		spans = append(spans, session.LedgerSpan{Text: strconv.Itoa(r.Base)})
	}

	// **No mark.** An underline under a multiplier in the middle of a sum reads as a typesetting
	// accident; the whole panel is bold already, and the figure's place in the line says what it is.
	spans = append(spans, session.LedgerSpan{
		Text: " x " + HandMultiplierText(r.Multiplier), Ink: session.InkHand,
	})

	// **A rung relic is a second multiplier, after the hand's own**, so it is a second `x` on the
	// line rather than a bigger figure in the first — which is what keeps the first reading as the
	// rung the player built.
	if r.HandScale != 0 && r.HandScale != 100 {
		spans = append(spans, session.LedgerSpan{
			Text: " x " + HandMultiplierText(r.HandScale), Ink: session.InkRelic,
		})
	}

	return append(spans,
		session.LedgerSpan{Text: " = "},
		session.LedgerSpan{Text: strconv.Itoa(r.Total), Ink: session.InkTotal},
	)
}

// scaleSpans writes relic multipliers as factors inside a term's bracket, in the relic pink the
// rest of the panel gives a relic's own figure.
func scaleSpans(scales []int) []session.LedgerSpan {
	out := make([]session.LedgerSpan, 0, len(scales))
	for _, pct := range scales {
		out = append(out, session.LedgerSpan{
			Text: " x " + HandMultiplierText(pct), Ink: session.InkRelic,
		})
	}
	return out
}

// afterLine is one line of what the player did between two fights: a marked verb and what it was
// done to.
//
// **The verb is marked exactly as an action's is**, so the aftermath can be scanned for what kind
// of thing happened before any of it is read. The rest goes through ElementSpans, so a fire card is
// named in the fire color here as it is everywhere else.
func afterLine(verb, clause string) session.LedgerLine {
	spans := []session.LedgerSpan{{Text: verb, Mark: true}}
	return session.LedgerLine{
		Voice: session.VoiceYou,
		Spans: append(spans, ElementSpans(" "+clause)...),
	}
}

// cardClause is what follows the verb on an act line: "with a fire strike", "and raises a brace".
//
// **The element goes after the article rather than in front of the phrase**, which is what makes it
// a sentence instead of a label, and **the article is corrected rather than followed** — two of the
// five elements begin with a vowel, so "a earth strike" is a third of the lines this writes.
func cardClause(r session.LedgerRecord) string {
	lead := "with a "
	if r.Raises {
		lead = "and raises a "
	}
	name := lower(r.Card)
	if r.Element == "" {
		return lead + name
	}
	el := lower(r.Element)
	if isVowel(el) {
		lead = lead[:len(lead)-2] + "an "
	}
	return lead + el + " " + name
}

func isVowel(s string) bool {
	if s == "" {
		return false
	}
	switch s[0] {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}

// cardWeightText is what an attack card multiplies its owner's DMG by, in brackets: ` (1.5x)`.
//
// **It is on the line rather than on a line of its own**, because it is a fact about the card that
// was just named and not a thing that happened. It exists for the opponent's turn: a creature's
// gnaw and its maul are two sentences that read identically and land wildly different figures.
//
// **Never the identity**, because `(1x)` on every ordinary swing is a bracket that says nothing on
// most lines in the game.
func cardWeightText(weight int) string {
	if weight == 0 || weight == 100 {
		return ""
	}
	return " (" + multiplierText(weight) + ")"
}

// voiceForSide maps a record's side onto the voice its line is spoken in.
func voiceForSide(side string) string {
	if side == session.SideFoe {
		return session.VoiceFoe
	}
	return session.VoiceYou
}

// verbWord is the verb a category is spoken with. **"defends" covers a guard and a shield alike**,
// which is a small stretch on the second and the right one: the word is a scanning aid saying which
// half of the turn a line belongs to, not a description of the card.
func verbWord(verb string) string {
	if verb == session.InkDefend {
		return "defends"
	}
	return "attacks"
}

// verbInkName is which ink that verb is written in, as the ledger names it.
func verbInkName(verb string) string {
	if verb == session.InkDefend {
		return session.InkDefend
	}
	return session.InkAttack
}
