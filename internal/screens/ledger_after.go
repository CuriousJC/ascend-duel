package screens

// **What the player did between two fights, written into the run's account.**
//
// The ledger recorded rounds and nothing else, so it went silent at the moment the duel ended and
// picked up again at the next one — with the deck, the relic row and the pouch all different and
// no line saying why. Everything the player *chose* happened in that gap. See
// session.LedgerFight.After, which is where these lines are filed, and session.Holdings, which is
// what they are derived from.
//
// **Read off the run, not announced by the screens** *(owner's call, 2026-09-12)*. The obvious
// version is a call beside every commit — the essence applied, the relic bought, the stone spent, the
// card taken — and that is a list a new mechanic gets left off, silently, because a missing
// announcement and a deliberate silence read identically. This watches the run instead and words
// whatever moved, which is combat_handmorph.go's rule one screen over: what changed is read off the
// faces rather than off the rune, so no rune has a case anywhere in the drawing and a new
// one cannot arrive with no picture.
//
// **One call site, in internal/game**, for the reason the ledger panel lives there: it is true of
// the whole run rather than of one screen, and a watcher ticked by two scenes is a watcher the
// third scene forgets.
//
// **It records nothing during a fight.** A rune spent mid-round is an event of that round and
// belongs to the round's own lines; this is the account of the gap between them.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/carddesc"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// RunWatch remembers what the run was holding a frame ago.
//
// **Exported because internal/game holds it**, exactly as it holds the ledger panel, and for the
// same reason: the thing being watched outlives every scene.
type RunWatch struct {
	// run is which session the snapshot belongs to. **Identity rather than a flag**, so a run
	// started fresh from the title screen cannot be diffed against the last one's deck — which
	// would write the whole of a new deck into the previous run's account.
	run  *session.Session
	last session.Holdings
}

// Note takes this frame's snapshot and writes down anything that moved since the last one.
func (w *RunWatch) Note(gs *state.GlobalState) {
	if gs == nil || gs.Run == nil {
		w.run, w.last = nil, session.Holdings{}
		return
	}

	now := gs.Run.Holdings()
	if w.run != gs.Run {
		w.run, w.last = gs.Run, now
		return
	}

	// A fight's own alterations are events of that fight. See the file comment.
	if gs.Run.Phase() == session.PhaseFight {
		w.last = now
		return
	}

	if lines := afterLines(gs, w.last, now); len(lines) > 0 {
		gs.Run.RecordAfter(lines)
	}
	w.last = now
}

// afterLines is the whole diff, worded.
//
// **The order is the order a player reads it in**: what happened to the deck first, because that is
// what the block is for, then what they are wearing and carrying, then what it cost.
func afterLines(gs *state.GlobalState, before, after session.Holdings) []session.LedgerLine {
	var out []session.LedgerLine
	out = append(out, deckLines(before.Cards, after.Cards)...)
	out = append(out, keyLines(gs, "wore", "sold", before.Relics, after.Relics)...)
	out = append(out, keyLines(gs, "took", "spent", before.Held, after.Held)...)
	out = append(out, keyLines(gs, "took", "spent", before.Pouch, after.Pouch)...)
	out = append(out, stoneLines(before.Stones, after.Stones)...)
	out = append(out, vitaeLines(before.Vitae, after.Vitae)...)
	return out
}

// deckLines is what happened to the cards: one taken, one cut, one turned into something else.
//
// **Cards are told apart by combat.Card.ID**, which is what makes the third case sayable at all: a
// card that was altered and a card that was cut with another taken in its place look identical to
// anything counting faces. The ids are exactly why they exist — see CLAUDE.md on runes naming
// card identities rather than deck positions.
func deckLines(before, after []combat.Card) []session.LedgerLine {
	was := make(map[int]combat.Card, len(before))
	for _, c := range before {
		was[c.ID] = c
	}
	now := make(map[int]combat.Card, len(after))
	for _, c := range after {
		now[c.ID] = c
	}

	var out []session.LedgerLine
	for _, c := range after {
		old, held := was[c.ID]
		switch {
		case !held:
			out = append(out, afterLine("took", cardWords(c)))
		case !sameFace(old, c):
			out = append(out, afterLine("changed", cardWords(old)+" into "+cardWords(c)))
		}
	}
	for _, c := range before {
		if _, held := now[c.ID]; !held {
			out = append(out, afterLine("cut", cardWords(c)))
		}
	}
	return out
}

// sameFace reports whether two records of one card say the same thing.
//
// **The id is set aside and everything else compared**, rather than a field list: combat.Card is
// comparable on purpose — the render cache and TestRoundIsDeterministic both depend on it — so the
// whole card is one equality, and a field added to it joins this check without being added to it.
func sameFace(a, b combat.Card) bool {
	a.ID, b.ID = 0, 0
	return a == b
}

// cardWords is what a card is called in a sentence about the deck.
//
// **Not the card's title**, which is set in capitals because a card face shouts and a line in the
// account is prose. The form is named only when an essence has moved it, since that is the one thing
// about an altered card the name would otherwise not say — the same gap that had an altered Crush
// drawing a spear over the word CRUSH.
func cardWords(c combat.Card) string {
	name := c.Label()
	if word := carddesc.ElementWord(c); word != "" {
		name = strings.ToLower(word) + " " + name
	}

	var notes []string
	if c.Form() != c.Spec().Form && c.Form() != combat.FormNone {
		notes = append(notes, strings.ToLower(attackVerb(c.Form())))
	}
	for _, line := range carddesc.RiderLines(c) {
		notes = append(notes, strings.ToLower(line))
	}
	if len(notes) > 0 {
		name += " (" + strings.Join(notes, ", ") + ")"
	}
	return name
}

// keyLines is the diff of a list of keys — the relic row, the runes in hand, the stones in the
// pouch — with a verb for one arriving and one for one leaving.
//
// **By count rather than by set**, because all three lists can hold the same key twice and a set
// would report a second copy as nothing having happened.
func keyLines(gs *state.GlobalState, gained, lost string, before, after []string) []session.LedgerLine {
	delta := map[string]int{}
	for _, k := range before {
		delta[k]--
	}
	for _, k := range after {
		delta[k]++
	}

	var out []session.LedgerLine
	for _, k := range sortedKeys(delta) {
		n := delta[k]
		verb := gained
		if n < 0 {
			verb, n = lost, -n
		}
		for i := 0; i < n; i++ {
			out = append(out, afterLine(verb, goodsName(gs, k)))
		}
	}
	return out
}

// stoneLines is a rung the run has raised. The stone leaves the pouch and lands here, so the pair
// reads as one act in two lines — spent, then what it bought.
func stoneLines(before, after map[string]int) []session.LedgerLine {
	var out []session.LedgerLine
	for _, hand := range sortedKeys(after) {
		if after[hand] > before[hand] {
			out = append(out, afterLine("raised",
				fmt.Sprintf("%s to +%d", handWords(hand), after[hand])))
		}
	}
	return out
}

// vitaeLines is what the gap cost or paid.
//
// **One line per change rather than a net figure for the block**, because the block is an account:
// a purse reporting only its balance at the end would leave a relic bought and a stone bought as
// one number neither of them explains.
func vitaeLines(before, after int) []session.LedgerLine {
	switch {
	case after > before:
		return []session.LedgerLine{afterLine("gained", fmt.Sprintf("%d vitae", after-before))}
	case after < before:
		return []session.LedgerLine{afterLine("spent", fmt.Sprintf("%d vitae", before-after))}
	}
	return nil
}

// afterLine is one line of the block: a marked verb and what it was done to.
//
// **The verb is marked exactly as an action's is**, so the aftermath can be scanned for what kind
// of thing happened before any of it is read — the rule the round lines are already under. The rest
// goes through elementSpans, so a fire card is named in the fire colour here as it is everywhere
// else.
func afterLine(verb, clause string) session.LedgerLine {
	spans := []session.LedgerSpan{{Text: verb, Mark: true}}
	spans = append(spans, elementSpans(" "+clause)...)
	return session.LedgerLine{Voice: session.VoiceYou, Spans: spans}
}

// goodsName is what a relic, rune or stone is called on screen, falling back to its key.
//
// **A key is named rather than hidden**, for relicName's reason: a line in a saved account has to
// read as something, and a record this build no longer has is better admitted than dropped.
func goodsName(gs *state.GlobalState, key string) string {
	if gs != nil {
		if record, ok := gs.Relics[key]; ok {
			return record.Name
		}
	}
	if p, ok := session.RuneByKey(key); ok {
		return p.Name
	}
	if st, ok := session.StoneByKey(key); ok {
		return st.Name
	}
	return key
}

// handWords is a rung's name, falling back to its key on goodsName's terms.
func handWords(key string) string {
	if id, ok := combat.HandIDForKey(key); ok {
		if h, found := combat.HandByID(id); found {
			return h.Name
		}
	}
	return key
}

// sortedKeys walks a map in one order, because map iteration order may never decide anything —
// here, the order two lines are written into a run's permanent account in.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
