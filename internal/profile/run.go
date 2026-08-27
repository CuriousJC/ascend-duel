package profile

// **The run in progress, written down.**
//
// A run used to be explicitly un-persistable: `session.Session` said so, on the grounds that two
// runs from one seed can hold different decks — a deck edit is a *choice*, not something the seed
// derives — so replaying a run means a seed plus a choice log rather than a snapshot. That reasoning
// is still correct and it is an argument about *replay*, not about *resume*. Resuming needs the
// state the player is in, not the path they took to it, and a snapshot is exactly that. See
// MECHANICS.md, where the reversal is recorded.
//
// **The climb is not in here, and that is deliberate.** `session.Start` builds the pyramid from
// `seeds.For(runSeed, EnemySelect)`, so the run code rebuilds the same opponents in the same order.
// Storing the seed rather than the order keeps the file small and, more importantly, keeps one
// answer to "who is in room four". **The day the room-choice screen lets the player pick what is
// ahead, the climb stops being derivable and has to be written down here** — session's round-trip
// test is what should fail on that day rather than a player quietly resuming against different
// enemies.
//
// **Everything is a name or a count.** A concept is its registry key, an element is its name, a
// phase is its name, a ring is its record key. No ordinals — see doc.go.

// runFile is the run snapshot's name inside the store's directory.
const runFile = "run.json"

// RunSnapshot is a run, frozen at a phase transition.
//
// **Saved between stations of the loop and never inside a duel** *(owner's call, 2026-08-25)*. A
// half-resolved round would mean serialising both piles, the queued actions, the opponent's hidden
// hand and the playback position — and the combat screen is the part of the game still being built,
// so its shape moves week to week. Quitting mid-duel therefore loses that duel and puts the player
// back at the start of the room, which is what `Retry` already does after a defeat.
type RunSnapshot struct {
	// Version is the snapshot format, on the same terms as the profile's.
	Version int `json:"version"`

	// Seed is the run code — six Crockford base32 characters, the spelling a player reads off the
	// screen. **A code rather than the int64** because it is the one field of this file a person
	// might type, and because `seeds.Parse` refuses a malformed one where a raw number would
	// silently resume a different tower.
	Seed string `json:"seed"`

	// Fight is how many rooms in the run has got, zero-based, and Floor is not stored: it is
	// arithmetic on this, and a second field saying the same thing is a second field to keep in
	// step.
	Fight int `json:"fight"`

	// Phase is the station of the loop, by name — "fight", "reward", "shop", "choice".
	Phase string `json:"phase"`

	// Vitae is the purse, and LifeLeft is what the fighter walked out of the last fight with. The
	// reward screen draws the duelist card from LifeLeft, so a run resumed at the reward station
	// needs it.
	Vitae    int `json:"vitae"`
	LifeLeft int `json:"lifeLeft"`

	// Worn is the rings, by record key, **in worn order** — which is a rule and not a presentation
	// detail, since rings fire left to right and compound. A list rather than a set for that
	// reason.
	Worn []string `json:"worn"`

	// Grown is each growing ring's accumulator, by record key. Keyed by record rather than by
	// position, which is the reason `Session.grown` was already keyed that way: a position means
	// nothing in a file.
	Grown map[string]int `json:"grown"`

	// Stones is how many stones the run has put on each rung of the hand ladder, by **hand key**.
	// A name rather than a seat, on the same terms `Worn` names a ring record: a seat is a position
	// in the catalogue this build loaded, and a file outlives the build that wrote it.
	Stones map[string]int `json:"stones"`

	// Held is the bucket of parasites the run is carrying, by record key, **in acquisition
	// order** — which is the order the board piece draws them in, and so the only order the
	// player can see. A list rather than a count per key, because two of the same parasite are
	// two things to spend.
	Held []string `json:"held,omitempty"`

	// Spoils is what the last win still owes, unclaimed. A run saved at the reward station has a
	// payout part-narrated, and dropping it would pay the player less for quitting.
	Spoils SpoilsSnapshot `json:"spoils"`

	// NextCardID is the identity counter. **It is saved rather than recomputed from the deck's
	// highest id**, because the counter only ever goes up: a card removed by a worm must not have
	// its number handed out again to a card added later, and the highest surviving id has forgotten
	// that the removed one existed.
	NextCardID int `json:"nextCardID"`

	// Deck is every card the run owns, in order.
	Deck []CardSnapshot `json:"deck"`
}

// SpoilsSnapshot is one win's unclaimed payout, split the way the reward screen reads it out.
type SpoilsSnapshot struct {
	Propagated int `json:"propagated"`
	FromLife   int `json:"fromLife"`
	FromRoom   int `json:"fromRoom"`
}

// CardSnapshot is one owned card.
//
// **Its identity is stored.** Two cards that look identical are still two cards, and the number is
// what lets a drawn card be asked what it looked like before a ring touched it — see combat.Card.ID.
// A resumed run whose ids were reassigned would be a run where that question got a different answer.
type CardSnapshot struct {
	ID int `json:"id"`

	// Concept is the registry key — "strike", "jab" — never the ConceptID, which is a position in
	// a registry built at init and would re-point the moment a card was added to the deck.
	Concept string `json:"concept"`

	// Element is the element's name.
	Element string `json:"element"`

	// CostDelta and AmountPct are the per-card modifiers a worm writes, with zero meaning
	// unmodified. Omitted when empty, so an unaltered deck reads as a plain list.
	CostDelta int `json:"costDelta,omitempty"`
	AmountPct int `json:"amountPct,omitempty"`

	// Riders are the lasting rules a parasite has attached to this card, in the order they were
	// attached — which is the order they fire. Omitted when empty, so an unridden deck reads as a
	// plain list.
	Riders []RiderSnapshot `json:"riders,omitempty"`
}

// LoadRun reads the run in progress, and reports whether there was one.
//
// **A corrupt or future-versioned run is no run at all**, which is a harder line than the profile
// takes and the right one: a profile half-read still names achievements worth keeping, where a run
// half-read is a tower the player would resume into wrong. Losing it costs one run, and the profile
// beside it is untouched.
func LoadRun(s Store) (*RunSnapshot, bool, error) {
	var r RunSnapshot
	found, err := s.read(runFile, &r)
	if err != nil || !found {
		return nil, false, err
	}
	if r.Version > Version {
		return nil, false, nil
	}
	return &r, true, nil
}

// SaveRun writes the run in progress.
func SaveRun(s Store, r *RunSnapshot) error {
	r.Version = Version
	return s.write(runFile, r)
}

// DeleteRun throws the run away, and is content for there not to be one.
//
// **Nothing calls it yet, and that is a statement about the game rather than an oversight.** A
// defeat currently offers `Retry` and puts the same opponent back up — no run ever *ends*, so there
// is no moment at which a save should be dropped. When a death exists this is what it calls; until
// then a resumed loss lands at the start of the room it was lost in, which is what Retry does.
func DeleteRun(s Store) error { return s.remove(runFile) }

// RiderSnapshot is one rule attached to one card.
//
// **The kind is its name, never combat.RiderKind's number.** The enum is append-only and indexes
// nothing today, but a file outlives the build that wrote it — the same rule that keeps a concept
// key, an element name and a phase name in here rather than their ordinals.
type RiderSnapshot struct {
	Kind   string `json:"kind"`
	Amount int    `json:"amount"`
}
