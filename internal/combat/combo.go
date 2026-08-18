package combat

import "sort"

// Combos are the layer where a round stops being a pile of cards and starts being a plan.
// Throwing whatever you drew at the opponent works; *choosing* a shape and building toward it is
// meant to work better, and this file is the machinery that pays for that choice.
//
// **A turn produces exactly one attack** *(2026-08-14)*. The attack phase reads every attack card
// queued, forms the best hand it can, and resolves a single blow — so five Strikes are not five
// hits, they are one Four of a Kind. Attack cards that do not contribute are ignored outright:
// `Strike, Jab, Strike` is a Pair and the Jab is not in it.
//
// **A hand counts copies of a card, and buys damage and nothing else** *(2026-08-17)*. The ladder
// wears poker's names because it is poker's question — high card, pair, two pair, three of a kind,
// full house, four of a kind — and the whole of what forming one does is multiply the blow.
//
// **The multiplier multiplies the hand's own cards** *(2026-08-18, owner's call)*. A Pair of Lunges
// is `(20 + 20) x 1.5`, so what a hand is worth is a proportion of what its cards deal. It used to
// be applied to a separate reference swing of one 1x attack at the attacker's DMG, added on top of
// the cards, which made a percent buy a *fixed* figure: 500% was worth 2.5x the base on Jabs and
// 0.6x on Lunges, so the ladder paid least to the decks that had climbed furthest. The High Card
// carries 100 for the same reason, and it is what makes a lone attack land its own face damage.
//
// **That is deliberately narrow.** Combos used to carry a second axis counting the distinct colours
// in the formed hand, and a reward vocabulary that could bank action points or take actions off the
// opponent's next turn. Both are gone: statuses come from **elements and the rings that arm them**,
// so a combo is one number and there is exactly one place to look for what a hand is worth.
//
// **Exactly one hand applies.** A hand wins on its multiplier — four Strikes are a Four of a Kind
// rather than also the pair and the trips inside it — so a turn produces one combo with no ranking
// machinery beyond that comparison.
//
// **Matching is on the resolved order, never on the queue.** ResolutionOrder regroups a queue by
// category, so the cards you dragged into place are not the cards in the order they happen. It
// matters little now that hands are counted rather than ordered, but the turn the matcher reads is
// the resolved one.
//
// **Matching is on cards used, never on what they achieved.** A hand is known before the blow is
// worked out, which is what lets its multiplier apply to the cards that formed it. Matching on
// damage dealt would mean a plan could be silently invalidated by the opponent's defenses after
// the player had committed to it.
//
// Combos are eventually **discovered rather than given**, persisting on the profile as part of
// the unlock structure — see MECHANICS.md. No profile exists yet, so everything in the catalogue
// is always live. When one does, discovery gates the *catalogue*, not the matcher.

// HandID identifies a hand. It travels on the KindCombo event so the screen can name what fired
// without knowing the rule that fired it.
type HandID int

// HandNone is the zero value and means "no hand". Every real ID is written in combos.json.
const HandNone HandID = 0

// multiplierScale is the denominator the percent multipliers in combos.json are held over. They
// are integers because this package is integer arithmetic throughout, and a float in the file
// would be the one number in the game that rounds differently from every other.
const multiplierScale = 100

// Hand is one rung of the ladder, after combos.json has been read.
type Hand struct {
	ID   HandID
	Key  string
	Name string

	// Groups is how many cards of each *distinct* concept the hand wants. `[3,2]` is a full
	// house; the groups naming distinct values is why five of one card can never be one.
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

// catalogue is every hand in the game, read from data/combos.json at package init.
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
// against profile discovery: a combo ID no longer derives from a concept's position, so reordering
// the cards cannot renumber a combo the player has already found.
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
	// group, or the High Card itself. It is what the combo event reports as the card the blow
	// led with.
	Lead int

	// Hand is what formed. A lone attack that forms no hand carries HandNone.
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

// Formed reports whether a hand of two or more cards was made, as opposed to the High Card
// falling through.
//
// **The High Card is a hand and still answers false here** *(2026-08-15)*. It has a name, an ID
// and a line in the catalogue, so the feed can say what fired; what it does not have is anything
// a player *built*, and this is the question the screen's combo preview asks — whether the cards
// currently queued amount to more than the best of them. The combo dialog reads it too, to decide
// whether there is a hand to shout.
func (b Blow) Formed() bool { return b.Hand.ID != HandNone && b.Hand.Cards() > 1 }

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

// highCardKey is the catalogue entry naming the fallback. **It is in `combos.json` rather than
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
		if !found || h.Multiplier > best.Multiplier {
			best, bestCards, bestLead, found = h, cards, lead, true
		}
	}
	return bestCards, best, bestLead, found
}

// matchCountOf reads the turn as a set: how many cards carry each concept, and whether that
// satisfies the hand's groups.
//
// **Only the cards that form a blow are counted, and that is the matcher's rule rather than the
// catalogue's** *(2026-08-17)*. An entry used to name the categories it counted, but `formsBlow`
// is strictly narrower than "is an attack" — it also excludes recoil — so the field could never
// change what was counted and only invited an entry to claim otherwise.
//
// **Groups are filled largest-count-first**, and a tie goes to the concept registered first. A
// full house asked for `[3,2]` against three Jabs and two Strikes has only one reading, but the
// rule has to be written down for the cases that do not — and it has to be a rule rather than a
// map walk, per the determinism note in CLAUDE.md.
//
// **It tallies the turn rather than indexing a fixed-width array** *(2026-08-16)*. The array was
// `len(AllActions)` wide, which is a number that stopped existing when concepts became data: a
// registry holding every enemy's cards is hundreds wide and grows as decks are built, and a turn
// is at most five cards. Tallying what is actually in the turn, in the order it was played, is
// both smaller and deterministic without depending on how many enemies happen to be loaded.
//
// The second return is the turn index of the first card of the *first* group — the concept the
// hand is named after, and what the combo event reports as the card the blow led with.
func matchCountOf(turn []Slot, h Hand) ([]int, int, bool) {
	type tally struct {
		id      ConceptID
		members []int
		spent   bool
	}

	var tallies []tally
	for i, s := range turn {
		if !s.Card.formsBlow() {
			continue
		}
		at := -1
		for j := range tallies {
			if tallies[j].id == s.Card.Concept {
				at = j
				break
			}
		}
		if at < 0 {
			tallies = append(tallies, tally{id: s.Card.Concept})
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
// blow is. The player's three families ladder identically — a Lunge, a Cleave and a Smash all deal
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

// formsBlow reports whether this card contributes to its side's one attack.
//
// **Recoil is an attack that does not** *(2026-08-16)*. It resolves in the attack phase and is
// announced like any other card, but it is aimed at its own owner — so counting it toward a hand
// would let a duelist earn a Four of a Kind by hurting themselves four times.
func (c Card) formsBlow() bool {
	s := c.Spec()
	return s.Verb == VerbAttack && s.Target == TargetOpponent
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
