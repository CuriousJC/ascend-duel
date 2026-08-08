package combat

// Combos are the layer where a round stops being a pile of cards and starts being a plan.
// Throwing whatever you drew at the opponent works; *choosing* a shape and building toward
// it is meant to work better, and this file is the machinery that pays for that choice.
//
// **One pattern, three kinds of reward.** Every combo is the same shape — a run of cards
// that must appear consecutively in your own turn, and an Effect that says what forming it
// buys. The reward vocabulary is deliberately small and fixed:
//
//   - a **damage multiplier**, applied to the rest of the turn,
//   - **action points banked** for the round after, and
//   - an **alteration to the opponent**, of which stagger is the first.
//
// Adding a combo is adding one entry to the table at the bottom of this file. Adding a new
// *kind* of reward is a new field on Effect and one place that applies it in ResolveRound —
// which is the cost the framework is deliberately paying, because a reward vocabulary that
// grows without limit is one no player can hold in their head.
//
// **Matching is on the resolved order, never on the queue.** ResolutionOrder regroups a
// queue by category, so the cards you dragged into place are not the cards in the order they
// happen. Matching the queue would let the Resolution pane show one thing and the engine score
// another, which is the exact failure the pane's design exists to make impossible.
//
// **Matching is on cards used, never on what they achieved.** A combo is known before the
// round is played, which is what lets a damage multiplier apply to the cards that formed it
// rather than arriving after they have landed. Matching on damage dealt would make that
// impossible, and would mean a plan could be silently invalidated by the opponent's defenses
// after the player had committed to it.
//
// Combos are eventually **discovered rather than given**, persisting on the profile as part
// of the unlock structure — see MECHANICS.md. No profile exists yet, so every combo in the
// table is always live. When one does, discovery gates the *table*, not the matcher: an
// undiscovered combo is one that is not passed to matchCombos, and nothing else changes.

// ComboID identifies a combo. It travels on the KindCombo event so the screen can name what
// fired without knowing the rule that fired it.
type ComboID int

// ComboNone is the zero value and means no combo. Every real ID is derived from the card it
// is built on — see FlurryID and OnslaughtID.
const ComboNone ComboID = 0

// The flurry/onslaught family is **generated, one pair per attack card**, rather than written
// out. A new attack card gets its own Flurry and Onslaught by existing, which is the whole
// point: the family is a shape the game keeps, not a list somebody has to remember to extend.
//
// **Per card, not per category**, decided 2026-08-07. An earlier version matched
// `AnyOf(CategoryAttack)`, so any three attacks combo'd and a Jab was as good as a Heavy.
// Naming them for the card they are built on is what makes them worth *building toward*: three
// Strikes is a deck you assembled, three attacks is whatever you happened to draw. It is also
// what leaves room for the effects to differ per card later — a Heavy Flurry has every reason
// to hit harder than a Jab one, and a category-wide combo could never say so.
const (
	flurryRun    = 3
	onslaughtRun = 5
)

// The two bases keep a combo's ID stable as cards are added, since each is offset by the card
// rather than by its position in a list. Discovery will persist on the profile, so an ID that
// shifted when a card was inserted would silently re-lock combos the player had already found.
const (
	comboFlurryBase    ComboID = 100
	comboOnslaughtBase ComboID = 200
)

// FlurryID and OnslaughtID name the combo built on a given attack card.
func FlurryID(a ActionKind) ComboID    { return comboFlurryBase + ComboID(a) }
func OnslaughtID(a ActionKind) ComboID { return comboOnslaughtBase + ComboID(a) }

// Step is one position in a combo's pattern. It matches either an exact card or any card of
// a category, and the two constructors below are the only correct way to build one — a
// zero-valued Step reads as "exactly a Gather", which is never what a caller means.
type Step struct {
	action     ActionKind
	category   Category
	byCategory bool
}

// Card matches one specific action.
func Card(a ActionKind) Step { return Step{action: a} }

// AnyOf matches any action resolving in the given category. `AnyOf(CategoryAttack)` is what
// makes "three attacks" mean three attacks rather than three Strikes.
func AnyOf(c Category) Step { return Step{category: c, byCategory: true} }

func (s Step) matches(a ActionKind) bool {
	if s.byCategory {
		return a.Category() == s.category
	}
	return s.action == a
}

// StaggerAll is the Stagger value meaning "every action of the opponent's next turn". A
// count would have to guess at a cap that MaxActions is already allowed to raise, and a
// combo describing itself as taking the whole round should not quietly stop doing so the
// first time a ring hands somebody a sixth action.
const StaggerAll = -1

// Effect is what forming a combo buys. Every field is inert at its zero value, so a combo
// only pays out what it names.
//
// **A combo fires on the card that completes its run, and its effects last for the rest of
// that side's turn** *(changed 2026-08-07; it fired on the first card until then)*. Two
// reasons, and the second is the one that makes it correct rather than merely nicer:
//
//   - It reads in the order it happens. Three strikes land, and *then* the flurry is
//     recognised — rather than "COMBO!" appearing above cards that have not resolved yet.
//   - **A run that never finishes now pays nothing.** Matching happens up front against the
//     whole turn, so a turn cut short — a Riposte killing the attacker on the second of three
//     strikes — used to fire the combo off cards that never resolved. Firing on completion
//     makes that impossible by construction instead of by a check somebody has to remember.
//
// The cost, recorded honestly: **a damage multiplier can only boost what comes after the
// run**, never the cards that earned it. "Complete the combo, get a bonus for the rest of the
// turn" is a coherent reading, but it is not the one the multiplier was designed against, and
// neither shipping combo uses one — so nothing real exercises it yet.
type Effect struct {
	// DamageNum/DamageDen scale damage dealt for the rest of the turn, as a fraction so the
	// package stays pure integer arithmetic. Zero values mean no change; 3/2 is half again.
	DamageNum, DamageDen int

	// BankAP is added to the round's banked points, arriving as budget in the round after.
	// It cannot apply to the round that formed the combo — those points were committed when
	// the cards were queued — so it rides the same GatheredAP/BonusAP path a Gather does.
	BankAP int

	// Stagger is how many actions the *opponent* loses from their next turn, or StaggerAll.
	Stagger int
}

// Combo is one entry in the table: a name, a pattern, and what forming it buys.
type Combo struct {
	ID     ComboID
	Name   string
	Run    []Step
	Effect Effect
}

// comboTable is every combo in the game. Order matters only as a tie-break — matching takes
// the longest run that starts at a position, so five Strikes form a Strike Onslaught rather
// than a Strike Flurry plus two spare cards.
//
// **Built rather than written out**, so the flurry/onslaught family covers whatever attack
// cards exist. Deckbuilding is expected to add many; a table listing them by hand would be a
// list that goes stale the first time somebody adds a card and forgets this file.
//
// The staggers themselves are recorded decisions, not invention: MECHANICS.md decides three
// staggers one action and five takes the whole round.
var comboTable = buildComboTable()

func buildComboTable() []Combo {
	// AllActions is a slice, not a map, so this is a fixed order. Determinism matters even
	// here: matching walks the table and a shuffled one could pick a different combo for the
	// same turn. See the determinism rules in CLAUDE.md.
	var out []Combo
	for _, a := range AllActions {
		if a.Category() != CategoryAttack {
			continue
		}
		out = append(out,
			Combo{
				ID:     FlurryID(a),
				Name:   a.String() + " Flurry",
				Run:    runOf(a, flurryRun),
				Effect: Effect{Stagger: 1},
			},
			Combo{
				ID:     OnslaughtID(a),
				Name:   a.String() + " Onslaught",
				Run:    runOf(a, onslaughtRun),
				Effect: Effect{Stagger: StaggerAll},
			},
		)
	}
	return out
}

// runOf is n of the same card in a row.
func runOf(a ActionKind, n int) []Step {
	run := make([]Step, n)
	for i := range run {
		run[i] = Card(a)
	}
	return run
}

// Combos is every combo currently live. It exists so a reference screen can list them
// without reaching into the table, and so the discovery gate has one place to land.
func Combos() []Combo {
	out := make([]Combo, len(comboTable))
	copy(out, comboTable)
	return out
}

// ComboByID looks up a combo for narration. The bool reports whether it was found rather
// than returning a zero Combo that would silently narrate as an unnamed effect.
func ComboByID(id ComboID) (Combo, bool) {
	for _, c := range comboTable {
		if c.ID == id {
			return c, true
		}
	}
	return Combo{}, false
}

// ComboHit is one combo formed by one side's turn: which combo, whose turn, and where in
// that turn it begins.
//
// Start is an index into the side's *own* resolved turn, not into the round. A round has two
// turns and the second one starts wherever the first runs out, so a hit that carried a
// round-wide index would have to be recomputed the moment either queue changed length.
type ComboHit struct {
	ID     ComboID
	Side   Side
	Start  int
	Length int
	Effect Effect
}

// MatchCombos finds every combo formed by one side's queued actions.
//
// The actions are regrouped into resolution order first, so a caller passes the queue and
// gets hits against the order the round will actually be played in. **This is what the combat
// screen calls while the player is still planning** — it is the same function the engine uses,
// so a combo previewed on the hand is the combo that will fire, by construction rather than by
// two pieces of code agreeing.
//
// It matches the *whole* queue. ResolveRound matches what survives a stagger instead, which
// is why that path goes through matchSlots directly and why this one cannot be used there.
func MatchCombos(side Side, actions []ActionKind) []ComboHit {
	return matchSlots(appendTurn(nil, side, actions), comboTable)
}

// matchSlots is the matcher proper, over one side's turn as it will actually be played. The
// table is a parameter so a test can pin the matching rules against a pattern of its own
// rather than against whatever the game currently ships.
//
// **Left to right, longest first, and a card forms at most one combo.** Without the last of
// those, three attacks would form Flurry starting at every position it fits and pay out
// three times for one decision. Without longest-first, five attacks would score as a Flurry
// and two leftovers, and Onslaught could never fire at all.
func matchSlots(turn []Slot, table []Combo) []ComboHit {
	var hits []ComboHit
	for i := 0; i < len(turn); {
		best := -1
		bestLen := 0
		for c := range table {
			run := table[c].Run
			if len(run) <= bestLen || i+len(run) > len(turn) {
				continue
			}
			ok := true
			for k, step := range run {
				if !step.matches(turn[i+k].Action) {
					ok = false
					break
				}
			}
			if ok {
				best, bestLen = c, len(run)
			}
		}

		if best < 0 {
			i++
			continue
		}

		hits = append(hits, ComboHit{
			ID:     table[best].ID,
			Side:   turn[i].Side,
			Start:  i,
			Length: bestLen,
			Effect: table[best].Effect,
		})
		i += bestLen
	}
	return hits
}

// scaleDamage applies a combo's damage multiplier. A zero or negative denominator means the
// combo named no multiplier, which is the common case and must not divide by zero.
//
// Rounding is deliberately toward zero, matching guardDivisor: the package is integer
// arithmetic throughout and every other reduction already truncates, so a multiplier that
// rounded the other way would be the one rule a player could not predict from the others.
func scaleDamage(dmg, num, den int) int {
	if num <= 0 || den <= 0 {
		return dmg
	}
	return dmg * num / den
}
