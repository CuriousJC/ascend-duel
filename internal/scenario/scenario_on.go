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
}

// handCard is one card of a plugged hand: a concept by its label, and a colour.
type handCard struct {
	Card    string `json:"Card"`
	Element string `json:"Element"`
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
	if len(r.Hand) == 0 {
		return fmt.Errorf("has no hand, so there is nothing to look at")
	}
	if r.Enemy != "" {
		if _, ok := data.LoadEnemies()[r.Enemy]; !ok {
			return fmt.Errorf("%q is in no enemy record", r.Enemy)
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
