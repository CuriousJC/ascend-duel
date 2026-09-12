package session

// Parasites: the alterations a run can make to its own deck **while a fight is going on.**
//
// A worm is won between rooms and spent on the spot. A parasite is bought, carried in a bucket,
// and spent in the gap between one turn and the next — so the deck a duel started with is not
// necessarily the deck it ends with. The catalogue is `data/parasites.json`; this file is where a
// record becomes something applicable, and where a bad record is refused.
//
// **It lives here rather than in `internal/combat` for the reason worms do**: a parasite acts on
// the *run's* deck, and the rules have no deck. The one exception is a rider, whose vocabulary is
// a Go enum in `internal/combat` because the rules have to read it while a round resolves — this
// file resolves the name and hands over the value.
//
// **Targets are card *identities*, not positions.** That is the whole difference from worm.go,
// which takes an index and says in its own comment that a caller may not hold two across a call.
// A parasite may eat two cards, and it is spent while a hand, a draw pile and a discard pile are
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

// ParasiteTarget is what a parasite does.
//
// **A closed vocabulary**, the same posture WormTarget and combat.Verb take. A new target is a Go
// change plus one place applying it, never something a JSON file can assert into existence.
type ParasiteTarget int

const (
	// ParasiteRider attaches a lasting rule to a card — see combat.Rider. The card goes on being
	// the card it was; what changes is that it now does something extra when it is played.
	ParasiteRider ParasiteTarget = iota

	// ParasiteRemove takes cards out of the run for good. **Count may be more than one**, which is
	// the field a worm never had.
	ParasiteRemove

	// ParasiteSwap turns a card into a different card the game already defines. The identity is
	// kept, so a card the player has already spent two parasites on does not become a stranger —
	// its riders and its per-card modifiers travel with it.
	ParasiteSwap

	// ParasiteVitae fills the purse and touches no card. **Count is zero for it**, and the board
	// piece asks for no target at all.
	ParasiteVitae

	// ParasiteDuplicate copies a card. **It is the worm `spawn` spent in the middle of a fight**,
	// which is the whole difference: the copy goes into the dealt hand rather than only into the
	// run's deck, so the player has two of the card *this turn* — see `Session.Duplicated`, and
	// `CombatScene.takeParasite`, which is what puts it in the row.
	ParasiteDuplicate

	// ParasiteElement recolours cards. **Count is two**, which is the difference from the
	// elemental worms: a worm buys one card of a colour and this buys a pair, which is a hand.
	ParasiteElement

	// ParasiteForm changes what a card counts as on the form axis, without changing the card.
	//
	// **A defend card is a legal target** *(owner's call, 2026-09-02)*. A Brace told to be a crush
	// still shields and now matches crushes, which is a card doing something no card in the
	// catalogue does. That is the point of it rather than a hole in the checking.
	ParasiteForm

	// ParasiteStones hands the run a shower of random stones. **Count is zero** — it touches no
	// card at all, exactly as ParasiteVitae does — and Value is how many stones.
	//
	// **All of them are kept, and there is no pick** *(owner's call, 2026-09-02)*. A bag of rocks
	// offers four and keeps one; this shows three and keeps three. It is the one consumable that
	// rolls while it is being spent rather than while a shelf is being stocked, which is why it is
	// the only target that needs a source — see ApplyParasiteRolling.
	ParasiteStones

	// ParasiteClone turns one card into another card the player picked. **The first pick is the
	// card that changes and the second is the template**, which is the one target whose two seats
	// are not interchangeable — see ApplyParasite.
	ParasiteClone

	// ParasiteChimera fires whatever the run spent last, again.
	//
	// **It carries no effect of its own** and is resolved through `Session.Echoes` before anything
	// reads it: what it costs, how many cards it names and what it does to them all come from the
	// parasite it is copying. So a chimera behind an Emberbore asks for two cards and paints them
	// fire, and a chimera behind a Hoard asks for none.
	//
	// **The memory is the run's, not the fight's** *(owner's call, 2026-09-07)*. A chimera carried
	// out of one duel and into the next still copies what was spent in the first. It refuses only
	// on a run where nothing has ever been spent — there is no effect to copy, and a consumable
	// that landed and did nothing is something bought and taken away.
	//
	// **The targets are picked again rather than inherited.** The copied parasite's cards are long
	// gone from the hand by the time a chimera is spent — a different turn, sometimes a different
	// fight — and re-firing against the same identities would be a no-op wherever the effect was
	// idempotent, which is most of the catalogue.
	//
	// **A chimera never becomes the thing to copy.** `lastParasite` records the *resolved* record,
	// so a chimera behind a Goad leaves Goad behind it, and two chimeras in a row both fire Goad
	// rather than the second one copying the first into nothing.
	ParasiteChimera
)

// ParasiteChange is what class of alteration a parasite makes to a card.
//
// **The grammar it belongs to** *(owner's call, 2026-09-09)*: a card has a form, an element and an
// action, and those three compose freely — a Jab painted fire and reformed to crush is all three at
// once, and no parasite that moves one of them has any opinion about the others. Then it has **one
// upgrade**, which is a rider, and there is only ever one: writing it discards whatever was there.
//
// **The distinction is worth naming because it is invisible in the effect.** Bulwark turns a card
// into a Guard and Golden makes it gold, and from the outside both are "a parasite changed my
// card". What separates them is what the *next* parasite does — the second normal change leaves the
// first standing, and the second upgrade erases it — and a player who cannot tell which class a
// card in the shop belongs to cannot plan two purchases ahead.
//
// **A closed vocabulary and a required field**, the posture every other word a data file may write
// is under. It is authored rather than derived: see data.ParasiteData.Change.
type ParasiteChange int

const (
	// ParasiteNormal moves the form, the element or the action, and leaves the card's upgrade
	// exactly where it was. It is the zero value because it is what almost every parasite is.
	ParasiteNormal ParasiteChange = iota

	// ParasiteUpgrade writes the card's one upgrade slot, discarding whatever it held.
	ParasiteUpgrade
)

// ParasiteChanges is every change class in a fixed order, for anything that walks them.
func ParasiteChanges() []ParasiteChange {
	return []ParasiteChange{ParasiteNormal, ParasiteUpgrade}
}

func (c ParasiteChange) String() string {
	if c == ParasiteUpgrade {
		return "upgrade"
	}
	return "normal"
}

// ParseParasiteChange resolves a change class from its name, reporting failure rather than falling
// back to one — including for the empty string, so a record that never declared its class is
// refused at load rather than quietly filed as normal.
func ParseParasiteChange(name string) (ParasiteChange, bool) {
	for _, c := range ParasiteChanges() {
		if c.String() == name {
			return c, true
		}
	}
	return ParasiteNormal, false
}

// changeFor is the class a target actually makes, which is what an authored Change is checked
// against.
//
// **A rider is the only upgrade there is.** Everything else in the catalogue moves one of the three
// facts a card composes freely, or touches no card at all — and a parasite that touches no card is
// normal by the same argument, since there is nothing for it to overwrite.
func changeFor(t ParasiteTarget) ParasiteChange {
	if t == ParasiteRider {
		return ParasiteUpgrade
	}
	return ParasiteNormal
}

// ParasiteTargets is every target in a fixed order, for anything that walks them.
func ParasiteTargets() []ParasiteTarget {
	return []ParasiteTarget{
		ParasiteRider, ParasiteRemove, ParasiteSwap, ParasiteVitae,
		ParasiteDuplicate, ParasiteElement, ParasiteForm, ParasiteStones, ParasiteClone,
		ParasiteChimera,
	}
}

func (t ParasiteTarget) String() string {
	switch t {
	case ParasiteRemove:
		return "remove"
	case ParasiteSwap:
		return "swap"
	case ParasiteVitae:
		return "vitae"
	case ParasiteDuplicate:
		return "duplicate"
	case ParasiteElement:
		return "element"
	case ParasiteForm:
		return "form"
	case ParasiteStones:
		return "stones"
	case ParasiteClone:
		return "clone"
	case ParasiteChimera:
		return "chimera"
	default:
		return "rider"
	}
}

// ParseParasiteTarget resolves a target from its name, reporting failure rather than falling back
// to one. A parasite quietly registered as a rider because its target was misspelled is a
// mechanic nobody designed.
func ParseParasiteTarget(name string) (ParasiteTarget, bool) {
	for _, t := range ParasiteTargets() {
		if t.String() == name {
			return t, true
		}
	}
	return ParasiteRider, false
}

// MaxParasiteTargets is the most cards one parasite may name.
//
// **Two, because the board piece shows the targets side by side** and a picker that scrolled would
// be a menu to read rather than a decision to make — the same argument the two-worm offer is
// under. It is a layout number as much as a rules one.
const MaxParasiteTargets = 2

// Parasite is one consumable, resolved against the rules.
//
// Comparable, so a screen can hold one by value — the same property Worm has and for the same
// reason.
type Parasite struct {
	Record string
	Name   string
	Text   string
	Target ParasiteTarget

	// Change is the class of alteration this parasite makes — see ParasiteChange. It is what says
	// whether spending it discards the card's existing upgrade.
	Change ParasiteChange

	// Count is how many cards of the run this parasite takes. Zero for vitae, at least one for
	// everything else.
	Count int

	// Number is the figure read against the target: life for a heal-on-play rider, vitae for
	// vitae. Meaningless elsewhere.
	Number int

	// Rider is the rule a rider parasite attaches, already resolved. RiderNone elsewhere.
	Rider combat.RiderKind

	// Concept is what a swap parasite turns a card into, already resolved. NoConcept elsewhere.
	Concept combat.ConceptID

	// Element is what an element parasite recolours to, already resolved. Basic elsewhere.
	Element combat.Element

	// Form is what a form parasite makes a card count as, already resolved. FormNone elsewhere.
	Form combat.Form
}

// parasites is the validated catalogue, built once at package init.
//
// **A bad record panics at init**, so it fails on launch rather than the first time a player opens
// a bucket — the same severity a bad worm record takes, and for the same reason: a consumable that
// does nothing is something bought and taken away.
var parasites, parasiteOrder = loadParasites()

// Parasites is every parasite in the catalogue, in a fixed sorted order.
func Parasites() []Parasite {
	out := make([]Parasite, 0, len(parasiteOrder))
	for _, key := range parasiteOrder {
		out = append(out, parasites[key])
	}
	return out
}

// ParasiteByKey finds one by its record key.
func ParasiteByKey(key string) (Parasite, bool) {
	p, ok := parasites[key]
	return p, ok
}

func loadParasites() (map[string]Parasite, []string) {
	recs := data.LoadParasites()

	out := make(map[string]Parasite, len(recs))
	for _, key := range data.ParasiteOrder(recs) {
		p, err := resolveParasite(recs[key])
		if err != nil {
			panic("parasites.json: " + err.Error())
		}
		out[key] = p
	}

	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		panic("parasites.json: the catalogue is empty, and a bucket has to hold something")
	}
	return out, keys
}

// resolveParasite turns a record into a parasite, or says why it cannot.
//
// **It refuses a field the target does not read**, rather than ignoring it — a remove parasite
// carrying a rider name is somebody expecting something the mechanic does not do, and accepting it
// silently is how a catalogue comes to disagree with the game.
func resolveParasite(r data.ParasiteData) (Parasite, error) {
	if r.ParasiteRecord == "" {
		return Parasite{}, fmt.Errorf("a parasite has no record key")
	}
	if r.Name == "" {
		return Parasite{}, fmt.Errorf("%s has no name", r.ParasiteRecord)
	}
	if r.Text == "" {
		// The card is a name and a line of text and nothing else, exactly as a worm's is, so a
		// parasite with no text is a card that does not say what it does.
		return Parasite{}, fmt.Errorf("%s has no text, so its card says nothing", r.ParasiteRecord)
	}

	target, ok := ParseParasiteTarget(r.Target)
	if !ok {
		return Parasite{}, fmt.Errorf("%s names target %q, which is not one of %s",
			r.ParasiteRecord, r.Target, parasiteTargetList())
	}

	change, ok := ParseParasiteChange(r.Change)
	if !ok {
		return Parasite{}, fmt.Errorf("%s declares change %q, which is not one of normal/upgrade",
			r.ParasiteRecord, r.Change)
	}
	// **The record's claim is checked against what its target actually does.** An authored field
	// that nothing verified would be a second source of truth free to drift from the first, which
	// is worse than deriving it — see data.ParasiteData.Change for why it is authored at all.
	if want := changeFor(target); change != want {
		return Parasite{}, fmt.Errorf("%s targets %s and calls itself a %s change, and that is a %s one",
			r.ParasiteRecord, target, change, want)
	}

	p := Parasite{Record: r.ParasiteRecord, Name: r.Name, Text: r.Text,
		Target: target, Change: change, Count: r.Count, Concept: combat.NoConcept,
		Element: combat.Basic, Form: combat.FormNone}

	// **The count is checked against the target rather than in general.** A parasite aimed at no
	// card and one aimed at two are both legal, and the mistake worth catching is the mismatch: a
	// remove that eats nothing, or a vitae that asks the player to pick a card it will not touch.
	if target == ParasiteVitae || target == ParasiteStones || target == ParasiteChimera {
		// **A chimera is in this group because its own record names no cards.** How many it
		// actually asks for comes from the parasite it copies, and is read through `Echoes` long
		// after this — so the record itself is a zero-target one and is checked as one.
		if r.Count != 0 {
			return Parasite{}, fmt.Errorf("%s touches no card and asks for %d of them",
				r.ParasiteRecord, r.Count)
		}
	} else if r.Count < 1 {
		return Parasite{}, fmt.Errorf("%s targets %s and takes %d cards, so it does nothing",
			r.ParasiteRecord, target, r.Count)
	}
	if r.Count > MaxParasiteTargets {
		return Parasite{}, fmt.Errorf("%s takes %d cards, and the board piece offers %d",
			r.ParasiteRecord, r.Count, MaxParasiteTargets)
	}

	if r.Rider != "" && target != ParasiteRider {
		return Parasite{}, fmt.Errorf("%s targets %s and names rider %q, which nothing reads",
			r.ParasiteRecord, target, r.Rider)
	}

	switch target {
	case ParasiteRider:
		kind, ok := combat.ParseRiderKind(r.Rider)
		if !ok {
			return Parasite{}, fmt.Errorf("%s names rider %q, which the rules do not have",
				r.ParasiteRecord, r.Rider)
		}
		// **One rider kind has no figure, and it says so itself** *(2026-09-07)*. A wildcard makes
		// a card count as every element, which has no quantity — so a record naming it must write
		// no value, and a record naming any other rider must write one. Asking the vocabulary
		// rather than special-casing a name here is what keeps the next amount-less rider from
		// needing an edit in this file.
		if !kind.CarriesAmount() {
			if r.Value != "" {
				return Parasite{}, fmt.Errorf("%s attaches the %s rider and gives it the value %q, "+
					"which nothing reads", r.ParasiteRecord, kind, r.Value)
			}
			p.Rider = kind
			return p, nil
		}
		n, err := strconv.Atoi(r.Value)
		if err != nil {
			return Parasite{}, fmt.Errorf("%s attaches a rider and its value %q is not a number",
				r.ParasiteRecord, r.Value)
		}
		if n <= 0 {
			return Parasite{}, fmt.Errorf("%s attaches a rider worth %d, which is nothing at all",
				r.ParasiteRecord, n)
		}
		// **The two metals read their figure as a denominator, so it has a floor.** Below the
		// number of outcomes there is no losing face left: a golden card on a d2 grants something
		// every single time it is played, which is a different card entirely and is the mistake a
		// number in a JSON file could make silently. See combat.LuckOutcomes.
		if kind == combat.RiderGolden || kind == combat.RiderSilver {
			if n < combat.LuckOutcomes {
				return Parasite{}, fmt.Errorf("%s gambles on 1 in %d, and %d outcomes always pay",
					r.ParasiteRecord, n, combat.LuckOutcomes)
			}
		}
		p.Rider, p.Number = kind, n
		return p, nil

	case ParasiteVitae:
		n, err := strconv.Atoi(r.Value)
		if err != nil {
			return Parasite{}, fmt.Errorf("%s pays vitae and its value %q is not a number",
				r.ParasiteRecord, r.Value)
		}
		if n <= 0 {
			return Parasite{}, fmt.Errorf("%s pays %d vitae", r.ParasiteRecord, n)
		}
		p.Number = n
		return p, nil

	case ParasiteSwap:
		// **Resolved against the registry, so a parasite cannot invent a card.** That is the same
		// safety property the worms have — the concept is never one internal/combat has not
		// registered — and it is what keeps a consumable from being a way to author cards in a
		// JSON file the rules never read.
		id, ok := combat.ConceptByKey(r.Value)
		if !ok {
			return Parasite{}, fmt.Errorf("%s turns a card into %q, which is not a card this build has",
				r.ParasiteRecord, r.Value)
		}
		p.Concept = id
		return p, nil

	case ParasiteElement:
		// **Resolved against the rules' own element list**, for the reason a swap resolves against
		// the concept registry: a colour this build does not have is a parasite that would land
		// and paint nothing.
		e, ok := combat.ParseElement(r.Value)
		if !ok {
			return Parasite{}, fmt.Errorf("%s recolours to %q, which is not an element the rules have",
				r.ParasiteRecord, r.Value)
		}
		if e == combat.Basic {
			// Basic is the absence of a colour rather than a colour, so a parasite painting cards
			// basic would be one that takes an element away — a different mechanic, and not one
			// anybody has asked for.
			return Parasite{}, fmt.Errorf("%s recolours to basic, which is no colour at all",
				r.ParasiteRecord)
		}
		p.Element = e
		return p, nil

	case ParasiteForm:
		f, ok := combat.ParseForm(r.Value)
		if !ok {
			return Parasite{}, fmt.Errorf("%s makes cards %q, which is not a form the rules have",
				r.ParasiteRecord, r.Value)
		}
		if f == combat.FormNone {
			return Parasite{}, fmt.Errorf("%s makes cards formless, which is not an alteration",
				r.ParasiteRecord)
		}
		p.Form = f
		return p, nil

	case ParasiteStones:
		n, err := strconv.Atoi(r.Value)
		if err != nil {
			return Parasite{}, fmt.Errorf("%s hands over stones and its value %q is not a number",
				r.ParasiteRecord, r.Value)
		}
		if n <= 0 {
			return Parasite{}, fmt.Errorf("%s hands over %d stones", r.ParasiteRecord, n)
		}
		if n > len(stoneOrder) {
			// **Without repeats**, on the bag's argument: the same rock twice is a seat spent
			// saying nothing. So a record asking for more than the catalogue holds is one that
			// could not be honoured, and it is refused rather than quietly shortened.
			return Parasite{}, fmt.Errorf("%s hands over %d stones and the catalogue holds %d",
				r.ParasiteRecord, n, len(stoneOrder))
		}
		p.Number = n
		return p, nil

	case ParasiteClone:
		// **The template is a card the player picks, so there is nothing to resolve here** — what
		// this checks is that the record did not try to name one. A clone carrying a Value is
		// somebody expecting the swap it is not.
		if r.Count != 2 {
			return Parasite{}, fmt.Errorf("%s clones one card into another and takes %d cards, which cannot be done",
				r.ParasiteRecord, r.Count)
		}
	}

	if r.Value != "" {
		return Parasite{}, fmt.Errorf("%s targets %s and carries the value %q, which nothing reads",
			r.ParasiteRecord, target, r.Value)
	}
	return p, nil
}

func parasiteTargetList() string {
	out := ""
	for i, t := range ParasiteTargets() {
		if i > 0 {
			out += ", "
		}
		out += t.String()
	}
	return out
}

// StartingParasites is what a run opens holding in its bucket, by record key.
//
// **Empty as shipped, and it is a debug seat**, the counterpart of StartingRelics and for the same
// reason: a parasite is bought from the shelf and spent in the fight after it, so the board piece
// is two screens away from any launch. `internal/scenario` is what fills it; nothing else may.
var StartingParasites []string

// Held is the bucket: every parasite the run is carrying, by record key, in the order they were
// acquired.
//
// **A list rather than a count per key**, because two of the same parasite are two things to spend
// and the board piece draws a card for each. Order is acquisition order, which is the only order
// the player can see.
func (s *Session) Held() []string {
	out := make([]string, len(s.held))
	copy(out, s.held)
	return out
}

// MaxHeld is how many parasites the run can carry at once.
//
// **Two, and it is a rule rather than a number the layout chose** — the same standing `MaxWornRelics`
// has, and for the same reason: the top row draws the bucket as `held/2` beside the relics' `worn/5`,
// and a row saying two while the run carried a third is exactly the drift a displayed cap invites.
//
// **The bucket was uncapped until 2026-09-06** *(owner's call)*. What that cost was not storage —
// the dialog's row already tightened its pitch to hold any number — it was that a consumable with
// no ceiling is one a rich run hoards rather than spends. A cap of two makes the third purchase a
// decision about the two you are holding.
const MaxHeld = 2

// HoldCount is how many parasites the run is carrying.
func (s *Session) HoldCount() int { return len(s.held) }

// HoldFull is whether the bucket has no room. **Asked before a parasite is paid for**, which is the
// shop's business: see the bucket seat, which goes unavailable rather than taking five vitae for a
// parasite that would be refused.
func (s *Session) HoldFull() bool { return len(s.held) >= MaxHeld }

// Hold puts a parasite in the bucket, and reports whether it went in.
//
// **A parasite the catalogue does not have is refused**, rather than held as a key nothing can
// resolve — a bucket carrying a name that means nothing is a slot the player cannot spend.
//
// **A full bucket refuses too.** It is the last line of defence rather than the control the player
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
// **The one caller is StartingParasites**, which plants a bucket rather than acquiring one — see
// NewSession, where the argument is written down. Nothing a player can do reaches this.
func (s *Session) hold(key string) bool {
	if _, ok := parasites[key]; !ok {
		return false
	}
	s.held = append(s.held, key)
	return true
}

// Drop takes one out of the bucket by position, and reports whether it was there.
//
// **Spending is Drop plus ApplyParasite, and they are separate on purpose.** A parasite naming two
// cards is not spent until both are picked and the player confirms, and the picker can be backed
// out of at any point — the same rule the worm morph is under. Dropping first would charge for a
// choice that was never made.
func (s *Session) Drop(i int) bool {
	if i < 0 || i >= len(s.held) {
		return false
	}
	s.held = append(s.held[:i], s.held[i+1:]...)
	return true
}

// ApplyParasite performs a parasite against the cards it names, by identity.
//
// **All or nothing.** A parasite that ate one of its two cards and then found the second gone
// would leave the run in a state the player did not choose, so every target is checked before any
// is touched. It reports whether it fired.
//
// **The one place the deck is altered by a parasite**, so there is one place that can get it
// wrong.
func (s *Session) ApplyParasite(p Parasite, ids []int) bool {
	return s.ApplyParasiteRolling(p, ids, nil)
}

// ApplyParasiteRolling is ApplyParasite with a source, for the one target that rolls.
//
// **Only `stones` reads it**, and it is refused outright without one rather than falling back to a
// default draw — a consumable that quietly handed out the same three rocks every time would be a
// mechanic nobody designed, and it would be invisible. Everything else ignores the source, which
// is what lets `ApplyParasite` go on being the call every other site makes.
//
// **The caller owns the seeding**, exactly as the shop's three sealed goods do: the run does not
// know its own seed — `Snapshot` is handed one — so the screen derives the stream and passes the
// source. See `seeds.StoneShower` for what has to be mixed into it.
func (s *Session) ApplyParasiteRolling(p Parasite, ids []int, rng *rand.Rand) bool {
	if !s.CanApplyParasite(p, ids) {
		return false
	}

	// **Resolved before anything reads it**, so a chimera is never asked what it does. From here
	// down `p` is the parasite that actually fires, and the chimera is gone — which is what keeps
	// every case below written once. See Session.Echoes.
	p, ok := s.Echoes(p)
	if !ok {
		return false
	}
	if p.Target == ParasiteStones && rng == nil {
		return false
	}
	// **Remembered on the way in rather than on the way out.** Every branch below returns from
	// inside itself, so a single recording after the switch would be a line nothing reaches; and
	// the only branch that can still fail from here is the rider's, which fails on a card already
	// carrying its maximum — a case CanApplyParasite has already refused.
	s.rememberParasite(p)

	switch p.Target {
	case ParasiteVitae:
		s.AddVitae(p.Number)
		return true

	case ParasiteRemove:
		// **Removed back to front, so the shifting indices cannot bite.** Positions move as the
		// deck thins; the identities they were resolved from do not.
		positions := s.positionsOf(ids)
		sort.Sort(sort.Reverse(sort.IntSlice(positions)))
		for _, i := range positions {
			s.Remove(i)
		}
		return true

	case ParasiteSwap:
		for _, i := range s.positionsOf(ids) {
			// **The identity is kept and so are the riders.** A card the player has already spent
			// parasites on stays the card they invested in; what changes is which card it is.
			s.deck[i].Concept = p.Concept
		}
		return true

	case ParasiteStones:
		// **Drawn without repeats and every one of them kept** *(owner's call, 2026-09-02)*. The
		// shuffle-and-take-a-prefix is the bag's own draw, and it is flat for the bag's reason: a
		// stone has no rarity, so weighting them would be pricing the *hand*, which the ladder
		// already does.
		//
		// **They go into the pouch, not onto the ladder** *(owner's call, 2026-09-02)*. A shower
		// hands over consumables to be spent or sold later; it does not decide which rungs the run
		// is raising. See `Session.Carry`, and `StoneSalePrice` for the other thing that can happen
		// to one.
		all := Stones()
		rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
		if p.Number < len(all) {
			all = all[:p.Number]
		}

		s.granted = s.granted[:0]
		for _, st := range all {
			if s.Carry(st.Record) {
				s.granted = append(s.granted, st)
			}
		}
		return true

	case ParasiteDuplicate:
		// **The copy is a new card of the run, with a new identity.** Two cards that look alike are
		// still two cards — see `Card.ID` — so the copy can be altered later without the original
		// changing under it. `Session.Duplicated` is where the screen reads what was minted, since
		// the point of spending this mid-fight is that the copy joins the hand.
		s.duplicated = s.duplicated[:0]
		for _, i := range s.positionsOf(ids) {
			card := s.deck[i]
			s.Add(card)
			s.duplicated = append(s.duplicated, s.deck[len(s.deck)-1])
		}
		return true

	case ParasiteElement:
		for _, i := range s.positionsOf(ids) {
			s.deck[i].Element = p.Element
		}
		return true

	case ParasiteForm:
		for _, i := range s.positionsOf(ids) {
			// **An override rather than a swap**, so the card goes on being the card it was and
			// only the axis it is counted on moves. See `combat.Card.FormOverride`.
			s.deck[i].FormOverride = p.Form
		}
		return true

	case ParasiteClone:
		// **The order of the picks is the whole rule**: the first becomes the second. Everything
		// else about a parasite treats its targets as a set, and this is the one that cannot —
		// `CanApplyParasite` refuses a pick that would change nothing, so there is always a
		// direction.
		positions := s.positionsOf(ids)
		if len(positions) != 2 {
			return false
		}

		// **Everything but the identity** *(owner's call, 2026-09-08)*. It copied the concept alone
		// until then, so grafting a fire Cut onto an ice Jab produced an ice Cut — a card whose name
		// said it had become the right-hand card and whose colour said it had not. "BECOMES" is not
		// a partial verb, and the same bug was waiting on the form override, the worm deltas and the
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
		// ParasiteChange, which is the declaration in the record that says this parasite is one of
		// the ones allowed to do it.
		for _, i := range s.positionsOf(ids) {
			s.deck[i] = s.deck[i].SetRider(combat.Rider{Kind: p.Rider, Amount: p.Number})
		}
		return true
	}
}

// CanApplyParasite reports whether this parasite would do anything to these cards.
//
// **The board piece asks before it offers**, on the same terms CanApply is asked for a worm: a
// parasite that lands and changes nothing is something bought and taken away. It also refuses the
// wrong number of targets, which is what stops a two-card parasite being spent on one.
func (s *Session) CanApplyParasite(p Parasite, ids []int) bool {
	// **The chimera is resolved first, so the count and the legality asked about below are the
	// copied parasite's.** A chimera with nothing to copy is refused here rather than lower down,
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
			// two-card parasite spent on one card for double the effect.
			return false
		}
		seen[id] = true

		card, ok := s.CardByID(id)
		if !ok {
			return false
		}

		switch p.Target {
		case ParasiteSwap:
			if card.Concept == p.Concept {
				return false
			}
		case ParasiteElement:
			if card.Element == p.Element {
				return false
			}
		case ParasiteForm:
			// **Asked of the card's *current* form, not of its concept's**, so a card already
			// overridden to crush is not a legal target for a second crush parasite. A defend card
			// is legal and deliberately so — see ParasiteForm.
			if card.Form() == p.Form {
				return false
			}
		case ParasiteRider:
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
	// of different colours an illegal pick — the pick a player reaching for this most obviously
	// wants, now that the colour travels with the name.
	if p.Target == ParasiteClone && len(ids) == 2 {
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

// Duplicated is the cards the last duplicate parasite minted, in the order they were made.
//
// **It exists because the copy has to reach the dealt hand** *(owner's call, 2026-09-02)*. A worm
// copying a card between fights only has to put it in the deck; a parasite is spent in the middle
// of one, and a copy that could not be played until the next fight would read as a dud. The screen
// reads this straight after `ApplyParasite` and seats what it finds — see
// `CombatScene.takeParasite`.
//
// **It is cleared by the next duplicate rather than by the reader**, so it is only ever the most
// recent one and never a queue somebody has to remember to drain.
// Granted is the stones the last rock shower handed over, in the order they were drawn.
//
// **It exists so the dialog can show them** *(owner's call, 2026-09-02)*: "show them all and put
// them all in". A stone is applied the moment it is owned, so without this the player would watch
// a parasite disappear and be told nothing about what it did. Cleared by the next shower rather
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
// Unexported, and only ever called after CanApplyParasite has proved every one of them is there —
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
