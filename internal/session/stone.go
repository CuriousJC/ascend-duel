package session

// Stones: the run's own opinion about what a hand is worth.
//
// **A worm alters a card; a stone alters a rung.** One stone raises one hand's multiplier by a
// tenth of the figure `data/hands.json` writes down, for the rest of the run — see
// `internal/combat/stone.go`, which owns the arithmetic and the seat a count sits in.
//
// **This file is where a record becomes something usable, and where a bad record is refused**,
// which is the same job `worm.go` does for the other catalogue. It lives here rather than in
// `internal/combat` because a stone is *held by a run*: the rules have no idea a run exists, and
// what they are handed is a fighter that already carries its counts.
//
// **A run holds counts, not stones.** Two Agates are not two objects to keep track of, they are
// `stones["concept-pair"] == 2` — which is what the ladder actually reads, and what a save file
// can hold without inventing an identity for a rock.

import (
	"fmt"
	"sort"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// Stone is one rung-raiser, resolved against the rules.
//
// Comparable, so a screen can hold one by value and compare two without reaching for the key —
// exactly as `Worm` is.
type Stone struct {
	Record string
	Name   string
	Text   string

	// Hand is the rung this stone raises, by catalogue key. **A key rather than a seat**, because
	// a seat is a position in the table this build loaded and a key is what a save file writes
	// down. `combat.HandSlot` is what turns one into the other, and it is asked once, here.
	Hand string
}

// stones is the validated catalogue, built once at package init.
//
// **A bad record panics at init**, exactly as a bad worm does: a stone naming a rung the rules have
// not got is a purchase that silently buys nothing, and the failure has to happen on launch rather
// than in the one shop that offered it.
var stones, stoneOrder = loadStones()

// Stones is every stone in the catalogue, in a fixed sorted order.
func Stones() []Stone {
	out := make([]Stone, 0, len(stoneOrder))
	for _, key := range stoneOrder {
		out = append(out, stones[key])
	}
	return out
}

// StoneByKey finds one by its record key.
func StoneByKey(key string) (Stone, bool) {
	s, ok := stones[key]
	return s, ok
}

// StoneForHand is the stone that raises one rung, by hand key. **One stone per rung and one rung
// per stone** — loadStones refuses a second — so this is a lookup rather than a choice.
func StoneForHand(hand string) (Stone, bool) {
	for _, key := range stoneOrder {
		if stones[key].Hand == hand {
			return stones[key], true
		}
	}
	return Stone{}, false
}

func loadStones() (map[string]Stone, []string) {
	recs := data.LoadStones()

	out := make(map[string]Stone, len(recs))
	byHand := map[string]string{}
	for _, key := range data.StoneOrder(recs) {
		s, err := resolveStone(recs[key])
		if err != nil {
			panic("stones.json: " + err.Error())
		}
		if prev, dup := byHand[s.Hand]; dup {
			panic(fmt.Sprintf("stones.json: %s and %s both raise %s", prev, s.Record, s.Hand))
		}
		byHand[s.Hand] = s.Record
		out[key] = s
	}

	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// **Every rung has a stone, and the check is here rather than left to a review** *(owner's
	// call, 2026-08-27)*. The mechanic is "a stone for every hand", so a rung with none is a rung
	// that can never be improved — invisible in play, because nothing fails and the bag simply
	// never offers it.
	for _, hand := range combat.HandKeys() {
		if _, ok := byHand[hand]; !ok {
			panic(fmt.Sprintf("stones.json: hand %q has no stone, so that rung can never be raised", hand))
		}
	}

	if len(keys) < bagSize {
		// The bag offers four, so a catalogue of three cannot fill it. Caught here rather than
		// producing a shelf with a gap in it.
		panic(fmt.Sprintf("stones.json: %d stones, and a bag of rocks needs %d", len(keys), bagSize))
	}
	return out, keys
}

// resolveStone turns a record into a stone, or says why it cannot.
func resolveStone(r data.StoneData) (Stone, error) {
	if r.StoneRecord == "" {
		return Stone{}, fmt.Errorf("a stone has no record key")
	}
	if r.Name == "" {
		return Stone{}, fmt.Errorf("%s has no name", r.StoneRecord)
	}
	if r.Text == "" {
		// The card is a name, a picture and a line of text. A stone with no text is a card that
		// does not say which rung it raises, which is the only thing distinguishing nineteen of
		// them.
		return Stone{}, fmt.Errorf("%s has no text, so its card says nothing", r.StoneRecord)
	}
	if _, ok := combat.HandSlot(r.Hand); !ok {
		return Stone{}, fmt.Errorf("%s raises hand %q, which the catalogue does not hold", r.StoneRecord, r.Hand)
	}
	return Stone{Record: r.StoneRecord, Name: r.Name, Text: r.Text, Hand: r.Hand}, nil
}

// StoneSalePrice is what one carried stone fetches when it is sold.
//
// **Five, which is what a whole bag of rocks costs** — the bag is 5 vitae for four stones of which
// one is kept, so a stone sold back at 5 pays for the next bag outright. That is deliberately
// generous rather than tuned: selling exists so a rung you will never build is worth something,
// and a price that made selling pointless would leave the pouch full of rocks nobody wants. It is
// one number in one place, and it is the obvious thing to move first if the pouch turns out to be
// a vitae fountain.
const StoneSalePrice = 5

// UseStone puts a stone on its rung, for the rest of the run, and reports whether the catalogue
// held it.
//
// **Using is no longer the same as owning** *(owner's call, 2026-09-02, reversing 2026-08-27)*. A
// stone used to be applied the moment it was chosen and there was no inventory at all. A run now
// **carries** stones — see Carry and the pouch — and this is what spending one does. The bag of
// rocks still applies its chosen stone on the spot, because that is a pick out of four rather than
// something handed over; what changed is that a stone can now also arrive as a thing to keep.
func (s *Session) UseStone(key string) bool {
	stone, ok := stones[key]
	if !ok {
		return false
	}
	if s.stones == nil {
		s.stones = map[string]int{}
	}
	s.stones[stone.Hand]++
	return true
}

// StonesOn is how many stones this run has put on one rung, by hand key.
func (s *Session) StonesOn(hand string) int { return s.stones[hand] }

// StoneCounts is every rung this run has raised, by hand key, as a copy.
//
// **A copy for the reason `Deck` hands one back**: the map is the run's, and a caller that wrote
// to it would be changing the ladder from outside the one method that is allowed to.
func (s *Session) StoneCounts() map[string]int {
	out := make(map[string]int, len(s.stones))
	for k, n := range s.stones {
		if n != 0 {
			out[k] = n
		}
	}
	return out
}

// HandMultiplier is what one rung pays this run: the catalogue's figure plus whatever stones are
// on it. It reports false for a rung the catalogue does not hold.
//
// **It is the run asking the rules rather than doing the arithmetic**, so the hands panel and the
// resolver read the same answer out of `combat.StoneValue`.
func (s *Session) HandMultiplier(hand string) (int, bool) {
	for _, h := range combat.Hands() {
		if h.Key != hand {
			continue
		}
		return h.Multiplier + combat.StoneValue(h.Multiplier, s.stones[hand]), true
	}
	return 0, false
}

// StoneWorth is what the *next* stone on a rung would be worth, in multiplier points. It is what
// a stone card writes on its face, and it does not depend on how many are already there — a tenth
// of the catalogue figure, every time.
func StoneWorth(hand string) int {
	for _, h := range combat.Hands() {
		if h.Key == hand {
			return combat.StoneValue(h.Multiplier, 1)
		}
	}
	return 0
}

// equipStones puts the run's counts onto a duelist. Called by Equip, which is the one place a run
// becomes a fighter.
func (s *Session) equipStones(d combat.Duelist) combat.Duelist {
	for _, hand := range sortedHands(s.stones) {
		for i := 0; i < s.stones[hand]; i++ {
			// **A rung the catalogue has dropped is skipped rather than fatal.** A resumed run is
			// refused outright for a name this build has not got — see save.go — so reaching here
			// with one would mean the catalogue changed under a live session, and losing a stone
			// is a better failure than losing the fight to a panic.
			d, _ = d.WithHandStone(hand)
		}
	}
	return d
}

// sortedHands is the map's keys in a fixed order. Go randomises map iteration and this decides a
// duelist's contents, so it is sorted for the reason `WormOrder` is — see the `randomness` skill.
func sortedHands(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// The pouch: stones the run is carrying, unspent.
//
// **A list of record keys rather than counts**, which is the opposite of `stones` and is right for
// the opposite reason. `stones` is what the ladder reads and two Agates there are genuinely one
// number; the pouch is a row of things to click, and two Agates in it are two cards to draw and two
// separate decisions to make. It is the parasite bucket's shape exactly, and for the same argument.

// Carry puts a stone in the pouch, and reports whether the catalogue held it.
//
// **A stone the catalogue does not have is refused** rather than carried as a key nothing can
// resolve — the posture `Hold` takes for a parasite, and for the same reason: a pouch slot that
// cannot be resolved is a slot the player cannot spend.
func (s *Session) Carry(key string) bool {
	if _, ok := stones[key]; !ok {
		return false
	}
	s.pouch = append(s.pouch, key)
	return true
}

// Carried is every stone the run is carrying, by record key, in the order they were acquired.
func (s *Session) Carried() []string {
	out := make([]string, len(s.pouch))
	copy(out, s.pouch)
	return out
}

// CarryCount is how many stones are in the pouch.
func (s *Session) CarryCount() int { return len(s.pouch) }

// SpendCarried takes one out of the pouch and puts it on its rung. It reports whether it was there.
//
// **Out of the pouch first, then onto the rung**, and by position rather than by key because the
// pouch may hold two of the same stone and spending one must not be ambiguous about which — the
// argument `parasiteToggle.armed` is under.
func (s *Session) SpendCarried(i int) bool {
	if i < 0 || i >= len(s.pouch) {
		return false
	}
	key := s.pouch[i]
	s.pouch = append(s.pouch[:i], s.pouch[i+1:]...)
	return s.UseStone(key)
}

// SellCarried takes one out of the pouch and pays for it. It reports whether it was there.
//
// **The rung is never touched.** A sold stone was never on the ladder, which is the whole point of
// carrying one: a rung you will never build is a thing to turn into vitae rather than a raise you
// are stuck with.
func (s *Session) SellCarried(i int) bool {
	if i < 0 || i >= len(s.pouch) {
		return false
	}
	s.pouch = append(s.pouch[:i], s.pouch[i+1:]...)
	s.AddVitae(StoneSalePrice)
	return true
}

// StartingStones is what a run opens carrying in its pouch, by record key.
//
// **Empty as shipped, and it is a debug seat**, the counterpart of StartingParasites and for the
// same reason: a stone reaches the pouch only from a rock-shower parasite, which is bought two
// screens away and spent in the fight after that. `internal/scenario` is what fills it; nothing
// else may.
var StartingStones []string
