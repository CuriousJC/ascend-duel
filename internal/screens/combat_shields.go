package screens

// The shields on the table: heaped over whatever raised them, and thrown to meet the attacks they eat.
//
// **A shield forms over the card that made it and stays there** — a defend card, an attack card
// with a shield rider, or the ring of a relic that raises one. Several on one card heap, which is
// how the count is read: see ui/shield_heap.go. **They form once the turn's arithmetic is done**
// *(owner's call)*: every hit's line is worked out first, a defense's coming to 0, and the turn's
// shields go up together on the way out of it.
//
// **When the creature swings at a shield, the hit sets off and a shield leaves its heap to meet it**,
// and the two break apart where they meet on its path — on the beat that attack is reached, so what
// stops a hit is watched stopping it. See combat_intercept.go.
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

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
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

	// from is where a shield flying in set off — a stagger's ring — and flies says to draw the
	// forming as that journey rather than as a drop into place. See flyStaggers.
	from  ui.ShieldPose
	flies bool
}

// shieldFlyTicks is a stagger's shield crossing from its ring to its card: the hit's own flight, so
// the shield and the figure leave together and the card has its shield by the time the blow lands.
func shieldFlyTicks() int { return hitFlyTicks() }

// flyStaggers sends every shield a stagger raised over one card out of its ring and onto that card,
// on the beat the card's hit sets off. **It reads ahead in the log** — the staggers are announced as
// the turn closes, after every hit — which is the hand dialog's licence and stays inside the turn:
// the walk stops at the first event of the other side's. A stagger flown here is walked past, so
// the turn's bundle does not form it a second time; one this does not reach (a card whose line was
// never thrown) still forms over its card with the rest.
func (s *CombatScene) flyStaggers(gs *state.GlobalState, side combat.Side, seat int) {
	for at := s.cursor; at < len(s.log); at++ {
		e := s.log[at]
		if e.Kind == combat.KindRoundEnd || e.Side != side {
			break
		}
		if e.Kind != combat.KindStaggered || e.Slot != seat || e.Amount <= 0 || s.Theater.walked[at] {
			continue
		}
		s.markShown(at)
		from, ok := s.relicShieldPose(gs, e.Relic)
		if !ok {
			// No ring on screen to leave from: the shield forms where it lands, like a raise.
			s.Theater.pendingRaises = append(s.Theater.pendingRaises, e)
			continue
		}
		if i, ok := s.wornSeatOf(gs, e.Relic); ok {
			s.shakeRelicAt(i)
		}
		p := s.pileAtSeat(e.Side, e.Slot)
		if p == nil {
			s.Theater.piles = append(s.Theater.piles, shieldPile{side: e.Side, seat: e.Slot})
			p = &s.Theater.piles[len(s.Theater.piles)-1]
		}
		for i := 0; i < e.Amount; i++ {
			p.shields = append(p.shields, heapShield{
				element: ui.ArtFor(e.Element),
				form:    ui.NewTravel(i*ui.ShieldStaggerTicks(), shieldFlyTicks()),
				from:    from,
				flies:   true,
			})
		}
	}
}

// relicShieldPose is a shield sitting on a worn ring, at the ring's own scale: where a stagger's
// shield sets off from.
func (s *CombatScene) relicShieldPose(gs *state.GlobalState, id combat.RelicID) (ui.ShieldPose, bool) {
	seat, ok := s.wornSeatOf(gs, id)
	if !ok {
		return ui.ShieldPose{}, false
	}
	r := s.relicRow(gs).RowSlot(gs, seat)
	scale := float64(r.Dy()) / cardHeight
	return ui.HeapPose(float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2, 0, 1, scale), true
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

// shieldFadeTicks is a shield going away where it stood.
func shieldFadeTicks() int { return ui.Beat(1, 2) }

// ---- the events ----

// noteShieldRaise holds a raise until the turn's arithmetic is done, and reports that it took it.
// **Each heap stands over its own card, named by `Event.Slot`** — nothing is lit during the defend
// phase, so there is no lit card to read a seat off.
func (s *CombatScene) noteShieldRaise(e combat.Event) bool {
	if e.Kind != combat.KindRaised && e.Kind != combat.KindStaggered {
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
//
// **A fight that has just been won or lost forms nothing** *(owner's call)*. Once a fighter is down
// there is no turn left to defend, so shields going up over the body would be the defend phase
// carrying on after the duel ended: they are dropped. The life is read off the screen's own model,
// which a killing line has already written.
func (s *CombatScene) formPendingShields() bool {
	pending := s.Theater.pendingRaises
	s.Theater.pendingRaises = nil
	if s.fightDecided() {
		return false
	}
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

// fightDecided reports that a fighter on screen is down, which is the end of anything defensive.
func (s *CombatScene) fightDecided() bool {
	return (s.fighter != nil && !s.fighter.Alive()) || (s.enemy != nil && !s.enemy.Alive())
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
	at := s.Theater.resolved[seat].at(gs, n)
	return image.Rect(at.X, at.Y, at.X+cardWidth, at.Y+cardHeight), true
}

// enemySeatRect is where one of the opponent's cards is drawn right now. **It asks the row's own
// layout rather than recomputing a seat**, so a shield lands where the card actually is.
func (s *CombatScene) enemySeatRect(gs *state.GlobalState, seat int) (image.Rectangle, bool) {
	if seat < 0 || seat >= len(s.Theater.enemyDealt) {
		return image.Rectangle{}, false
	}
	d := s.Theater.enemyDealt[seat]
	at := s.enemyCardAt(gs, d, len(s.Theater.enemyDealt))
	return image.Rect(at.X, at.Y, at.X+cardWidth, at.Y+cardHeight), true
}

// drawShieldPiles draws every heap and every shield fading. A shield thrown at a hit is the
// intercept's to draw — see drawIntercepts.
func (s *CombatScene) drawShieldPiles(gs *state.GlobalState, screen *ebiten.Image) {
	for _, p := range s.Theater.piles {
		for i, sh := range p.shields {
			pose, ok := s.shieldPose(gs, p, i)
			if !ok || sh.form.Waiting() {
				continue
			}
			if sh.flies {
				ui.DrawShield(screen, sh.element, ui.LerpPose(sh.from, pose, ui.EaseOut(sh.form.Progress())))
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
}
