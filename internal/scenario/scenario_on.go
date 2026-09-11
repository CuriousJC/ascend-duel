//go:build scenario

package scenario

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
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

// DummyLife and DummyRounds are what `Dummy` sets, and they are constants rather than fields
// because a training dummy is one thing rather than a dial.
//
// **The life is a big round number and not a very big one** *(2026-09-11)*. It is drawn on a health
// bar over a fraction, so a figure with eight digits in it is a fraction nothing on the card can
// read; six is enough that no hand in the game dents it. **The clock is 999 rather than off**,
// because `session.SetRoundLimit` refuses to stop the clock — see session/clock.go, where zero is
// a drawback reaching a number it must not reach — so the fixture goes the long way round rather
// than being given a back door into the rules.
const (
	DummyLife   = 999999
	DummyRounds = 999
)

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

	// Parasites is what the run opens with in its bucket, by record key.
	//
	// **The board piece is otherwise two shops away.** A parasite is bought from the shelf and
	// spent between the turns of the fight after it, so seeing the dialog at all meant playing to
	// a shop, buying the bucket, taking one of four, winning the room and opening it — which is
	// the twenty-minute question this package exists to answer. Keys are checked by the caller,
	// exactly as Rings are: a parasite key is internal/session's to resolve.
	Parasites []string `json:"Parasites"`

	// Stones is what the run opens carrying in its pouch, by record key. The caller resolves them,
	// on the terms Parasites is under.
	Stones []string `json:"Stones"`

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

	// Dummy makes the fight unkillable in both directions, so a scenario can be *played with*
	// rather than survived. The opponent gets DummyLife and so does the duelist, and the round
	// clock is lifted to DummyRounds.
	//
	// **It is not a creature in `data/enemies.json`, deliberately** *(owner's call, 2026-09-11)*.
	// A training dummy is a fixture, and `data/` is the game's own catalogue — loaded by every
	// build, drawn on the roster sheet, and reachable by the climb's own roll. So this changes the
	// *stats* of whichever opponent the fixture or the climb already named, which means the fight
	// still has a real portrait, a real deck and a real set of blows to watch. What it does not
	// have is an end.
	//
	// **Nothing else in the package changes an outcome this hard**, and it is allowed to for the
	// reason the whole package is: it is a fixture, compiled out, and the argument for a build tag
	// rather than a runtime flag. A dummy must never be reachable from a launched game.
	Dummy bool `json:"Dummy"`

	// Actions is the duelist's action-point budget, overriding the record's.
	//
	// **Zero is the record's own**, which is the six every duelist fights on. It exists because a
	// bench is a place to *pick the cards you want*, and a 6 AP budget means half the interesting
	// hands cannot be paid for — so looking at what a rung does turns into shopping for cheap
	// copies of it. **It does not lift `combat.MaxActions`**, the count bound, which is five cards
	// a turn whatever they cost: this buys the freedom to pick any five, not a sixth.
	Actions int `json:"Actions"`

	// RoundLimit is how many rounds a fight of this run gets, overriding combat.DefaultRoundLimit.
	//
	// **Zero means the run's own**, which is the five every fight in the tower is on. It is here
	// rather than only inside Dummy because the clock is a thing worth *looking at* on its own —
	// a one-round fight is how the clock's own death is reached without playing four rounds first
	// — and because `session.SetRoundLimit` refuses to stop the clock, so this cannot turn it off
	// either. See combat.DefaultRoundLimit and session/clock.go.
	RoundLimit int `json:"RoundLimit"`

	// RingSlots is how many rings the run may wear at once, overriding combat.DefaultRingSlots.
	//
	// **Zero means the run's own**, which is the five every climb is on. It exists because the
	// whole catalogue is looked at five at a time otherwise, and a batch of new art or a new
	// interaction is a batch: a sixth ring meant playing to a shop, selling one and buying
	// another, per ring, forever.
	//
	// **It is bounded by `combat.MaxWornRings`**, the width of the duelist's ring array — not by
	// the design cap, which is the thing this overrides. `session.SetRingSlots` clamps to the same
	// figure, so a fixture asking for more gets the width rather than a disagreement between the
	// shop and the fighter.
	RingSlots int `json:"RingSlots"`

	// Teach starts the tutorial on this run.
	//
	// **The real trigger is the profile now** — a player it has not recorded as taught is taught on
	// the first fight of a fresh run, see main.teachThisRun. **This field stayed rather than being
	// replaced by it**, because once the profile records a player as taught this is the only way to
	// see the lesson again: it forces the script whatever the profile says, and it is why the
	// tutorial can still be worked on after it has been finished once.
	Teach bool `json:"Teach"`
}

// handCard is one card of a plugged hand: a concept by its label, a colour, and any riders it
// has already been given.
type handCard struct {
	Card    string   `json:"Card"`
	Element string   `json:"Element"`
	Riders  []string `json:"Riders"`
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

	// Riders are the rules these cards arrive already carrying, by the names combat.RiderKind
	// writes — so a fixture can open on a deck a parasite has *already been spent on* rather than
	// on one a parasite has to be spent on first.
	//
	// **It exists because a card alteration is otherwise several shops and a duel away**
	// *(2026-09-07)*. `Parasites` puts the consumable in the bucket, which is the right fixture
	// for looking at the *dialog*; it is the wrong one for looking at what an altered card does to
	// a hand, because getting there means playing a turn to spend it and then reading a hand that
	// has already been half spent. This is the same argument `Deck` made against `Hand`, one
	// level further in.
	//
	// **A rider that carries a figure writes it after a colon** — `"damage-on-play:10"` — and one
	// that does not writes the bare name. The vocabulary and which of them take a figure are
	// `internal/combat`'s, so a misspelling fails the launch like every other word here.
	Riders []string `json:"Riders"`
}

// riders resolves the names on one line into real riders, and reports the first it cannot.
//
// **A shared helper rather than two copies**, because a deck line and a hand card want exactly
// the same thing and the check and the build have to agree about what a name means — a fixture
// whose validation and whose construction read a string differently is a fixture that passes its
// own check and tests something else.
func riders(names []string) ([]combat.Rider, error) {
	out := make([]combat.Rider, 0, len(names))
	for _, name := range names {
		kind, amount := name, ""
		if i := strings.IndexByte(name, ':'); i >= 0 {
			kind, amount = name[:i], name[i+1:]
		}
		k, ok := combat.ParseRiderKind(kind)
		if !ok {
			return nil, fmt.Errorf("%q is not a rider the rules have", kind)
		}
		if !k.CarriesAmount() {
			if amount != "" {
				return nil, fmt.Errorf("the %s rider carries no figure, and %q gives it one",
					k, name)
			}
			out = append(out, combat.Rider{Kind: k})
			continue
		}
		n, err := strconv.Atoi(amount)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("the %s rider wants a figure and %q is not one", k, name)
		}
		out = append(out, combat.Rider{Kind: k, Amount: n})
	}
	if len(out) > combat.MaxCardRiders {
		return nil, fmt.Errorf("%d riders on one card, and a card holds %d",
			len(out), combat.MaxCardRiders)
	}
	return out, nil
}

// ridden is a card with its upgrade on it, or the card unchanged if the fixture named none.
//
// **A card holds one, so this list is at most one long** — check() has already refused a fixture
// naming more, which is what keeps the loop honest rather than silently keeping the last.
func ridden(c combat.Card, names []string) combat.Card {
	list, err := riders(names)
	if err != nil {
		return c // unreachable: check() has already refused anything riders() would reject
	}
	for _, r := range list {
		c = c.SetRider(r)
	}
	return c
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
	if chosen.Dummy {
		log.Printf("scenario %s: DUMMY — %d life each way and a %d-round clock, so nothing ends",
			chosen.ScenarioRecord, DummyLife, DummyRounds)
	}
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
	// **And so does Teach** *(2026-09-06)*. The tutorial's seed moved into `data/tutorial.json`,
	// where the lesson's promises live, so the entry that starts it no longer pins one here — it
	// says only "teach it". That is the same guarantee at one further remove: the script names the
	// run, and the run deals the hand.
	if len(r.Hand) == 0 && len(r.Deck) == 0 && r.Seed == "" && !r.Teach && r.Screen == screenCombat {
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
		if _, err := riders(c.Riders); err != nil {
			return fmt.Errorf("deck: %q: %v", c.Card, err)
		}
	}
	if r.Screen != "" && r.Screen != screenCombat && r.Screen != screenReward && r.Screen != screenShop {
		return fmt.Errorf("%q is not a screen (want %q, %q or %q)",
			r.Screen, screenCombat, screenReward, screenShop)
	}
	if r.Actions < 0 {
		return fmt.Errorf("an action budget of %d is not a budget", r.Actions)
	}
	if r.RoundLimit < 0 {
		return fmt.Errorf("round limit %d is not a number of rounds", r.RoundLimit)
	}
	if r.RingSlots < 0 {
		return fmt.Errorf("%d ring slots is not a number of fingers", r.RingSlots)
	}
	if r.RingSlots > combat.MaxWornRings {
		return fmt.Errorf("%d ring slots, and a duelist's hand is %d wide — raise "+
			"combat.MaxWornRings if a fixture genuinely needs more", r.RingSlots, combat.MaxWornRings)
	}
	if n := len(r.Rings); n > 0 && n > r.effectiveRingSlots() {
		return fmt.Errorf("wears %d rings on %d fingers, so %d of them would never go on",
			n, r.effectiveRingSlots(), n-r.effectiveRingSlots())
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
		if _, err := riders(c.Riders); err != nil {
			return fmt.Errorf("hand: %q: %v", c.Card, err)
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

// Parasites is what the run opens holding in its bucket, by record key. The caller resolves them,
// for the reason it resolves the rings.
func Parasites() []string { return current.Parasites }

// Stones is what the run opens carrying in its pouch, by record key.
func Stones() []string { return current.Stones }

// Hand is the opening hand to deal, resolved into real cards.
func Hand() []combat.Card {
	out := make([]combat.Card, 0, len(current.Hand))
	for _, c := range current.Hand {
		id, _ := combat.ConceptByKey(c.Card)
		e, _ := combat.ParseElement(c.Element)
		out = append(out, ridden(combat.Of(id, e), c.Riders))
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
			out = append(out, ridden(combat.Of(id, e), line.Riders))
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

// Dummy reports whether this scenario's fight is unkillable in both directions.
func Dummy() bool { return current.Dummy }

// Actions is the action-point budget to fight on, or zero for the record's own.
func Actions() int { return current.Actions }

// effectiveRingSlots is the cap this record will actually fight on, so check() and RingSlots()
// cannot come to different conclusions about whether a list of rings fits.
func (r *record) effectiveRingSlots() int {
	if r.RingSlots > 0 {
		return r.RingSlots
	}
	return combat.DefaultRingSlots
}

// RingSlots is how many fingers this scenario wants, or zero for the run's own.
func RingSlots() int { return current.RingSlots }

// RoundLimit is the clock this scenario wants, or zero for the run's own.
//
// **A dummy implies one**, because a fight nobody can win is a fight the five-round clock kills
// the player at the end of — see combat.FightOver. An authored figure still wins, so a fixture
// can have a dummy *and* a short clock if what it is looking at is the clock.
func RoundLimit() int {
	if current.RoundLimit > 0 {
		return current.RoundLimit
	}
	if current.Dummy {
		return DummyRounds
	}
	return 0
}
