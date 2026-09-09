package data

// The achievements: **what the player has done, as opposed to what a run has.**
//
// It is the first catalogue in this directory whose records are not offered, bought or played. An
// achievement is a *record* — it changes nothing about a duel — and the thing it may hand out is an
// unlock, which is a different key on a different list. See `internal/profile`, which holds both
// and keeps them apart.
//
// **A record is a name, two pieces of prose, and a trigger.** The prose is deliberately two things
// rather than one: `How` is what you must do and is legible while the row is still locked, and
// `Said` is what the game says once it has happened. A single line would have to be both, and the
// two want opposite tenses.
//
// **The trigger is the whole design.** Eleven achievements that each needed their own line of Go
// would be eleven places to forget, so the trigger is a small closed grammar with exactly three
// kinds — see below, and `internal/achieve`, which is the seat every word in it is refused at.
//
// **The rules do not read this file, and could not**: an achievement is the *player's*, and
// `internal/combat` has never heard of a player. `internal/achieve` parses it, `internal/screens`
// connects it to the profile and to the toast. Same who-consumes-it test every file here answers.

import (
	_ "embed"
	"encoding/json"
)

//go:embed achievements.json
var achievementsJSON []byte

// The three kinds of trigger. **Closed, and refused at load if a file invents a fourth** — the
// posture combat.Verb and session.WormTarget both take.
//
// They are three rather than one because the things they watch are genuinely different shapes: a
// turn is gone the moment it resolves, a count accumulates across every run the player has ever
// played, and a moment is a named thing the code already reaches. One mechanism covering all three
// would be a predicate over state that has to remember a hundred turns to answer "300 times".
const (
	// TriggerTurn is a pattern over the cards played in one turn. See ClauseData.
	TriggerTurn = "turn"

	// TriggerCount is a lifetime tally reaching a figure. The counter names are made by
	// internal/achieve, not written here as free text — a counter nothing ever bumps is an
	// achievement that can never land, so the name is checked against the vocabulary at load.
	TriggerCount = "count"

	// TriggerMoment is a named thing having happened. The list is closed and grows only when the
	// code genuinely gains a new one.
	TriggerMoment = "moment"
)

// The clause modes. A clause is read against the cards of one category in the played turn.
const (
	// ModeDistinct is **at least** N different values on the clause's axis *(owner's call,
	// 2026-09-06)*. At-least rather than exactly, so a five-element turn also earns the
	// four-element achievement — the two are rungs of one ladder and a player who skipped to the
	// top should not be missing the step below it.
	ModeDistinct = "distinct"

	// ModeSame is every one of the filtered cards agreeing on the clause's axis. An empty
	// selection satisfies nothing, so `same` also asserts there is at least one card.
	ModeSame = "same"

	// ModeCount is at least N filtered cards, and is the one mode with no axis. It exists for the
	// half of Arsenal that is "and a defence" — a presence rather than a shape.
	ModeCount = "count"
)

// The categories a clause may filter the turn by. Empty means the whole turn.
const (
	OfAttack = "attack"
	OfDefend = "defend"
)

// ClauseData is one condition over the cards of one category in a turn.
//
// **The filter is on the clause rather than on the pattern**, which is the one shape decision worth
// arguing with. A pattern-level filter would be tidier for four of the five turn achievements and
// could not express the fifth at all: Arsenal counts forms among the attacks *and* asks for a
// defence beside them, which is two different selections of the same turn.
type ClauseData struct {
	// Of is which cards this clause looks at: `attack`, `defend`, or empty for the whole turn.
	Of string `json:"Of,omitempty"`

	// Axis is what the clause counts on — `concept`, `form` or `element`, the three combat.Axis
	// values. Required by every mode but `count`, which has nothing to count on.
	Axis string `json:"Axis,omitempty"`

	// Mode is one of the three above.
	Mode string `json:"Mode"`

	// N is how many, read against the mode. `same` ignores it.
	N int `json:"N,omitempty"`

	// Cost narrows the selection to cards of exactly this AP, on top of Of *(2026-09-09)*.
	//
	// **A pointer, because zero is a real filter.** `duelist_cards.json` ships a 0 AP rung on every
	// attack ladder — Poke, Nick, Tap — so "the free ones" and "any cost" are different questions
	// and an int could not tell them apart.
	//
	// **It reads the card's cost, not its concept's**, so a worm that made a Lunge dearer counts it
	// and one that made an Impale cheaper does not. That is the right way round: the achievement is
	// about what the player paid for the turn, and the card's own face is what says it.
	//
	// It is on the clause rather than on the pattern for the reason Of is — Godslayer wants five
	// cards that agree on concept *and* all cost 4, which is one selection asked two questions.
	Cost *int `json:"Cost,omitempty"`
}

// PatternData is a set of clauses that must all hold.
type PatternData struct {
	Clauses []ClauseData `json:"Clauses"`
}

// TriggerData is what makes an achievement land.
//
// **One struct with three readings rather than three structs**, because JSON has no unions and the
// alternative is three optional sub-objects that a record could fill in two of. The Kind says which
// fields are live and internal/achieve refuses a record carrying one that is not.
type TriggerData struct {
	Kind string `json:"Kind"`

	// Patterns is the turn trigger: **any** pattern matching earns it. Most records carry one; the
	// alternation exists for Prism, which is a form five-of-a-kind or a card five-of-a-kind and is
	// one achievement either way.
	Patterns []PatternData `json:"Patterns,omitempty"`

	// Counter is the count trigger's tally name. N below is the figure it must reach.
	Counter string `json:"Counter,omitempty"`

	// Moment is the moment trigger's name, and Value is what the moment carries where it carries
	// anything — a concept label, for `card-altered`.
	Moment string `json:"Moment,omitempty"`
	Value  string `json:"Value,omitempty"`

	// N is read by both `count` and `floor-reached`, and is a **threshold rather than an equality**
	// in each: reaching floor six earns the floor-five row, exactly as at-least does on a clause.
	N int `json:"N,omitempty"`
}

// AchievementData is one achievement as written in the file.
type AchievementData struct {
	// AchievementRecord is the key, and **it is the disk contract**: it is written into
	// `profile.json` and is the thing that must never be renamed once a build has shipped. Every
	// other field here can be rewritten any afternoon. Kebab-case, like a ring's and a worm's.
	AchievementRecord string `json:"AchievementRecord"`

	// Name is what the page calls it, in caps, like the rows already on that screen.
	Name string `json:"Name"`

	// How is what you must do, in the imperative. **It is legible while the row is still locked**,
	// which is the whole reason the achievements page lists what has not been earned.
	How string `json:"How"`

	// Said is what the game says once it lands, and **every line is shown at once** *(owner's call,
	// 2026-09-06)*. Picking one of several would be a roll, and a roll owes its own salted stream
	// and its own argument — see the `randomness` skill. Several lines shown together cost neither.
	Said []string `json:"Said,omitempty"`

	// Unlocks is what earning this opens up, by unlock key.
	//
	// **An achievement never gates anything itself** *(owner's call, 2026-09-06)*. profile.go draws
	// the line: an achievement is a record and changes nothing, an unlock is an input to the rules.
	// So a ring behind an achievement reads the *unlock*, and this field is the bridge — two keys
	// rather than one, which is what lets an achievement be retired without orphaning a ring.
	//
	// Nothing is behind an achievement yet, so every record ships without one.
	Unlocks []string `json:"Unlocks,omitempty"`

	Trigger TriggerData `json:"Trigger"`
}

// LoadAchievements parses the catalogue.
//
// **A slice rather than a map, deliberately** — the same call LoadDuelistCards makes. File order is
// page order: the achievements screen has listed its rows in an authored order since it was written,
// because a page that reshuffled itself between visits is a page you have to re-read each time. A
// map would hand that decision to Go's hashing, and a sorted walk would hand it to the alphabet,
// which puts "Arsenal" above "First Steps".
//
// **Shape only.** Whether a trigger says anything the game can act on is internal/achieve's
// question, exactly as a worm's target is internal/session's and a card's verb is
// internal/combat's. This file may not import either.
func LoadAchievements() []AchievementData {
	var list []AchievementData
	if err := json.Unmarshal(achievementsJSON, &list); err != nil {
		panic("Failed to unmarshal achievements.json: " + err.Error())
	}

	seen := make(map[string]bool, len(list))
	for _, a := range list {
		if a.AchievementRecord == "" {
			panic("achievements.json: a record has no AchievementRecord")
		}
		if seen[a.AchievementRecord] {
			panic("achievements.json: two records keyed " + a.AchievementRecord)
		}
		seen[a.AchievementRecord] = true
	}
	return list
}
