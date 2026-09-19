package session

// Runes: the alterations a run can make to its own deck **while a fight is going on.**
//
// An essence is won between rooms and spent on the spot. A rune is bought, carried in a sack,
// and spent in the gap between one turn and the next — so the deck a duel started with is not
// necessarily the deck it ends with. The catalog is `data/runes.json`; this file is where a
// record becomes something applicable, and where a bad record is refused.
//
// **It lives here rather than in `internal/combat` for the reason essences do**: a rune acts on
// the *run's* deck, and the rules have no deck. The one exception is a rider, whose vocabulary is
// a Go enum in `internal/combat` because the rules have to read it while a round resolves — this
// file resolves the name and hands over the value.
//
// **Targets are card *identities*, not positions.** That is the whole difference from essence.go,
// which takes an index and says in its own comment that a caller may not hold two across a call.
// A rune may eat two cards, and it is spent while a hand, a draw pile and a discard pile are
// all live and holding copies of the same cards — so a position would be meaningless by the time
// the player confirmed. `combat.Card.ID` is what makes this possible and it is what it was added
// for.

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// RuneTarget is what a rune does.
//
// **A closed vocabulary**, the same posture EssenceTarget and combat.Verb take. A new target is a Go
// change plus one place applying it, never something a JSON file can assert into existence.
type RuneTarget int

const (
	// RuneRider attaches a lasting rule to a card — see combat.Rider. The card goes on being
	// the card it was; what changes is that it now does something extra when it is played.
	RuneRider RuneTarget = iota

	// RuneRemove takes cards out of the run for good. **Count may be more than one**, which is
	// the field an essence never had.
	RuneRemove

	// RuneVitae fills the purse and touches no card. **Count is zero for it**, and the board
	// piece asks for no target at all.
	RuneVitae

	// RuneDuplicate copies a card. **It is the essence `spawn` spent in the middle of a fight**,
	// which is the whole difference: the copy goes into the dealt hand rather than only into the
	// run's deck, so the player has two of the card *this turn* — see `Session.Duplicated`, and
	// `CombatScene.takeRune`, which is what puts it in the row.
	RuneDuplicate

	// RuneElement recolors cards. **Count is two**, which is the difference from the
	// elemental essences: an essence buys one card of a color and this buys a pair, which is a hand.
	RuneElement

	// RuneForm changes what a card counts as on the form axis, without changing the card.
	//
	// **A defend card is a legal target** *(owner's call, 2026-09-02)*. A Brace told to be a crush
	// still shields and now matches crushes, which is a card doing something no card in the
	// catalog does. That is the point of it rather than a hole in the checking.
	RuneForm

	// RuneStones hands the run a shower of random stones. **Count is zero** — it touches no
	// card at all, exactly as RuneVitae does — and Value is how many stones.
	//
	// **All of them are kept, and there is no pick** *(owner's call, 2026-09-02)*. A bag of rocks
	// offers four and keeps one; this shows three and keeps three. It is the one consumable that
	// rolls while it is being spent rather than while a shelf is being stocked, which is why it is
	// the only target that needs a source — see ApplyRuneRolling.
	RuneStones

	// RuneClone turns one card into another card the player picked. **The first pick is the
	// card that changes and the second is the template**, which is the one target whose two seats
	// are not interchangeable — see ApplyRune.
	RuneClone

	// RuneChimera fires whatever the run spent last, again.
	//
	// **It carries no effect of its own** and is resolved through `Session.Echoes` before anything
	// reads it: what it costs, how many cards it names and what it does to them all come from the
	// rune it is copying. So a chimera behind an Embermark asks for two cards and paints them
	// fire, and a chimera behind a Hoard asks for none.
	//
	// **The memory is the run's, not the fight's** *(owner's call, 2026-09-07)*. A chimera carried
	// out of one duel and into the next still copies what was spent in the first. It refuses only
	// on a run where nothing has ever been spent — there is no effect to copy, and a consumable
	// that landed and did nothing is something bought and taken away.
	//
	// **The targets are picked again rather than inherited.** The copied rune's cards are long
	// gone from the hand by the time a chimera is spent — a different turn, sometimes a different
	// fight — and re-firing against the same identities would be a no-op wherever the effect was
	// idempotent, which is most of the catalog.
	//
	// **A chimera never becomes the thing to copy.** `lastRune` records the *resolved* record,
	// so a chimera behind a Goad leaves Goad behind it, and two chimeras in a row both fire Goad
	// rather than the second one copying the first into nothing.
	RuneChimera
)

// RuneChange is what class of alteration a rune makes to a card.
//
// **The grammar it belongs to** *(owner's call, 2026-09-09)*: a card has a form, an element and an
// action, and those three compose freely — a Jab painted fire and reformed to crush is all three at
// once, and no rune that moves one of them has any opinion about the others. Then it has **one
// upgrade**, which is a rider, and there is only ever one: writing it discards whatever was there.
//
// **The distinction is worth naming because it is invisible in the effect.** Bulwark turns a card
// into a Guard and Golden makes it gold, and from the outside both are "a rune changed my
// card". What separates them is what the *next* rune does — the second normal change leaves the
// first standing, and the second upgrade erases it — and a player who cannot tell which class a
// card in the shop belongs to cannot plan two purchases ahead.
//
// **A closed vocabulary and a required field**, the posture every other word a data file may write
// is under. It is authored rather than derived: see data.RuneData.Change.
type RuneChange int

const (
	// RuneNormal moves the form, the element or the action, and leaves the card's upgrade
	// exactly where it was. It is the zero value because it is what almost every rune is.
	RuneNormal RuneChange = iota

	// RuneUpgrade writes the card's one upgrade slot, discarding whatever it held.
	RuneUpgrade
)

// RuneChanges is every change class in a fixed order, for anything that walks them.
func RuneChanges() []RuneChange {
	return []RuneChange{RuneNormal, RuneUpgrade}
}

func (c RuneChange) String() string {
	if c == RuneUpgrade {
		return "upgrade"
	}
	return "normal"
}

// ParseRuneChange resolves a change class from its name, reporting failure rather than falling
// back to one — including for the empty string, so a record that never declared its class is
// refused at load rather than quietly filed as normal.
func ParseRuneChange(name string) (RuneChange, bool) {
	for _, c := range RuneChanges() {
		if c.String() == name {
			return c, true
		}
	}
	return RuneNormal, false
}

// changeFor is the class a target actually makes, which is what an authored Change is checked
// against.
//
// **A rider is the only upgrade there is.** Everything else in the catalog moves one of the three
// facts a card composes freely, or touches no card at all — and a rune that touches no card is
// normal by the same argument, since there is nothing for it to overwrite.
func changeFor(t RuneTarget) RuneChange {
	if t == RuneRider {
		return RuneUpgrade
	}
	return RuneNormal
}

// RuneTargets is every target in a fixed order, for anything that walks them.
func RuneTargets() []RuneTarget {
	return []RuneTarget{
		RuneRider, RuneRemove, RuneVitae,
		RuneDuplicate, RuneElement, RuneForm, RuneStones, RuneClone,
		RuneChimera,
	}
}

func (t RuneTarget) String() string {
	switch t {
	case RuneRemove:
		return "remove"
	case RuneVitae:
		return "vitae"
	case RuneDuplicate:
		return "duplicate"
	case RuneElement:
		return "element"
	case RuneForm:
		return "form"
	case RuneStones:
		return "stones"
	case RuneClone:
		return "clone"
	case RuneChimera:
		return "chimera"
	default:
		return "rider"
	}
}

// ParseRuneTarget resolves a target from its name, reporting failure rather than falling back
// to one. A rune quietly registered as a rider because its target was misspelled is a
// mechanic nobody designed.
func ParseRuneTarget(name string) (RuneTarget, bool) {
	for _, t := range RuneTargets() {
		if t.String() == name {
			return t, true
		}
	}
	return RuneRider, false
}

// MaxRuneTargets is the most cards one rune may name.
//
// **Two, because the board piece shows the targets side by side** and a picker that scrolled would
// be a menu to read rather than a decision to make — the same argument the two-essence offer is
// under. It is a layout number as much as a rules one.
const MaxRuneTargets = 2

// Rune is one consumable, resolved against the rules.
//
// Comparable, so a screen can hold one by value — the same property Essence has and for the same
// reason.
type Rune struct {
	Record string
	Name   string
	Text   string
	Target RuneTarget

	// Art is the assets key of the picture this rune draws, already resolved through
	// data.RuneData.ArtKey — never empty, and the placeholder for a record nobody has drawn.
	Art string

	// Family and Draw are authored, ignored by everything that plays the game, and read only by
	// tools/runesheet — the motif the record was written under, and the art brief for its
	// picture.
	Family string
	Draw   string

	// Change is the class of alteration this rune makes — see RuneChange. It is what says
	// whether spending it discards the card's existing upgrade.
	Change RuneChange

	// Count is how many cards of the run this rune takes. Zero for vitae, at least one for
	// everything else.
	Count int

	// Number is the figure read against the target: life for a heal-on-play rider, vitae for
	// vitae. Meaningless elsewhere.
	Number int

	// Rider is the rule a rider rune attaches, already resolved. RiderNone elsewhere.
	Rider combat.RiderKind

	// Element is what an element rune recolors to, already resolved. Basic elsewhere.
	Element combat.Element

	// Form is what a form rune makes a card count as, already resolved. FormNone elsewhere.
	Form combat.Form
}

// runes is the validated catalog, built once at package init.
//
// **A bad record panics at init**, so it fails on launch rather than the first time a player opens
// a sack — the same severity a bad essence record takes, and for the same reason: a consumable that
// does nothing is something bought and taken away.
var runes, runeOrder = loadRunes()

// Runes is every rune in the catalog, in a fixed sorted order.
func Runes() []Rune {
	out := make([]Rune, 0, len(runeOrder))
	for _, key := range runeOrder {
		out = append(out, runes[key])
	}
	return out
}

// RuneByKey finds one by its record key.
func RuneByKey(key string) (Rune, bool) {
	p, ok := runes[key]
	return p, ok
}

func loadRunes() (map[string]Rune, []string) {
	recs := data.LoadRunes()

	out := make(map[string]Rune, len(recs))
	for _, key := range data.RuneOrder(recs) {
		p, err := resolveRune(recs[key])
		if err != nil {
			panic("runes.json: " + err.Error())
		}
		out[key] = p
	}

	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		panic("runes.json: the catalog is empty, and a sack has to hold something")
	}
	return out, keys
}

// resolveRune turns a record into a rune, or says why it cannot.
//
// **It refuses a field the target does not read**, rather than ignoring it — a remove rune
// carrying a rider name is somebody expecting something the mechanic does not do, and accepting it
// silently is how a catalog comes to disagree with the game.
func resolveRune(r data.RuneData) (Rune, error) {
	if r.RuneRecord == "" {
		return Rune{}, fmt.Errorf("a rune has no record key")
	}
	if r.Name == "" {
		return Rune{}, fmt.Errorf("%s has no name", r.RuneRecord)
	}
	if r.Text == "" {
		// The card is a name and a line of text and nothing else, exactly as an essence's is, so a
		// rune with no text is a card that does not say what it does.
		return Rune{}, fmt.Errorf("%s has no text, so its card says nothing", r.RuneRecord)
	}

	target, ok := ParseRuneTarget(r.Target)
	if !ok {
		return Rune{}, fmt.Errorf("%s names target %q, which is not one of %s",
			r.RuneRecord, r.Target, runeTargetList())
	}

	change, ok := ParseRuneChange(r.Change)
	if !ok {
		return Rune{}, fmt.Errorf("%s declares change %q, which is not one of normal/upgrade",
			r.RuneRecord, r.Change)
	}
	// **The record's claim is checked against what its target actually does.** An authored field
	// that nothing verified would be a second source of truth free to drift from the first, which
	// is worse than deriving it — see data.RuneData.Change for why it is authored at all.
	if want := changeFor(target); change != want {
		return Rune{}, fmt.Errorf("%s targets %s and calls itself a %s change, and that is a %s one",
			r.RuneRecord, target, change, want)
	}

	p := Rune{Record: r.RuneRecord, Name: r.Name, Text: r.Text,
		Target: target, Change: change, Count: r.Count,
		Element: combat.Basic, Form: combat.FormNone,
		Art: r.ArtKey(), Family: r.Family, Draw: r.Draw}

	// **The count is checked against the target rather than in general.** A rune aimed at no
	// card and one aimed at two are both legal, and the mistake worth catching is the mismatch: a
	// remove that eats nothing, or a vitae that asks the player to pick a card it will not touch.
	if target == RuneVitae || target == RuneStones || target == RuneChimera {
		// **A chimera is in this group because its own record names no cards.** How many it
		// actually asks for comes from the rune it copies, and is read through `Echoes` long
		// after this — so the record itself is a zero-target one and is checked as one.
		if r.Count != 0 {
			return Rune{}, fmt.Errorf("%s touches no card and asks for %d of them",
				r.RuneRecord, r.Count)
		}
	} else if r.Count < 1 {
		return Rune{}, fmt.Errorf("%s targets %s and takes %d cards, so it does nothing",
			r.RuneRecord, target, r.Count)
	}
	if r.Count > MaxRuneTargets {
		return Rune{}, fmt.Errorf("%s takes %d cards, and the board piece offers %d",
			r.RuneRecord, r.Count, MaxRuneTargets)
	}

	if r.Rider != "" && target != RuneRider {
		return Rune{}, fmt.Errorf("%s targets %s and names rider %q, which nothing reads",
			r.RuneRecord, target, r.Rider)
	}

	switch target {
	case RuneRider:
		kind, ok := combat.ParseRiderKind(r.Rider)
		if !ok {
			return Rune{}, fmt.Errorf("%s names rider %q, which the rules do not have",
				r.RuneRecord, r.Rider)
		}
		// **One rider kind has no figure, and it says so itself** *(2026-09-07)*. A wildcard makes
		// a card count as every element, which has no quantity — so a record naming it must write
		// no value, and a record naming any other rider must write one. Asking the vocabulary
		// rather than special-casing a name here is what keeps the next amount-less rider from
		// needing an edit in this file.
		if !kind.CarriesAmount() {
			if r.Value != "" {
				return Rune{}, fmt.Errorf("%s attaches the %s rider and gives it the value %q, "+
					"which nothing reads", r.RuneRecord, kind, r.Value)
			}
			p.Rider = kind
			return p, nil
		}
		n, err := strconv.Atoi(r.Value)
		if err != nil {
			return Rune{}, fmt.Errorf("%s attaches a rider and its value %q is not a number",
				r.RuneRecord, r.Value)
		}
		if n <= 0 {
			return Rune{}, fmt.Errorf("%s attaches a rider worth %d, which is nothing at all",
				r.RuneRecord, n)
		}
		// **The two metals read their figure as a denominator, so it has a floor.** Below the
		// number of outcomes there is no losing face left: a golden card on a d2 grants something
		// every single time it is played, which is a different card entirely and is the mistake a
		// number in a JSON file could make silently. See combat.LuckOutcomes.
		if kind == combat.RiderGolden || kind == combat.RiderSilver {
			if n < combat.LuckOutcomes {
				return Rune{}, fmt.Errorf("%s gambles on 1 in %d, and %d outcomes always pay",
					r.RuneRecord, n, combat.LuckOutcomes)
			}
		}
		p.Rider, p.Number = kind, n
		return p, nil

	case RuneVitae:
		n, err := strconv.Atoi(r.Value)
		if err != nil {
			return Rune{}, fmt.Errorf("%s pays vitae and its value %q is not a number",
				r.RuneRecord, r.Value)
		}
		if n <= 0 {
			return Rune{}, fmt.Errorf("%s pays %d vitae", r.RuneRecord, n)
		}
		p.Number = n
		return p, nil

	case RuneElement:
		// **Resolved against the rules' own element list**, so that a color this build does not
		// have is refused rather than being a rune that lands and paints nothing.
		e, ok := combat.ParseElement(r.Value)
		if !ok {
			return Rune{}, fmt.Errorf("%s recolors to %q, which is not an element the rules have",
				r.RuneRecord, r.Value)
		}
		if e == combat.Basic {
			// Basic is the absence of a color rather than a color, so a rune painting cards
			// basic would be one that takes an element away — a different mechanic, and not one
			// anybody has asked for.
			return Rune{}, fmt.Errorf("%s recolors to basic, which is no color at all",
				r.RuneRecord)
		}
		p.Element = e
		return p, nil

	case RuneForm:
		f, ok := combat.ParseForm(r.Value)
		if !ok {
			return Rune{}, fmt.Errorf("%s makes cards %q, which is not a form the rules have",
				r.RuneRecord, r.Value)
		}
		if f == combat.FormNone {
			return Rune{}, fmt.Errorf("%s makes cards formless, which is not an alteration",
				r.RuneRecord)
		}
		p.Form = f
		return p, nil

	case RuneStones:
		n, err := strconv.Atoi(r.Value)
		if err != nil {
			return Rune{}, fmt.Errorf("%s hands over stones and its value %q is not a number",
				r.RuneRecord, r.Value)
		}
		if n <= 0 {
			return Rune{}, fmt.Errorf("%s hands over %d stones", r.RuneRecord, n)
		}
		if n > len(stoneOrder) {
			// **Without repeats**, on the bag's argument: the same rock twice is a seat spent
			// saying nothing. So a record asking for more than the catalog holds is one that
			// could not be honored, and it is refused rather than quietly shortened.
			return Rune{}, fmt.Errorf("%s hands over %d stones and the catalog holds %d",
				r.RuneRecord, n, len(stoneOrder))
		}
		p.Number = n
		return p, nil

	case RuneClone:
		// **The template is a card the player picks, so there is nothing to resolve here** — what
		// this checks is that the record did not try to name one. A clone carrying a Value is
		// somebody expecting it to name a card, which it never does.
		if r.Count != 2 {
			return Rune{}, fmt.Errorf("%s clones one card into another and takes %d cards, which cannot be done",
				r.RuneRecord, r.Count)
		}
	}

	if r.Value != "" {
		return Rune{}, fmt.Errorf("%s targets %s and carries the value %q, which nothing reads",
			r.RuneRecord, target, r.Value)
	}
	return p, nil
}

func runeTargetList() string {
	out := ""
	for i, t := range RuneTargets() {
		if i > 0 {
			out += ", "
		}
		out += t.String()
	}
	return out
}

// StartingRunes is what a run opens holding in its sack, by record key.
//
// **Empty as shipped, and it is a debug seat**, the counterpart of StartingRelics and for the same
// reason: a rune is bought from the shelf and spent in the fight after it, so the board piece
// is two screens away from any launch. `internal/scenario` is what fills it; nothing else may.
var StartingRunes []string

// Held is the sack: every rune the run is carrying, by record key, in the order they were
// acquired.
//
// **A list rather than a count per key**, because two of the same rune are two things to spend
// and the board piece draws a card for each. Order is acquisition order, which is the only order
// the player can see.
func (s *Session) Held() []string {
	out := make([]string, len(s.held))
	copy(out, s.held)
	return out
}

// MaxHeld is how many runes the run can carry at once.
//
// **Two, and it is a rule rather than a number the layout chose** — the same standing `MaxWornRelics`
// has, and for the same reason: the top row draws the sack as `held/2` beside the relics' `worn/5`,
// and a row saying two while the run carried a third is exactly the drift a displayed cap invites.
//
// **The sack was uncapped until 2026-09-06** *(owner's call)*. What that cost was not storage —
// the dialog's row already tightened its pitch to hold any number — it was that a consumable with
// no ceiling is one a rich run hoards rather than spends. A cap of two makes the third purchase a
// decision about the two you are holding.
const MaxHeld = 2

// HoldCount is how many runes the run is carrying.
func (s *Session) HoldCount() int { return len(s.held) }

// HoldFull is whether the sack has no room. **Asked before a rune is paid for**, which is the
// shop's business: see the sack seat, which goes unavailable rather than taking five vitae for a
// rune that would be refused.
func (s *Session) HoldFull() bool { return len(s.held) >= MaxHeld }

// Hold puts a rune in the sack, and reports whether it went in.
//
// **A rune the catalog does not have is refused**, rather than held as a key nothing can
// resolve — a sack carrying a name that means nothing is a slot the player cannot spend.
//
// **A full sack refuses too.** It is the last line of defense rather than the control the player
// meets: a seat that could be bought and then silently dropped would be the purchase-for-nothing
// this returns false to prevent, and the shop is where it is actually stopped.
func (s *Session) Hold(key string) bool {
	if s.HoldFull() {
		return false
	}
	return s.hold(key)
}

// hold is Hold without the cap: it checks the key and nothing else.
//
// **The one caller is StartingRunes**, which plants a sack rather than acquiring one — see
// NewSession, where the argument is written down. Nothing a player can do reaches this.
func (s *Session) hold(key string) bool {
	if _, ok := runes[key]; !ok {
		return false
	}
	s.held = append(s.held, key)
	return true
}

// MoveRune slides the rune at `from` to sit at `to`, shuffling everything between them along. It
// reports whether the sack actually changed.
//
// **It is an arrangement, not a rule** *(owner's call, 2026-09-17)*, and that is the difference from
// MoveRelic one file over: worn order is the order relics *fire* in, so moving one changes what a
// duel does, where the sack is a row of things waiting to be picked. What reordering buys is
// finding them — a sack can hold far more runes than it has comfortable seats, and the pane packs
// them into slivers, so being able to bring one to the front is the difference between carrying a
// rune and being able to spend it.
//
// **Positions rather than keys**, exactly as Drop takes a position: a sack may hold two of the same
// rune and a reorder must not be ambiguous about which one moved.
//
// An out-of-range index is a no-op rather than a panic — this is driven by a drag, and a drop
// resolved against a sack that changed underneath it must not take the frame with it. The same
// courtesy MoveRelic extends.
func (s *Session) MoveRune(from, to int) bool {
	n := len(s.held)
	if from < 0 || from >= n || to < 0 || to >= n || from == to {
		return false
	}

	key := s.held[from]
	if from < to {
		copy(s.held[from:to], s.held[from+1:to+1])
	} else {
		copy(s.held[to+1:from+1], s.held[to:from])
	}
	s.held[to] = key
	return true
}

// Drop takes one out of the sack by position, and reports whether it was there.
//
// **Spending is Drop plus ApplyRune, and they are separate on purpose.** A rune naming two
// cards is not spent until both are picked and the player confirms, and the picker can be backed
// out of at any point — the same rule the essence morph is under. Dropping first would charge for a
// choice that was never made.
func (s *Session) Drop(i int) bool {
	if i < 0 || i >= len(s.held) {
		return false
	}
	s.held = append(s.held[:i], s.held[i+1:]...)
	return true
}

// ApplyRune performs a rune against the cards it names, by identity.
//
// **All or nothing.** A rune that ate one of its two cards and then found the second gone
// would leave the run in a state the player did not choose, so every target is checked before any
// is touched. It reports whether it fired.
//
// **The one place the deck is altered by a rune**, so there is one place that can get it
// wrong.
func (s *Session) ApplyRune(p Rune, ids []int) bool {
	return s.ApplyRuneRolling(p, ids, nil)
}

// ApplyRuneRolling is ApplyRune with a source, for the one target that rolls.
//
// **Only `stones` reads it**, and it is refused outright without one rather than falling back to a
// default draw — a consumable that quietly handed out the same three rocks every time would be a
// mechanic nobody designed, and it would be invisible. Everything else ignores the source, which
// is what lets `ApplyRune` go on being the call every other site makes.
//
// **The caller owns the seeding**, exactly as the shop's three sealed goods do: the run does not
// know its own seed — `Snapshot` is handed one — so the screen derives the stream and passes the
// source. See `seeds.StoneShower` for what has to be mixed into it.
func (s *Session) ApplyRuneRolling(p Rune, ids []int, rng *rand.Rand) bool {
	if !s.CanApplyRune(p, ids) {
		return false
	}

	// **Resolved before anything reads it**, so a chimera is never asked what it does. From here
	// down `p` is the rune that actually fires, and the chimera is gone — which is what keeps
	// every case below written once. See Session.Echoes.
	p, ok := s.Echoes(p)
	if !ok {
		return false
	}
	if p.Target == RuneStones && rng == nil {
		return false
	}
	// **Remembered on the way in rather than on the way out.** Every branch below returns from
	// inside itself, so a single recording after the switch would be a line nothing reaches; and
	// the only branch that can still fail from here is the rider's, which fails on a card already
	// carrying its maximum — a case CanApplyRune has already refused.
	s.rememberRune(p)

	// **Both handovers are emptied here, by the rune that is firing, rather than by the branch
	// that fills them** *(2026-09-17)*. They say what *this* rune did, and the combat screen reads
	// both after every rune it spends — `Duplicated` to seat the copy in the dealt hand, `Granted`
	// to fly the stones to the pouch. Cleared only by the next rune *of the same kind*, a copy went
	// on being handed over to every rune spent after it and was seated again each time: one Mimic
	// and three more runes is four cards in the hand, from one card minted in the deck. Clearing
	// where the spending happens is what makes "what did the last rune do" answerable by a rune
	// that did neither.
	s.duplicated = s.duplicated[:0]
	s.granted = s.granted[:0]

	switch p.Target {
	case RuneVitae:
		s.AddVitae(p.Number)
		return true

	case RuneRemove:
		// **Removed back to front, so the shifting indices cannot bite.** Positions move as the
		// deck thins; the identities they were resolved from do not.
		positions := s.positionsOf(ids)
		sort.Sort(sort.Reverse(sort.IntSlice(positions)))
		for _, i := range positions {
			s.Remove(i)
		}
		return true

	case RuneStones:
		// **Drawn without repeats and every one of them kept** *(owner's call, 2026-09-02)*. The
		// shuffle-and-take-a-prefix is the bag's own draw, and it is flat for the bag's reason: a
		// stone has no rarity, so weighting them would be pricing the *hand*, which the ladder
		// already does.
		//
		// **They go straight onto the ladder** *(owner's call, 2026-09-19)*. A shower is a handful
		// of rocks arriving at once, and a pouch filling with four stones the player then has to
		// spend one at a time is an inventory chore in front of a rung that was already decided by
		// the draw — there is nothing to choose, so there is nothing to hold. `Granted` still says
		// which ones came, which is what the flight to the duelist card is drawn from.
		all := Stones()
		rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
		if p.Number < len(all) {
			all = all[:p.Number]
		}

		for _, st := range all {
			if s.UseStone(st.Record) {
				s.granted = append(s.granted, st)
			}
		}
		return true

	case RuneDuplicate:
		// **The copy is a new card of the run, with a new identity.** Two cards that look alike are
		// still two cards — see `Card.ID` — so the copy can be altered later without the original
		// changing under it. `Session.Duplicated` is where the screen reads what was minted, since
		// the point of spending this mid-fight is that the copy joins the hand.
		for _, i := range s.positionsOf(ids) {
			card := s.deck[i]
			s.Add(card)
			s.duplicated = append(s.duplicated, s.deck[len(s.deck)-1])
		}
		return true

	case RuneElement:
		for _, i := range s.positionsOf(ids) {
			s.deck[i].Element = p.Element
		}
		return true

	case RuneForm:
		for _, i := range s.positionsOf(ids) {
			// **An override rather than a replacement**, so the card goes on being the card it
			// was and only the axis it is counted on moves. See `combat.Card.FormOverride`.
			s.deck[i].FormOverride = p.Form
		}
		return true

	case RuneClone:
		// **The order of the picks is the whole rule**: the first becomes the second. Everything
		// else about a rune treats its targets as a set, and this is the one that cannot —
		// `CanApplyRune` refuses a pick that would change nothing, so there is always a
		// direction.
		positions := s.positionsOf(ids)
		if len(positions) != 2 {
			return false
		}

		// **Everything but the identity** *(owner's call, 2026-09-08)*. It copied the concept alone
		// until then, so grafting a fire Cut onto an ice Jab produced an ice Cut — a card whose name
		// said it had become the right-hand card and whose color said it had not. "BECOMES" is not
		// a partial verb, and the same bug was waiting on the form override, the essence deltas and the
		// riders.
		//
		// **`ID` is what does not travel**, because it is the one field that says *which* card this
		// is rather than what it is — see `combat.Card.ID`. A graft leaves the run holding the same
		// number of cards it held before, each still findable by the handle it has always had.
		id := s.deck[positions[0]].ID
		s.deck[positions[0]] = s.deck[positions[1]]
		s.deck[positions[0]].ID = id
		return true

	default:
		// **The upgrade replaces whatever the card was carrying** — see combat.Card.SetRider, and
		// RuneChange, which is the declaration in the record that says this rune is one of
		// the ones allowed to do it.
		for _, i := range s.positionsOf(ids) {
			s.deck[i] = s.deck[i].SetRider(combat.Rider{Kind: p.Rider, Amount: p.Number})
		}
		return true
	}
}

// CanApplyRune reports whether this rune would do anything to these cards.
//
// **The board piece asks before it offers**, on the same terms CanApply is asked for an essence: a
// rune that lands and changes nothing is something bought and taken away. It also refuses the
// wrong number of targets, which is what stops a two-card rune being spent on one.
func (s *Session) CanApplyRune(p Rune, ids []int) bool {
	// **The chimera is resolved first, so the count and the legality asked about below are the
	// copied rune's.** A chimera with nothing to copy is refused here rather than lower down,
	// which is what makes the consumables pane draw it dim.
	p, ok := s.Echoes(p)
	if !ok {
		return false
	}

	if len(ids) != p.Count {
		return false
	}

	seen := make(map[int]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			// The same card named twice is a picker bug, and letting it through would mean a
			// two-card rune spent on one card for double the effect.
			return false
		}
		seen[id] = true

		card, ok := s.CardByID(id)
		if !ok {
			return false
		}

		switch p.Target {
		case RuneElement:
			if card.Element == p.Element {
				return false
			}
		case RuneForm:
			// **Asked of the card's *current* form, not of its concept's**, so a card already
			// overridden to crush is not a legal target for a second crush rune. A defend card
			// is legal and deliberately so — see RuneForm.
			if card.Form() == p.Form {
				return false
			}
		case RuneRider:
			// **A card already carrying exactly this upgrade is the only illegal pick.** It used
			// to be a card carrying its maximum, which is a different question and stopped making
			// sense the day the maximum became one: a gold card the player wants to make silver is
			// full and is also the pick they came for. What is refused now is the pick that would
			// change nothing, which is the rule every other target here is under.
			if card.Rider() == (combat.Rider{Kind: p.Rider, Amount: p.Number}) {
				return false
			}
		}
	}

	// **The clone is checked as a pair rather than card by card**, which is the only target that
	// can be: whether it does anything is a fact about the two picks together.
	//
	// **Compared on everything but the identity**, which is exactly what the apply copies. It asked
	// only about the concept while only the concept was copied, and that made two same-named cards
	// of different colors an illegal pick — the pick a player reaching for this most obviously
	// wants, now that the color travels with the name.
	if p.Target == RuneClone && len(ids) == 2 {
		first, ok1 := s.CardByID(ids[0])
		second, ok2 := s.CardByID(ids[1])
		if !ok1 || !ok2 {
			return false
		}
		first.ID, second.ID = 0, 0
		if first == second {
			return false
		}
	}
	return true
}

// Duplicated is the cards the last duplicate rune minted, in the order they were made.
//
// **It exists because the copy has to reach the dealt hand** *(owner's call, 2026-09-02)*. An essence
// copying a card between fights only has to put it in the deck; a rune is spent in the middle
// of one, and a copy that could not be played until the next fight would read as a dud. The screen
// reads this straight after `ApplyRune` and seats what it finds — see
// `CombatScene.takeRune`.
//
// **It is cleared by the next rune rather than by the reader**, so it is only ever the most
// recent one and never a queue somebody has to remember to drain. **By the next rune of any kind**
// *(2026-09-17)*: it was cleared by the next *duplicate* until then, so a copy was still standing
// when the screen read this after the graft spent after it, and got seated in the hand a second
// time. See ApplyRuneRolling, where both handovers are emptied.
// Granted is the stones the last rock shower handed over, in the order they were drawn.
//
// **It exists so the dialog can show them** *(owner's call, 2026-09-02)*: "show them all and put
// them all in". A stone is applied the moment it is owned, so without this the player would watch
// a rune disappear and be told nothing about what it did. Cleared by the next rune rather
// than by the reader, on the same terms Duplicated is.
func (s *Session) Granted() []Stone {
	out := make([]Stone, len(s.granted))
	copy(out, s.granted)
	return out
}

func (s *Session) Duplicated() []combat.Card {
	out := make([]combat.Card, len(s.duplicated))
	copy(out, s.duplicated)
	return out
}

// positionsOf turns identities into deck positions, in the order the identities were given.
//
// Unexported, and only ever called after CanApplyRune has proved every one of them is there —
// which is what lets it skip a miss rather than have to report one.
func (s *Session) positionsOf(ids []int) []int {
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		for i, c := range s.deck {
			if c.ID == id {
				out = append(out, i)
				break
			}
		}
	}
	return out
}
