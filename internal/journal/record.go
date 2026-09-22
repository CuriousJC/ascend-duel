package journal

// **What the player chose, as flat named data — the thing the journal actually stores.**
//
// A record is one click, written as fields rather than as a sentence. Nothing here is worded and
// nothing here is a color: the audience is a person retracing a run with the game open, and a
// machine that one day replays it.

// The record kinds. **One word each, append-only, and a name rather than an ordinal**, on the rule
// every file this game writes is under — and this one outlives its build harder than a save does,
// because a journal is read by whatever is on the other end of a bug report.
//
// **All of them are choices.** A kind naming something the engine decided belongs in the ledger;
// see session/record.go, and the file comment above for why the two may not be mixed.
const (
	// KindHeader opens the file: the schema, the build and the run. **Written by Begin and Resume
	// and by nothing else**, so a reader can take the first line as the thing it describes.
	KindHeader = "header"

	// Where the player is. **Both are diffed once a frame rather than announced**, because the
	// thing being watched is true of the whole game and a watcher ticked by one scene is a watcher
	// the next scene forgets. See internal/game.
	KindRun    = "run"
	KindScreen = "screen"
	KindPhase  = "phase"

	// A turn being built. KindSelect and KindUnselect are the same click, and the pair is kept
	// rather than one `select` carrying a flag: a line that has to be read twice to know which way
	// it went is a line nobody can scan.
	KindSelect   = "select"
	KindUnselect = "unselect"
	KindDiscard  = "discard"
	KindDuel     = "duel"

	// A consumable spent. **A verb each**, because what the journal is for is saying what the
	// player reached for, and the verb is the thing being scanned for.
	KindRune    = "rune"
	KindEssence = "essence"
	KindStone   = "stone"
	KindPotion  = "potion"

	// The shop. KindGood is a sealed good paid for and KindTake is the one thing taken out of it,
	// which is two choices and two lines: the good is bought before its contents are seen.
	KindBuy  = "buy"
	KindSell = "sell"
	KindGood = "good"
	KindTake = "take"
)

// The actions a KindRun record can carry. **A word rather than a kind each**, because what a reader
// wants off the run line is the code, and these are the same fact about it at different moments.
//
// **A run that ended carries the ending's own word** — session.EndedInDefeat or EndedByChoice —
// rather than a fourth constant here plus a reason beside it. The four words are already distinct
// and one field saying which is one field to read; a run *ended* is never the thing worth knowing
// on its own, since there is no retry and how it ended is the whole of the news.
const (
	RunStarted = "started"
	RunResumed = "resumed"
)

// What a KindStone or KindSell choice did with the thing it names. A stone in the pouch can be put
// on its rung or sold back, and the line has to say which.
const (
	StoneUsed = "used"
	StoneSold = "sold"
)

// Record is one choice.
//
// **One struct rather than one per kind**, on session.LedgerRecord's argument: a reader switches on
// the kind, and a Go interface here would be a type assertion at every call site for no gain.
// **Every field but the first two is `omitempty`**, so what reaches the file is the handful of
// fields the kind actually uses — which is the narrow record the format promises, whatever the
// struct looks like in Go.
type Record struct {
	// T is the simulation tick, from At. **Never a wall clock.**
	T int `json:"t"`

	// Kind is one of the names above.
	Kind string `json:"kind"`

	// Key is what a catalog calls the thing chosen — a relic, a rune, a stone, an essence, a
	// potion, a sealed good. **A key rather than a name**, because a name is wording and a key is
	// what data/ is indexed by.
	Key string `json:"key,omitempty"`

	// Card is the identity of the card the choice was about, and Label what it reads as.
	//
	// **An identity rather than a deck position**, for the reason combat.Card.ID exists at all:
	// three piles hold copies of the same cards, so a position names a different card a moment
	// later. The label rides along because the id is meaningless to a person and the point of this
	// file is that a person can read it.
	Card  int    `json:"card,omitempty"`
	Label string `json:"label,omitempty"`

	// Seat is where in the row it was standing, which is what a person retracing this actually
	// clicks.
	//
	// **An absent seat is the leftmost one**, not an unknown one. Every kind that sets this always
	// has a seat, so `omitempty` swallowing a zero costs a reader nothing — which is the price of
	// keeping the envelope flat rather than giving one field a pointer nothing else here needs.
	Seat int `json:"seat,omitempty"`

	// Targets is the cards a consumable was aimed at, by identity. **The one array in the
	// envelope**, and it is here because a rune may name two cards and an essence a hand of them.
	Targets []int `json:"targets,omitempty"`

	// Amount is the figure the choice moved: a price paid, a purse left, the action points a turn
	// committed.
	Amount int `json:"amount,omitempty"`

	// Action is which way a two-way choice went — see RunStarted and StoneUsed above.
	Action string `json:"action,omitempty"`

	// Screen is the scene being drawn and Phase the station the run stands at, both as words
	// rather than as the append-only ordinals they are held as. Fight is which room of the climb.
	Screen string `json:"screen,omitempty"`
	Phase  string `json:"phase,omitempty"`
	Fight  int    `json:"fight,omitempty"`
}

// Header is the first line of the file: enough to know which build wrote it and which run it is
// about, before a single choice is read.
//
// **It carries no path, no machine name and no user name**, the rule internal/crashlog is under
// and for the same reason: a copy of this file travels beside a crash report, so a field added here
// on the assumption that it stays local is a field that leaves the machine the day a send button
// lands.
type Header struct {
	T    int    `json:"t"`
	Kind string `json:"kind"`

	// Schema is the journal format this build writes. **It earns its place before there is
	// anything to migrate**, on the profile's argument: the one version number that cannot be
	// added later is the one a reader already has files without.
	Schema int `json:"schema"`

	Version  string `json:"version"`
	RunCode  string `json:"runCode"`
	Install  string `json:"installId,omitempty"`
	Platform string `json:"platform"`
	UTC      string `json:"utc"`

	// Resumed says this header opened a file that already had a run in it, which is the one thing
	// a reader cannot work out from the records underneath: two headers in one file is a climb
	// played across two launches, not two runs.
	Resumed bool `json:"resumed,omitempty"`
}
