package combat

import "sort"

// Combos are the layer where a round stops being a pile of cards and starts being a plan.
// Throwing whatever you drew at the opponent works; *choosing* a shape and building toward it is
// meant to work better, and this file is the machinery that pays for that choice.
//
// **A turn produces exactly one attack** *(2026-08-14)*. The attack phase reads every attack card
// queued, forms the best hand it can, and resolves a single blow — so five Strikes are not five
// hits, they are one Strike Barrage. Attack cards that do not contribute are ignored outright:
// `Strike, Jab, Strike` is a Strike Pair and the Jab is not in it.
//
// **Two independent axes, multiplied at load rather than enumerated.**
//
//   - The **hand** counts copies of a card — pair, two pair, flurry, full house, barrage,
//     barrage. It is what poker counts, and it wears poker's names honestly.
//   - The **mix** counts distinct non-basic *colours* in whatever hand formed — drab, mono, duo,
//     trio, rainbow. Basic is not a colour and never counts in either direction, so two basic
//     Strikes and an ice Strike is mono.
//
// Six hands by five mixes is 27 combos, of which the sizes that cannot hold four colours strike
// a few. Authoring that as a grid would be 27 sets of numbers nobody could keep consistent;
// authoring it as two axes is 11.
//
// **Exactly one hand and exactly one mix apply**, which is what retired the family/tier
// machinery this file used to carry. A hand wins on its multiplier — five Strikes are an
// Barrage rather than also the pair and flurry inside it — and the mixes partition every hand
// by an *exact* colour count, so no two can both be true.
//
// **Matching is on the resolved order, never on the queue.** ResolutionOrder regroups a queue by
// category, so the cards you dragged into place are not the cards in the order they happen. It
// matters less than it did now that hands are counted rather than ordered, but scope is still
// read off the resolved card.
//
// **Matching is on cards used, never on what they achieved.** A hand is known before the blow is
// worked out, which is what lets its multiplier apply to the cards that formed it. Matching on
// damage dealt would mean a plan could be silently invalidated by the opponent's defenses after
// the player had committed to it.
//
// Combos are eventually **discovered rather than given**, persisting on the profile as part of
// the unlock structure — see MECHANICS.md. No profile exists yet, so everything in the catalogue
// is always live. When one does, discovery gates the *catalogue*, not the matcher.

// HandID identifies a hand, and MixID an element makeup. Both travel on the KindCombo event so
// the screen can name what fired without knowing the rule that fired it.
type HandID int
type MixID int

// HandNone and MixNone are the zero values and mean "no hand" and "no makeup". Every real ID is
// written in combos.json; an expanded hand adds the card's enum value to the ID declared there.
const (
	HandNone HandID = 0
	MixNone  MixID  = 0
)

// StaggerAll is the Stagger value meaning "every action of the opponent's next turn". A count
// would have to guess at a cap that MaxActions is already allowed to raise, and a combo
// describing itself as taking the whole round should not quietly stop doing so the first time a
// ring hands somebody a sixth action.
const StaggerAll = -1

// multiplierScale is the denominator the percent multipliers in combos.json are held over. They
// are integers because this package is integer arithmetic throughout, and a float in the file
// would be the one number in the game that rounds differently from every other.
const multiplierScale = 100

// Effect is what forming a hand buys *besides* damage. Damage is the multiplier on Hand and Mix;
// this is everything else, and every field is inert at its zero value.
type Effect struct {
	// BankAP is added to the round's banked points, arriving as budget in the round after. It
	// cannot apply to the round that formed the hand — those points were committed when the
	// cards were queued — so it rides the same GatheredAP/BonusAP path a Prepare does.
	BankAP int

	// Stagger is how many actions the *opponent* loses from their next turn, or StaggerAll.
	Stagger int
}

// Hand is one rung of the ladder, after combos.json has been read and expanded.
type Hand struct {
	ID   HandID
	Key  string // the catalogue key, shared by every expansion of one file entry
	Name string

	// Groups is how many cards of each *distinct* concept the hand wants. `[3,2]` is a full
	// house; the groups naming distinct values is why five of one card can never be one.
	Groups []int

	// Scope is which categories this hand counts. Empty means every card in the turn.
	Scope []Category

	// Pin fixes the hand to one concept: the Strike Flurry is the flurry entry pinned to Strike.
	// Set by expansion, so a generic entry like Two Pair carries none.
	Pin    ActionKind
	HasPin bool

	// Multiplier is this hand's contribution to the turn's damage, in percent.
	Multiplier int

	Effect Effect
}

// Cards is how many cards this hand is formed from.
func (h Hand) Cards() int {
	n := 0
	for _, g := range h.Groups {
		n += g
	}
	return n
}

// inScope reports whether a card of this category is counted by this hand.
func (h Hand) inScope(cat Category) bool {
	if len(h.Scope) == 0 {
		return true
	}
	for _, s := range h.Scope {
		if s == cat {
			return true
		}
	}
	return false
}

// Mix is one element makeup: how many distinct non-basic colours the formed hand showed.
type Mix struct {
	ID         MixID
	Key        string
	Name       string
	Colours    int
	Multiplier int
}

// catalogue is every hand and mix in the game, read from data/combos.json at package init.
var handTable, mixTable = loadCatalogue()

// Hands and Mixes are the live catalogue. They exist so a reference screen can list them without
// reaching into the tables, and so the discovery gate has one place to land.
func Hands() []Hand {
	out := make([]Hand, len(handTable))
	copy(out, handTable)
	return out
}

func Mixes() []Mix {
	out := make([]Mix, len(mixTable))
	copy(out, mixTable)
	return out
}

// HandByID and MixByID look one up for narration. The bool reports whether it was found rather
// than returning a zero value that would silently narrate as an unnamed effect.
func HandByID(id HandID) (Hand, bool) {
	for _, h := range handTable {
		if h.ID == id {
			return h, true
		}
	}
	return Hand{}, false
}

func MixByID(id MixID) (Mix, bool) {
	for _, m := range mixTable {
		if m.ID == id {
			return m, true
		}
	}
	return Mix{}, false
}

// HandByName and MixByName find one by the name it carries on screen. For tests and for a
// reference screen; nothing in the rules looks one up this way.
func HandByName(name string) (Hand, bool) {
	for _, h := range handTable {
		if h.Name == name {
			return h, true
		}
	}
	return Hand{}, false
}

func MixByName(name string) (Mix, bool) {
	for _, m := range mixTable {
		if m.Name == name {
			return m, true
		}
	}
	return Mix{}, false
}

// HandIDFor names the hand an expanded entry produced for one card — the `flurry` entry pinned to
// Strike. It reads the catalogue rather than recomputing `base + value`, so the file stays the
// single source of the number.
func HandIDFor(key string, card ActionKind) (HandID, bool) {
	for _, h := range handTable {
		if h.Key == key && h.HasPin && h.Pin == card {
			return h.ID, true
		}
	}
	return HandNone, false
}

// Attack is what one side's attack phase amounts to: which cards were spent on it, what hand and
// mix they formed, and the multiplier that follows.
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

	// Hand and Mix are what formed. A lone attack that forms no hand carries HandNone and still
	// carries a Mix, because one card is still one colour.
	Hand Hand
	Mix  Mix

	// Multiplier is `Hand.Multiplier + Mix.Multiplier`, in percent. **Additive, not
	// multiplicative** — a pair (150) that is duo (200) is 350, not 300. That is what keeps the
	// top of the ladder at a few hundred damage instead of several thousand.
	Multiplier int

	// Elements is every distinct non-basic colour in the hand, in element order. The mix is its
	// count; this is the list, and it is what decides which statuses land.
	Elements []Element
}

// Formed reports whether a hand of two or more cards was made, as opposed to the High Card
// falling through.
//
// **The High Card is a hand and still answers false here** *(2026-08-15)*. It has a name, an ID
// and a line in the catalogue, so the feed can say what fired; what it does not have is anything
// a player *built*, and this is the question the screen's combo preview asks — whether the cards
// currently queued amount to more than the best of them.
func (b Blow) Formed() bool { return b.Hand.ID != HandNone && b.Hand.Cards() > 1 }

// BlowFor works out one side's attack phase from the cards it resolved.
//
// **The best hand wins, and best means the biggest multiplier.** Four Strikes hold a pair and a
// flurry as well as a barrage; the barrage is worth the most, so it is the hand, and nothing else
// pays.
//
// **When no hand of two or more forms, the High Card is the blow**: the single attack that hits
// hardest, at no multiplier at all, so what lands is the card's own face damage. Ties go to the
// card queued first, which needs no tie-break rule beyond the order the turn is already in.
func BlowFor(turn []Slot) Blow {
	return blowFor(turn, handTable, mixTable)
}

func blowFor(turn []Slot, hands []Hand, mixes []Mix) Blow {
	cards, hand, formed := matchHand(turn, hands)
	if !formed {
		cards, hand = biggestAttack(turn), highCard(hands)
	}
	if len(cards) == 0 {
		return Blow{}
	}

	elems := elementsOf(turn, cards)
	mix := mixFor(len(elems), mixes)

	return Blow{
		Cards:      cards,
		Hand:       hand,
		Mix:        mix,
		Multiplier: hand.Multiplier + mix.Multiplier,
		Elements:   elems,
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
func matchHand(turn []Slot, hands []Hand) ([]int, Hand, bool) {
	var (
		best      Hand
		bestCards []int
		found     bool
	)

	for _, h := range hands {
		if h.Cards() < 2 {
			continue
		}
		cards, ok := matchCountOf(turn, h)
		if !ok {
			continue
		}
		if !found || h.Multiplier > best.Multiplier {
			best, bestCards, found = h, cards, true
		}
	}
	return bestCards, best, found
}

// matchCountOf reads the turn as a set: how many cards carry each concept, and whether that
// satisfies the hand's groups.
//
// **Groups are filled largest-count-first**, and a tie goes to the lower enum value. A full house
// asked for `[3,2]` against three Jabs and two Strikes has only one reading, but the rule has to
// be written down for the cases that do not — and it has to be a rule rather than a map walk, per
// the determinism note in CLAUDE.md.
func matchCountOf(turn []Slot, h Hand) ([]int, bool) {
	width := len(AllActions)
	members := make([][]int, width)
	for i, s := range turn {
		if !h.inScope(s.Card.Category()) {
			continue
		}
		v := int(s.Card.Action)
		if v < 0 || v >= width {
			continue
		}
		members[v] = append(members[v], i)
	}

	var out []int
	spent := make([]bool, width)
	for _, g := range h.Groups {
		best, bestCount := -1, 0
		for v := 0; v < width; v++ {
			if spent[v] || len(members[v]) < g {
				continue
			}
			// A pinned hand is the one entry in its rung that names a concept: the Strike Flurry
			// counts Strikes and nothing else.
			if h.HasPin && v != int(h.Pin) {
				continue
			}
			if len(members[v]) > bestCount {
				best, bestCount = v, len(members[v])
			}
		}
		if best < 0 {
			return nil, false
		}
		spent[best] = true
		out = append(out, members[best][:g]...)
	}

	// The cards are collected group by group, so a full house arrives as its three and then its
	// two. Sorting puts them back into the order they resolve, which is what a bracket wants.
	sort.Ints(out)
	return out, true
}

// biggestAttack is the High Card: the single attack that hits hardest, or nothing if the turn
// queued no attacks at all.
//
// **It is compared on the concept's damage rather than its cost**, because damage is what the
// blow is. The three families ladder identically — a Lunge, a Cleave and a Smash all deal double
// — so ties are now the common case rather than the exception, and they go to the card queued
// first. The earliest slot wins, which is deterministic without inventing a rule.
func biggestAttack(turn []Slot) []int {
	best, bestDamage := -1, 0
	for i, s := range turn {
		if s.Card.Category() != CategoryAttack {
			continue
		}
		// A fixed reference DMG: this is a comparison between concepts, and every concept
		// scales from the same DMG, so any positive value ranks them identically.
		if d := s.Card.Action.Damage(damageRankDMG); d > bestDamage {
			best, bestDamage = i, d
		}
	}
	if best < 0 {
		return nil
	}
	return []int{best}
}

// damageRankDMG is the DMG `biggestAttack` ranks concepts at. It never reaches a life total
// — it exists only so Heavy sorts above Strike sorts above Jab — and it is deliberately large
// enough that Jab's `dmg/2` floor of 1 cannot flatten the ladder.
const damageRankDMG = 100

// elementsOf is every distinct non-basic colour among the cards that formed the hand, in element
// order.
//
// **Basic is skipped in both directions.** It is the absence of an element, so a basic card
// neither adds a colour nor spoils one — two basic Strikes and an ice Strike show one colour and
// are mono. That is what makes a plain draw neutral rather than a punishment.
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

// mixFor is the makeup for a given number of distinct colours. The mixes name an *exact* count
// and partition every hand, so exactly one matches — which is what lets a turn produce one combo
// with no ranking at all.
func mixFor(colours int, mixes []Mix) Mix {
	for _, m := range mixes {
		if m.Colours == colours {
			return m
		}
	}
	return Mix{}
}

// scaleDamage applies a turn's multiplier to a base figure.
//
// Rounding is deliberately toward zero, matching guardDivisor and the defend reductions: the
// package is integer arithmetic throughout, so a multiplier that rounded the other way would be
// the one rule a player could not predict from the others.
func scaleDamage(base, pct int) int {
	if pct <= 0 {
		return 0
	}
	return base * pct / multiplierScale
}
