package screens

// combat_consume.go — a consumable spent mid-fight, played as `ui.Consume`: lifted out of its seat
// in the pane to the middle of the table, used up there in its own way, and its motes carried to
// whatever it changed.
//
//   - **A rune or an essence** sends its motes to the hand cards it changed, and their morphs are
//     put back to land with the motes rather than a beat after the click — `raiseHandMorphs` raises
//     them as it always did and this only moves their start.
//   - **A stone** sends its motes to the hands button, the ladder it raised.
//   - **A cantrip** sends its embers to the seat its relic takes in the band, and the relic arrives
//     there out of nothing as they land; the pane leaves that seat empty until it has.
//
// **The run changed before any of this is drawn**, the rule every mover here is under: the card has
// left the pane, the cards in the hand are already the new cards and are selectable while they
// change, and nothing waits on the sequence.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// combatUse is one consumable being used on this screen.
type combatUse struct {
	ui.Consume

	// relicSeat is the band seat a cantrip's relic is arriving in, which the pane draws empty until
	// the arrival has finished; -1 for every other kind.
	relicSeat int
}

// useStage is the card-sized seat in the middle of the table a consumable is used in.
func useStage(gs *state.GlobalState) image.Rectangle {
	c := tableCenter(gs)
	return image.Rect(c.X-cardWidth/2, c.Y-cardHeight/2, c.X+cardWidth/2, c.Y+cardHeight/2)
}

// useFrom is the pane seat the consumable being spent stood in, read before the pane closes over
// it; the stage itself when no seat was recorded, so the card is simply used where it stands.
func (s *CombatScene) useFrom(gs *state.GlobalState) image.Rectangle {
	if s.spendSeat < 0 {
		return useStage(gs)
	}
	r := consumableSlotRect(s.consumablePaneRect(gs), s.spendSeat, consumableSeats(gs))
	return image.Rect(r.Min.X, r.Min.Y, r.Min.X+cardWidth, r.Min.Y+cardHeight)
}

// raiseUse starts one consumable's sequence and returns where it was put.
func (s *CombatScene) raiseUse(gs *state.GlobalState, exit ui.ConsumeExit, face cards.Spec,
	to []image.Rectangle, changes []ui.Morph, motes ui.Emitter) *combatUse {

	c := ui.NewConsume(ui.Consume{
		Exit:    exit,
		Face:    face,
		Style:   cards.EssenceStyle,
		From:    s.useFrom(gs),
		Stage:   useStage(gs),
		To:      to,
		Changes: changes,
		Motes:   motes,
		Seed:    cards.MarkSeed(face.Name) + uint32(s.round),
	})
	s.uses = append(s.uses, combatUse{Consume: c, relicSeat: -1})
	return &s.uses[len(s.uses)-1]
}

// useHandMotes sends a rune's or an essence's motes to every hand card raiseHandMorphs has just
// started changing, and puts those morphs back to begin as the motes land. `from` is how many morphs
// were running before this spend, so only this spend's are moved.
func (s *CombatScene) useHandMotes(gs *state.GlobalState, exit ui.ConsumeExit, face cards.Spec,
	from int, motes ui.Emitter) {

	var to []image.Rectangle
	for _, h := range s.Theater.morphs[from:] {
		to = append(to, image.Rect(h.at.Min.X, h.at.Min.Y, h.at.Min.X+cardWidth, h.at.Min.Y+cardHeight))
	}
	u := s.raiseUse(gs, exit, face, to, nil, motes)
	for i := from; i < len(s.Theater.morphs); i++ {
		s.Theater.morphs[i].m = s.Theater.morphs[i].m.Delayed(u.ArrivalTicks())
	}
}

// useStoneMotes sends a stone's motes to the hands button: the ladder the stone raised.
func (s *CombatScene) useStoneMotes(gs *state.GlobalState, face cards.Spec) {
	s.raiseUse(gs, ui.ExitCrumble, face, []image.Rectangle{handsTarget(gs)}, nil, stoneMotes())
}

// handsTarget is the hands button as a destination: the ladder a stone raises. **The same slot on
// every screen that has the button**, which is why the shop and the fight can share it.
func handsTarget(gs *state.GlobalState) image.Rectangle {
	c := ControlColumnSlotCenter(gs, SlotHands)
	return image.Rect(c.X-30, c.Y-30, c.X+30, c.Y+30)
}

// stoneMotes is the dust a crumbled stone sends to the ladder.
func stoneMotes() ui.Emitter {
	return ui.Emitter{Element: cards.Earth, Rate: 1.2, Size: 22,
		Sprites: ui.FirstDrawn(ui.RubbleSprites, []string{"particle-earth-1", "particle-earth-2"})}
}

// stoneUseFrom is a stone being used out of a seat on a screen with no table: lifted to the middle
// of the screen, crumbled there, its dust sent to the hands button. The shop's pouch.
func stoneUseFrom(gs *state.GlobalState, face cards.Spec, from image.Rectangle) ui.Consume {
	cx, cy := gs.ScreenWidth/2, gs.ScreenHeight/2
	return ui.NewConsume(ui.Consume{
		Exit:  ui.ExitCrumble,
		Face:  face,
		Style: cards.EssenceStyle,
		From:  image.Rect(from.Min.X, from.Min.Y, from.Min.X+cardWidth, from.Min.Y+cardHeight),
		Stage: image.Rect(cx-cardWidth/2, cy-cardHeight/2, cx+cardWidth/2, cy+cardHeight/2),
		To:    []image.Rectangle{handsTarget(gs)},
		Motes: stoneMotes(),
		Seed:  cards.MarkSeed(face.Name),
	})
}

// useCantripMotes burns the scroll and carries its relic into the band seat it now takes.
func (s *CombatScene) useCantripMotes(gs *state.GlobalState, face cards.Spec) {
	seat := -1
	for i, r := range s.relicRowSeats(gs) {
		if !r.runRelic() && r.cast == len(s.cast)-1 {
			seat = i
		}
	}
	motes := ui.Emitter{Element: cards.Fire, Set: ui.ParticleMixed, Rate: 1.2, Size: 22}
	if seat < 0 {
		s.raiseUse(gs, ui.ExitBurn, face, nil, nil, motes)
		return
	}
	worn := s.paneRelics(gs)
	at := relicSlotAt(s.relicPaneRect(gs), seat, len(worn))
	rect := image.Rect(at.X, at.Y, at.X+cards.RelicStyle.Width, at.Y+cards.RelicStyle.Height)
	relic := ui.RelicSpec(gs, worn[seat], s.countersNow()[worn[seat].RelicRecord], true, false)
	u := s.raiseUse(gs, ui.ExitBurn, face, []image.Rectangle{rect},
		[]ui.Morph{ui.MorphIn(relic, cards.RelicStyle)}, motes)
	u.relicSeat = seat
}

// relicSeatArriving reports whether a cantrip's relic is still on its way into this band seat, which
// the pane then leaves empty.
func (s *CombatScene) relicSeatArriving(seat int) bool {
	for _, u := range s.uses {
		if u.relicSeat == seat && len(u.Changes) > 0 && !u.Changes[0].Done() {
			return true
		}
	}
	return false
}

// updateUses advances every consumable being used and drops the finished ones.
func (s *CombatScene) updateUses() {
	kept := s.uses[:0]
	for i := range s.uses {
		s.uses[i].Tick()
		if !s.uses[i].Done() {
			kept = append(kept, s.uses[i])
		}
	}
	s.uses = kept
}

// drawUses draws them, over everything: a card crossing to the stage passes the band, the table and
// the hand, and one that went behind any of them would read as having been dropped.
func (s *CombatScene) drawUses(gs *state.GlobalState, screen *ebiten.Image) {
	for _, u := range s.uses {
		u.Draw(gs, screen)
	}
}
