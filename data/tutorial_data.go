package data

// The tutorial script: **what Bob says, what he points at, and what makes him move on.**
//
// **The steps are an ordered list rather than a map**, for the reason `duelist_cards.json` is one:
// a script is a sequence, file order is play order, and a map would put the run's first lesson
// wherever Go's hashing felt like putting it.
//
// **The file wraps that list in the run the lesson needs** *(2026-08-25)* — a seed and an opening
// opponent. Bob promises the player a hand of five matching cards and a fight ended in one blow,
// and both are facts about one particular deal against one particular creature, not about the game
// in general. They lived in `internal/scenario/scenarios.json` while a fixture was the only way to
// start the tutorial; the moment the profile became a real trigger, a lesson running on whatever
// the clock rolled promised five Jabs and dealt two. **A promise and the thing that makes it true
// belong in the same file**, which is the whole reason this stopped being a bare array.
//
// **The rules do not read this file and neither does `internal/combat`.** A tutorial step names a
// thing on a screen and a moment in a run; both are as far from a resolved round as a portrait
// key is. `internal/tutorial` is what consumes it — the same who-consumes-it test every file in
// this package answers — and it is also where the two closed vocabularies below are enforced.
//
// **How much of the screen a step locks is not a field.** It is derived from `Until` by
// `tutorial.lockFor`, because the rule — lock everything that is not the thing being asked for —
// has exactly three cases and the condition already says which one applies. It *was* a field for a
// few hours on 2026-08-25, and a step about which room you are standing in used it to leave the
// screen live while the player queued two cards nobody had mentioned.
//
// **Both vocabularies are closed and neither is defaulted.** An `Anchor` naming nothing and an
// `Until` naming nothing are refused at load rather than skipped, exactly as a hand's `match` is:
// a step that quietly pointed at nowhere would be a lesson the player is left staring at, and a
// step whose condition never fires is a tutorial with no way out.

import (
	_ "embed"
	"encoding/json"
)

//go:embed tutorial.json
var tutorialJSON []byte

// TutorialData is the whole script: the run it needs, and the steps.
type TutorialData struct {
	// Seed is the run code the lesson is written against — six Crockford base32 characters, resolved
	// by `internal/seeds`. **It is what makes the taught hand the hand Bob describes**, since the
	// shuffle is deterministic: `00H602` deals all five Jabs in the opening eight off the shipping
	// deck, and the other three cards are three different forms so the Jabs are the only set.
	//
	// **Nothing checks that it still does.** Change `duelist_cards.json`, `startingDeck` or
	// `handSize` and this silently deals something else — exactly as it does for `tools/seeds`,
	// which is the tool to re-run when any of those move.
	Seed string `json:"Seed"`

	// Enemy is the record the first room stands, by key. **The lesson promises a kill in one blow**,
	// which is a fact about the taught hand's damage against one creature's HP —
	// `TestTheTutorialsBlowKillsTheTutorialsEnemy` in `internal/combat` is what holds it.
	//
	// Empty means whoever the climb put there, which is a lesson that has stopped promising
	// anything about the fight.
	Enemy string `json:"Enemy"`

	// Match is which axis the `matching-cards` anchor counts a set on — `concept`, `form` or
	// `element`, resolved by `tutorial.ParseMatchAxis`. **Required by any script that points at a
	// matching set**, and refused when absent rather than defaulted: the lit square and the
	// condition that lets the player past it are the same set of cards, and which cards those are
	// depends entirely on this word.
	Match string `json:"Match"`

	// Steps is what Bob says, in order.
	Steps []TutorialStepData `json:"Steps"`
}

// TutorialStepData is one thing Bob says.
type TutorialStepData struct {
	// StepRecord is the key, kebab-case like a ring's or a worm's. Nothing stores it yet — the
	// cursor is an index, because a script is walked rather than looked up — but a step needs a
	// name to be talked about in a bug report, and the day a save file records how far a player
	// got, this is the thing it writes rather than the number.
	StepRecord string `json:"StepRecord"`

	// Text is what goes in the speech bubble. Prose rather than the clipped register the cards
	// use: the bubble is a paragraph's worth of room, and this is the one place in the game
	// allowed to explain something at length.
	Text string `json:"Text"`

	// Anchor is what the step points at — a closed vocabulary resolved by
	// `tutorial.ParseAnchor`, and turned into an actual rectangle by `internal/screens`.
	//
	// **Empty means the step points at nothing**, which is a real and common case: an opening
	// line and a closing one are about the run rather than about a control. The bubble then sits
	// in the middle of the screen instead of beside something.
	Anchor string `json:"Anchor,omitempty"`

	// Until is what advances the step: a closed vocabulary resolved by
	// `tutorial.ParseCondition`. `next` is the Next button and is the only one that asks the
	// player to acknowledge rather than to act.
	Until string `json:"Until"`
}

// LoadTutorial parses the script, steps in file order.
//
// **The steps are a slice and there is no `TutorialOrder`.** The sorted-key walk every
// map-returning loader here carries exists because Go randomises map iteration and an outcome must
// not depend on it; a slice is already ordered, and this one's order is the whole meaning of the
// file.
func LoadTutorial() TutorialData {
	var out TutorialData
	if err := json.Unmarshal(tutorialJSON, &out); err != nil {
		panic("Failed to unmarshal tutorial.json: " + err.Error())
	}
	return out
}
