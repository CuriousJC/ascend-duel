package screens

// The shields on the table: heaped over whatever raised them, and thrown onto the attacks they eat.
//
// **A shield forms over the card that made it and stays there** — a defend card, an attack card
// with a shield rider, or the ring of a relic that raises one. Several on one card heap, which is
// how the count is read: see ui/shield_heap.go. **They form once the turn's arithmetic is done**
// *(owner's call)*: every hit's line is worked out first, a defense's coming to 0, and the turn's
// shields go up together on the way out of it.
//
// **When the creature swings at a shield, the hit it would have landed shows on its card, and a
// shield leaves its heap and lands on the card, smothering it** while the figure fades — on the
// beat that attack is reached, so what stops a hit is watched stopping it. A matched block writes
// SURGE across the card it smothered, since that block banked an action point for the next turn.
//
// **It is a view over the resolved log and cannot change an outcome.** `combat.Duelist.Shields` is
// not written until the round's end state is adopted, so the heaps are moved on the beat each event
// is reached rather than read off the model; at the end of the round they are squared up with the
// model, which is the authority.
//
// **The engine counts shields per element and never says which card's shield it spent**, so the
// screen picks: the newest heap holding a shield of the spent element gives up its top one. That is
// a choice about a picture, not about a rule.

import (
	"image"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// shieldPile is the heap over one thing that raised shields.
type shieldPile struct {
	// side is whose shields these are, and seat the table seat they stand over — or -1 for a heap
	// standing on a relic.
	side combat.Side
	seat int

	// relic is the ring the heap stands on, and onRelic says to use it rather than the seat. **A
	// bool beside the id**, because the zero RelicID is a real relic.
	relic   combat.RelicID
	onRelic bool

	// atDuelist is the heap of shields standing with nothing on the table to stand over: what the
	// model says is up that no card or ring raised this round. It stands on the duelist card.
	atDuelist bool

	shields []heapShield
}

// heapShield is one standing shield: the element it was raised in, which is a rule — a shield
// eating a hit of its own element banks an action point — and its forming clock.
type heapShield struct {
	element cards.Element
	form    ui.Travel
}

// fadingShield is a shield going away where it stood: lapsing unspent as its card leaves the
// table, or squared away by an announcement. It carries its own pose, because whatever it stood on
// may already have gone.
type fadingShield struct {
	element cards.Element
	pose    ui.ShieldPose
	t       ui.Travel
}

func (f *fadingShield) Tick()     { f.t.Tick() }
func (f fadingShield) Done() bool { return f.t.Done() }

// smother is a shield thrown onto the attack it ate, and then lying on it for the rest of the round.
type smother struct {
	// seat is the attack's seat in the opponent's row.
	seat int

	element cards.Element

	// from is where the shield was when it left its heap.
	from ui.ShieldPose

	// surged says the block banked an action point, which writes SURGE across the card.
	surged bool

	// would is the hit the shield stopped, shown on the card until the shield lands on it, in the
	// attack card's own element.
	would int
	ink   cards.Element

	t ui.Travel
}

// The clocks, in the game's own beats so they slow down and speed up with the round.
// smotherLeadTicks is the blocked hit's figure on its card before the shield sets off.
func smotherLeadTicks() int  { return ui.Beat(1, 2) }
func smotherFlyTicks() int   { return ui.Beat(4, 5) }
func smotherHoldTicks() int  { return ui.Beat(3, 5) }
func smotherTotalTicks() int { return smotherLeadTicks() + smotherFlyTicks() + smotherHoldTicks() }
func shieldFadeTicks() int   { return ui.Beat(1, 2) }

// flying is how far the shield is through its flight, 0 while the figure is still being read.
func (m smother) flying() float64 {
	return ui.Clamp01(float64(m.t.Age-smotherLeadTicks()) / float64(smotherFlyTicks()))
}

// landed reports that the shield has reached the attack card.
func (m smother) landed() bool { return m.t.Age >= smotherLeadTicks()+smotherFlyTicks() }

// sinceLanding is how many ticks the shield has been lying on the card.
func (m smother) sinceLanding() int { return m.t.Age - smotherLeadTicks() - smotherFlyTicks() }

// holding reports that the cursor should still wait on this one: the figure being read, the shield
// in the air, or the beat after it lands that lets the player see which card was stopped.
func (m smother) holding() bool { return m.t.Age < smotherTotalTicks() }

// smotherDeg is the turn a shield lies at on the card it smothered: a little off true, so it reads
// as thrown down on the card rather than printed on it.
const smotherDeg = -8

// ---- the events ----

// noteShieldRaise holds a raise until the turn's arithmetic is done, and reports that it took it.
// **Each heap stands over its own card, named by `Event.Slot`** — nothing is lit during the defend
// phase, so there is no lit card to read a seat off.
func (s *CombatScene) noteShieldRaise(e combat.Event) bool {
	if e.Kind != combat.KindRaised {
		return false
	}
	s.Theater.pendingRaises = append(s.Theater.pendingRaises, e)
	return true
}

// noteWard holds a helm's shields for its ring, on the same terms as a card's.
func (s *CombatScene) noteWard(e combat.Event) {
	s.Theater.pendingRaises = append(s.Theater.pendingRaises, e)
}

// turnOver reports that the acting side's turn has nothing left to reach: the next event belongs
// to the other side, ends the round, or there is none.
func (s *CombatScene) turnOver() bool {
	if len(s.Theater.pendingRaises) == 0 {
		return false
	}
	if s.cursor >= len(s.log) {
		return true
	}
	next := s.log[s.cursor]
	return next.Kind == combat.KindRoundEnd || next.Side != s.Theater.pendingRaises[0].Side
}

// formPendingShields forms every shield the turn raised, all on one frame — **the turn's shields
// go up as one gesture**, each heap over its own card or ring — and reports whether it formed any.
// The cursor then waits for them: see combatTheater.shieldsMoving.
func (s *CombatScene) formPendingShields() bool {
	pending := s.Theater.pendingRaises
	s.Theater.pendingRaises = nil
	formed := false
	for _, e := range pending {
		if e.Amount <= 0 {
			continue
		}
		var p *shieldPile
		if e.Kind == combat.KindWarded {
			if p = s.pileOnRelic(e.Side, e.Relic); p == nil {
				s.Theater.piles = append(s.Theater.piles, shieldPile{side: e.Side, seat: -1, relic: e.Relic, onRelic: true})
				p = &s.Theater.piles[len(s.Theater.piles)-1]
			}
		} else if p = s.pileAtSeat(e.Side, e.Slot); p == nil {
			s.Theater.piles = append(s.Theater.piles, shieldPile{side: e.Side, seat: e.Slot})
			p = &s.Theater.piles[len(s.Theater.piles)-1]
		}
		p.add(ui.ArtFor(e.Element), e.Amount)
		formed = true
	}
	return formed
}

// noteBlock throws a shield onto the attack it ate. `Target` is whose shield was spent and the
// attack is the other side's, at `Slot` in its row.
func (s *CombatScene) noteBlock(gs *state.GlobalState, e combat.Event) {
	el := ui.ArtFor(e.Element)
	from, ok := s.takeShield(gs, e.Target, el)
	if e.Target != combat.SideA || e.Slot < 0 || e.Slot >= len(s.Theater.enemyDealt) {
		// Only the player raises shields and only a creature swings at them, so an attack on the
		// player's own row has no seat to land on. The shield goes where it stood.
		if ok {
			s.fadeShield(el, from)
		}
		return
	}
	if !ok {
		// A block with no heap to come out of — a shield nothing on screen raised. It leaves the
		// duelist card, which is whose it was.
		from = s.duelistShieldPose(gs, 0, 1)
	}
	s.Theater.smothers = append(s.Theater.smothers, smother{
		seat: e.Slot, element: el, from: from, surged: e.Surged,
		would: e.Would, ink: ui.ArtFor(s.Theater.enemyDealt[e.Slot].card.Element),
		t: ui.NewTravel(0, smotherTotalTicks()),
	})
}

// noteExpired squares a side's heaps with what an expiry leaves standing: `Life`, which is what a
// worn relic kept and nothing for nearly every run.
func (s *CombatScene) noteExpired(gs *state.GlobalState, e combat.Event) {
	s.trimShields(gs, e.Target, e.Life)
}

// ---- the heaps ----

// add puts n shields of one element on top of a heap, forming one after another.
func (p *shieldPile) add(el cards.Element, n int) {
	start := 0
	for _, sh := range p.shields {
		if !sh.form.Done() {
			start++
		}
	}
	for i := 0; i < n; i++ {
		p.shields = append(p.shields, heapShield{
			element: el,
			form:    ui.NewTravel((start+i)*ui.ShieldStaggerTicks(), ui.ShieldFormTicks()),
		})
	}
}

func (s *CombatScene) pileAtSeat(side combat.Side, seat int) *shieldPile {
	for i := range s.Theater.piles {
		p := &s.Theater.piles[i]
		if p.side == side && !p.onRelic && !p.atDuelist && p.seat == seat {
			return p
		}
	}
	return nil
}

func (s *CombatScene) pileOnRelic(side combat.Side, id combat.RelicID) *shieldPile {
	for i := range s.Theater.piles {
		p := &s.Theater.piles[i]
		if p.side == side && p.onRelic && p.relic == id {
			return p
		}
	}
	return nil
}

func (s *CombatScene) pileAtDuelist(side combat.Side) *shieldPile {
	for i := range s.Theater.piles {
		p := &s.Theater.piles[i]
		if p.side == side && p.atDuelist {
			return p
		}
	}
	s.Theater.piles = append(s.Theater.piles, shieldPile{side: side, seat: -1, atDuelist: true})
	return &s.Theater.piles[len(s.Theater.piles)-1]
}

// takeShield lifts one shield of an element off a side's heaps and reports where it was: the
// newest heap holding one gives up its topmost, or — on a heap that has drifted from the engine —
// the newest heap's top shield whatever it is.
func (s *CombatScene) takeShield(gs *state.GlobalState, side combat.Side, el cards.Element) (ui.ShieldPose, bool) {
	for _, loose := range []bool{false, true} {
		for pi := len(s.Theater.piles) - 1; pi >= 0; pi-- {
			p := &s.Theater.piles[pi]
			if p.side != side {
				continue
			}
			for si := len(p.shields) - 1; si >= 0; si-- {
				if !loose && p.shields[si].element != el {
					continue
				}
				pose, ok := s.shieldPose(gs, *p, si)
				p.shields = append(p.shields[:si:si], p.shields[si+1:]...)
				s.dropEmptyPiles()
				return pose, ok
			}
		}
	}
	return ui.ShieldPose{}, false
}

// fadeShield lets one shield go where it is.
func (s *CombatScene) fadeShield(el cards.Element, pose ui.ShieldPose) {
	s.Theater.fades = append(s.Theater.fades, fadingShield{element: el, pose: pose,
		t: ui.NewTravel(0, shieldFadeTicks())})
}

// fadePile lets every shield on one heap go where it stands.
func (s *CombatScene) fadePile(gs *state.GlobalState, p shieldPile) {
	for i, sh := range p.shields {
		if pose, ok := s.shieldPose(gs, p, i); ok {
			s.fadeShield(sh.element, pose)
		}
	}
}

func (s *CombatScene) dropEmptyPiles() {
	kept := s.Theater.piles[:0]
	for _, p := range s.Theater.piles {
		if len(p.shields) > 0 {
			kept = append(kept, p)
		}
	}
	s.Theater.piles = kept
}

// trimShields leaves a side standing behind n shields: the ones on rings and the duelist card are
// kept first, since those are the ones that outlast a card, and the rest fade.
func (s *CombatScene) trimShields(gs *state.GlobalState, side combat.Side, n int) {
	for _, offTable := range []bool{true, false} {
		for pi := range s.Theater.piles {
			p := &s.Theater.piles[pi]
			if p.side != side || (p.onRelic || p.atDuelist) != offTable {
				continue
			}
			for si := 0; si < len(p.shields); si++ {
				if n > 0 {
					n--
					continue
				}
				if pose, ok := s.shieldPose(gs, *p, si); ok {
					s.fadeShield(p.shields[si].element, pose)
				}
				p.shields = append(p.shields[:si:si], p.shields[si+1:]...)
				si--
			}
		}
	}
	s.dropEmptyPiles()
}

// settleShields squares the player's heaps with the model at the end of a round, as the played
// cards leave the table.
//
// **What the model still has standing outlasts the card that raised it**, and it has to stand on
// something: the relic that kept it — the Tower Shield — or the duelist card when nothing worn
// explains it. Everything else lapses, and fades where it stood. A ring's own heap stays on its
// ring for as long as the model still holds its element.
func (s *CombatScene) settleShields(gs *state.GlobalState, side combat.Side, standing combat.ShieldStack) {
	budget := map[cards.Element]int{}
	for _, e := range combat.AllElements {
		if n := standing[e]; n > 0 {
			budget[ui.ArtFor(e)] += n
		}
	}

	var carried []cards.Element
	piles := s.Theater.piles
	s.Theater.piles = nil
	for _, p := range piles {
		if p.side != side {
			s.Theater.piles = append(s.Theater.piles, p)
			continue
		}
		keep := p
		keep.shields = nil
		for i, sh := range p.shields {
			if budget[sh.element] <= 0 {
				if pose, ok := s.shieldPose(gs, p, i); ok {
					s.fadeShield(sh.element, pose)
				}
				continue
			}
			budget[sh.element]--
			if p.onRelic || p.atDuelist {
				keep.shields = append(keep.shields, sh)
				continue
			}
			carried = append(carried, sh.element)
			if pose, ok := s.shieldPose(gs, p, i); ok {
				s.fadeShield(sh.element, pose)
			}
		}
		if len(keep.shields) > 0 {
			s.Theater.piles = append(s.Theater.piles, keep)
		}
	}
	for _, e := range combat.AllElements {
		for budget[ui.ArtFor(e)] > 0 {
			budget[ui.ArtFor(e)]--
			carried = append(carried, ui.ArtFor(e))
		}
	}
	if len(carried) == 0 {
		return
	}

	var to *shieldPile
	if id, ok := s.keeperRelic(side); ok {
		to = s.pileOnRelic(side, id)
		if to == nil {
			s.Theater.piles = append(s.Theater.piles, shieldPile{side: side, seat: -1, relic: id, onRelic: true})
			to = &s.Theater.piles[len(s.Theater.piles)-1]
		}
	} else {
		to = s.pileAtDuelist(side)
	}
	for _, el := range carried {
		to.add(el, 1)
	}
}

// keeperRelic is the worn relic that keeps shields past their lapse — the Tower Shield — if any.
func (s *CombatScene) keeperRelic(side combat.Side) (combat.RelicID, bool) {
	c := s.fighter
	if side == combat.SideB {
		c = s.enemy
	}
	if c == nil {
		return 0, false
	}
	for _, w := range c.Duelist.WornRelics() {
		if combat.KeptShields([]combat.WornRelic{w}) > 0 {
			return w.Relic, true
		}
	}
	return 0, false
}

// smotheredSeats is which of the opponent's seats a shield has landed on this round.
func (s *CombatScene) smotheredSeats() map[int]bool {
	out := map[int]bool{}
	for _, m := range s.Theater.smothers {
		if m.landed() {
			out[m.seat] = true
		}
	}
	return out
}

// ---- the drawing ----

// shieldPose is where shield i of a heap stands right now, following its card as it lifts and
// moves.
func (s *CombatScene) shieldPose(gs *state.GlobalState, p shieldPile, i int) (ui.ShieldPose, bool) {
	switch {
	case p.atDuelist:
		return s.duelistShieldPose(gs, i, len(p.shields)), true
	case p.onRelic:
		seat, ok := s.wornSeatOf(gs, p.relic)
		if !ok {
			return ui.ShieldPose{}, false
		}
		r := s.relicRow(gs).RowSlot(gs, seat)
		scale := float64(r.Dy()) / cardHeight
		return ui.HeapPose(float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2, i, len(p.shields), scale), true
	}
	r, ok := s.seatRect(gs, p.side, p.seat)
	if !ok {
		return ui.ShieldPose{}, false
	}
	return ui.HeapPose(float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2, i, len(p.shields), 1), true
}

// duelistShieldPose is shield i of n standing on the duelist card.
func (s *CombatScene) duelistShieldPose(gs *state.GlobalState, i, n int) ui.ShieldPose {
	r := ui.DuelistCardRect(gs)
	scale := float64(r.Dy()) / cardHeight
	return ui.HeapPose(float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2, i, n, scale)
}

// seatRect is where a table card is drawn right now, lift included — the same placement its row
// draws it at, so a heap moves with its card.
func (s *CombatScene) seatRect(gs *state.GlobalState, side combat.Side, seat int) (image.Rectangle, bool) {
	if side == combat.SideB {
		return s.enemySeatRect(gs, seat)
	}
	n := len(s.Theater.resolved)
	if n == 0 {
		return image.Rectangle{}, false
	}
	// **A seat the row no longer holds stands over the row's first card** rather than nowhere: a
	// heap appearing over slightly the wrong card is a far smaller lie than shields the player
	// raised not being drawn at all.
	if seat < 0 || seat >= n {
		seat = 0
	}
	at := s.Theater.resolved[seat].at(gs, seat, n, s.playedSplit(), lit(s.Theater.firingSeats, seat))
	return image.Rect(at.X, at.Y, at.X+cardWidth, at.Y+cardHeight), true
}

// enemySeatRect is where one of the opponent's cards is drawn right now. **It asks the row's own
// layout rather than recomputing a seat**, so a shield lands where the card actually is.
func (s *CombatScene) enemySeatRect(gs *state.GlobalState, seat int) (image.Rectangle, bool) {
	if seat < 0 || seat >= len(s.Theater.enemyDealt) {
		return image.Rectangle{}, false
	}
	d := s.Theater.enemyDealt[seat]
	at := s.enemyCardAt(gs, d, seat, len(s.Theater.enemyDealt), s.enemySplit(),
		lit(s.Theater.enemyFiringSeats, seat))
	return image.Rect(at.X, at.Y, at.X+cardWidth, at.Y+cardHeight), true
}

// drawShieldPiles draws every heap, every shield fading, and every shield thrown onto an attack.
func (s *CombatScene) drawShieldPiles(gs *state.GlobalState, screen *ebiten.Image) {
	for _, p := range s.Theater.piles {
		for i, sh := range p.shields {
			pose, ok := s.shieldPose(gs, p, i)
			if !ok || sh.form.Waiting() {
				continue
			}
			ui.DrawShield(screen, sh.element, pose.Forming(sh.form.Progress()))
		}
	}
	for _, f := range s.Theater.fades {
		pose := f.pose
		pose.Alpha *= float32(1 - f.t.Progress())
		ui.DrawShield(screen, f.element, pose)
	}
	for _, m := range s.Theater.smothers {
		r, ok := s.enemySeatRect(gs, m.seat)
		if !ok {
			continue
		}
		drawSmother(gs, screen, m, r)
	}
}

// drawSmother draws one blocked attack: the hit it would have landed on its card, a shield on its
// way to the card or lying on it, the figure fading as the shield lands, and SURGE across the card
// once a matched block has landed.
//
// **The shield grows a little as it lands**, where a damage figure shrinks into a card: a shield is
// not going into the card, it is coming down over it. **The figure is drawn over the shield**, so
// it is seen going rather than simply covered.
func drawSmother(gs *state.GlobalState, screen *ebiten.Image, m smother, card image.Rectangle) {
	mid := image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2)
	if m.t.Age >= smotherLeadTicks() {
		to := ui.ShieldPose{X: float64(mid.X), Y: float64(mid.Y), Deg: smotherDeg, Scale: 1.08, Alpha: 1}
		ui.DrawShield(screen, m.element, ui.LerpPose(m.from, to, ui.EaseOut(m.flying())))
	}
	if m.would > 0 {
		alpha := float32(1)
		if m.landed() {
			alpha = float32(1 - ui.Clamp01(float64(m.sinceLanding())/float64(smotherHoldTicks())))
		}
		if alpha > 0 {
			drawMathText(gs, screen, "-"+strconv.Itoa(m.would), hitFigureSize, hitInkFor(m.ink),
				mid, hitToScale, alpha, false)
		}
	}
	if m.surged && m.landed() {
		drawSurge(gs, screen, card, m.sinceLanding())
	}
}

// surgeSize is SURGE's type size: as big as fits corner to corner across a card.
const surgeSize = 64

// drawSurge writes SURGE across a card at 45 degrees, rising left to right, popping in as it
// arrives. It is in the action-point bar's own amber, because what a surge banked is budget.
func drawSurge(gs *state.GlobalState, screen *ebiten.Image, card image.Rectangle, age int) {
	pop := ui.Clamp01(float64(age) / float64(ui.Beat(1, 4)))
	scale := 1.35 - 0.35*ui.EaseOut(pop)

	op := &text.DrawOptions{}
	op.PrimaryAlign, op.SecondaryAlign = text.AlignCenter, text.AlignCenter
	op.GeoM.Scale(scale, scale)
	op.GeoM.Rotate(-45 * 3.141592653589793 / 180)
	op.GeoM.Translate(float64(card.Min.X+card.Max.X)/2, float64(card.Min.Y+card.Max.Y)/2)
	op.ColorScale.ScaleWithColor(apSpentColor)
	op.ColorScale.ScaleAlpha(float32(pop))
	systems.DrawUI(screen, "SURGE", &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: surgeSize}, op)
}
