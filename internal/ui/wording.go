package ui

// **The words: which sentence of data/wording.json each line of the game is, and what fills it.**
//
// The sentences themselves are in data/wording.json, where they are edited. This file names each
// one and declares the holes its code fills — so the template can be reworded and reordered freely,
// and what cannot change without a code change is the list beside it.
//
// **Both halves are checked at launch.** A key the file lacks panics, and so does a sentence whose
// `{holes}` are not exactly the ones listed here: a hole the template lost is a figure that silently
// stops being shown, and one it gained would draw as its own braces. `{ink:word}` and `{mark:word}`
// are words rather than holes and may be added anywhere. TestEveryWordingKeyIsUsed holds the
// other direction — a sentence in the file nothing here asks for.
//
// **What is not here** is the arithmetic: a blow's working is `(10 x 2) + 4 = 24`, notation rather
// than a sentence, and it is still built a figure at a time in ledger_prose.go.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
)

// The reward screen's payout, read out a line at a time. See screens.payoutLines. The heading, the
// three claims and the total are rows of one block; the code puts the amounts — SayPayoutGain and
// SayPayoutPurse, `¤` being the vitae mark — at its right edge and fills the gap with
// SayPayoutLeader.
var (
	SayPayoutHeading  = word("payout.heading")
	SayPayoutInterest = word("payout.interest", "per", "max")
	SayPayoutLife     = word("payout.life")
	SayPayoutRoom     = word("payout.room")
	SayPayoutTotal    = word("payout.total")
	SayPayoutGain     = word("payout.gain", "n")
	SayPayoutPurse    = word("payout.purse", "n")
	SayPayoutLeader   = word("payout.leader")
	SayPayoutEssence  = word("payout.essence")
)

// A card being played: "Duelist attacks with a fire strike (1.5x)". {verb} is the marked verb and
// {card} arrives with its article already chosen — "a" or "an" — by the code.
var (
	SayAct        = word("act.line", "who", "verb", "clause")
	SayActWith    = word("act.with", "card", "weight")
	SayActRaising = word("act.raising", "card", "weight")
	SayVerbAttack = word("act.attacks")
	SayVerbDefend = word("act.defends")
)

// Something spent from the consumables pane: "Duelist casts Cantrip of Might - DMG 10 to 20",
// "Duelist uses Embermark on a jab and a bash" — and the cantrip-relic a cast puts on, "Duelist wears
// Mighty - +10 DMG for this fight.".
var (
	SayUsed     = word("used.line", "who", "verb", "subject")
	SayUsedInto = word("used.into", "into")
	SayUsedOn   = word("used.on", "into")
	SayVerbCast = word("used.casts")
	SayVerbUse  = word("used.uses")
	SayVerbWear = word("used.wears")
)

// What became of a card, hung on the end of its line. The first outcome on a line follows
// SayOutcomeFirst and every later one SayOutcomeNext.
var (
	SayOutcomeFirst = word("outcome.first")
	SayOutcomeNext  = word("outcome.next")
	SayFizzled      = word("outcome.fizzled")
	SayDrained      = word("outcome.drained", "relic", "n")
	SayRaised       = word("outcome.raised", "shields")
	SayHeld         = word("outcome.held", "n")
	SayReflected    = word("outcome.reflected", "relic", "n")
	SayTithed       = word("outcome.tithed", "relic", "n")
	SaySilver       = word("outcome.silver", "n")
	SayLapsed       = word("outcome.lapsed", "shields")
	SayBlocked      = word("outcome.blocked", "shields")
	SayHitBack      = word("outcome.hit-back", "n")
	SayDamage       = word("outcome.damage", "n")
)

// Lines of their own, which nothing attaches to.
var (
	SayRegenerated = word("announce.regenerated", "who", "n", "relic")
	SayWarded      = word("announce.warded", "who", "shields", "relic")
	SayTimeUp      = word("announce.time-up", "who", "n")
	SayDefeated    = word("announce.defeated", "who")
)

// What the player did between two fights.
var (
	SayAfterTook    = word("after.took", "subject")
	SayAfterCut     = word("after.cut", "subject")
	SayAfterWore    = word("after.wore", "subject")
	SayAfterSold    = word("after.sold", "subject")
	SayAfterSpent   = word("after.spent", "subject")
	SayAfterChanged = word("after.changed", "subject", "into")
	SayAfterRaised  = word("after.raised", "hand", "n")
	SayAfterGained  = word("after.gained", "n")
	SayAfterPaid    = word("after.paid", "n")
)

// The words inside a blow's working. The name column is padded in code, so these are only the
// words around the figures.
var (
	SayTermDMG     = word("term.dmg", "n")
	SayTermFlat    = word("term.flat", "n")
	SayTermTotal   = word("term.total")
	SayFactorLands = word("term.lands", "relic")
	SayFactorGrown = word("term.grown", "n")
)

// wording is the file, loaded once; wordingAsked is every key something here named.
var (
	wording      = data.LoadWording()
	wordingAsked = map[string]bool{}
)

// word is one sentence of the file, by `section.key`, refused unless its holes are exactly these.
func word(key string, holes ...string) string {
	section, name, _ := strings.Cut(key, ".")
	tmpl, ok := wording[section][name]
	if !ok {
		panic(fmt.Sprintf("wording.json: no sentence %q", key))
	}
	got, want := templateHoles(tmpl), slices.Clone(holes)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		panic(fmt.Sprintf("wording.json: %q has holes %v, and its code fills %v", key, got, want))
	}
	wordingAsked[key] = true
	return tmpl
}
