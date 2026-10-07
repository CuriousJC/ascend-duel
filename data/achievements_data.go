package data

// The achievements: **what the player has done, as opposed to what a run has.**
//
// It is the first catalog in this directory whose records are not offered, bought or played. An
// achievement is a *record* — it changes nothing about a duel — and the thing it may hand out is an
// unlock, which is a different key on a different list. See `internal/profile`, which holds both
// and keeps them apart.
//
// **A record is Steam's achievement fields, one line of our own prose, and a trigger.** The prose is
// deliberately two things rather than one: `Description` is what you must do and is legible while
// the row is still locked, and `Said` is what the game says once it has happened. A single line
// would have to be both, and the two want opposite tenses.
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
)

//go:embed achievements.json
var achievementsJSON []byte

// The three kinds of trigger. **Closed, and refused at load if a file invents a fourth** — the
// posture combat.Verb and session.EssenceTarget both take.
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
	// half of Arsenal that is "and a defense" — a presence rather than a shape.
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
// defense beside them, which is two different selections of the same turn.
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
	// anything — a rung's key for `hand-formed`, a direction for `ladder-wrapped`.
	Moment string `json:"Moment,omitempty"`
	Value  string `json:"Value,omitempty"`

	// N is read by both `count` and `realm-reached`, and is a **threshold rather than an equality**
	// in each: reaching realm six earns the realm-five row, exactly as at-least does on a clause.
	N int `json:"N,omitempty"`
}

// SetByClient is the one `SetBy` the game writes. **Steam's three values are Client, GS and
// Official GS**, and the two server ones mean a game server vouches for the unlock. Every
// achievement here is earned on the player's own machine, so a record naming a server would be a
// promise nothing keeps — the vocabulary is closed at this one word, and internal/achieve refuses
// any other.
const SetByClient = "client"

// AchievementIconDir is where the achieved icons are committed, and AchievementIconPrefix the
// prefix their asset keys carry — `assets/achievement/first-steps.png` is `achievement-first-steps`.
// **Prefixed** for the upgrade art's reason: the image map is flat, and an achievement named
// `prism` or `arsenal` is a name a relic could take.
const (
	AchievementIconDir    = "achievement"
	AchievementIconPrefix = "achievement-"

	// DefaultAchievementIcon is the stem an achievement with no icon of its own draws.
	DefaultAchievementIcon = "default"

	// AchievementIconSize is the side of the square every icon is committed at — **Steam's own
	// upload size**, which it shows reduced to 64. The game's page draws it at that 64 too, so the
	// icon a player sees in the game is the one Steam will show.
	AchievementIconSize = 256
)

// AchievementData is one achievement as written in the file.
//
// **The first five fields are Steamworks' own achievement fields, under Steamworks' own names** —
// API Name, Display Name, Description, Set By, Hidden — and the icon pair is the sixth and seventh.
// The file is the source the Steamworks admin page is filled in from, so a field that means the same
// thing as one there is called what it is called there.
type AchievementData struct {
	// APIName is the key, and **it is the disk contract**: it is written into `profile.json` and is
	// the thing that must never be renamed once a build has shipped — on Steam as well, where it is
	// the string the API unlocks by. Every other field here can be rewritten any afternoon.
	// Kebab-case, like a relic's and an essence's.
	APIName string `json:"APIName"`

	// DisplayName is what the page and the toast call it, in caps, like the rows already on that
	// screen — and the name Steam's pop-up and the Community page show.
	DisplayName string `json:"DisplayName"`

	// Description is what you must do, in the imperative. **It is legible while the row is still
	// locked**, which is the whole reason the achievements page lists what has not been earned.
	Description string `json:"Description"`

	// SetBy is who may unlock it, and is always SetByClient. Written on every record anyway, because
	// it is a Steamworks field and this file is what that form is filled in from.
	SetBy string `json:"SetBy"`

	// Hidden keeps a locked achievement's name, description and icon off the page until it is
	// earned — Steam's own meaning, where a hidden achievement does not appear on the Community page
	// at all until it is achieved.
	Hidden bool `json:"Hidden"`

	// AchievedIcon is the icon's stem under assets/achievement/, and **empty draws the default**.
	// It is the Steamworks "Achieved Icon"; **there is no UnachievedIcon field**, because the
	// unachieved icon is this one in grayscale, derived rather than drawn — a second picture per
	// record would be a second thing to keep in step and a generator's chance to draw it differently.
	AchievedIcon string `json:"AchievedIcon"`

	// Draw is the subject the icon is generated from, pasted after
	// docs/art/achievement_art_prompt.MD. **Ignored by the game**, like every catalog's Draw.
	Draw string `json:"Draw"`

	// Said is what the game says once it lands, and **every line is shown at once** *(owner's call,
	// 2026-09-06)*. Picking one of several would be a roll, and a roll owes its own salted stream
	// and its own argument — see the `randomness` skill. Several lines shown together cost neither.
	Said []string `json:"Said,omitempty"`

	// Unlocks is what earning this opens up, by unlock key.
	//
	// **An achievement never gates anything itself** *(owner's call, 2026-09-06)*. profile.go draws
	// the line: an achievement is a record and changes nothing, an unlock is an input to the rules.
	// So a relic behind an achievement reads the *unlock*, and this field is the bridge — two keys
	// rather than one, which is what lets an achievement be retired without orphaning a relic.
	//
	// **Every key here must be read by something** — a relic's `Unlock` today — and every relic's
	// `Unlock` must be granted here; `internal/session` refuses either half at load. Several
	// achievements may grant one key, and the first of them earned opens it.
	Unlocks []string `json:"Unlocks,omitempty"`

	Trigger TriggerData `json:"Trigger"`
}

// LoadAchievements parses the catalog.
//
// **A slice rather than a map, deliberately** — the same call LoadDuelistCards makes. File order is
// page order: the achievements screen has listed its rows in an authored order since it was written,
// because a page that reshuffled itself between visits is a page you have to re-read each time. A
// map would hand that decision to Go's hashing, and a sorted walk would hand it to the alphabet,
// which puts "Arsenal" above "First Steps".
//
// **Shape only.** Whether a trigger says anything the game can act on is internal/achieve's
// question, exactly as an essence's target is internal/session's and a card's verb is
// internal/combat's. This file may not import either.
func LoadAchievements() []AchievementData {
	return mustBeUniquelyKeyed(
		parse[AchievementData](achievementsJSON, "achievements.json"),
		"achievements.json",
		func(a AchievementData) string { return a.APIName },
	)
}

// AchievedIconKey is the asset key the achieved icon draws from: the record's own, or the default.
func (a AchievementData) AchievedIconKey() string {
	if a.AchievedIcon == "" {
		return AchievementIconPrefix + DefaultAchievementIcon
	}
	return AchievementIconPrefix + a.AchievedIcon
}
