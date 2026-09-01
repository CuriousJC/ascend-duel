package combat

import "sort"

// Hands are the layer where a round stops being a pile of cards and starts being a plan.
// Throwing whatever you drew at the opponent works; *choosing* a shape and building toward it is
// meant to work better, and this file is the machinery that pays for that choice.
//
// **A turn produces exactly one attack** *(2026-08-14)*. The attack phase reads every attack card
// queued, forms the best hand it can, and resolves a single blow — so five Strikes are not five
// hits, they are one Four of a Kind. Attack cards that do not contribute are ignored outright:
// `Strike, Jab, Strike` is a Pair and the Jab is not in it.
//
// **A hand counts cards that agree, and buys damage and nothing else** *(2026-08-17)*. The ladder
// wears poker's names because it is poker's question — high card, pair, two pair, three of a kind,
// full house, four of a kind — and the whole of what forming one does is multiply the blow.
//
// **A hand is what you played, not what you hit with** *(owner's call, 2026-08-23)*. Defend cards
// carry an element, so they are counted like anything else: two Wards are a Card Pair, and a turn of
// four fire cards is an Elemental Four of a Kind whether any of them swung. They bring no damage
// into the sum, since a defend card's `Damage` is zero — so a hand of nothing but shields multiplies
// nothing and lands nothing, and a Ward beside two attacks raises the rung the two attacks are paid
// at.
// That last case is the whole of what the change buys, and the whole of what it costs.
//
// **The colours a hand shows include its defences**, so a fire Ward arms a burn on a turn with no
// fire attack in it. That follows from the same decision and is the sharper half of it.
//
// **What "agree" means is the hand's own business** *(2026-08-19)*. Every rung exists three times
// over, once per `Axis`: two Bashes are a Card Pair, a Bash and a Cleave are a Form Pair only if
// they share a form — they do not — and an ice Bash beside an ice Thrust is an Elemental Pair
// though the two agree on nothing else. The three are separate catalogue entries rather than one
// entry with three readings, so each is priced on how often it can actually be built.
//
// **The multiplier multiplies the hand's own cards** *(2026-08-18, owner's call)*. A Pair of Lunges
// is `(20 + 20) x 1.5`, so what a hand is worth is a proportion of what its cards deal. It used to
// be applied to a separate reference swing of one 1x attack at the attacker's DMG, added on top of
// the cards, which made a percent buy a *fixed* figure: 500% was worth 2.5x the base on Jabs and
// 0.6x on Lunges, so the ladder paid least to the decks that had climbed furthest. The High Card
// carries 100 for the same reason, and it is what makes a lone attack land its own face damage.
//
// **That is deliberately narrow.** Hands used to carry a second axis counting the distinct colours
// in the formed hand, and a reward vocabulary that could bank action points or take actions off the
// opponent's next turn. Both are gone: statuses come from **elements and the rings that arm them**,
// so a hand is one number and there is exactly one place to look for what a hand is worth.
//
// **Exactly one hand applies.** A hand wins on its multiplier — four Strikes are a Four of a Kind
// rather than also the pair and the trips inside it — so a turn produces one hand with no ranking
// machinery beyond that comparison.
//
// **Matching is on the resolved order, never on the queue.** ResolutionOrder regroups a queue by
// category, so the cards you dragged into place are not the cards in the order they happen. It
// matters little now that hands are counted rather than ordered, but the turn the matcher reads is
// the resolved one.
//
// **Matching is on cards used, never on what they achieved.** A hand is known before the blow is
// worked out, which is what lets its multiplier apply to the cards that formed it. Matching on
// damage dealt would mean a hand could be silently invalidated by the opponent's defenses after
// the player had committed to it.
//
// Hands are eventually **discovered rather than given**, persisting on the profile as part of
// the unlock structure — see MECHANICS.md. **The profile exists as of 2026-08-25 and does not gate
// this**: `profile.Profile.HandsDiscovered` is the field waiting for it, and everything in the
// catalogue is still always live. When it does, discovery gates the *catalogue*, not the matcher.

// HandID identifies a hand. It travels on the KindHand event so the screen can name what fired
// without knowing the rule that fired it.
type HandID int

// HandNone is the zero value and means "no hand". Every real ID is written in hands.json.
const HandNone HandID = 0

// multiplierScale is the denominator the percent multipliers in hands.json are held over. They
// are integers because this package is integer arithmetic throughout, and a float in the file
// would be the one number in the game that rounds differently from every other.
const multiplierScale = 100

// Axis is what a hand counts copies *of* *(2026-08-19)*. The same rung exists once per axis —
// three Bashes are a Card Three of a Kind, three crushes are a Form Three of a Kind, three ice
// cards are an Elemental Three of a Kind — so each can be priced on its own rarity.
//
// **The order is the tie-break, narrowest first, and that is a rule rather than an accident.**
// A concept fixes a form, so every card hand is also a form hand and two of them can be live at
// the same multiplier; the narrower one is what the player aimed at, so it wins. Element is
// independent of both — an ice Bash and a fire Bash are a card hand and not an elemental one.
//
// It is safe to order this enum meaningfully because **an axis is never serialized**: hands.json
// writes `"match": "form"` and `ParseAxis` resolves it, exactly as elements and forms are named
// rather than numbered. Reordering would change the tie-break and nothing else.
type Axis int

const (
	// AxisConcept counts copies of the same card. It is the narrowest axis: four cards can share
	// a concept only by being its four elemental copies.
	AxisConcept Axis = iota

	// AxisForm counts cards of the same form — stab, slash, crush or defend. `FormNone` never
	// counts, so an enemy's formless deck cannot build one.
	//
	// **Defend is a fourth form as of 2026-08-23**, since those cards join hands too. Twelve of the
	// player's forty-eight cards share it, which makes it the commonest value on this axis.
	AxisForm

	// AxisElement counts cards of the same colour. `Basic` never counts, for the same reason.
	AxisElement
)

// axisNames are the strings hands.json writes. Index by Axis.
var axisNames = [...]string{
	AxisConcept: "concept",
	AxisForm:    "form",
	AxisElement: "element",
}

// AllAxes is every axis, in tie-break order. It exists so a test and a reference screen can walk
// them without knowing how many there are.
var AllAxes = []Axis{AxisConcept, AxisForm, AxisElement}

func (a Axis) String() string {
	if int(a) < 0 || int(a) >= len(axisNames) {
		return "unknown"
	}
	return axisNames[a]
}

// ParseAxis resolves the axis names written in hands.json. It reports failure rather than falling
// back to a default, for the reason ParseElement does: a hand quietly counting the wrong thing is
// a balance change nobody made.
func ParseAxis(name string) (Axis, bool) {
	for i, n := range axisNames {
		if n == name {
			return Axis(i), true
		}
	}
	return AxisConcept, false
}

// matchValue is what one card counts as on this axis, and whether it counts at all.
//
// **`FormNone` and `Basic` are absences rather than values** *(2026-08-19)*, so a card carrying
// one matches nothing on that axis. Every enemy card is both, which is what stops a formless,
// colourless deck from reading as a table full of elemental hands.
//
// **The player has no basic card left** *(2026-08-23)*. The defences used to be the exception and
// were excluded before this was asked anyway; they now ship in the five colours like every attack,
// so `FormDefend` and every element are live values here and the absences belong to the enemies.
func matchValue(c Card, a Axis) (int, bool) {
	switch a {
	case AxisForm:
		f := c.Form()
		return int(f), f != FormNone
	case AxisElement:
		e := c.Element
		return int(e), e != Basic
	default:
		return int(c.Concept), true
	}
}

// spread is how many different values a hand can spread its groups across on this axis, or 0 for
// an axis wide enough that the question cannot bite.
//
// It is what refuses a hand asking for more distinct values than the axis has. **Every form now
// reaches a blow** *(2026-08-23)* — `FormDefend` used to be filtered out before the matcher saw it,
// which is why this was `len(Forms()) - 1` — so the form axis is as wide as the enum. Concepts are
// hundreds wide once the enemy decks are registered and a turn holds five cards, so that axis is
// left unchecked rather than tied to a registry that grows.
func (a Axis) spread() int {
	switch a {
	case AxisForm:
		return len(Forms())
	case AxisElement:
		return ElementCount - 1
	default:
		return 0
	}
}

// Hand is one rung of the ladder, after hands.json has been read.
type Hand struct {
	ID   HandID
	Key  string
	Name string

	// Match is the axis this hand counts on.
	Match Axis

	// Groups is how many cards of each *distinct value on the hand's own axis* the hand wants.
	// `[3,2]` is a full house; the groups naming distinct values is why five cards sharing one
	// value can never be one.
	Groups []int

	// Multiplier is this hand's damage multiplier, in percent, and is the whole of what forming
	// it buys.
	Multiplier int
}

// Cards is how many cards this hand is formed from.
func (h Hand) Cards() int {
	n := 0
	for _, g := range h.Groups {
		n += g
	}
	return n
}

// catalogue is every hand in the game, read from data/hands.json at package init.
var handTable = loadCatalogue()

// Hands is the live catalogue. It exists so a reference screen can list the ladder without
// reaching into the table, and so the discovery gate has one place to land.
func Hands() []Hand {
	out := make([]Hand, len(handTable))
	copy(out, handTable)
	return out
}

// HandByID looks one up for narration. The bool reports whether it was found rather than
// returning a zero value that would silently narrate as an unnamed effect.
func HandByID(id HandID) (Hand, bool) {
	for _, h := range handTable {
		if h.ID == id {
			return h, true
		}
	}
	return Hand{}, false
}

// HandByName finds one by the name it carries on screen. For tests and for a reference screen;
// nothing in the rules looks one up this way.
func HandByName(name string) (Hand, bool) {
	for _, h := range handTable {
		if h.Name == name {
			return h, true
		}
	}
	return Hand{}, false
}

// HandIDForKey is the number the catalogue gives one entry.
//
// **One ID per key, written in the file.** An entry used to produce one hand *per attack concept*
// with `base + int(concept)` for an ID, which held twelve concepts and could not hold the four
// hundred a per-enemy deck list produces. It also closes the open question MECHANICS.md recorded
// against profile discovery: a hand ID no longer derives from a concept's position, so reordering
// the cards cannot renumber a hand the player has already found.
func HandIDForKey(key string) (HandID, bool) {
	for _, h := range handTable {
		if h.Key == key {
			return h.ID, true
		}
	}
	return HandNone, false
}

// Blow is what one side's attack phase amounts to: which cards were spent on it, what hand they
// formed, and the multiplier that follows.
//
// **A turn has at most one of these.** It is the whole of the attack phase — there is no second
// blow behind it, however many attack cards were queued.
type Blow struct {
	// Cards are indices into the side's own resolved turn, naming the cards that formed the
	// hand. Attack cards that contributed nothing are not here.
	//
	// **A list rather than a start and a length**, because a counted hand is not contiguous: Two
	// Pair can be two cards, a card that earned nothing, and two more.
	Cards []int

	// Lead is the turn index of the card the hand is named after: the first card of its first
	// group, or the High Card itself. It is what the hand event reports as the card the blow
	// led with.
	Lead int

	// Hand is what formed, and **it is always a hand**: a turn that builds nothing bigger falls
	// back to the catalogue's High Card. A blow with no cards in it at all — a turn with no attack
	// — is the zero Blow, and `len(Cards) == 0` is how a caller asks that.
	Hand Hand

	// Multiplier is the hand's, in percent, and exists as its own field because the blow is what
	// the resolver and the feed both read — neither should have to know the multiplier has only
	// one source today. 100 is the identity: it is what the High Card carries.
	Multiplier int

	// Elements is every distinct non-basic colour in the hand, in element order. It is what
	// decides which statuses land, and it is the *only* thing colour does to a blow — it buys no
	// damage.
	Elements []Element
}

// BlowFor works out one side's attack phase from the cards it resolved.
//
// **The best hand wins, and best means the biggest multiplier.** Four Strikes hold a pair and
// trips as well as a four of a kind; the four of a kind is worth the most, so it is the hand, and
// nothing else pays.
//
// **When no hand of two or more forms, the High Card is the blow**: the single attack that hits
// hardest, at the catalogue's identity multiplier, so what lands is the card's own face damage.
// Ties go to the card queued first, which needs no tie-break rule beyond the order the turn is
// already in.
func BlowFor(turn []Slot) Blow {
	return blowFor(turn, handTable)
}

func blowFor(turn []Slot, hands []Hand) Blow {
	cards, hand, lead, formed := matchHand(turn, hands)
	if !formed {
		cards, hand = biggestAttack(turn), highCard(hands)
		if len(cards) > 0 {
			lead = cards[0]
		}
	}
	if len(cards) == 0 {
		return Blow{}
	}

	return Blow{
		Cards:      cards,
		Lead:       lead,
		Hand:       hand,
		Multiplier: hand.Multiplier,
		Elements:   elementsOf(turn, cards),
	}
}

// highCardKey is the catalogue entry naming the fallback. **It is in `hands.json` rather than
// written out here** so the one thing every turn can produce is named and numbered where the rest
// of the ladder is, and the feed can look it up like any other hand.
const highCardKey = "high-card"

// highCard is the catalogue's fallback entry. It is required to exist — loadCatalogue panics
// without it — because a turn with an attack in it always produces a hand, and one the engine
// could not name is the single failure this model can have.
func highCard(hands []Hand) Hand {
	for _, h := range hands {
		if h.Key == highCardKey {
			return h
		}
	}
	return Hand{}
}

// matchHand finds the best-paying hand of **two or more cards** the turn can form.
//
// **Best is the biggest multiplier, and a tie goes to the narrowest axis** *(2026-08-19)*. Two
// Bashes satisfy the card pair and the form pair at once, so the comparison needs a second key or
// it would be decided by file order; `Axis` is written narrowest-first for exactly this.
//
// **The one-card hand is skipped rather than matched** *(2026-08-15)*. The High Card is in the
// catalogue and would match against any attack at all, but counting is the wrong way to pick it:
// `matchCountOf` fills groups largest-count-first, so it would hand back whichever concept
// appeared most rather than the card that hits hardest. Which card is the High Card is a question
// about damage, and `biggestAttack` is what answers it.
func matchHand(turn []Slot, hands []Hand) ([]int, Hand, int, bool) {
	var (
		best      Hand
		bestCards []int
		found     bool
	)

	bestLead := -1
	for _, h := range hands {
		if h.Cards() < 2 {
			continue
		}
		cards, lead, ok := matchCountOf(turn, h)
		if !ok {
			continue
		}
		if !found || h.Multiplier > best.Multiplier ||
			(h.Multiplier == best.Multiplier && h.Match < best.Match) {
			best, bestCards, bestLead, found = h, cards, lead, true
		}
	}
	return bestCards, best, bestLead, found
}

// matchCountOf reads the turn as a set: how many cards carry each value on the hand's own axis,
// and whether that satisfies its groups. A card with no value on that axis — a formless or
// colourless one — is skipped rather than tallied under a zero everything else would join.
//
// **Which cards are counted is the matcher's rule rather than the catalogue's** *(2026-08-17)*. An
// entry used to name the categories it counted, and it could never change what was counted — it
// only invited an entry to claim otherwise.
//
// **It counts every card in the turn** *(2026-08-23)*. Defences are in, so a pair of Wards is a Card
// Pair and a turn of one colour is an elemental hand whether it swung or not; they bring no damage
// with them, since `Card.Damage` is zero for every verb that is not an attack. What is left out is
// decided by `matchValue` — a card with no value on the hand's own axis — and by nothing else.
//
// **Groups are filled largest-count-first**, and a tie goes to the value whose first card was
// played first. A full house asked for `[3,2]` against three Jabs and two Strikes has only one
// reading, but the rule has to be written down for the cases that do not — and it has to be a rule rather than a
// map walk, per the determinism note in CLAUDE.md.
//
// **It tallies the turn rather than indexing a fixed-width array** *(2026-08-16)*. The array was
// `len(AllActions)` wide, which is a number that stopped existing when concepts became data: a
// registry holding every enemy's cards is hundreds wide and grows as decks are built, and a turn
// is at most five cards. Tallying what is actually in the turn, in the order it was played, is
// both smaller and deterministic without depending on how many enemies happen to be loaded.
//
// The second return is the turn index of the first card of the *first* group — the concept the
// hand is named after, and what the hand event reports as the card the blow led with.
func matchCountOf(turn []Slot, h Hand) ([]int, int, bool) {
	type tally struct {
		value   int
		members []int
		spent   bool
	}

	var tallies []tally
	for i, s := range turn {
		v, counts := matchValue(s.Card, h.Match)
		if !counts {
			continue
		}
		at := -1
		for j := range tallies {
			if tallies[j].value == v {
				at = j
				break
			}
		}
		if at < 0 {
			tallies = append(tallies, tally{value: v})
			at = len(tallies) - 1
		}
		tallies[at].members = append(tallies[at].members, i)
	}

	var out []int
	lead := -1
	for _, g := range h.Groups {
		best, bestCount := -1, 0
		for j := range tallies {
			if tallies[j].spent || len(tallies[j].members) < g {
				continue
			}
			// Strictly greater, so a tie goes to the earlier tally — which is the concept whose
			// first card was played first, since tallies are built by walking the turn.
			if len(tallies[j].members) > bestCount {
				best, bestCount = j, len(tallies[j].members)
			}
		}
		if best < 0 {
			return nil, -1, false
		}
		tallies[best].spent = true
		if lead < 0 {
			lead = tallies[best].members[0]
		}
		out = append(out, tallies[best].members[:g]...)
	}

	// The cards are collected group by group, so a full house arrives as its three and then its
	// two. Sorting puts them back into the order they resolve, which is what a bracket wants.
	sort.Ints(out)
	return out, lead, true
}

// biggestAttack is the High Card: the single attack that hits hardest, or nothing if the turn
// queued no attacks at all.
//
// **It is compared on the concept's damage rather than its cost**, because damage is what the
// blow is. The player's three forms ladder identically — a Lunge, a Cleave and a Smash all deal
// double — so ties are common rather than exceptional, and they go to the card queued first. The
// earliest slot wins, which is deterministic without inventing a rule.
func biggestAttack(turn []Slot) []int {
	best, bestDamage := -1, 0
	for i, s := range turn {
		if !s.Card.formsBlow() {
			continue
		}
		// A fixed reference DMG: this is a comparison between concepts, and every concept
		// scales from the same DMG, so any positive value ranks them identically.
		if d := s.Card.Damage(damageRankDMG); d > bestDamage {
			best, bestDamage = i, d
		}
	}
	if best < 0 {
		return nil
	}
	return []int{best}
}

// formsBlow reports whether this card contributes damage to its side's one attack.
//
// **It is about damage, not about membership** *(2026-08-23)*. Every card a duelist can queue is
// counted toward a hand — see `matchCountOf` — and this is the narrower question `biggestAttack`
// asks: which of them can be the High Card. A defend card cannot, because it deals nothing.
func (c Card) formsBlow() bool {
	return c.Spec().Verb == VerbAttack
}

// damageRankDMG is the DMG `biggestAttack` ranks concepts at. It never reaches a life total
// — it exists only so Heavy sorts above Strike sorts above Jab — and it is deliberately large
// enough that Jab's `dmg/2` floor of 1 cannot flatten the ladder.
const damageRankDMG = 100

// elementsOf is every distinct non-basic colour among the cards that formed the hand, in element
// order.
//
// **Colour buys no damage, only statuses** *(2026-08-17)*. This list is read by the resolver to
// decide what lands, gated on the rings the attacker wears; the count of it used to be a second
// multiplier and is not any more.
//
// **Basic is skipped.** It is the absence of an element, so a basic card neither adds a colour nor
// spoils one — two basic Strikes and an ice Strike show one colour. That is what makes a plain
// draw neutral rather than a punishment.
func elementsOf(turn []Slot, cards []int) []Element {
	var seen [ElementCount]bool
	for _, i := range cards {
		if i < 0 || i >= len(turn) {
			continue
		}
		if e := turn[i].Card.Element; e != Basic {
			seen[e] = true
		}
	}

	var out []Element
	for _, e := range AllElements {
		if e != Basic && seen[e] {
			out = append(out, e)
		}
	}
	return out
}

// scaleDamage applies a turn's multiplier to a base figure — since 2026-08-18 that figure is the
// sum of the hand's own cards, so this is the whole of the blow rather than a bonus term.
//
// Rounding is deliberately toward zero, matching guardDivisor and the defend reductions: the
// package is integer arithmetic throughout, so a multiplier that rounded the other way would be
// the one rule a player could not predict from the others. A hand at an odd percent can therefore
// lose a point off its total, which is why the dialog prints the figures rather than inviting the
// player to multiply them out.
func scaleDamage(base, pct int) int {
	if pct <= 0 {
		return 0
	}
	return base * pct / multiplierScale
}
