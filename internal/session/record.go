package session

// **What happened, as flat named data — the thing the ledger actually stores.**
//
// A record is one thing the round or the gap between two of them did, written as fields rather
// than as a sentence. The words are derived from it when a panel draws it or a file is written —
// see internal/ui/ledger_prose.go, which is the one translator — so a change to the wording
// reaches every account already on disk, and a machine reading a record gets data rather than
// English.
//
// **Records are outputs, never inputs.** A record is what the resolver produced and what the run
// watcher saw move; it cannot re-drive a round, and nothing may treat it as though it could. What
// puts a run back where it was is the player's own choices, which are a different file.
//
// **Every field is a name, a number or an already-resolved label**, because a record outlives the
// build that wrote it harder than a save does. A concept, a relic, a status and a hand are written
// as the words they were called, so a catalog edit leaves an old account readable rather than
// blank. Nothing here is an ordinal and nothing here is a color.
//
// **It is not combat.Event.** `HandGrown`, `HandRelicScale` and `HandLanding` are 25x5 arrays each,
// so an event is about 2.5 KB and nearly all of it zero; a record carries the terms a blow actually
// had. The events are still the source — `internal/screens` turns a resolved round into records the
// moment it ends — and the resolver is still the only thing that decides anything.

// The record kinds. **One word each, append-only, and a name rather than an ordinal**, on the rule
// every saved file in this game is under.
//
// **A kind with no translation is a blank line**, which is why RecordKinds and the translator are
// checked against each other by a test: a record nobody worded and a record nobody wrote look
// identical on the panel.
const (
	// A round, as the resolver decided it.

	// KindAct is one duelist playing one card. It is the line every outcome below attaches to.
	KindAct = "act"

	// KindBlow is the attack phase: the hand that formed and what it came to. **A hand-forming
	// side plays several cards and lands one figure**, so the cards write no act of their own and
	// this is the phase's line. A solo attacker has no blow and writes an act per card.
	KindBlow = "blow"

	// KindTerm is one line of a blow's working: a card's own figure, what a relic priced it at, a
	// flat term a relic paid, or the sum underneath. Which one is Term.Role.
	KindTerm = "term"

	// The outcomes. **Each attaches to the act or blow above it** rather than opening a line,
	// because they are what became of a card that was played and the sentence for it is already
	// written.
	KindDamage  = "damage"
	KindStatus  = "status"
	KindDrained = "drained"
	KindMissed  = "missed"
	KindBlocked = "blocked"
	KindRaised  = "raised"
	KindLapsed  = "lapsed"
	KindHeld    = "held"
	KindSilver  = "silver"

	// The announcements. **Each opens a line of its own**, because there is nothing above them to
	// attach to: they happen at the top of a turn, at the end of a round, or to a duelist rather
	// than to a card.
	KindChilled     = "chilled"
	KindRegenerated = "regenerated"
	KindTicked      = "ticked"
	KindTimeUp      = "time-up"
	KindDefeated    = "defeated"

	// The gap between two fights, as the run watcher saw it. **A verb each**, because what the
	// block is for is saying what the player chose, and the verb is the thing being scanned for.
	KindTook       = "took"
	KindCut        = "cut"
	KindChanged    = "changed"
	KindWore       = "wore"
	KindSold       = "sold"
	KindSpent      = "spent"
	KindRaisedRung = "raised-rung"
	KindGained     = "gained"
	KindPaid       = "paid"
)

// The sides a record can belong to. **Words rather than combat.Side**, because a record is saved
// and an ordinal in a file eventually means something else.
const (
	SideYou = "you"
	SideFoe = "foe"
)

// The roles a term line can take. A blow's working is several kinds of line in one column, and the
// role is what tells them apart without the translator having to infer it from which fields are
// filled.
const (
	// RoleCard is one landing: a card that was in the hand, what it was worth, and what each relic
	// did to it. **A landing rather than a card** — an echoed card lands twice and writes two.
	RoleCard = "card"

	// RoleFlat is a term no card paid: the cards kept back, or the purse. It names the relic that
	// put it in the sum.
	RoleFlat = "flat"

	// RoleDMG is what a relic did to the DMG the blow was swung at. **It is not a term and carries
	// no figure in the sum's column** — the bigger figure is already inside every card term below
	// it, and this says where it came from. Without it the relic would be invisible, which is the
	// one thing a relic may never be.
	RoleDMG = "dmg"

	// RoleSum is the blow written out as the sum it is, under the terms it adds up.
	RoleSum = "sum"
)

// LedgerRecord is one thing that happened.
//
// **One struct rather than one per kind**, because the panel switches on the kind and a Go
// interface here would be a type assertion at every reader for no gain. **Every field is
// `omitempty`**, so what reaches a file is the handful of fields the kind actually uses — which is
// the narrow record the format promises, whatever the struct looks like in Go.
type LedgerRecord struct {
	// Kind is one of the names above.
	Kind string `json:"kind"`

	// Side is whose record it is, and Target who it happened to. **Both, because they differ**: a
	// burn ticks on the duelist it was put on, and a blow belongs to whoever swung it.
	Side   string `json:"side,omitempty"`
	Target string `json:"target,omitempty"`

	// Name is what the duelist was called at the time. **Stored rather than looked up**, because
	// the account is read back after the fight and the creature that was standing there is gone.
	Name string `json:"name,omitempty"`

	// Card is a concept's label, Element the element it carried, and Verb which half of the turn
	// it belonged to — "attack" or "defend".
	Card    string `json:"card,omitempty"`
	Element string `json:"element,omitempty"`
	Verb    string `json:"verb,omitempty"`

	// Raises says the card puts something up rather than being swung at somebody, which is the one
	// fact the clause around it turns on: "and raises a fire brace" against "with a fire jab".
	// **A property of the card, not a choice about the wording** — the translator owns the phrase
	// and this is what it needs to pick one.
	Raises bool `json:"raises,omitempty"`

	// Weight is what an attack card multiplies its owner's DMG by, as a percentage. **Attacks
	// only and never the identity**, which is what keeps `(1x)` off every ordinary swing.
	Weight int `json:"weight,omitempty"`

	// Status is a status's key and Relic a relic's name — the two things that name themselves in a
	// line, so that a second status or a second relic doing the same thing cannot narrate
	// identically to the first.
	Status string `json:"status,omitempty"`
	Relic  string `json:"relic,omitempty"`

	// Amount is the figure the record is about: damage dealt, shields standing, vitae paid.
	Amount int `json:"amount,omitempty"`

	// Hand is what the blow formed, already carrying its axis, and Multiplier what the rung paid
	// as a percentage. HandScale is a rung relic's second multiplier, which is a second factor on
	// the line rather than a bigger figure in the first.
	Hand       string `json:"hand,omitempty"`
	Multiplier int    `json:"multiplier,omitempty"`
	HandScale  int    `json:"handScale,omitempty"`

	// Role is which kind of term line this is, from the list above. Set on KindTerm and nothing
	// else.
	Role string `json:"role,omitempty"`

	// Base is a term's own figure before its relics, and Note what a flat term counted — "4 cards
	// kept back", "the purse".
	Base int    `json:"base,omitempty"`
	Note string `json:"note,omitempty"`

	// Factors is what the relics did to one term, in worn order, which is firing order. **A
	// landing and a multiplier are different things** and say so, because a relic that buys a term
	// contributes no arithmetic and a line crediting it with some would be wrong.
	Factors []LedgerFactor `json:"factors,omitempty"`

	// Terms is the sum's own figures, one per landing, so the sum line can be written out without
	// re-reading the terms above it.
	Terms []LedgerSum `json:"terms,omitempty"`

	// Flats are the flat terms inside the sum, in the order the resolver adds them.
	Flats []int `json:"flats,omitempty"`

	// Total is what the blow came to.
	Total int `json:"total,omitempty"`

	// Subject is what an after-the-fight record was done to: a card in words, a relic's name, a
	// rung's name. **Already worded**, because a deck line describes a card by more than its
	// label — its element, a form an essence moved, the odds a rider carries — and a record
	// storing the pieces would be a second card renderer.
	Subject string `json:"subject,omitempty"`

	// Into is what a changed card became, which is the one after-record with two subjects.
	Into string `json:"into,omitempty"`
}

// LedgerFactor is one relic's contribution to one term.
//
// **Landed and a multiplier are kept apart.** An echo relic buys a landing and contributes no
// figure, so a factor written as `x Echo 1x` would credit it with arithmetic it did not do.
type LedgerFactor struct {
	// Relic is the relic's name.
	Relic string `json:"relic"`

	// Landed says this relic bought the term rather than priced it. Scale is then unread.
	Landed bool `json:"landed,omitempty"`

	// Scale is what it priced the term at, as a percentage. **A relic firing at the identity still
	// fired** and is written, because leaving it out is how a growing relic's climb off 1x becomes
	// invisible.
	Scale int `json:"scale,omitempty"`

	// Grown is what a growing relic stood at after this term, written only where it moved — the
	// one case in which the same relic prices two terms of one blow differently.
	Grown int `json:"grown,omitempty"`
}

// LedgerSum is one landing's figures as the sum reads them.
//
// **The term is the product the game worked out, not its answer**: the DMG the hand was swung at,
// times the card's own multiplier, times whatever a relic priced it at. A term the split cannot
// describe — an echo's rounding, or a card that hit the damage floor — carries Base instead and
// says so with Split false.
type LedgerSum struct {
	// Element is the card's, so a figure in the sum wears its own card's color.
	Element string `json:"element,omitempty"`

	// Split says the first two figures are real. DMG is what the blow was swung at and Weight the
	// card's own multiplier.
	Split  bool `json:"split,omitempty"`
	DMG    int  `json:"dmg,omitempty"`
	Weight int  `json:"weight,omitempty"`

	// Base is the flat form, for a term the split cannot describe.
	Base int `json:"base,omitempty"`

	// Scales is what the relics priced it at, in worn order.
	Scales []int `json:"scales,omitempty"`
}

// RecordKinds is every kind a record can be, which is what makes the translator checkable.
//
// **A list rather than a derived set**, because the kinds are constants and Go cannot walk them.
// The cost is that a kind added above has to be added here too, and the payment is
// internal/ui.TestEveryRecordKindReadsAsSomething, which fails on a kind nobody worded — where a
// record with no case draws as a blank line and looks exactly like a record nobody wrote.
func RecordKinds() []string {
	return []string{
		KindAct, KindBlow, KindTerm,
		KindDamage, KindStatus, KindDrained, KindMissed, KindBlocked,
		KindRaised, KindLapsed, KindHeld, KindSilver,
		KindChilled, KindRegenerated, KindTicked, KindTimeUp, KindDefeated,
		KindTook, KindCut, KindChanged, KindWore, KindSold, KindSpent,
		KindRaisedRung, KindGained, KindPaid,
	}
}
