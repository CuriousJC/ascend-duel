//go:build scenario

package scenario

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

//go:embed scenarios.json
var scenariosJSON []byte

// scenarioEnvVar names which scenario to plug in. **Unset takes the first in the file**, so the
// common case — edit the top entry, relaunch — needs no environment at all.
//
//	ASCEND_DUEL_SCENARIO=echo-flurry go run -tags scenario .
const scenarioEnvVar = "ASCEND_DUEL_SCENARIO"

// record is one scenario as the file writes it.
type record struct {
	// ScenarioRecord is the key the environment variable names.
	ScenarioRecord string `json:"ScenarioRecord"`

	// Note says what this scenario is *for*: the question it was built to answer. It is printed at
	// startup, because a fixture nobody can remember the purpose of is a fixture that gets deleted.
	Note string `json:"Note"`

	// Rings is what the run opens wearing, in worn order — and worn order matters, since rings
	// fire left to right and two multiplicative ones do not commute.
	Rings []string `json:"Rings"`

	// Hand is the opening hand, dealt over whatever the shuffle produced.
	Hand []handCard `json:"Hand"`

	// Enemy is a record key from enemies.json. **Empty means the climb's own**, so a scenario that
	// is only about the hand does not have to pick a fight.
	Enemy string `json:"Enemy"`

	// Screen is which scene to open on: `combat` (the default), `reward` or `shop`.
	//
	// **It exists because a between-fights screen is otherwise a twenty-minute question**
	// *(owner's call, 2026-08-22)*. Looking at the reward screen's narration or the shop's shelf
	// meant playing a duel to reach it, every time, and a screen under construction is looked at
	// dozens of times an afternoon. It is the same argument the plugged hand was built on, applied
	// one station further along the loop.
	//
	// **It sets the run's phase, not the scene directly.** The run owns where it is — see
	// session/flow.go — so a jump that named a scene could put a screen up that the run does not
	// think it is on, and leaving that screen would advance from the wrong station.
	Screen string `json:"Screen"`

	// Fight is which room the run has reached: 0 is floor 1's outer room, 2 its stairway. **It is
	// what makes a jumped-in reward screen pay the right room award**, and what the enemy is scaled
	// against.
	Fight int `json:"Fight"`

	// Vitae is the purse to arrive with, and Life the life the last fight is treated as having
	// ended on — a tenth of which is part of what the reward screen pays out. **Zero means the
	// run's own**: a fresh purse, and full life.
	Vitae int `json:"Vitae"`
	Life  int `json:"Life"`

	// Deck replaces the run's whole deck, rather than dealing over the shuffle the way Hand does.
	//
	// **Empty means the authored deck**, which is what almost every fixture wants: a scenario about
	// one interaction should not have to restate sixty cards.
	//
	// **It exists because a lesson has to be able to promise what the player is holding**
	// *(2026-08-25)*. `Hand` deals over the top of a normal shuffle, so the cards behind it are
	// still the ordinary deck — fine for looking at an interaction, useless for "these five all
	// match, play them all", which stops being true the moment the refill deals a sixth card
	// nobody mentioned.
	Deck []deckLine `json:"Deck"`

	// Seed pins the run seed, so the same fixture deals the same cards and meets the same
	// opponents every launch. **Empty means the clock**, which is the ordinary case.
	//
	// It is a **run code** — six Crockford base32 characters — the same spelling the game prints and a
	// player will one day type in, so a fixture and a bug report name a run the same way. A
	// string that is not a code fails the launch; see main.go.
	//
	// It is the per-scenario counterpart of `fixedRunSeed` in main.go and takes precedence over
	// it: a fixture that is *about* a particular deal cannot be at the mercy of whether somebody
	// left that constant at zero.
	Seed string `json:"Seed"`

	// Teach starts the tutorial on this run.
	//
	// **The real trigger is the profile now** — a player it has not recorded as taught is taught on
	// the first fight of a fresh run, see main.teachThisRun. **This field stayed rather than being
	// replaced by it**, because once the profile records a player as taught this is the only way to
	// see the lesson again: it forces the script whatever the profile says, and it is why the
	// tutorial can still be worked on after it has been finished once.
	Teach bool `json:"Teach"`
}

// handCard is one card of a plugged hand: a concept by its label, and a colour.
type handCard struct {
	Card    string `json:"Card"`
	Element string `json:"Element"`
}

// deckLine is one line of a replacement deck: a card, a colour, and how many copies.
//
// **Copies rather than repeating the line**, because a deck list is read to be counted and five
// identical lines is a list nobody checks. Absent or zero is one copy, so the common case writes
// nothing.
type deckLine struct {
	Card    string `json:"Card"`
	Element string `json:"Element"`
	Copies  int    `json:"Copies"`
}

var current = resolve()

// resolve reads the file once, at package init, and **fails the launch on anything it cannot
// resolve**. A misspelled ring or card in a fixture is a scenario that quietly tests something
// else, which is worse than a game that will not start: the whole point of this package is to
// look at a specific combination, and it must never be allowed to look at a different one.
//
// It runs at init rather than lazily so the failure lands before a window opens.
func resolve() *record {
	var list []record
	if err := json.Unmarshal(scenariosJSON, &list); err != nil {
		log.Fatalf("scenarios.json: %v", err)
	}
	if len(list) == 0 {
		log.Fatal("scenarios.json holds no scenarios")
	}

	want := os.Getenv(scenarioEnvVar)
	chosen := &list[0]
	if want != "" {
		chosen = nil
		for i := range list {
			if list[i].ScenarioRecord == want {
				chosen = &list[i]
				break
			}
		}
		if chosen == nil {
			log.Fatalf("%s=%s names no scenario in scenarios.json (have %s)",
				scenarioEnvVar, want, strings.Join(keysOf(list), ", "))
		}
	}

	if err := check(chosen); err != nil {
		log.Fatalf("scenario %s: %v", chosen.ScenarioRecord, err)
	}

	log.Printf("scenario %s: %s", chosen.ScenarioRecord, chosen.Note)
	log.Printf("scenario %s: wearing %v, hand of %d, enemy %q",
		chosen.ScenarioRecord, chosen.Rings, len(chosen.Hand), chosen.Enemy)
	return chosen
}

func keysOf(list []record) []string {
	out := make([]string, 0, len(list))
	for _, r := range list {
		out = append(out, r.ScenarioRecord)
	}
	return out
}

// check resolves every word the scenario uses, so a typo fails the launch rather than the
// interaction being tested.
//
// **A hand longer than the game's own is allowed on purpose.** This deals over the shuffle rather
// than through it, and a fixture wanting nine cards to show an interaction is a fixture, not a
// rules change — the action-point budget still refuses to play them all.
//
// **The rings are checked by the caller, not here.** A ring key is `internal/session`'s to resolve
// and this package sits below it — `main` hands the list to `session.StartingRings`, which already
// refuses a key the catalogue does not hold.
func check(r *record) error {
	// **A hand is only required of a scenario that opens on a duel.** One jumping straight to the
	// reward screen or the shop has nothing to deal it to.
	// **A replacement deck counts as having something to look at.** A fixture that hands over a
	// five-card deck has said exactly what the player will be holding; requiring it to restate the
	// same five as a Hand would be two lists to keep in step.
	//
	// **So does a pinned seed** *(2026-08-25)*. A seed *is* an opening hand, the shuffle being
	// deterministic, so a fixture naming one has said what the player will be holding just as
	// exactly as a Deck has — by reference rather than by list. The tutorial is what wanted it:
	// its lesson has to happen on the real deck and the real shuffle, so the fixture pins the run
	// that deals the hand rather than replacing the deck that cannot.
	if len(r.Hand) == 0 && len(r.Deck) == 0 && r.Seed == "" && r.Screen == screenCombat {
		return fmt.Errorf("has no hand and no deck, so there is nothing to look at")
	}
	for _, c := range r.Deck {
		if _, ok := combat.ConceptByKey(c.Card); !ok {
			return fmt.Errorf("deck: %q is not a card in the player's deck", c.Card)
		}
		if _, ok := combat.ParseElement(c.Element); !ok {
			return fmt.Errorf("deck: %q is not an element", c.Element)
		}
		if c.Copies < 0 {
			return fmt.Errorf("deck: %q has %d copies", c.Card, c.Copies)
		}
	}
	if r.Screen != "" && r.Screen != screenCombat && r.Screen != screenReward && r.Screen != screenShop {
		return fmt.Errorf("%q is not a screen (want %q, %q or %q)",
			r.Screen, screenCombat, screenReward, screenShop)
	}
	if r.Fight < 0 {
		return fmt.Errorf("fight %d is before the first room", r.Fight)
	}
	if r.Enemy != "" {
		_, enemy := data.LoadEnemies()[r.Enemy]
		_, boss := data.LoadBosses()[r.Enemy]
		if !enemy && !boss {
			return fmt.Errorf("%q is in no enemy or boss record", r.Enemy)
		}
	}
	for _, c := range r.Hand {
		if _, ok := combat.ConceptByKey(c.Card); !ok {
			return fmt.Errorf("%q is not a card in the player's deck", c.Card)
		}
		if _, ok := combat.ParseElement(c.Element); !ok {
			return fmt.Errorf("%q is not an element", c.Element)
		}
	}
	return nil
}

// Active reports whether a scenario is plugged in.
func Active() bool { return current != nil }

// Name is which scenario is running, for a log line.
func Name() string { return current.ScenarioRecord }

// Note is the authored sentence saying what the scenario is for.
func Note() string { return current.Note }

// Rings is what the run should open wearing, in worn order.
func Rings() []string { return current.Rings }

// Hand is the opening hand to deal, resolved into real cards.
func Hand() []combat.Card {
	out := make([]combat.Card, 0, len(current.Hand))
	for _, c := range current.Hand {
		id, _ := combat.ConceptByKey(c.Card)
		e, _ := combat.ParseElement(c.Element)
		out = append(out, combat.Of(id, e))
	}
	return out
}

// Enemy is the record key to fight instead of the climb's own, or empty for the climb's.
func Enemy() string { return current.Enemy }

// Teach reports whether this scenario starts the tutorial.
func Teach() bool { return current.Teach }

// Seed is the run code to pin, or empty for the clock.
func Seed() string { return current.Seed }

// Deck is the replacement deck, resolved into real cards, or nil for the authored one.
func Deck() []combat.Card {
	if len(current.Deck) == 0 {
		return nil
	}
	out := make([]combat.Card, 0, len(current.Deck))
	for _, line := range current.Deck {
		id, _ := combat.ConceptByKey(line.Card)
		e, _ := combat.ParseElement(line.Element)
		n := line.Copies
		if n == 0 {
			n = 1
		}
		for i := 0; i < n; i++ {
			out = append(out, combat.Of(id, e))
		}
	}
	return out
}

// The screens a scenario may open on. Written as keys rather than as `state.ActiveScreen` values,
// because this package sits below `internal/state` and must stay there.
const (
	screenCombat = "combat"
	screenReward = "reward"
	screenShop   = "shop"
)

// Screen is which scene to open on, defaulting to the duel.
func Screen() string {
	if current.Screen == "" {
		return screenCombat
	}
	return current.Screen
}

// Fight, Vitae and Life are the run state a jumped-in screen needs to have anything to show.
func Fight() int { return current.Fight }
func Vitae() int { return current.Vitae }
func Life() int  { return current.Life }
