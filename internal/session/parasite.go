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
	// **A defend card is a legal target** *(owner's call, 2026-09-02)*. A Ward told to be a crush
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
)

// ParasiteTargets is every target in a fixed order, for anything that walks them.
func ParasiteTargets() []ParasiteTarget {
	return []ParasiteTarget{
		ParasiteRider, ParasiteRemove, ParasiteSwap, ParasiteVitae,
		ParasiteDuplicate, ParasiteElement, ParasiteForm, ParasiteStones, ParasiteClone,
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

	p := Parasite{Record: r.ParasiteRecord, Name: r.Name, Text: r.Text,
		Target: target, Count: r.Count, Concept: combat.NoConcept,
		Element: combat.Basic, Form: combat.FormNone}

	// **The count is checked against the target rather than in general.** A parasite aimed at no
	// card and one aimed at two are both legal, and the mistake worth catching is the mismatch: a
	// remove that eats nothing, or a vitae that asks the player to pick a card it will not touch.
	if target == ParasiteVitae || target == ParasiteStones {
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
		n, err := strconv.Atoi(r.Value)
		if err != nil {
			return Parasite{}, fmt.Errorf("%s attaches a rider and its value %q is not a number",
				r.ParasiteRecord, r.Value)
		}
		if n <= 0 {
			return Parasite{}, fmt.Errorf("%s attaches a rider worth %d, which is nothing at all",
				r.ParasiteRecord, n)
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
// **Empty as shipped, and it is a debug seat**, the counterpart of StartingRings and for the same
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

// HoldCount is how many parasites the run is carrying.
func (s *Session) HoldCount() int { return len(s.held) }

// Hold puts a parasite in the bucket, and reports whether it went in.
//
// **A parasite the catalogue does not have is refused**, rather than held as a key nothing can
// resolve — a bucket carrying a name that means nothing is a slot the player cannot spend.
func (s *Session) Hold(key string) bool {
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
	if p.Target == ParasiteStones && rng == nil {
		return false
	}

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
		// `CanApplyParasite` refuses two identical picks, so there is always a direction.
		positions := s.positionsOf(ids)
		if len(positions) != 2 {
			return false
		}
		s.deck[positions[0]].Concept = s.deck[positions[1]].Concept
		return true

	default:
		for _, i := range s.positionsOf(ids) {
			card, ok := s.deck[i].AddRider(combat.Rider{Kind: p.Rider, Amount: p.Number})
			if !ok {
				return false
			}
			s.deck[i] = card
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
			if card.RiderCount() >= combat.MaxCardRiders {
				return false
			}
		}
	}

	// **The clone is checked as a pair rather than card by card**, which is the only target that
	// can be: whether it does anything is a fact about the two picks together. Two cards of the
	// same concept would leave the deck exactly as it was found.
	if p.Target == ParasiteClone && len(ids) == 2 {
		first, ok1 := s.CardByID(ids[0])
		second, ok2 := s.CardByID(ids[1])
		if !ok1 || !ok2 || first.Concept == second.Concept {
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
