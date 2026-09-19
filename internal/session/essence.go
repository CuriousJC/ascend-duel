package session

// Essences: the alterations a run can make to its own deck.
//
// **An essence targets one aspect of a card and gives it a new value**, which is the card language's
// shape pointed at a card that already exists rather than at a card being defined. The catalog
// is `data/essences.json`; this file is where a record becomes something applicable, and where a bad
// record is refused.
//
// **It lives here rather than in `internal/combat` because an essence acts on the *run's* deck.** The
// rules resolve rounds and have no deck; the run has one. That is the same who-consumes-it test
// every file in `data/` answers.

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// EssenceTarget is which aspect of a card an essence changes.
//
// **A closed vocabulary, and closing it is the point** — the same posture `combat.Verb` takes.
// The set is short for a structural reason rather than a lack of imagination: `combat.Card` is a
// concept plus an element, and **the element is the only per-instance field**. Cost, damage,
// form and label all live on the shared concept, so an essence targeting one of those would change
// every copy of that card in the deck rather than the one the player picked.
//
// **Cost and amount became per-card on 2026-08-17** — `Card.CostDelta` and `Card.AmountPct` — and
// it was cheaper than it looked: `Cost()` and `Damage()` were already methods on the card, so the
// override went in one place each, and `Amount()` was added beside them for the three sites that
// read `Spec().Amount` directly. **Form and label are still concept-wide**, and an essence reaching
// for one of those is the moment to make the same argument again from scratch.
type EssenceTarget int

const (
	// TargetElement recolors a card. The concept is untouched, so what changes is which color
	// it counts as in a mix and which status it can apply.
	TargetElement EssenceTarget = iota

	// TargetRemove takes a card out of the run for good.
	TargetRemove

	// TargetDuplicate puts a second copy of a card into the run. **Copies are the sharpest dial
	// in the game** — four of one concept in a turn is a Barrage — so this is the essence most
	// likely to need a cost.
	TargetDuplicate

	// TargetCost changes what a card costs, by a signed delta. **Floored at zero, not at one**
	// (owner's call, 2026-08-17): a free card is still bounded, by the count cap rather than by
	// the budget, and that shift was taken with its eyes open.
	TargetCost

	// TargetAmount scales a card's figure, as a percentage — 150 is half again. What the figure
	// *is* depends on the verb, which is what makes one essence reach every card in the deck: a
	// defense percentage, a shield count, or a damage multiplier. A defense is clamped
	// under 100 by `Card.Amount`, because nothing stops a blow outright.
	TargetAmount

	// TargetPromote moves a card one rung up its form's ladder — Jab to Bash to Smash. It
	// costs more and hits harder, and the ladder is a consequence of what `duelist_cards.json`
	// declares rather than a table beside it.
	TargetPromote

	// TargetDemote moves a card one rung down: cheaper and weaker. **Not a downgrade so much as a
	// consistency play** — a hand of cheap cards plays more of itself, and the hand ladder counts
	// copies rather than damage.
	TargetDemote

	// TargetForm changes what one card counts as on the form axis — the element essences' trick on
	// the other matching axis.
	//
	// **It writes `Card.FormOverride`, which a rune already owns**, so this is the vocabulary
	// arriving at a mechanic rather than a new one: form is concept-wide and an essence swapping the
	// concept's form would change every copy in the deck, which is the argument the target list
	// was closed against. The override is per-card and only `Card.Form` reads it.
	//
	// **A Brace told to be a crush still defends**, and now matches on an attack axis — the legal
	// weird thing the owner ruled in when runes got the field.
	TargetForm
)

// EssenceTargets is every target in a fixed order, for anything that walks them.
func EssenceTargets() []EssenceTarget {
	return []EssenceTarget{TargetElement, TargetRemove, TargetDuplicate,
		TargetCost, TargetAmount, TargetPromote, TargetDemote, TargetForm}
}

func (t EssenceTarget) String() string {
	switch t {
	case TargetRemove:
		return "remove"
	case TargetDuplicate:
		return "duplicate"
	case TargetCost:
		return "cost"
	case TargetAmount:
		return "amount"
	case TargetPromote:
		return "promote"
	case TargetDemote:
		return "demote"
	case TargetForm:
		return "form"
	default:
		return "element"
	}
}

// ParseEssenceTarget resolves a target from its name. It reports failure rather than falling back: a
// essence quietly registered as a recolor because its target was misspelled is a mechanic nobody
// designed.
func ParseEssenceTarget(name string) (EssenceTarget, bool) {
	for _, t := range EssenceTargets() {
		if t.String() == name {
			return t, true
		}
	}
	return TargetElement, false
}

// Essence is one alteration, resolved against the rules.
//
// Comparable, so a screen can hold one by value and compare two without reaching for the key.
type Essence struct {
	Record string
	Name   string
	Text   string
	Target EssenceTarget

	// Art is the assets key of the picture this essence draws, already resolved through
	// data.EssenceData.ArtKey — so it is never empty, and an essence nobody has drawn carries the
	// placeholder rather than a hole. Carried here so the reward screen and tools/essencesheet read
	// one answer.
	Art string

	// Family and Draw are authored, ignored by everything that plays the game, and read only by
	// tools/essencesheet — the motif the record was written under, and the art brief for its picture.
	// They ride along here so the sheet does not have to open data/essences.json a second time.
	Family string
	Draw   string

	// Element is the new color, and is only meaningful for TargetElement.
	Element combat.Element

	// Form is the axis a card is told to count on, and is only meaningful for TargetForm.
	Form combat.Form

	// Number is the value read against a numeric target: a signed delta for `cost`, a percentage
	// for `amount`. Meaningless for the rest, which are refused if they carry one.
	Number int
}

// essences is the validated catalog, built once at package init.
//
// **A bad record panics at init**, so it fails on launch rather than the first time a player wins
// a fight — the same severity a bad card record takes, and for the same reason: an essence that does
// nothing is a reward that silently is not one.
var essences, essenceOrder = loadEssences()

// Essences is every essence in the catalog, in a fixed sorted order.
func Essences() []Essence {
	out := make([]Essence, 0, len(essenceOrder))
	for _, key := range essenceOrder {
		out = append(out, essences[key])
	}
	return out
}

// EssenceByKey finds one by its record key.
func EssenceByKey(key string) (Essence, bool) {
	w, ok := essences[key]
	return w, ok
}

func loadEssences() (map[string]Essence, []string) {
	recs := data.LoadEssences()

	out := make(map[string]Essence, len(recs))
	for _, key := range data.EssenceOrder(recs) {
		w, err := resolveEssence(recs[key])
		if err != nil {
			panic("essences.json: " + err.Error())
		}
		out[key] = w
	}

	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) < 2 {
		// The offer is two essences, so a catalog of one cannot fill it. Caught here rather than
		// producing a screen with a gap in it.
		panic(fmt.Sprintf("essences.json: %d essences, and an offer needs two", len(keys)))
	}
	return out, keys
}

// resolveEssence turns a record into an essence, or says why it cannot.
//
// **It refuses a value on a target that takes none**, rather than ignoring it. A `remove` essence
// carrying `"Value": "fire"` is somebody expecting something the mechanic does not do, and
// accepting it silently is how a catalog comes to disagree with the game.
func resolveEssence(r data.EssenceData) (Essence, error) {
	if r.EssenceRecord == "" {
		return Essence{}, fmt.Errorf("an essence has no record key")
	}
	if r.Name == "" {
		return Essence{}, fmt.Errorf("%s has no name", r.EssenceRecord)
	}
	if r.Text == "" {
		// The card is a name and a line of text and nothing else — there is no glyph and no
		// figure — so an essence with no text is a card that does not say what it does.
		return Essence{}, fmt.Errorf("%s has no text, so its card says nothing", r.EssenceRecord)
	}

	target, ok := ParseEssenceTarget(r.Target)
	if !ok {
		return Essence{}, fmt.Errorf("%s names target %q, which is not one of %s",
			r.EssenceRecord, r.Target, targetList())
	}

	w := Essence{Record: r.EssenceRecord, Name: r.Name, Text: r.Text, Target: target,
		Art: r.ArtKey(), Family: r.Family, Draw: r.Draw}

	switch target {
	case TargetCost:
		n, err := strconv.Atoi(r.Value)
		if err != nil {
			return Essence{}, fmt.Errorf("%s targets cost and its value %q is not a number",
				r.EssenceRecord, r.Value)
		}
		if n == 0 {
			return Essence{}, fmt.Errorf("%s changes a cost by nothing", r.EssenceRecord)
		}
		w.Number = n
		return w, nil

	case TargetAmount:
		n, err := strconv.Atoi(r.Value)
		if err != nil {
			return Essence{}, fmt.Errorf("%s targets amount and its value %q is not a percentage",
				r.EssenceRecord, r.Value)
		}
		if n <= 0 {
			return Essence{}, fmt.Errorf("%s scales an amount to %d%%, which is nothing at all",
				r.EssenceRecord, n)
		}
		if n == 100 {
			return Essence{}, fmt.Errorf("%s scales an amount to 100%%, which changes nothing",
				r.EssenceRecord)
		}
		w.Number = n
		return w, nil
	}

	if target == TargetForm {
		f, ok := combat.ParseForm(r.Value)
		if !ok {
			return Essence{}, fmt.Errorf("%s names form %q, which the rules do not have",
				r.EssenceRecord, r.Value)
		}
		if f == combat.FormNone {
			// A card counting on no axis is a card that can never join a form hand, which is a
			// essence that takes something away rather than giving it.
			return Essence{}, fmt.Errorf("%s takes a card's form away", r.EssenceRecord)
		}
		w.Form = f
		return w, nil
	}

	if target == TargetElement {
		e, ok := combat.ParseElement(r.Value)
		if !ok {
			return Essence{}, fmt.Errorf("%s names element %q, which the rules do not have",
				r.EssenceRecord, r.Value)
		}
		if e == combat.Basic {
			// An essence that grayed a card out would be a way to *lose* a color rather than choose
			// one, and no card in the player's deck is drab — the defenses stopped being the
			// exception on 2026-08-23.
			return Essence{}, fmt.Errorf("%s turns a card basic, which takes a color away", r.EssenceRecord)
		}
		w.Element = e
		return w, nil
	}

	if r.Value != "" {
		return Essence{}, fmt.Errorf("%s targets %s and carries the value %q, which nothing reads",
			r.EssenceRecord, target, r.Value)
	}
	return w, nil
}

func targetList() string {
	out := ""
	for i, t := range EssenceTargets() {
		if i > 0 {
			out += ", "
		}
		out += t.String()
	}
	return out
}

// Apply performs an essence on one card of the run, by its position in the deck.
//
// **The one place the deck is altered by an essence**, so there is one place that can get it wrong.
// It reports whether anything happened: an index the deck does not hold is refused rather than
// silently landing on a neighbor, which matters because the offer hands out positions and the
// deck thins under them.
func (s *Session) Apply(w Essence, i int) bool {
	card, ok := s.Card(i)
	if !ok {
		return false
	}

	switch w.Target {
	case TargetRemove:
		return s.Remove(i)

	case TargetDuplicate:
		s.Add(card)
		return true

	case TargetCost:
		card.CostDelta += w.Number
		s.deck[i] = card
		return true

	case TargetAmount:
		// **Percentages compound rather than replace.** A card scaled twice by 150 is at 225, not
		// back at 150, so a second essence on the same card is worth something — and `Card.Amount`
		// does the clamping, so a Defend walked up repeatedly stops at its ceiling instead of
		// being refused.
		if card.AmountPct == 0 {
			card.AmountPct = 100
		}
		card.AmountPct = card.AmountPct * w.Number / 100
		s.deck[i] = card
		return true

	case TargetForm:
		card.FormOverride = w.Form
		s.deck[i] = card
		return true

	case TargetPromote, TargetDemote:
		step := 1
		if w.Target == TargetDemote {
			step = -1
		}
		next, ok := combat.NeighborWrapping(card.Concept, step)
		if !ok {
			return false
		}
		card.Concept = next
		s.deck[i] = card
		return true

	default:
		return s.SetElement(i, w.Element)
	}
}

// CanApply reports whether this essence can be spent on this card at all.
//
// **A pick that would change nothing is legal, and the burden is the player's** *(owner's call,
// 2026-09-19)*. Painting a lightning card lightning is a wasted essence, and it is wasted by a
// player who chose it over the other cards in their deck — where a refusal is a card sitting dead
// under the cursor with the screen declining to say why. The ladder wraps for the same reason, so
// there is no rung an Exalt or a Debase cannot reach.
//
// What is still refused is a card that is not there: an index off the end of the deck is a caller
// bug, not a choice.
func (s *Session) CanApply(w Essence, i int) bool {
	card, ok := s.Card(i)
	if !ok {
		return false
	}

	switch w.Target {
	case TargetPromote:
		_, ok := combat.NeighborWrapping(card.Concept, 1)
		return ok
	case TargetDemote:
		_, ok := combat.NeighborWrapping(card.Concept, -1)
		return ok
	default:
		return true
	}
}

// The satchel: essences the run is carrying, unspent.
//
// **An essence is a consumable now** *(owner's call, 2026-09-19)*. It is still what a won fight
// offers and still what a vial sells, and taking one there still spends it on a card there — but an
// essence that goes into the satchel instead is carried into a duel and aimed at a card in the hand,
// out of the same pane a rune and a stone are spent from.
//
// **Why it can be**: an essence edits the run's deck, and so does a rune. `resyncHandFromRun` is
// what makes either legible mid-fight, and the gate is `planning()` for the reason every consumable
// is under — `ResolveRound` decides a whole round before a frame of it is drawn, so a card altered
// during playback would show a face disagreeing with a blow already computed.
//
// **There is no cap, unlike the sack.** `MaxHeld` exists because the consumables pane draws `held/2`
// and a fraction has to be a rule; the satchel is counted with the pouch, which has never had one.

// StartingEssences is what a run opens carrying in its satchel, by record key.
//
// **Empty as shipped, and it is a debug seat** — the counterpart of StartingRunes and
// StartingStones, written only by `internal/scenario`, which is compiled out of every normal build.
var StartingEssences []string

// Stow puts an essence in the satchel, and reports whether the catalog held it.
//
// **An essence the catalog does not have is refused** rather than carried as a key nothing can
// resolve, which is the posture `Hold` and `Carry` both take: a seat that cannot be resolved is a
// seat the player cannot spend.
func (s *Session) Stow(key string) bool {
	if _, ok := essences[key]; !ok {
		return false
	}
	s.satchel = append(s.satchel, key)
	return true
}

// Stowed is every essence the run is carrying, by record key, in the order they were acquired.
func (s *Session) Stowed() []string {
	out := make([]string, len(s.satchel))
	copy(out, s.satchel)
	return out
}

// StowCount is how many essences are in the satchel.
func (s *Session) StowCount() int { return len(s.satchel) }

// DropStowed takes one out of the satchel by position, and reports whether it was there.
//
// **By position rather than by key**, because the satchel may hold two of the same essence and
// spending one must not be ambiguous about which — the rule a rune's sack seat and a stone's pouch
// position are both under.
//
// **Spending is ApplyTo plus DropStowed, and they are separate on purpose**, exactly as a rune's
// spend is Drop plus ApplyRune: the card is picked before the essence is clicked and the picker can
// be backed out of, so dropping first would charge for a choice that was never made.
func (s *Session) DropStowed(i int) bool {
	if i < 0 || i >= len(s.satchel) {
		return false
	}
	s.satchel = append(s.satchel[:i], s.satchel[i+1:]...)
	return true
}

// CanApplyTo reports whether this essence can be spent on the card with this identity.
//
// **Aimed by identity rather than by deck position, because a fight holds copies.** A card in the
// hand was dealt off the deck and three piles hold cards that look alike; `combat.Card.ID` is the
// handle that says which one, which is the argument `Session.CanApplyRune` is already under.
func (s *Session) CanApplyTo(w Essence, id int) bool {
	i, ok := s.positionOf(id)
	if !ok {
		return false
	}
	return s.CanApply(w, i)
}

// ApplyTo performs an essence against the card with this identity. It reports whether it fired.
//
// **What it mints is handed over the way a rune's copy is**, through `Session.Duplicated`: an
// essence spent between fights only has to put the copy in the deck, and one spent in the middle of
// a duel has to reach the hand it was aimed at or it reads as a dud. The handover is emptied here
// rather than by the reader, so it only ever says what the last consumable did.
func (s *Session) ApplyTo(w Essence, id int) bool {
	i, ok := s.positionOf(id)
	if !ok {
		return false
	}

	s.duplicated = s.duplicated[:0]
	before := len(s.deck)
	if !s.Apply(w, i) {
		return false
	}
	for j := before; j < len(s.deck); j++ {
		s.duplicated = append(s.duplicated, s.deck[j])
	}
	return true
}

// positionOf turns one identity into a deck position.
func (s *Session) positionOf(id int) (int, bool) {
	for i, c := range s.deck {
		if c.ID == id {
			return i, true
		}
	}
	return 0, false
}

// EssenceTargets is how many cards one essence is spent on: **one, and whatever the relics make of
// it** — the `essence-spent` moment. See combat.EssenceTargets, which is where the compounding and
// the floor of one live.
//
// **The one seat all three spend sites ask through**: the reward screen's offer, the shop's vial and
// a satchel essence aimed at the hand. Three screens reading three counts is three screens that can
// disagree about what the player was promised.
func (s *Session) EssenceTargets() int {
	return combat.EssenceTargets(s.WornRelics())
}

// ApplyToAll performs one essence against every card named, and reports whether it fired.
//
// **All or nothing, the rule ApplyRune is under**: every card is checked before any of them is
// changed, so an essence that lit up cannot land on two cards and refuse the third. A consumable
// half spent is a consumable the player paid for and did not get.
//
// **The deck is walked from the back**, which is what makes several targets safe in one call: a
// removal shifts every position above it and leaves everything below it alone, so a descending walk
// never aims at a card that has moved. Identities are resolved to positions up front for the same
// reason a rune resolves them — three piles hold copies of the same card.
//
// **What it mints is accumulated rather than replaced.** Session.Apply appends a duplicate to the
// end of the deck, and a caller wanting the copies has to be handed all of them: Duplicated is
// emptied here once, at the top, rather than once per card.
func (s *Session) ApplyToAll(w Essence, ids []int) bool {
	if len(ids) == 0 {
		return false
	}

	seen := make(map[int]bool, len(ids))
	at := make([]int, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			// **A card named twice is refused rather than changed twice.** The row a selection is
			// made in cannot produce one, so this is a caller bug, and changing one card for two
			// would be an essence quietly doing half of what the player was shown.
			return false
		}
		seen[id] = true
		i, ok := s.positionOf(id)
		if !ok || !s.CanApply(w, i) {
			return false
		}
		at = append(at, i)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(at)))

	s.duplicated = s.duplicated[:0]
	for _, i := range at {
		before := len(s.deck)
		if !s.Apply(w, i) {
			return false
		}
		for j := before; j < len(s.deck); j++ {
			s.duplicated = append(s.duplicated, s.deck[j])
		}
	}
	return true
}

// CanApplyToAll reports whether one essence can be spent on every one of these cards at once.
//
// **The question the lit state asks and the question ApplyToAll asks**, so a consumable drawn lit
// cannot then be refused — the rule consumableTarget is built around.
func (s *Session) CanApplyToAll(w Essence, ids []int) bool {
	if len(ids) == 0 {
		return false
	}
	seen := make(map[int]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return false
		}
		seen[id] = true
		i, ok := s.positionOf(id)
		if !ok || !s.CanApply(w, i) {
			return false
		}
	}
	return true
}

// What an essence says about itself when it reaches more than one card.
//
// **The authored line is written for one card and the wider ones are derived from it** *(owner's
// call, 2026-09-19)*. The alternative was a plural string beside `Text` on every record, and it is
// not one string but one per count — a relic may put an essence on two cards or on five — so the
// catalog would be carrying a sentence per reach for fifteen records that all say the same thing a
// different way. What is authored is the essence; what is derived is the arithmetic on top of it.
//
// **The shapes are closed, and a line that fits none of them is left alone.** Every record today is
// either `CARD <verb> ...` or `<verb> CARD`, so those are the two rewrites; an author who writes a
// third shape gets their sentence printed as they wrote it rather than mangled, which is the
// failure worth choosing between.

// essencePlurals is the verb agreement the first rewrite needs: a sentence about one card says
// BECOMES and a sentence about two says BECOME.
//
// **A closed table rather than a rule**, because English does not have one that is safe on three
// words — and three is what the catalog uses.
var essencePlurals = map[string]string{
	"BECOMES": "BECOME",
	"GAINS":   "GAIN",
	"LOSES":   "LOSE",
}

// TextAt is what this essence says when it is spent on this many cards.
//
// **One card is the authored line, unchanged.** That is the shipped reach and the reach every
// review sheet shows, so nothing rewrites anything until a relic has moved the number.
func (w Essence) TextAt(targets int) string {
	if targets <= 1 {
		return w.Text
	}

	lines := strings.Split(w.Text, "\n")
	for i, line := range lines {
		lines[i] = w.lineAt(line, targets)
	}
	return strings.Join(lines, "\n")
}

// lineAt rewrites one authored line for a wider reach.
func (w Essence) lineAt(line string, targets int) string {
	// `CARD BECOMES FIRE` is about the cards, so the count leads and the verb agrees with it.
	if rest, ok := strings.CutPrefix(line, "CARD "); ok {
		words := strings.SplitN(rest, " ", 2)
		if plural, ok := essencePlurals[words[0]]; ok {
			words[0] = plural
		}
		return fmt.Sprintf("%d CARDS %s", targets, strings.Join(words, " "))
	}

	// `DESTROY CARD` is about the doing, so the count lands on what is being done to.
	if verb, ok := strings.CutSuffix(line, " CARD"); ok {
		// **A copy is counted in times rather than in cards** *(owner's call, 2026-09-19)*.
		if w.Target == TargetDuplicate {
			return verb + " CARD " + essenceTimes(targets)
		}
		return fmt.Sprintf("%s %d CARDS", verb, targets)
	}

	return line
}

// essenceTimes is how often something happens, said the way a player would say it.
func essenceTimes(n int) string {
	if n == 2 {
		return "TWICE"
	}
	return fmt.Sprintf("%d TIMES", n)
}
