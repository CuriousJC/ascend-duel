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
// **The attaching is here rather than in the record stream.** An outcome — damage, a drain, a
// shield going up — is what became of a card that was played, and the sentence for it is the act
// above it. So the records stay one-per-thing-that-happened and the state machine that folds them
// into readable lines lives with the wording, where it can be changed without rewriting an account.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/session"
)

// LedgerLines is a block of records read back as lines: a round's, or the gap after a fight.
func LedgerLines(recs []session.LedgerRecord) []session.LedgerLine {
	w := lineWriter{blowRow: -1}
	for _, r := range recs {
		w.write(r)
	}
	w.closeBlow()
	return w.rows
}

// lineWriter folds a record stream into lines.
//
// **cur is the line an outcome attaches to**, or -1 when the last thing written was an announcement
// with nothing to hang off. curSide is whose line it is, tracked rather than read back off the
// row's swatch: a damage record compared against a swatch would read every hit under the hand's
// amber as belonging to the wrong duelist.
//
// **hits is where each hit's line was written**, keyed by its Hit, and an outcome that names a hit
// attaches there instead: a blow writes every hit's line before any of them has landed, so the last
// line written is the total and not the hit the outcome is about. outcomes counts per row for the
// same reason — two hits' outcomes interleave.
type lineWriter struct {
	rows     []session.LedgerLine
	cur      int
	curSide  string
	hits     map[int]int
	hitSide  string
	outcomes map[int]int

	// hand is the name of the blow being written, which labels its multiplier in each hit's
	// working on an account whose hit records do not carry it themselves.
	hand string

	// blowRow is the blow's heading, where its grand total is written, or -1 outside a blow.
	blowRow int

	// hitSum is what each hit's working came to and the element it was dealt in, keyed by Hit, so
	// the damage that hit landed can be written as that sum applied.
	hitSum map[int]session.LedgerRecord

	// settled is which hits of the blow something became of; lastLanded is the last hit whose
	// damage was applied, which a fall that follows it is written on; and fell says the blow's
	// target went down, after which a hit nothing became of was over damage.
	settled    map[int]bool
	lastLanded int
	fell       bool
}

// closeBlow says what the blow's leftover hits were: **a hit that came to something, landed
// nothing and had no shield or fizzle to say why, after the target fell, was over damage** — the
// target was already down. Called when the next blow begins and when the block ends.
func (w *lineWriter) closeBlow() {
	if w.fell {
		for hit, rec := range w.hitSum {
			if w.settled[hit] || rec.Total <= 0 || !w.onHit(hit) {
				continue
			}
			row := &w.rows[w.hits[hit]]
			row.Spans = append(row.Spans, session.LedgerSpan{Text: " " + SayText(SayTermOver, nil)})
		}
	}
	w.hitSum, w.settled, w.lastLanded, w.fell = nil, nil, 0, false
}

// attach adds an outcome to the tail of the line it belongs to, after the verb, so the colored verb
// never moves as a line grows. `hit` is the record's Hit, and zero attaches to the open line.
func (w *lineWriter) attach(hit int, what []session.LedgerSpan) {
	row := w.cur
	if w.onHit(hit) {
		row = w.hits[hit]
		w.settle(hit)
	}
	if row < 0 {
		return
	}
	if w.outcomes == nil {
		w.outcomes = map[int]int{}
	}
	sep := SayOutcomeFirst
	if w.outcomes[row] > 0 {
		sep = SayOutcomeNext
	}
	w.rows[row].Spans = append(w.rows[row].Spans, Say(sep+"{what}", Slots{"what": what})...)
	w.outcomes[row]++
}

// applied writes a hit's damage as its sum applied — `= 288 Arcane Damage Applied`, underlined —
// and reports whether it could. **Only when the hit landed the whole of what its working came to**;
// anything else is said as an outcome beside the sum, so a figure the hit did not land is never
// written as though it had.
func (w *lineWriter) applied(hit, amount int) bool {
	rec, ok := w.hitSum[hit]
	if !ok || !w.onHit(hit) || amount != rec.Total {
		return false
	}
	// **A hit that came to nothing applied nothing** — a defense's line is `= 0` — so it is
	// settled with nothing said rather than written as damage applied.
	if amount == 0 {
		w.settle(hit)
		return true
	}
	// The figure and the words after it, all in the hit's element and underlined; the `=` is not.
	row := &w.rows[w.hits[hit]]
	for i := 1; i < len(row.Spans); i++ {
		row.Spans[i].Mark = true
	}
	words := SayText(SayTermApplied, nil)
	if rec.Element != "" {
		words = capitalize(rec.Element) + " " + words
	}
	row.Spans = append(row.Spans, session.LedgerSpan{Text: " " + words, Ink: rec.Element, Mark: true})
	w.settle(hit)
	w.lastLanded = hit
	return true
}

// settle records that something became of a hit, so the blow's close does not call it over damage.
func (w *lineWriter) settle(hit int) {
	if w.settled == nil {
		w.settled = map[int]bool{}
	}
	w.settled[hit] = true
}

// grandTotal is what every hit of a blow came to together, written after the hand's name: the words
// in the grand total's own amber and **the figure in red, the one red figure in the working**.
func grandTotal(total int) []session.LedgerSpan {
	spans := Say(SayTermGrand, Slots{"n": {{Text: strconv.Itoa(total), Ink: session.InkTotal}}})
	for i := range spans {
		if spans[i].Ink == "" {
			spans[i].Ink = session.InkGrand
		}
	}
	return spans
}

// capitalize is a word with its first letter raised: `arcane` is `Arcane`.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// onHit reports whether an outcome naming this hit has a hit's line to attach to.
func (w *lineWriter) onHit(hit int) bool {
	_, ok := w.hits[hit]
	return ok && hit > 0
}

// sideOf is whose line an outcome is attaching to: the hit's, where it names one.
func (w *lineWriter) sideOf(hit int) string {
	if w.onHit(hit) {
		return w.hitSide
	}
	return w.curSide
}

// announce opens a line belonging to nobody's card, which nothing may then attach to.
func (w *lineWriter) announce(voice string, spans []session.LedgerSpan) {
	w.rows = append(w.rows, session.LedgerLine{Voice: voice, Spans: spans})
	w.cur = -1
}

// open starts a line outcomes may attach to.
func (w *lineWriter) open(line session.LedgerLine, side string) {
	w.rows = append(w.rows, line)
	w.cur, w.curSide = len(w.rows)-1, side
}

func (w *lineWriter) write(r session.LedgerRecord) {
	switch r.Kind {

	// ---- the lines something can attach to ----

	case session.KindAct:
		// "<who> <verb> <phrase>": "Duelist attacks with a fire strike". The verb is its own span
		// so it can carry its category's color.
		w.open(session.LedgerLine{
			Voice: voiceForSide(r.Side),
			Spans: Say(SayAct, Slots{
				"who":    ElementSpans(r.Name),
				"verb":   Marked(verbWord(r.Verb), verbInkName(r.Verb)),
				"clause": cardClause(r),
			}),
		}, r.Side)

	case session.KindBlow:
		// **The hand's name is the whole line.** The amber swatch says a hand formed and in a
		// player's ledger the duelist is who forms them, so every word in front of the name was
		// already said by something on the row. **And it is not marked**: bold is the whole panel,
		// and an underline under a name alone on its line reads as a mistake rather than emphasis.
		//
		// **A new blow forgets the last one's hits**, since a Hit counts within one blow.
		w.closeBlow()
		w.hits, w.hitSide, w.hand = map[int]int{}, r.Side, r.Hand
		w.hitSum = map[int]session.LedgerRecord{}
		w.open(session.LedgerLine{
			Voice: session.VoiceHand,
			Spans: []session.LedgerSpan{{Text: r.Hand, Ink: session.InkHand}},
		}, r.Side)
		w.blowRow = w.cur

	// ---- the outcomes ----

	case session.KindFizzled:
		// **Naming the reason is the point**: a hit that landed nothing with no word beside it
		// would read as a bug.
		// "Ice attack fizzles, no damage applied." — the hit's own element, in its colour.
		what := "Attack"
		if el := w.hitSum[r.Hit].Element; el != "" {
			what = capitalize(el) + " attack"
		}
		w.attach(r.Hit, Say(SayFizzled, Slots{"what": ElementSpans(what)}))

	case session.KindDrained:
		// **The relic names itself**, so a second drain relic cannot narrate identically to the
		// first.
		w.attach(r.Hit, Say(SayDrained, Slots{"relic": Plain(r.Relic), "n": figure(r.Amount)}))

	case session.KindRaised:
		// **The count that is standing, not the count this card added.** Two Guards in a turn is
		// one duelist behind six shields, and a line saying "+3" twice makes the reader do the
		// arithmetic the readout has already done.
		w.attach(0, Say(SayRaised, Slots{"shields": Plain(ShieldCount(r.Amount))}))

	case session.KindHeld:
		w.attach(0, Say(SayHeld, Slots{"n": figure(r.Amount)}))

	case session.KindReflected:
		// **The relic names itself**, the drain's rule, so a second thorn relic cannot narrate as
		// the first.
		w.attach(r.Hit, Say(SayReflected, Slots{"relic": Plain(r.Relic), "n": figure(r.Amount)}))

	case session.KindTithed:
		w.attach(r.Hit, Say(SayTithed, Slots{"relic": Plain(r.Relic), "n": figure(r.Amount)}))

	case session.KindSilver:
		// **Two riders pay vitae and they are different sentences.** A held card is paid for being
		// kept back; a played silver card gambled and came up.
		w.attach(0, Say(SaySilver, Slots{"n": figure(r.Amount)}))

	case session.KindLapsed:
		// **Shields that were never spent are the player's own decision coming back**, and a
		// readout that simply went blank would read as a bug.
		w.attach(0, Say(SayLapsed, Slots{"shields": Plain(ShieldCount(r.Amount))}))

	case session.KindBlocked:
		// **The only record that the hit happened at all**, since it landed nothing and there is no
		// damage line coming.
		w.attach(r.Hit, Say(SayBlocked, Slots{"shields": Plain(ShieldCount(r.Amount))}))

	case session.KindDamage:
		// **Damage whose side does not match the line it is attaching to is damage running the
		// other way**, which reads as something done back rather than as a hit of its own. Nothing
		// produces it today; it costs one branch and catches the case rather than mis-narrating it.
		if (w.cur >= 0 || w.onHit(r.Hit)) && w.sideOf(r.Hit) != r.Side {
			w.attach(r.Hit, Say(SayHitBack, Slots{"n": figure(r.Amount)}))
			return
		}
		if w.applied(r.Hit, r.Amount) {
			return
		}
		w.attach(r.Hit, Say(SayDamage, Slots{"n": figure(r.Amount)}))
		if w.onHit(r.Hit) {
			w.lastLanded = r.Hit
		}

	// ---- the announcements ----

	case session.KindRegenerated:
		// **A line of its own, where a drain attaches to one.** This happens at the top of a turn
		// with nothing before it, so there is nothing to attach to.
		w.announce(voiceForSide(r.Side), Say(SayRegenerated, Slots{
			"who": ElementSpans(r.Name), "n": figure(r.Amount), "relic": Plain(r.Relic),
		}))

	case session.KindWarded:
		// **A line of its own**, like a regeneration: the top of a turn has nothing to attach to.
		w.announce(voiceForSide(r.Side), Say(SayWarded, Slots{
			"who":     ElementSpans(r.Name),
			"shields": ElementSpans(strings.Replace(ShieldCount(r.Amount), " ", " "+r.Element+" ", 1)),
			"relic":   Plain(r.Relic),
		}))

	case session.KindTimeUp:
		// **A line of its own, and it opens one.** Nobody swung, so there is no attacker's sentence
		// for this to attach to — and the fall on the next record would otherwise be the only
		// account of the biggest thing that can happen in a fight.
		w.announce(voiceForSide(r.Target), Say(SayTimeUp, Slots{"who": ElementSpans(r.Name), "n": figure(r.Amount)}))

	case session.KindDMG:
		// **The duelist's DMG for the turn, built once at its top**: their own figure, each plus a
		// relic or a held card put into it, and what it came to — the figure every hit below starts
		// from, so a hit swung at 238 has already said why.
		w.announce(voiceForSide(r.Side), Say(SayDMGBuilt, Slots{"who": ElementSpans(r.Name)}))
		w.rows = append(w.rows, workStep("", strconv.Itoa(r.DMGBase), "", SayText(SayStepBase, nil), ""))
		for _, f := range r.DMGParts {
			if f.Add != 0 {
				w.rows = append(w.rows, workStep("+", strconv.Itoa(f.Add), session.InkRelic, f.Relic, session.InkRelic))
			} else {
				w.rows = append(w.rows, workStep("x", HandMultiplierText(f.Scale), session.InkRelic, f.Relic, session.InkRelic))
			}
		}
		w.rows = append(w.rows, workStep("=", strconv.Itoa(r.Amount), "", SayText(SayStepDMG, nil), ""))

	case session.KindDefeated:
		// **A fall to a blow is written on the hit that landed it**, the last hit of the blow whose
		// damage went in, because that is where the fight was won. Any other fall — the clock, a
		// creature's own blow — is a line of its own.
		if w.lastLanded > 0 && w.onHit(w.lastLanded) && r.Target != w.hitSide {
			w.attach(w.lastLanded, Say(SayDefeated, Slots{"who": ElementSpans(r.Name)}))
			w.fell = true
			return
		}
		w.announce(voiceForSide(r.Target), Say(SayDefeated, Slots{"who": ElementSpans(r.Name)}))

	case session.KindUsed:
		// "Duelist casts Cantrip of Might - DMG 10 to 20", "Duelist uses Embermark on a jab and a
		// bash". **The verb is marked like an act's** so a round's spending can be scanned for, and
		// takes the row's own ink: a consumable is neither of the two verbs that own a color.
		spans := Say(SayUsed, Slots{
			"who":     ElementSpans(r.Name),
			"verb":    Marked(usedVerb(r.Note), ""),
			"subject": Plain(r.Subject),
		})
		if r.Into != "" {
			join := SayUsedInto
			if r.Note == "rune" || r.Note == "essence" {
				join = SayUsedOn
			}
			spans = append(spans, Say(join, Slots{"into": ElementSpans(r.Into)})...)
		}
		w.rows = append(w.rows, session.LedgerLine{Voice: voiceForSide(r.Side), Spans: spans})
		w.cur = -1

	// ---- the working under a blow ----

	case session.KindTerm:
		w.cur = -1
		// **Every hit together is the blow's grand total, said on the blow's own heading**, where
		// the hand is named, rather than under the last hit.
		if r.Role == session.RoleTotal && w.blowRow >= 0 && w.blowRow < len(w.rows) {
			head := &w.rows[w.blowRow]
			for i := range head.Spans {
				head.Spans[i].Ink = session.InkGrand
			}
			head.Spans = append(head.Spans, grandTotal(r.Total)...)
			return
		}
		if r.Role == session.RoleDMG || r.Role == session.RoleTotal {
			w.rows = append(w.rows, termLine(r))
			return
		}
		// RoleHit, and any role this build does not write — a line read back from an older account
		// draws as the hit it was part of.
		w.rows = append(w.rows, hitLines(r, w.hand)...)
		// **A hit's `=` line is where its outcomes go**, however many lines are written after it:
		// what the hit did sits beside what it came to.
		if r.Hit > 0 {
			if w.hits == nil {
				w.hits = map[int]int{}
			}
			if w.hitSum == nil {
				w.hitSum = map[int]session.LedgerRecord{}
			}
			w.hits[r.Hit] = len(w.rows) - 1
			w.hitSum[r.Hit] = r
		}

	// ---- the gap between two fights ----

	case session.KindChanged:
		w.rows = append(w.rows, afterLine(SayAfterChanged, Slots{
			"subject": ElementSpans(r.Subject), "into": ElementSpans(r.Into),
		}))
		w.cur = -1

	case session.KindRaisedRung:
		w.rows = append(w.rows, afterLine(SayAfterRaised, Slots{
			"hand": ElementSpans(r.Hand), "n": figure(r.Amount),
		}))
		w.cur = -1

	case session.KindGained, session.KindPaid:
		tmpl := SayAfterGained
		if r.Kind == session.KindPaid {
			tmpl = SayAfterPaid
		}
		w.rows = append(w.rows, afterLine(tmpl, Slots{"n": figure(r.Amount)}))
		w.cur = -1

	case session.KindTook, session.KindCut, session.KindWore, session.KindSold, session.KindSpent:
		w.rows = append(w.rows, afterLine(afterTemplates[r.Kind], Slots{"subject": ElementSpans(r.Subject)}))
		w.cur = -1
	}
}

// termLine is a line of a blow's working that is not a hit: what a relic did to the DMG, or every
// hit's total.
//
// **The name column is one width for both**, so the relic's figure and the total land in the same
// place.
func termLine(r session.LedgerRecord) session.LedgerLine {
	switch r.Role {

	case session.RoleDMG:
		// **It carries no figure in a hit's line's column**, because the bigger figure is already inside
		// every term below it. This says where it came from, which is the whole reason it is drawn:
		// a relic folded into a number the game already shows is a relic the player cannot see.
		// A rung's raise names the hand that paid it; the purse's names the purse.
		why := r.Hand
		if why == "" {
			why = r.Note
		}
		return session.LedgerLine{Voice: session.VoiceTerm, Spans: []session.LedgerSpan{
			{Text: fmt.Sprintf("%-14s", r.Relic+" ("+why+")"), Ink: session.InkRelic},
			{Text: SayText(SayTermDMG, Slots{"n": figure(r.Amount)}), Ink: session.InkRelic},
		}}

	default: // RoleTotal, on an account with no blow heading to write it on
		return session.LedgerLine{Voice: session.VoiceTerm, Spans: grandTotal(r.Total)}
	}
}

// workCol is the column a step's label starts at, in digit widths from where the step's text
// begins: room for an operator and the widest figure a step writes, a hit's total included.
const workCol = 9

// hitLines is one hit written down a column: the card's line, then **one step per figure that
// went into it, each labelled with what produced it**, then the `=` the hit came to.
//
//	Thrust (lightning)
//	      20   DMG
//	    x 1    Thrust
//	    x 2    Lightning Ring
//	    x 4    Prism Crown
//	    x 1.4  Two Pair
//	    = 448
//
// **The order is the card's term outward, with the hand last**: the DMG, what an Awakening woke
// into it, the card's own multiplier, the card's played riders, each relic that priced the card,
// the flat terms, each relic that multiplies the hand, then the hand itself — so the column ends on
// the rung the player built. The flats are added before any multiplier, as the resolver adds them;
// past that every step multiplies, so where the hand stands changes no figure. A landing a relic
// bought — an echo — is said on the card's line, because it is why this hit exists rather than a
// figure in it.
//
// **It computes nothing.** Every figure is a field of the record; the labels are the names the
// record carries beside them.
func hitLines(r session.LedgerRecord, hand string) []session.LedgerLine {
	var landed []string
	var priced []session.LedgerFactor
	for _, f := range r.Factors {
		if f.Landed {
			landed = append(landed, f.Relic)
		} else {
			priced = append(priced, f)
		}
	}

	head := []session.LedgerSpan{{Text: termCardName(r), Ink: r.Element}}
	// "Earth Jab Repeats - Twisted Points": the word in the card's own colour, since it is the card
	// landing again, and the relic that made it in the relic's.
	if len(landed) > 0 {
		head = append(head,
			session.LedgerSpan{Text: " " + SayText(SayHitAgain, nil), Ink: r.Element},
			session.LedgerSpan{Text: " - " + strings.Join(landed, " + "), Ink: session.InkRelic})
	}
	return append([]session.LedgerLine{{Voice: session.VoiceTerm, Spans: head}}, hitSteps(r, priced, hand)...)
}

// hitSteps is the column under a hit's line. `priced` is the record's factors less its landings,
// in the order hitRecord wrote them — what an Awakening added, the card's played riders, then each
// relic's price — which is the order the steps take them in.
func hitSteps(r session.LedgerRecord, priced []session.LedgerFactor, hand string) []session.LedgerLine {
	var out []session.LedgerLine
	add := func(op, figure, figureInk, label, labelInk string) {
		out = append(out, workStep(op, figure, figureInk, label, labelInk))
	}
	// next is the factor the next step is labelled by, or a relic nobody named on an account that
	// carries fewer factors than figures.
	next := func() session.LedgerFactor {
		if len(priced) == 0 {
			return session.LedgerFactor{Relic: "a relic"}
		}
		f := priced[0]
		priced = priced[1:]
		return f
	}

	for _, t := range r.Terms {
		if t.Split {
			// The DMG the card was swung at. What built it is written once, at the top of the turn —
			// see KindDMG — so every hit starts from the figure that block came to.
			add("", strconv.Itoa(t.DMG), "", SayText(SayStepBase, nil), "")
			if t.Awaken != 0 {
				add("+", strconv.Itoa(t.Awaken), session.InkRelic, next().Relic, session.InkRelic)
			}
			add("x", HandMultiplierText(t.Weight), t.Element, r.Card, t.Element)
		} else {
			// **The flat form already holds what the card and an Awakening made of the DMG**, so
			// its Awakening is passed over rather than added a second time.
			if t.Awaken != 0 {
				next()
			}
			add("", strconv.Itoa(t.Base), t.Element, r.Card, t.Element)
		}
		if t.PlayAdd != 0 {
			add("+", strconv.Itoa(t.PlayAdd), t.Element, next().Relic, t.Element)
		}
		if t.PlayPct != 0 {
			add("x", HandMultiplierText(t.PlayPct), t.Element, next().Relic, t.Element)
		}
		for _, pct := range t.Scales {
			f := next()
			label := f.Relic
			// **What the relic stood at after this hit**, written only where it moved — the one
			// case in which the same relic prices two hits of one blow differently.
			if f.Grown > 0 {
				label += SayText(SayFactorGrown, Slots{"n": figure(f.Grown)})
			}
			add("x", HandMultiplierText(pct), session.InkRelic, label, session.InkRelic)
		}
	}

	for _, flat := range r.Flats {
		add("+", strconv.Itoa(flat), "", SayText(SayStepFlat, nil), "")
	}

	// A hit whose record carries no terms still has its figure: an account nothing produces today,
	// written as the base it was rather than as a multiplier with nothing in front of it.
	if len(r.Terms) == 0 && len(r.Flats) == 0 {
		add("", strconv.Itoa(r.Base), r.Element, r.Card, r.Element)
	}

	// **A hand relic is its own step**, never a bigger figure in the hand's, which keeps the hand
	// reading as the rung the player built — one step per relic, each beside what it alone
	// multiplied by; an account that kept only the compounded figure writes it once.
	if r.HandScale != 0 && r.HandScale != 100 {
		if len(r.HandRelics) == 0 {
			add("x", HandMultiplierText(r.HandScale), session.InkRelic, "a relic", session.InkRelic)
		}
		for _, f := range r.HandRelics {
			add("x", HandMultiplierText(f.Scale), session.InkRelic, f.Relic, session.InkRelic)
		}
	}

	// **The hand is the last step**, labelled by name — which a hit record carries and an older one
	// leaves to the blow above it — so the column ends on the rung the player built.
	name := r.Hand
	if name == "" {
		name = hand
	}
	add("x", HandMultiplierText(r.Multiplier), session.InkHand, name, session.InkHand)

	// **The hit's figure is in its card's element**; red is kept for the blow's grand total alone.
	return append(out, session.LedgerLine{Voice: session.VoiceWork, Spans: []session.LedgerSpan{
		{Text: "= "},
		{Text: strconv.Itoa(r.Total), Ink: r.Element},
	}})
}

// workStep is one step of a hit's working: an operator, the figure in the ink of what produced it,
// and on the label column the name of what that was. **The operator is the panel's own ink**, the
// one thing on the line the game supplied rather than the player. The first step has no operator
// and is set in by one, so its figure sits over the figures below it.
func workStep(op, figure, figureInk, label, labelInk string) session.LedgerLine {
	if op == "" {
		op = " "
	}
	return session.LedgerLine{Voice: session.VoiceWork, Spans: []session.LedgerSpan{
		{Text: op + " "},
		{Text: figure, Ink: figureInk},
		{Text: label, Ink: labelInk, Col: workCol},
	}}
}

// termCardName is what a landing is called in a column of figures: the card, its element in front
// of it when it has one — `Earth Jab`, `Arcane Brace`. **Not a clause** — that is what an act line
// writes.
func termCardName(r session.LedgerRecord) string {
	if r.Element == "" {
		return r.Card
	}
	return capitalize(r.Element) + " " + r.Card
}

// afterLine is one line of what the player did between two fights: a marked verb and what it was
// done to.
//
// **The verb is marked exactly as an action's is**, so the aftermath can be scanned for what kind
// of thing happened before any of it is read — every SayAfter template opens on a `{mark:...}`.
// What it was done to goes through ElementSpans, so a fire card is named in the fire color here as
// it is everywhere else.
func afterLine(tmpl string, slots Slots) session.LedgerLine {
	return session.LedgerLine{Voice: session.VoiceYou, Spans: Say(tmpl, slots)}
}

// afterTemplates is the sentence for each record kind that names one thing the player did.
var afterTemplates = map[string]string{
	session.KindTook:  SayAfterTook,
	session.KindCut:   SayAfterCut,
	session.KindWore:  SayAfterWore,
	session.KindSold:  SayAfterSold,
	session.KindSpent: SayAfterSpent,
}

// figure is a number as a slot, in the panel's own ink.
func figure(n int) []session.LedgerSpan { return Plain(strconv.Itoa(n)) }

// cardClause is what follows the verb on an act line: "with a fire strike", "and raises a brace".
//
// **The element goes after the article rather than in front of the phrase**, which is what makes it
// a sentence instead of a label, and **the article is corrected rather than followed** — two of the
// five elements begin with a vowel, so "a earth strike" is a third of the lines this writes.
func cardClause(r session.LedgerRecord) []session.LedgerSpan {
	tmpl := SayActWith
	if r.Raises {
		tmpl = SayActRaising
	}
	card := lower(r.Card)
	article := "a "
	if r.Element != "" {
		el := lower(r.Element)
		if isVowel(el) {
			article = "an "
		}
		card = el + " " + card
	}
	return Say(tmpl, Slots{
		"card":   ElementSpans(article + card),
		"weight": Plain(cardWeightText(r.Weight)),
	})
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
		return SayVerbDefend
	}
	return SayVerbAttack
}

// usedVerb is what the duelist does with a consumable: a cantrip is cast, everything else is used.
func usedVerb(kind string) string {
	switch kind {
	case "cantrip":
		return SayVerbCast
	case "relic":
		return SayVerbWear
	}
	return SayVerbUse
}

// verbInkName is which ink that verb is written in, as the ledger names it.
func verbInkName(verb string) string {
	if verb == session.InkDefend {
		return session.InkDefend
	}
	return session.InkAttack
}
