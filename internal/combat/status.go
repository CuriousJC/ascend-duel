package combat

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/curiousjc/ascend-duel/data"
)

// Statuses are what a landed attack leaves standing on a duelist, and they are the first thing in
// this package that outlives the action that caused it. A burn ticks at a moment no action owns; a
// chill is read when a turn is taken; a miss is rolled against the attack it interrupts.
//
// **A status is data, and it is no longer the same object as an element** *(2026-08-17)*. It was
// four constants indexed by colour until then, which made two things inexpressible that the ring
// grammar needs: a second fire status, and a status arriving from something that is not a colour at
// all. `data/statuses.json` is the catalogue; this file registers it and holds the lifecycle.
//
// **Nothing applies a status by itself.** A fire attack is a plain attack with a red border unless
// something says otherwise, and the only thing that says otherwise today is a ring the attacker is
// wearing — an `apply-status` effect at the `attack-lands` moment, naming the status by its record
// key. See ring.go. The reason is unchanged from 2026-08-16: statuses given away with the colour
// left the first rings with nothing to *be*.
//
// **One lifecycle for all of them, so it is learned once**:
//
//   - **Applied by a landed attack, and by nothing else.**
//   - **It does not stack; a second hit resets the clock.** *(2026-08-16)* Two chills chill for the
//     same one card as one did, and the second simply buys two more round-ends of it. Amounts
//     stacked until then, which made a status something to pile on rather than something to keep
//     up — and with one blow a turn, four stacks of a thing was four cards spent saying one word
//     louder. A ring that *does* stack is a ring that can be designed later; the base rule being
//     "no" is what leaves it somewhere to go.
//   - **Cleared at the end of the round after the one that applied them.** A record's `Rounds` is
//     counted in round *ends*, and every status in the file says 2 for one reason: a status applied
//     during round N has to survive round N's ending to be felt in round N+1, so a duration of 1
//     applied by whoever acts second would never bite anything at all.
//
// **Two statuses of the same kind add up.** Nothing in the game applies two yet — there is one
// record per kind — but the queries below sum rather than pick, because the grammar can express a
// second one and a silent choice between them would be a rule nobody wrote down.

// StatusID identifies a registered status. Like ConceptID it is an index into a registry, so it is
// cheap to compare and **meaningless outside the process that assigned it** — a save file writes
// `StatusRecord`, never this.
//
// **Append-only, the hazard Element and GlyphKind carry**, and here it is the file that decides:
// inserting a record into the middle of `statuses.json` re-points every status a duelist is
// carrying, because `Duelist.Statuses` is indexed by this value.
type StatusID int

// NoStatus is the absence of one. Negative rather than zero, because a zero StatusID is the first
// registered record — the same shape NoConcept has.
const NoStatus StatusID = -1

// MaxStatuses is the width of `Duelist.Statuses`, and therefore the most statuses the game may
// define.
//
// **It is an array width rather than a design cap.** A duelist has to stay comparable — see the
// note on the struct — so the statuses standing on one are a fixed array indexed by StatusID, and a
// fixed array needs a compile-time size. Registration panics past this rather than growing, because
// silently dropping the fifth record would be a status that never fires.
//
// **What the card can *show* is a separate and smaller number**: `cards.MaxEffects` is the badge
// row, and `internal/screens` holds the check that every registered status has a seat in it.
const MaxStatuses = 8

// StatusEffect is what carrying a status does. **Four kinds, closed** — the same posture Verb and
// the ring effects take. A status is a file entry; a kind of status is a Go change here plus the one
// place reading it.
type StatusEffect int

const (
	// EffectDamageOverTime ticks damage at the end of every round it survives. Amount is a percent
	// of the *attacker's* DMG, read off them and frozen when the status lands — see statusAmount.
	EffectDamageOverTime StatusEffect = iota

	// EffectLoseActions takes cards off the front of the carrier's turn, for as long as it lasts.
	EffectLoseActions

	// EffectMissChance is percentage points of chance that the carrier's attack never happens.
	EffectMissChance

	// EffectDamageReduction is percentage points off the damage the carrier deals. It is the only
	// kind that reaches forward into what its victim does rather than what happens to them.
	EffectDamageReduction
)

// StatusEffects is every effect kind in a fixed order, for anything that walks them.
func StatusEffects() []StatusEffect {
	return []StatusEffect{EffectDamageOverTime, EffectLoseActions, EffectMissChance, EffectDamageReduction}
}

func (e StatusEffect) String() string {
	switch e {
	case EffectLoseActions:
		return "lose-actions"
	case EffectMissChance:
		return "miss-chance"
	case EffectDamageReduction:
		return "damage-reduction"
	default:
		return "damage-over-time"
	}
}

// ParseStatusEffect resolves an effect kind from its name, reporting failure rather than falling
// back: a status quietly registered as a burn because its kind was misspelled is a balance change
// nobody made.
func ParseStatusEffect(name string) (StatusEffect, bool) {
	for _, e := range StatusEffects() {
		if e.String() == name {
			return e, true
		}
	}
	return EffectDamageOverTime, false
}

// StatusSpec is one status's rules, whole. The badge and the long-press text stay in `data` — they
// are what a screen needs and this package has no opinion about either.
type StatusSpec struct {
	// Key is the record key, and the identity anything outside the process has to use.
	Key string

	// Name is the word the Resolution feed shouts.
	Name string

	Effect StatusEffect
	Amount int
	Rounds int
}

// The status registry, in file order. Package state for exactly the reason the concept registry is:
// it is loaded once from embedded data, never rewritten, and threading a catalogue through every
// method and every test would describe nothing that changes.
var (
	statusRegistry []StatusSpec
	statusBy       = map[string]StatusID{}
)

// registeredStatuses loads `statuses.json` at package init. A record the rules cannot resolve panics
// here rather than mid-duel, the same contract RegisterConcept has.
var registeredStatuses = registerStatuses()

func registerStatuses() int {
	for _, s := range data.LoadStatuses() {
		if err := registerStatus(s); err != nil {
			panic("statuses.json: " + err.Error())
		}
	}
	return len(statusRegistry)
}

func registerStatus(s data.StatusData) error {
	if s.StatusRecord == "" {
		return fmt.Errorf("a status has no StatusRecord")
	}
	if _, taken := statusBy[s.StatusRecord]; taken {
		return fmt.Errorf("%s is registered twice", s.StatusRecord)
	}
	if len(statusRegistry) == MaxStatuses {
		return fmt.Errorf("%s is status number %d, and Duelist.Statuses holds %d",
			s.StatusRecord, len(statusRegistry)+1, MaxStatuses)
	}

	effect, ok := ParseStatusEffect(s.Effect)
	if !ok {
		return fmt.Errorf("%s names effect %q, which is not one of the four kinds", s.StatusRecord, s.Effect)
	}
	if s.Amount <= 0 {
		return fmt.Errorf("%s has Amount %d, so it does nothing", s.StatusRecord, s.Amount)
	}
	if s.Rounds <= 0 {
		return fmt.Errorf("%s lasts %d rounds, so it is over before it is felt", s.StatusRecord, s.Rounds)
	}

	// Nothing in the game stops a blow outright and nothing misses every time: a defence that
	// always works deletes a whole opposing turn for the price of one card, which is what one blow
	// per turn cannot afford. The same bound RegisterConcept holds against a 100% defence.
	if effect == EffectMissChance && s.Amount >= 100 {
		return fmt.Errorf("%s misses %d%% of attacks, and nothing may stop a blow outright", s.StatusRecord, s.Amount)
	}
	if effect == EffectDamageReduction && s.Amount >= 100 {
		return fmt.Errorf("%s blunts damage by %d%%, and nothing may reduce a blow to zero", s.StatusRecord, s.Amount)
	}

	statusBy[s.StatusRecord] = StatusID(len(statusRegistry))
	statusRegistry = append(statusRegistry, StatusSpec{
		Key:    s.StatusRecord,
		Name:   s.Name,
		Effect: effect,
		Amount: s.Amount,
		Rounds: s.Rounds,
	})
	return nil
}

// StatusOf is the record behind an ID. An ID the registry does not hold returns a zero spec, which
// applies nothing and is named "?" — a guard rather than a path, since the engine never invents one.
func StatusOf(id StatusID) StatusSpec {
	if id < 0 || int(id) >= len(statusRegistry) {
		return StatusSpec{Key: "?", Name: "?"}
	}
	return statusRegistry[id]
}

// StatusByKey finds a registered status by its record key.
func StatusByKey(key string) (StatusID, bool) {
	id, ok := statusBy[key]
	return id, ok
}

// MustStatus is StatusByKey for callers that would rather fail at startup than carry a status that
// does nothing — the tools and the tests.
func MustStatus(key string) StatusID {
	id, ok := statusBy[key]
	if !ok {
		panic("combat: no status named " + key)
	}
	return id
}

// StatusCount is how many statuses are registered. Unlike ConceptCount it is fixed for a build:
// nothing registers a status after init.
func StatusCount() int { return registeredStatuses }

// AllStatuses is every registered status in file order — the order a badge row is drawn in, which is
// what keeps a badge from moving along the row as another status comes and goes.
func AllStatuses() []StatusID {
	out := make([]StatusID, len(statusRegistry))
	for i := range statusRegistry {
		out[i] = StatusID(i)
	}
	return out
}

// StatusKeys is every record key, sorted, for a tool or a test that wants to walk the catalogue
// without depending on file order.
func StatusKeys() []string {
	out := make([]string, 0, len(statusRegistry))
	for _, s := range statusRegistry {
		out = append(out, s.Key)
	}
	sort.Strings(out)
	return out
}

// Status is one status's hold on a duelist: how much, and how much longer.
//
// Amount means a different thing per effect kind and each is documented at its constant above. A
// generic amount is what lets every status share one array and one lifecycle; a generic *meaning*
// would be a rule nobody could state.
type Status struct {
	Amount int
	Rounds int
}

// Active reports whether this status is doing anything. A zero Status is the absence of one.
func (s Status) Active() bool { return s.Rounds > 0 && s.Amount > 0 }

// statusAmount is how much of a status one hit applies, given the duelist throwing it.
//
// **The attacker is a parameter because a burn is a share of their DMG**, read off them and frozen
// when it lands rather than recomputed each tick: a burn is what that blow lit, and a duelist whose
// DMG changes mid-duel does not retroactively burn harder. Every other kind is flat, and passing the
// attacker to all of them is what keeps "how much" one function rather than a special case.
//
// The floor is the same rule the cheapest attack card follows: a duelist under 10 DMG would
// otherwise light a burn worth nothing at all, and a status that lands and does nothing is worse
// than one that does not land.
func statusAmount(spec StatusSpec, by Duelist) int {
	if spec.Effect != EffectDamageOverTime {
		return spec.Amount
	}
	tick := by.DMG * spec.Amount / 100
	if tick < 1 {
		tick = 1
	}
	return tick
}

// applyStatus lands one status on a duelist: the amount is set and the duration refreshed.
// **Set, not added** — see the file comment on why nothing stacks.
//
// It returns the duelist by value like everything else in this package, plus the amount applied, so
// the caller can log what happened without recomputing it. The bool reports whether the ID named a
// status at all.
func applyStatus(d Duelist, id StatusID, by Duelist) (Duelist, int, bool) {
	if id < 0 || int(id) >= len(statusRegistry) {
		return d, 0, false
	}
	spec := statusRegistry[id]

	amount := statusAmount(spec, by)
	d.Statuses[id] = Status{Amount: amount, Rounds: spec.Rounds}
	return d, amount, true
}

// totalOf sums the amounts of every active status of one kind. **Summed rather than picked**: the
// ring grammar can put two chills on a duelist, and choosing between them silently would be a rule
// nobody wrote down.
func (d Duelist) totalOf(kind StatusEffect) int {
	total := 0
	for id := 0; id < len(statusRegistry); id++ {
		if statusRegistry[id].Effect != kind || !d.Statuses[id].Active() {
			continue
		}
		total += d.Statuses[id].Amount
	}
	return total
}

// maxStatusPct is the ceiling on any summed percentage a status can reach. Nothing stops a blow
// outright and nothing misses every time — see the two registration checks, which hold the same line
// for a single record.
const maxStatusPct = 99

// capPct holds a summed percentage under the ceiling.
func capPct(pct int) int {
	if pct > maxStatusPct {
		return maxStatusPct
	}
	return pct
}

// chillCards is how many actions this duelist loses off the front of its turn, for as long as the
// status lasts. It adds to `Duelist.Chilled`, which is what a chilling hit banked for the turn to
// come — see playTurn, where the two are read together.
func (d Duelist) chillCards() int { return d.totalOf(EffectLoseActions) }

// weight is the percentage this duelist's outgoing attack damage is blunted by. A percentage rather
// than a fraction because MECHANICS.md specified one; the arithmetic in blunt is still integer and
// still truncates.
//
// **Clamped below 100**, which is the bound registration holds one record to, applied again to a sum
// two records could reach: nothing in the game reduces a blow to zero.
func (d Duelist) weight() int { return capPct(d.totalOf(EffectDamageReduction)) }

// blunt applies a weight to one outgoing blow.
//
// **Rounding is toward zero**, matching guardDivisor, the defend reductions and scaleDamage. The rule
// being the same as every other reduction is what keeps it predictable from the others rather than
// being the one number a player cannot work out.
func blunt(dmg, pct int) int {
	if pct <= 0 {
		return dmg
	}
	return dmg * (100 - pct) / 100
}

// missChance is how likely this duelist's attack is to come to nothing, in percentage points.
func (d Duelist) missChance() int { return capPct(d.totalOf(EffectMissChance)) }

// attackMisses rolls a duelist's attack and reports whether it misses.
//
// **Nothing is consumed** *(2026-08-16)*: the status rolls again on every attack until its rounds run
// out. It therefore takes the duelist by value and gives nothing back — a status that neither stacks
// nor depletes is read, not spent.
//
// **This is the only randomness in `internal/combat` and it arrives the way CLAUDE.md requires**: an
// injected `*rand.Rand` on `ResolveRound`, never a package-level source. The costs are real and were
// accepted — `tools/balance` becomes a distribution rather than an exact answer, and the stream is
// advanced per attack phase, so a change early in a duel reshuffles every roll after it.
//
// A nil source means "no rolls", which is what keeps a caller that has no business being random — a
// preview, a test pinning the deterministic parts — from silently getting one.
func attackMisses(d Duelist, rng *rand.Rand) bool {
	pct := d.missChance()
	if pct <= 0 || rng == nil {
		return false
	}
	return rng.Intn(100) < pct
}

// tickingStatuses is every damage-over-time status standing on this duelist, as its ID and what it
// deals at the end of the round. **One entry per status rather than a sum**, because each one
// produces its own line in the feed.
//
// Walked in registration order, per the determinism rules: the order damage lands in decides which
// tick killed a duelist, and an unordered walk would decide that differently every launch.
func (d Duelist) tickingStatuses() []StatusID {
	var out []StatusID
	for id := 0; id < len(statusRegistry); id++ {
		if statusRegistry[id].Effect != EffectDamageOverTime || !d.Statuses[id].Active() {
			continue
		}
		out = append(out, StatusID(id))
	}
	return out
}

// tickStatuses counts every status down one round end and clears the ones that run out. The
// damage-over-time damage is *not* here — it is a life change that has to produce events, so
// endRound does it.
func tickStatuses(d Duelist) Duelist {
	for id := 0; id < len(statusRegistry); id++ {
		s := d.Statuses[id]
		if s.Rounds <= 0 {
			d.Statuses[id] = Status{}
			continue
		}
		s.Rounds--
		if s.Rounds <= 0 {
			s = Status{}
		}
		d.Statuses[id] = s
	}
	return d
}
