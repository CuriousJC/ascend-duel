package session

// **The ledger: the whole run's account of itself, in already-worded lines.**
//
// The fight log used to be `CombatScene.rounds` — this fight's events, thrown away by the next
// `Init` — so the account of a run was something the player had to have been watching. The ledger
// is the same record kept for the length of the run and read back from anywhere: what a blow came
// to, which ring priced which term of it, and, at run scale, how a climb went and where it went
// wrong. *(owner's call, 2026-09-02)*
//
// **Lines, not events, and that is the whole design decision.** `combat.Event` is a fat comparable
// struct — `HandGrown`, `HandRingScale` and `HandLanding` are 25x5 arrays each, about 2.5 KB an
// event, nearly all of it zero on everything that is not a `KindHand` — so keeping a run of them
// would be several megabytes held for a session to say what a few hundred kilobytes of sentences
// say. The events are still the source: `internal/screens` words them the instant a round ends,
// through the same walk the log always used, and hands the result here. **The purpose is reading
// back, not replaying** — a ledger cannot re-run a round and is not meant to.
//
// **Nothing here computes anything and nothing here knows what a card is.** A line arrives as
// strings; this package stores them and hands them back. That is what keeps the run's record
// structurally unable to disagree with the round it reports — see `internal/screens/prose.go`,
// which is the one place the words are decided.
//
// **Every field is a name or a number, never an ordinal**, because the ledger is saved: `Voice`
// and `Category` are short closed vocabularies written as words, on the rule every other snapshot
// is under. See save.go.

// The voices a line can be spoken in. **A name rather than a colour**, because a colour in a save
// file is a palette decision frozen into a run — the screen maps these onto its swatches, so a
// re-coloured pane re-colours every ledger already written.
const (
	// VoiceYou and VoiceFoe are one duelist doing something. They carry that side's swatch.
	VoiceYou = "you"
	VoiceFoe = "foe"

	// VoiceHand is the blow: the attack phase's one line, what the hand formed and what it came to.
	VoiceHand = "hand"

	// VoiceTerm is one term of that blow's arithmetic — a card's own figure, what a ring did to it,
	// the sum underneath. **It is what the log never had**: the total was printed and none of the
	// working was, so a multiplier read as a number the game had decided rather than one the player
	// had built.
	VoiceTerm = "term"

	// VoicePlain is a line belonging to nobody: a heading, or an empty ledger saying so.
	VoicePlain = "plain"
)

// The inks a run of text can be written in. **A name rather than a colour**, because a colour in a
// save file is a palette decision frozen into a run — the screen maps these onto what it paints
// with today, so a re-coloured panel re-colours every ledger already written.
//
// The five **element names are inks too**, spelled as `combat.Element.String()` spells them, so the
// figures in a blow's working carry their own cards' colours exactly as the hand dialog's do.
const (
	// InkAttack and InkDefend are an action line's marked verb, which is how a round can be scanned
	// for what *kind* of thing happened before any of it is read.
	InkAttack = "attack"
	InkDefend = "defend"

	// InkHand is the hand: its name, and the multiplier in the sum that came off it.
	//
	// **It resolves to no colour at all** *(owner's call, 2026-09-02)*. Hue belongs to the elements
	// — five of them, plus pink for a ring and red and blue for the two verbs — and there is none
	// left that is not a near-collision. A hand is marked by weight and by the amber swatch on its
	// row instead, which is why these runs carry Mark. See screens.inkNamed.
	InkHand = "hand"

	// InkRing is what a worn ring did — the figure it priced a term at, or the landing it bought.
	InkRing = "ring"

	// InkTotal is what the blow came to. The damage colour rather than the hand's, because that is
	// the figure that leaves the sum and lands in a life bar.
	InkTotal = "total"
)

// The outcomes a fight record can end on. An empty outcome is a fight still being fought — the one
// record the screen draws expanded.
const (
	OutcomeWon  = "won"
	OutcomeLost = "lost"
)

// LedgerRun is one run of text inside a line, with the colour it is written in.
//
// **A line is runs rather than one string**, because the ledger is trying to look like the screen
// it is an account of: the verb coloured by category, a figure by its card's element, a ring's
// multiplier in the ring pink. One ink a line could say none of that.
type LedgerRun struct {
	Text string

	// Ink is one of the names above, an element's name, or empty for the panel's own ink.
	Ink string

	// Mark draws the run bold and underlined. **It marks what happened** — an action's verb, and a
	// hand's name and multiplier, which have no hue of their own. One mark would be ambiguous on a
	// panel that also uses colour for the side; two together are not, and no line carries both a
	// verb and a hand.
	Mark bool
}

// LedgerLine is one line of the account, already worded and already coloured.
//
// **Worded once, when it happened**, so a line read back three fights later is the line that was on
// screen while it was happening. See internal/screens/prose.go, the one place the words are chosen.
type LedgerLine struct {
	Runs []LedgerRun

	// Voice is who is speaking, from the list above. An unrecognised voice draws plain rather than
	// failing anything — see resumeLedger in save.go for why a line may never refuse a run.
	Voice string
}

// Text is the whole line as one string, for a caller reading it rather than drawing it: a test, or
// the scripted demo's report.
func (l LedgerLine) Text() string {
	var out string
	for _, r := range l.Runs {
		out += r.Text
	}
	return out
}

// Line is a whole line in one voice and one ink, which is what a heading and most sentences are.
func Line(voice, text string) LedgerLine {
	return LedgerLine{Voice: voice, Runs: []LedgerRun{{Text: text}}}
}

// LedgerRound is one round of one fight.
type LedgerRound struct {
	// Number is the round's place in its fight, 1-based, which is what the heading prints.
	Number int
	Lines  []LedgerLine
}

// LedgerFight is one duel, from the first round to the outcome.
//
// **A retry is a new record.** A defeat that is fought again is a different duel with a different
// shuffle, and folding the two would make the run's account claim a fight was won that was lost
// first.
type LedgerFight struct {
	// Number is which duel of the run this was, 1-based, counting retries. Floor is where it was
	// fought and Enemy is who stood there.
	Number int
	Floor  int
	Enemy  string

	// Outcome is "won", "lost", or empty while it is still being fought.
	Outcome string

	Rounds []LedgerRound

	// dealt is what the player's blows came to across the fight. Unexported and read through
	// Dealt(), so nothing outside this package can add to a total the rounds do not support.
	dealt int
}

// Dealt is what the player's blows came to across the whole fight, which is the figure a collapsed
// summary line prints. **It is accumulated as rounds arrive rather than parsed back out of the
// lines**, because the words are presentation and a summary reading them would be a second
// arithmetic over a first one's prose.
func (f LedgerFight) Dealt() int { return f.dealt }

// RoundCount is how many rounds the fight ran to.
func (f LedgerFight) RoundCount() int { return len(f.Rounds) }

// Ledger is every fight of the run, oldest first.
type Ledger struct {
	Fights []LedgerFight
}

// LedgerFights hands back the run's account, oldest first.
//
// **A copy of the slice header, not of the rounds.** A caller reading it every frame to draw a
// panel must not be copying a run's worth of lines sixty times a second, and nothing outside this
// package appends to it.
func (s *Session) LedgerFights() []LedgerFight { return s.ledger.Fights }

// BeginFight opens a record for the duel about to be fought.
//
// **An unfought record is replaced rather than added to.** `CombatScene.Init` runs on entering the
// screen and again on a retry, and a player who reached the room and left it without throwing a
// round should not leave an empty heading in the run's account.
func (s *Session) BeginFight(floor int, enemy string) {
	if n := len(s.ledger.Fights); n > 0 && len(s.ledger.Fights[n-1].Rounds) == 0 {
		s.ledger.Fights = s.ledger.Fights[:n-1]
	}
	s.ledger.Fights = append(s.ledger.Fights, LedgerFight{
		Number: len(s.ledger.Fights) + 1,
		Floor:  floor,
		Enemy:  enemy,
	})
}

// RecordRound adds a finished round to the fight in progress, with what the player's blows in it
// came to.
//
// **A round is recorded when it finishes, not while it plays.** The screen is already drawing the
// round in progress — the table, the cards, the sum acted out — so a ledger racing playback would
// be the one place in the game that says what is about to happen. It also keeps this out of the
// beat: one conversion per round rather than per frame.
//
// A round arriving with no fight open is dropped rather than opening one, because a record with no
// enemy and no floor would be a heading the panel could not write.
func (s *Session) RecordRound(lines []LedgerLine, dealt int) {
	n := len(s.ledger.Fights)
	if n == 0 || len(lines) == 0 {
		return
	}
	f := &s.ledger.Fights[n-1]
	f.Rounds = append(f.Rounds, LedgerRound{Number: len(f.Rounds) + 1, Lines: lines})
	f.dealt += dealt
}

// EndFight closes the record with what became of the duel.
//
// **A record with no rounds is dropped**, on BeginFight's terms: a fight left before a round was
// thrown is not a fight the run had.
func (s *Session) EndFight(outcome string) {
	n := len(s.ledger.Fights)
	if n == 0 {
		return
	}
	if len(s.ledger.Fights[n-1].Rounds) == 0 {
		s.ledger.Fights = s.ledger.Fights[:n-1]
		return
	}
	s.ledger.Fights[n-1].Outcome = outcome
}

// LedgerOpenFight reports whether a fight is still being fought, and which record it is. The panel
// asks, because that is the one record it draws expanded.
func (s *Session) LedgerOpenFight() (int, bool) {
	n := len(s.ledger.Fights)
	if n == 0 || s.ledger.Fights[n-1].Outcome != "" {
		return 0, false
	}
	return s.ledger.Fights[n-1].Number, true
}
