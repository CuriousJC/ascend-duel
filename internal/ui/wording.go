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

// The reward screen's payout, read out a line at a time. See screens.payoutLines. The three claims
// are rows of one box titled SayPayoutHeading; the code puts each amount — as that many vitae dots —
// at its row's right edge and fills the gap with SayPayoutLeader.
var (
	SayPayoutHeading  = word("payout.heading")
	SayPayoutInterest = word("payout.interest", "per", "max")
	SayPayoutLife     = word("payout.life")
	SayPayoutRoom     = word("payout.room")
	SayPayoutLeader   = word("payout.leader")
	SayPayoutEssence  = word("payout.essence")
)

// A fighter card's tooltip: every figure on the card, in words. See DuelistTip and PlayerTip.
var (
	SayFighterDMG     = word("fighter.dmg", "n")
	SayFighterAP      = word("fighter.ap", "n")
	SayFighterAPSurge = word("fighter.ap-surge", "n", "surge")
	SayFighterVitae   = word("fighter.vitae", "n")
	SayFighterHP      = word("fighter.hp", "life", "max")
	SayFighterWhere   = word("fighter.where", "realm", "room")
)

// Where the journey is: the three rooms of a realm, and a realm by its ordinal — written out to the
// twenty-fifth, and by its number past that. See RoomNames and RealmName.
var (
	SayRoomOuter     = word("journey.outer")
	SayRoomInner     = word("journey.inner")
	SayRoomPortal    = word("journey.portal")
	SayRealm         = word("journey.realm", "ordinal")
	SayRealmDeep     = word("journey.realm-deep", "n")
	SayRealmOrdinals = ordinals("journey.ordinal-")
)

// ordinals is every `prefix-1`, `prefix-2`, … sentence the file carries, in order, stopping at the
// first number it lacks.
func ordinals(prefix string) []string {
	var out []string
	for n := 1; ; n++ {
		key := fmt.Sprintf("%s%d", prefix, n)
		section, name, _ := strings.Cut(key, ".")
		if _, ok := wording[section][name]; !ok {
			return out
		}
		out = append(out, word(key))
	}
}

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
	SayFizzled      = word("outcome.fizzled", "what")
	SayDrained      = word("outcome.drained", "relic", "n")
	SayRaised       = word("outcome.raised", "shields")
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
	SayHeld        = word("announce.held", "card", "n")
	SayHeldCard    = word("announce.held-card")
	SayTimeUp      = word("announce.time-up", "who", "n")
	SayDefeated    = word("announce.defeated", "who")
	SayDMGBuilt    = word("announce.dmg", "who")
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

// The words inside a blow's working. A hit is written down a column — a figure, then what produced
// it — so most labels are a name the record already carries; these are the rest.
var (
	SayTermDMG     = word("term.dmg", "n")
	SayTermGrand   = word("term.grand", "n")
	SayTermApplied = word("term.applied")
	SayTermOver    = word("term.over")
	SayHitAgain    = word("term.again")
	SayStepBase    = word("term.base")
	SayStepDMG     = word("term.dmg-total")
	SayStepFlat    = word("term.flat")
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
