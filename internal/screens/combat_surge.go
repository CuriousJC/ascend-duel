package screens

// combat_surge.go — a surge paid into the AP orb. A matched block banks an action point for its
// owner's next turn, and the SURGE break throws a small lapis orb carrying it into the duelist
// card's AP orb, which takes the point on the frame it lands and swells there — see ui.OrbSwells.
//
// **The card's AP reads what the next turn will have** once DUEL! is pressed: the base budget, plus
// each surge as it lands. The surge banked last round was spent the moment the round was committed,
// so leaving it on the card through the playback would put a point there that the turn being bought
// does not hold. At the round's end the duelist is adopted with the surge it banked, which is the
// figure the landed orbs already drew.
//
// **It is a view over the resolved log and cannot change an outcome.** The event says the block
// surged; the orb is the picture of the point going where it is spent.

import (
	"image"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// surgeOrbSize is the orb as it leaves the break; it grows to the AP orb's own size on the way.
const surgeOrbSize = 40

// surgeAmount is what one matched block banks.
const surgeAmount = 1

// apStat is the AP row in ui.DuelistSpec's Stats.
const apStat = 1

// surgeFlight is one banked point on its way from the break to the AP orb. `from` is the break,
// which does not move.
type surgeFlight struct {
	from   image.Point
	landed bool
	t      ui.Travel
}

func (f *surgeFlight) Tick()     { f.t.Tick() }
func (f surgeFlight) Done() bool { return f.t.Age >= signalFlyTicks() }

// raiseSurges throws an orb out of every SURGE break that has just met. Called each tick, after the
// intercepts have advanced.
func (t *combatTheater) raiseSurges() {
	for i := range t.intercepts {
		ic := &t.intercepts[i]
		if ic.surge && !ic.paid && ic.met() {
			ic.paid = true
			t.surges = append(t.surges, surgeFlight{from: ic.meetAt, t: ui.NewTravel(0, signalFlyTicks())})
		}
	}
}

// advanceSurges ticks the orbs in the air and pays each into the card's tally on the frame it lands.
func (t *combatTheater) advanceSurges() {
	live := t.surges[:0]
	for i := range t.surges {
		f := &t.surges[i]
		f.Tick()
		if !f.landed && f.Done() {
			f.landed = true
			t.shown[combat.SideA].ap += surgeAmount
		}
		if !f.Done() {
			live = append(live, *f)
		}
	}
	t.surges = live
}

// shownAP is the AP the duelist card draws: the whole budget while the player plans, and the base
// budget plus every surge landed so far while a round plays back.
func (s *CombatScene) shownAP() int {
	if s.planning() {
		return s.fighter.ActionPoints()
	}
	return s.fighter.Actions + s.Theater.shownFor(combat.SideA).ap
}

// drawSurges draws every banked point on its way to the AP orb.
func (s *CombatScene) drawSurges(gs *state.GlobalState, screen *ebiten.Image) {
	card := ui.DuelistCardRect(gs)
	to, ok := ui.StatOrbCenter(card, cards.DuelistStyle, apStat)
	if !ok {
		return
	}
	land := float64(surgeOrbSize)
	if box, ok := cards.StatOrbBox(cards.DuelistStyle, apStat); ok {
		land = float64(box.Dx())
	}
	for _, f := range s.Theater.surges {
		p := ui.EaseOut(ui.Clamp01(float64(f.t.Age) / float64(signalFlyTicks())))
		at := ui.LerpPoint(f.from, to, p)
		size := surgeOrbSize + (land-surgeOrbSize)*p
		ui.DrawOrb(gs.Fonts["kubasta"], screen, cards.OrbAP, strconv.Itoa(surgeAmount),
			float64(at.X), float64(at.Y), size)
	}
}
