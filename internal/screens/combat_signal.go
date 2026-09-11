package screens

// Card signals: what a card does when one of its riders fires.
//
// **A rider used to be silent** *(2026-09-10)*. A golden card came up on its one face in five, the
// run gained a permanent point of DMG, and the screen said nothing whatsoever — no figure, no
// mark, no line in the fight log. The choreography table had promised a flight for both grants and
// for the heal since the day those kinds were added, and nothing had ever drawn one. This is that
// promise kept, plus the part the table had no vocabulary for.
//
// # Two halves, and the burst is the new one
//
// **The burst** is the firework: rays thrown out of the card that fired, in the colour of the rider
// that threw them. It says *this card just did something* at the card, which is the one thing a
// figure landing on a duelist card three hundred pixels away cannot say.
//
// **The flight** is the figure travelling into whatever it changed — the DMG row, the vitae row or
// the health bar of the duelist card. That is `anchorActorSeat` to `anchorActorCard`, exactly as
// the theatre table has said all along.
//
// The burst is deliberately not a `gesture` in that table. It is an *emphasis at the source* rather
// than a journey of its own, so it composes with whatever the row already says a kind does — which
// is what lets the next thing that wants fireworks (a scored card, a relic firing) reuse it without
// the table gaining a row per decoration. A `gestureBurst` is worth adding the day something bursts
// and sends nothing anywhere.
//
// # Why the colour is not a colour of its own
//
// **A signal is drawn in the tint of the rider that threw it** — `upgradeForRider` into
// `systems.UpgradeTint`, the same table the card's own face is washed with. The wheel is full, so a
// firework in a hue of its own would be claiming one; and more usefully, the burst on the card is
// then the same colour as the card, so the two read as one object rather than as a card and an
// effect that happened near it.
//
// # When they fire, which is the part that took a design decision
//
// **Riders resolve before the attack phase** — `playTurn` runs chill, then riders, then the blow —
// so every one of these events sits in the log *ahead* of `KindHand` and the sum. Drawing them
// where they sit would put four fireworks on screen before the hand had even been named.
//
// So the screen defers *(owner's call, 2026-09-10)*:
//
//   - **A played card signals as it scores.** The signal is collected here, parked against its
//     seat, and released by the hand dialog on the beat that card's own term starts. That is
//     `takeShields` generalised — a defend card's pips already leave with its figure for the same
//     reason, and `mathItem` already carried a per-item payload for it.
//   - **Every held card signals at once, as the sum begins.** They are one statement about what the
//     hand kept back rather than several about cards, and nothing sequences them because nothing
//     about them is sequential.
//
// **The resolver was not reordered to achieve this.** A heal arriving before the blow is a rules
// decision with its own argument in `playRiders`, and moving it so a screen could draw it in order
// would be the rules bending to the picture. The screen owns when it draws; the log owns what
// happened.
//
// **What happens when there is no scoring:** a turn of nothing but defences forms no hand, so the
// box never runs and there is no sequence to hang anything on. Anything still parked is flushed
// when the acting side changes or the round ends, which fires it at its own place in the log. Same
// fallback `noteShieldRaise` already keeps for pips, and for the same reason.
//
// # The rules every mover here inherits
//
// **The model has already moved.** `ResolveRound` decided all of this before a frame of it was
// drawn; a signal in the air is a ghost of something that has happened.
//
// **It holds the playback cursor** — `combatTheatre.running` — which is pacing, and pacing is
// allowed. *(Owner's call, 2026-09-10: every signal of a card firing holds playback.)*
//
// **It cannot change an outcome.**

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// The signal's clock, in the game's own beats — see `beat` in clock.go, and the note there about
// every clock on this screen being a fraction of the one speed.
var (
	// signalBurstTicks is how long the burst is out. **Most of a beat** *(owner's call,
	// 2026-09-10)*: it was a fifth of one, on the argument that a firework is a bang — and a bang
	// nobody catches is a bang nobody had. What a signal has to say is *that this card hit*, and it
	// says it at the card while the figure is still setting off, so it has to outlast a glance.
	signalBurstTicks = beat(4, 5)

	// signalFlyTicks is the figure's journey to the duelist card. The damage figure's own, since
	// it crosses comparable screen and is read the same way — see hitFlyTicks.
	signalFlyTicks = beat(1, 1)

	// signalHoldTicks is the pause on the card after landing, with the figure it changed already
	// showing the new value. The overlap is the causal link, exactly as hitHoldTicks is.
	signalHoldTicks = beat(7, 10)
)

const (
	// signalFigureSize is the type size of a travelling figure, and it is the sum's total size for
	// hitFigureSize's reason: every figure that crosses this screen is the same figure.
	signalFigureSize = mathTotalSize

	// signalFromScale and signalToScale: a signal *recedes* into the card it lands on, like a blow
	// and unlike a term flying into the sum.
	signalFromScale = 1.0
	signalToScale   = 0.74

	// signalRays is how many arms the outer relic throws, and signalSparks how many the inner one
	// does. **Both odd, and coprime** — an even count reads as a star sitting still, and two relics
	// whose counts share a factor line up into spokes.
	signalRays   = 15
	signalSparks = 11

	// signalRayLen is how far the longest outer arm reaches past its own inner radius, and
	// signalSparkLen how far the inner relic's do.
	//
	// **Both are large deliberately** *(owner's call, 2026-09-10)*. A burst the size of the card it
	// came out of is a decoration on that card; one that reaches well past it is the card doing
	// something to the table.
	signalRayLen   = 190.0
	signalSparkLen = 96.0

	// signalRayWidth is how thick an arm starts, and signalRayTaper how much of that it keeps at
	// full reach. **An arm that thins as it goes reads as a spark**, where a constant-width one
	// reads as a spoke — and fifteen spokes are a wheel rather than an explosion.
	signalRayWidth = 8.0
	signalRayTaper = 0.15

	// signalCoreSize is the hot centre, as a fraction of the card's width.
	//
	// **There is no flash disc behind the arms** *(owner's call, 2026-09-10)*. There was, at nearly
	// two card widths — and on a pale tint it lifted so far toward white that what reached the
	// screen was a big white circle with some colour round the edge, which is a *flash* and not an
	// *explosion*. What holds fifteen arms together as one object is a small bright core at the one
	// place they are all still touching, not a disc large enough to hide the card that threw them.
	signalCoreSize = 0.30
)

// signalDest is what a signal lands on: which figure of the acting duelist's card it changes.
//
// **It is not the same question as which rider fired.** One rider — gold — moves two different
// figures on two different rows, so the destination has to be its own field rather than a reading
// of the rider.
type signalDest int

const (
	// signalDMG is the DMG stat row, and signalVitae the VITAE row, both on the duelist card.
	signalDMG signalDest = iota
	signalVitae

	// signalLife is the health bar. Two riders land here — a heal and gold's life face — and they
	// are told apart by their colour rather than by where they go.
	signalLife
)

// cardSignal is one rider's firing, on its way from the card that fired it to the figure it moved.
//
// **It stores no coordinates**, like every other mover on this screen: both ends are recomputed
// every frame from the geometry that owns them, so a signal survives the row it left re-laying out
// underneath it — a sort, a slide, or the hand closing up.
type cardSignal struct {
	rider  combat.RiderKind // what fired: the colour comes from this and nothing else
	dest   signalDest
	amount int
	side   combat.Side // whose card it lands on, and whose row the seat is measured in

	// seat is where it came from: a played card's seat on the table, or a card's seat in the hand
	// when held is true. A seat the row no longer holds draws no burst and no figure — see
	// signalOrigin.
	seat int
	held bool

	t travel
}

func (c *cardSignal) tick()     { c.t.tick() }
func (c cardSignal) done() bool { return c.t.done() }

// arrived reports whether the figure has reached the card, which is the frame the figure it
// changed starts showing the new value.
func (c cardSignal) arrived() bool { return c.t.age >= signalFlyTicks }

// signalShown is what a fighter card draws **on top of** its model, because a signal has landed on
// it and the model does not catch up until the round is adopted.
//
// **It is `shownLife`'s idea pointing the other way.** A damage figure lands on a life the model
// has already spent, so the drawing lags; a rider's grant lands on a figure the *screen's* copy of
// the duelist will not hold until `endOfRound`, so the drawing leads. Both are views over a model
// that is already correct — neither is a second copy of it — and both are dropped the moment the
// authoritative state arrives.
type signalShown struct {
	dmg     int
	life    int
	maxLife int
	vitae   int
}

// riderDrawing is one rider's account of itself: whether firing it reaches the event log at all,
// which kind it arrives as, and — when it draws no signal — why not.
type riderDrawing struct {
	// emits says the rider announces something. Four of the ten change only the arithmetic of a
	// blow, so nothing about them reaches the log: the sum is where they are read.
	emits bool

	// kind is the event it arrives as, meaningful only when emits. Gold announces two and this
	// names the first; both go to the same place by the same rule.
	kind combat.EventKind

	// why is empty for a rider that throws a signal, and the reason otherwise.
	why string
}

// riderDraws is every rider kind and what firing it puts on screen.
//
// **It is the choreography table's argument one layer down.** An absent entry and a deliberate
// silence read identically in a switch, and the failure this catches is the same one: a rider added
// tomorrow that emits an event nobody drew. `TestEverySignalRiderIsAccountedFor` walks
// `combat.RiderKinds()` against this and against `noteSignal` itself, so the table cannot drift
// from the code it describes.
//
// **A silent rider is not an undrawn one.** Six of the ten are silent because what they do is
// already a figure in the sum, a pip on a row, or a question asked while the hand is matched — and
// a firework beside a term that is already flying would be the "said twice" failure.
var riderDraws = map[combat.RiderKind]riderDrawing{
	combat.RiderGolden:      {emits: true, kind: combat.KindGrantedDMG},
	combat.RiderSilver:      {emits: true, kind: combat.KindVitae},
	combat.RiderHealOnPlay:  {emits: true, kind: combat.KindHealed},
	combat.RiderVitaeInHand: {emits: true, kind: combat.KindVitae},

	combat.RiderShieldOnPlay: {emits: true, kind: combat.KindRaised,
		why: "the pips are the drawing, and they already fly out of the card with its figure"},

	combat.RiderDamageOnPlay: {why: "it moves the duelist's DMG for the blow, so every term of the sum is already bigger"},
	combat.RiderDamageInHand: {why: "the same, for a card the turn kept back"},
	combat.RiderScaleInHand:  {why: "it scales the blow, and the sum is where a multiplier is read"},
	combat.RiderScaleInCombo: {why: "the same, for a card that made the hand"},
	combat.RiderWildElement:  {why: "it is read while the hand is matched rather than while the turn resolves - what it does is the rung's name"},
}

// noteSignal parks one rider event, and reports whether it took it.
//
// **A true return means applyEvent is finished with the event**: the picture is this file's now,
// and it happens later than here.
func (s *CombatScene) noteSignal(e combat.Event) bool {
	if e.Rider == combat.RiderNone || e.Amount <= 0 {
		return false
	}

	var dest signalDest
	switch e.Kind {
	case combat.KindGrantedDMG:
		dest = signalDMG
	case combat.KindVitae:
		dest = signalVitae
	case combat.KindGrantedLife, combat.KindHealed:
		dest = signalLife
	default:
		// A shielding attack's KindRaised carries a rider too, and its picture is the pip row's
		// own — the pips already fly out of the card with its figure. A second gesture for the
		// same fact is the "said twice" failure the choreography table's reasons are full of.
		return false
	}

	// **Held is read off the rider, not guessed from the turn.** RiderVitaeInHand is the only rider
	// that pays for a card that was *not* played, which is why Event.Rider was worth a field: a
	// KindVitae from a played silver card and one from a held card were indistinguishable before.
	held := e.Rider == combat.RiderVitaeInHand
	seat := e.Slot
	if held {
		seat = s.heldSeatOf(e)
	}

	s.theatre.pending = append(s.theatre.pending, cardSignal{
		rider:  e.Rider,
		dest:   dest,
		amount: e.Amount,
		side:   e.Side,
		seat:   seat,
		held:   held,
		t:      newTravel(0, signalFlyTicks+signalHoldTicks),
	})
	return true
}

// heldSeatOf finds which card in the hand a held rider's payment came out of.
//
// **The event names a concept and an element, which is a kind of card rather than one of them**, so
// two copies of the same held card would claim one seat twice: each match is taken once, counted
// against the signals already parked. That is the problem `Event.Slot` was added to `KindBlocked`
// to solve on the table, and it is solved by counting here rather than by a second field because a
// held card has no slot — it was never in the turn.
//
// **A card the hand does not hold answers -1**, which flies from the hand row's own middle rather
// than from a seat. That is the honest picture for a payment whose card cannot be pointed at.
func (s *CombatScene) heldSeatOf(e combat.Event) int {
	taken := 0
	for _, p := range s.theatre.pending {
		if p.held && p.rider == e.Rider {
			taken++
		}
	}
	for i, c := range s.hand {
		if c.actionCard.Concept != e.Action || c.actionCard.Element != e.Element {
			continue
		}
		if taken > 0 {
			taken--
			continue
		}
		return i
	}
	return -1
}

// releaseHeldSignals sends every parked held-card signal at once, and reports how many went.
//
// **All of them on one beat** *(owner's call, 2026-09-10)*: what the hand kept back is one fact
// about the turn, and a hand holding four paying cards would otherwise be four pauses in front of a
// sum that has not started.
func (s *CombatScene) releaseHeldSignals() int {
	return s.releaseSignals(func(c cardSignal) bool { return c.held })
}

// releaseSeatSignals sends whatever the played card in this seat parked, on the beat its own term
// starts in the sum. See handMathBox.takeSignals, which is what asks.
func (s *CombatScene) releaseSeatSignals(side combat.Side, seat int) int {
	return s.releaseSignals(func(c cardSignal) bool {
		return !c.held && c.side == side && c.seat == seat
	})
}

// flushSignalsAtBoundary throws whatever is still parked when the turn it belongs to is over.
//
// **The boundary is the acting side changing, or the round ending**, and nothing else: everything
// inside one turn is either scored by the sum or flushed when the sum finishes. It is checked
// against the parked signals' own side rather than against a remembered one, so a screen re-entered
// mid-round cannot flush a turn it never watched.
func (s *CombatScene) flushSignalsAtBoundary(e combat.Event) {
	if len(s.theatre.pending) == 0 {
		return
	}
	if e.Kind == combat.KindRoundEnd {
		s.flushSignals()
		return
	}
	s.releaseSignals(func(c cardSignal) bool { return c.side != e.Side })
}

// flushSignals sends everything still parked, for the turn that never scored.
func (s *CombatScene) flushSignals() int {
	return s.releaseSignals(func(cardSignal) bool { return true })
}

// releaseSignals moves the parked signals matching a predicate onto the stage, keeping log order.
func (s *CombatScene) releaseSignals(want func(cardSignal) bool) int {
	if len(s.theatre.pending) == 0 {
		return 0
	}
	kept, sent := s.theatre.pending[:0], 0
	for _, c := range s.theatre.pending {
		if !want(c) {
			kept = append(kept, c)
			continue
		}
		s.theatre.signals = append(s.theatre.signals, c)
		sent++
	}
	s.theatre.pending = kept
	return sent
}

// advanceSignals ticks every signal and applies the one that lands on this frame.
//
// **It is its own loop rather than `advance`, for advanceBreaks' reason**: something has to happen
// on the frame a figure *arrives*, and a mover that ticks and is dropped in one pass gives nobody a
// chance to notice. The tally is what the fighter cards draw on top of their model until the round
// is adopted — see signalShown.
func (t *combatTheatre) advanceSignals() []cardSignal {
	if len(t.signals) == 0 {
		return t.signals
	}
	live := t.signals[:0]
	for i := range t.signals {
		was := t.signals[i].arrived()
		t.signals[i].tick()
		if !was && t.signals[i].arrived() {
			t.land(t.signals[i])
		}
		if !t.signals[i].done() {
			live = append(live, t.signals[i])
		}
	}
	return live
}

// land adds one arrived signal to what its side's card is drawing on top of its model.
//
// **Gold's life face raises the ceiling and the floor together**, exactly as `playRiders` does and
// as `session.Equip` does: a maximum that rose while the bar stayed where it was would read as
// nothing having happened. A heal has no ceiling to move, and the engine has already capped the
// amount at the one that exists.
func (t *combatTheatre) land(c cardSignal) {
	if c.side < 0 || int(c.side) >= len(t.shown) {
		return
	}
	sh := &t.shown[c.side]
	switch c.dest {
	case signalDMG:
		sh.dmg += c.amount
	case signalVitae:
		sh.vitae += c.amount
	case signalLife:
		sh.life += c.amount
		if c.rider == combat.RiderGolden {
			sh.maxLife += c.amount
		}
	}
}

// adopted clears what the cards were drawing over their model, because the model now holds it.
// Called from endOfRound on the frame the authoritative duelists are taken up.
func (t *combatTheatre) adopted() { t.shown = [2]signalShown{} }

// shownDMG, shownMaxLife and shownVitae are the figures a fighter card draws, which are not always
// the figures the model holds. See signalShown, and shownLife, which is the same idea for the bar.
func (s *CombatScene) shownDMG(side combat.Side, actual int) int {
	return actual + s.theatre.shownFor(side).dmg
}

func (s *CombatScene) shownMaxLife(side combat.Side, actual int) int {
	return actual + s.theatre.shownFor(side).maxLife
}

func (s *CombatScene) shownVitae(actual int) int {
	return actual + s.theatre.shownFor(combat.SideA).vitae
}

func (t *combatTheatre) shownFor(side combat.Side) signalShown {
	if side < 0 || int(side) >= len(t.shown) {
		return signalShown{}
	}
	return t.shown[side]
}

// signalInk is what a signal is drawn in, sparks and figure alike: **the colour of the rider that
// threw it**, asked for rather than restated — `upgradeForRider` into `systems.UpgradeTint`, the
// same table the card's own face is washed with. A colour of its own would be a second vocabulary
// for something the card already says, and the wheel has no room left to spend on one.
//
// So a gold card throws gold sparks, a silver card silver ones, and the heal its rose. **One card,
// one colour, all the way from the burst to the figure landing** — which is what makes the thing on
// the duelist card readable as having come out of the card it came out of.
//
// **The vitae-in-hand rider is the exception, and it is the game's rather than this file's**
// *(owner's call, 2026-09-10)*. `vitaeInk` is the crimson vitae is written in **everywhere it is
// written** — the purse on the duelist card, the word in the reward screen's prose — and it is the
// only red on the table precisely so a figure in it says "money" before it is read. That rider's
// whole subject is vitae, so its placeholder blue-grey was the one tint saying the wrong thing.
//
// **Silver is deliberately not swept up in that**, although it also pays into the purse. What is
// being said there is *the metal came up*, and the metal is what the card is; the row it lands on
// is already crimson and does not need the figure to agree with it. The vitae card has no metal to
// be, which is exactly why it takes the currency's colour instead.
func signalInk(rider combat.RiderKind) color.RGBA {
	if rider == combat.RiderVitaeInHand {
		return vitaeInk
	}
	if ink := systems.UpgradeTint(upgradeForRider[rider]); ink.A > 0 {
		return ink
	}
	return groundInk
}

// drawSignals draws every burst and every figure at wherever it has got to.
func (s *CombatScene) drawSignals(gs *state.GlobalState, screen *ebiten.Image) {
	for _, c := range s.theatre.signals {
		from, ok := s.signalOrigin(gs, c)
		if !ok {
			continue
		}
		ink := signalInk(c.rider)
		drawBurst(screen, from, c, ink)

		to := s.signalTarget(gs, c)
		p := easeOut(clamp01(float64(c.t.age) / float64(signalFlyTicks)))
		at := image.Pt(
			from.X+int(float64(to.X-from.X)*p),
			from.Y+int(float64(to.Y-from.Y)*p),
		)
		scale := signalFromScale + (signalToScale-signalFromScale)*p
		drawMathText(gs, screen, "+"+strconv.Itoa(c.amount), signalFigureSize, ink,
			at, scale, signalAlpha(c), false)
	}
}

// drawBurst throws the sparks. **They grow out of the card and fade rather than travelling**,
// because what a firework says is "here", and an arm that drifts is an arm going somewhere.
//
// **Two relics and a core, drawn back to front.** The long relic carries the reach, the short relic
// carries the density — a single relic of fifteen is legible as fifteen lines, where two relics at
// different lengths read as a scatter — and the core is the one place they are all still touching,
// which is what makes them one object instead of twenty-six.
func drawBurst(screen *ebiten.Image, at image.Point, c cardSignal, ink color.RGBA) {
	if c.t.age >= signalBurstTicks {
		return
	}
	p := easeOut(clamp01(float64(c.t.age) / float64(signalBurstTicks)))
	fade := 1 - p*p

	x, y := float32(at.X), float32(at.Y)
	inner := float64(cardWidth) * 0.26

	arm := ink
	arm.A = uint8(255 * fade)
	drawSparkRelic(screen, x, y, inner, signalRayLen, signalRayWidth, p, arm, burstRays(c, 0))

	// The inner relic is offset half a step so its arms sit between the long ones rather than under
	// them, and it is thinner: it is the shower, not the reach.
	drawSparkRelic(screen, x, y, inner*0.7, signalSparkLen, signalRayWidth*0.6, p, arm,
		burstRays(c, 1))

	// **The core is the ink lifted a little toward white**, not white itself — a saturated colour
	// has nowhere to climb by scaling, the same reason `BevelEdges` derives its light edge with
	// ColorToward. It shrinks as the arms extend, so the burst empties outward.
	//
	// **A quarter of the way and no further** *(2026-09-10)*. It was more than half, which on a pale
	// tint like silver's put a white disc back in the middle of the thing that had just stopped
	// being one — and the whole point of a card's signal is that it is that card's colour.
	core := systems.ColorToward(ink, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 25)
	core.A = uint8(255 * fade)
	vector.DrawFilledCircle(screen, x, y,
		float32(float64(cardWidth)*signalCoreSize*(1-0.55*p)), core, true)
}

// drawSparkRelic throws one relic of arms, each to its own fraction of the relic's reach.
func drawSparkRelic(screen *ebiten.Image, x, y float32, inner, reach, width float64,
	p float64, ink color.RGBA, rays []float64) {

	step := 2 * math.Pi / float64(len(rays))
	for i, far := range rays {
		a := -math.Pi/2 + float64(i)*step
		out := inner + reach*far*p
		w := float32(width * (1 - (1-signalRayTaper)*p))
		vector.StrokeLine(screen,
			x+float32(math.Cos(a)*inner), y+float32(math.Sin(a)*inner),
			x+float32(math.Cos(a)*out), y+float32(math.Sin(a)*out),
			w, ink, true)
	}
}

// burstRays is how far each arm of one relic reaches, as fractions of that relic's own length.
//
// **Derived from the rider, the seat and the relic, never rolled** — the crack pattern's rule and
// the dissolve's, and the explicit exception the randomness skill records. So one card's burst is
// the same burst through a resize, an interruption or a re-entry, and nothing here touches a
// stream.
//
// **The spread is wide on purpose.** Arms of nearly one length are a circle with a fringe; arms
// between two fifths and full reach are an explosion.
func burstRays(c cardSignal, relic int) []float64 {
	n := signalRays
	if relic > 0 {
		n = signalSparks
	}
	out := make([]float64, n)
	h := uint32(int(c.rider)*2654435761) ^ uint32(c.seat*40503+7) ^ uint32(relic*2246822519)
	for i := range out {
		h ^= h << 13
		h ^= h >> 17
		h ^= h << 5
		out[i] = 0.40 + float64(h%64)/64*0.60
	}
	return out
}

// signalAlpha fades the figure over its hold, the same shape hitAlpha has.
func signalAlpha(c cardSignal) float32 {
	if !c.arrived() {
		return 1
	}
	held := float64(c.t.age-signalFlyTicks) / float64(signalHoldTicks)
	return float32(clamp01(1 - held))
}

// signalOrigin is where a signal sets off from: the middle of the card that fired it.
//
// **A held card fires from its seat in the hand**, which is what finally gives `anchorHandRow` a
// user — the row was in the anchor enum from the start with nothing pointing at it, waiting for the
// first thing that happened to the *hand* rather than to a duelist.
func (s *CombatScene) signalOrigin(gs *state.GlobalState, c cardSignal) (image.Point, bool) {
	if c.held {
		if c.seat < 0 || c.seat >= len(s.hand) {
			r := handBand(gs, s.laidOutCount())
			return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2), true
		}
		r := s.cardSlot(gs, c.seat)
		return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2), true
	}

	var at image.Point
	if c.side == combat.SideA {
		if c.seat < 0 || c.seat >= len(s.theatre.resolved) {
			return image.Point{}, false
		}
		at = playedSeatAt(gs, c.seat, len(s.theatre.resolved), s.playedSplit())
	} else {
		if c.seat < 0 || c.seat >= len(s.theatre.enemyDealt) {
			return image.Point{}, false
		}
		at = enemySeatAt(gs, c.seat, len(s.theatre.enemyDealt), s.enemySplit())
	}
	return image.Pt(at.X+cardWidth/2, at.Y+cardHeight/2), true
}

// signalTarget is the figure the signal lands on — a stat row or the health bar of the acting
// side's fighter card.
//
// **The rows are read off the card's own style rather than measured here.** `cards.DuelistStyle`
// already says where DMG, AP and VITAE sit and where the bar is, and the card is drawn at 1:1 into
// its rectangle, so the arithmetic is the rectangle's corner plus the style's own offset. A
// constant typed in here would be a second opinion about a layout that has one.
func (s *CombatScene) signalTarget(gs *state.GlobalState, c cardSignal) image.Point {
	r := s.enemyCardRect(gs)
	st := cards.EnemyStyle
	if c.side == combat.SideA {
		r = s.duelistCardRect(gs)
		st = cards.DuelistStyle
	}
	x := (r.Min.X + r.Max.X) / 2

	// **The enemy card has no stat rows**, so anything aimed at one lands on its bar instead. That
	// cannot happen today — nothing deals a creature a metal — but a figure flying to a row that is
	// not drawn would be a number landing on nothing, and the check is cheaper than the assumption.
	if c.dest != signalLife && st.StatRowPitch > 0 {
		row := 0
		if c.dest == signalVitae {
			row = 2
		}
		return image.Pt(x, r.Min.Y+st.StatsTop+row*st.StatRowPitch+st.StatRowPitch/3)
	}
	return image.Pt(x, r.Min.Y+st.HealthBarTop+st.HealthBarHeight/2)
}
