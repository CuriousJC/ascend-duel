package screens

// **The band in a fight**: the same controls every screen showing it has, and the three answers only
// a fight gives — the duelist standing in the room has to be kept in step with the row, a relic sold
// mid-fight stops counting at once, and a carried card can be used here as well as sold.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/scenario"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// updateBand runs the band for one frame.
//
// **The drags are live while a round resolves and the sale is not** *(owner's calls, 2026-08-26 and
// 2026-10-04)*. A reorder made during playback cannot reach the round being played — it was decided
// at DUEL! — so it lands on the next one; a sale or a use would change the duelist out from under a
// round already decided, so both wait for planning. A modal covering the screen, or a tutorial step
// holding input elsewhere, takes the whole band with it.
func (s *CombatScene) updateBand(gs *state.GlobalState) {
	s.band.update(gs, bandHooks{
		live:      !s.modalUp(),
		closed:    !s.canSpendRunes(gs),
		moveRelic: func(from, to int) { s.moveRelic(gs, from, to) },
		sellRelic: func(key string) { s.sellRelic(gs, key) },
		canUse: func(seat int) bool {
			return heldUsable(s.consumableSpendable(gs), heldConsumables(gs), seat)
		},
		use:         func(seat int) { s.spendConsumable(gs, seat) },
		forget:      s.tip.Forget,
		extraRelics: len(s.cast),
		relicKeyAt:  func(seat int) (string, bool) { return s.runRelicAt(gs, seat) },
		relicSeat:   func(key string) (image.Rectangle, bool) { return s.runRelicSeat(gs, key) },
	})
}

// sellRelic sells a worn relic in the middle of a fight, and **it stops counting at once** *(owner's
// call, 2026-10-04)*: the duelist standing in the room is refit from the run without it, so the
// next turn resolves as though it had never been worn — its life, its DMG, its clock and its rules
// all go. What has already happened stays happened: the cards it dealt are in the hand as they were
// dealt, and the rounds already played were played.
func (s *CombatScene) sellRelic(gs *state.GlobalState, key string) {
	if !sellWorn(gs, key) {
		return
	}
	s.refit(gs)
	trace.Logf("relics", "sold %s mid-fight, refit: %d/%d life, %d DMG", key,
		s.fighter.CurrentLife, s.fighter.MaxLife, s.fighter.DMG)
}

// equippedFighter is the duelist the run puts in this room: built from the record, equipped with the
// fight's whole relic row — the run's relics and every cantrip-relic cast, in the order the row
// stands in — and given whatever a scenario overrides. **No fight state** — no wound, no shields —
// which is what refit carries over from the duelist standing here.
func (s *CombatScene) equippedFighter(gs *state.GlobalState) combat.Duelist {
	return s.equippedWorn(gs, s.fightWorn(gs))
}

// equippedWorn is equippedFighter with a chosen worn row: refit asks for the run's alone as well, to
// measure what the casts added.
func (s *CombatScene) equippedWorn(gs *state.GlobalState, worn []combat.WornRelic) combat.Duelist {
	d := ui.DuelistFromRecord(gs, ui.PlayerRecord).Duelist
	if gs.Run != nil {
		d = gs.Run.EquipWorn(d, worn)
	}
	if scenario.Active() && scenario.Dummy() {
		d.MaxLife, d.CurrentLife = scenario.DummyLife, scenario.DummyLife
	}
	if scenario.Active() && scenario.Actions() > 0 {
		d.Actions = scenario.Actions()
	}
	return d
}

// refit rebuilds the duelist standing in the fight from what the run now holds, and carries the
// fight across.
//
// **Rebuilt rather than re-equipped.** `Session.Equip` adds what the run carries — the potions, the
// boss bonus, every relic's flat figures — to whatever it is handed, so equipping the fighter
// already standing here would pay all of it a second time. Every mid-fight change to what the run
// carries goes through here: a relic sold, a stone spent, a shower of stones landing.
//
// **What the fight has done is carried, and what the run carries is not.** The purse is the run's,
// which is where a sale's proceeds have just gone — the rules take it back at the next DUEL!. The wound stays a
// wound — life is the new ceiling less what the fight has taken, never below one — and the standing
// shields, the banked surge and each remaining relic's growth this fight come across
// as they are. Every cantrip-relic cast this fight is worn again, in cast order after the run's,
// because a cantrip lasts the fight and is not the run's — and **a cast is itself a refit**, which is
// how a relic put on mid-fight answers `equipped` over the whole worn set.
func (s *CombatScene) refit(gs *state.GlobalState) {
	live := s.fighter.Duelist
	d := s.equippedFighter(gs)

	// **What the casts added is measured against the same fighter built without them**, so a
	// cantrip-relic's life and DMG are told apart from the run's however the two compound.
	var bare combat.Duelist
	if gs.Run != nil {
		bare = s.equippedWorn(gs, gs.Run.WornRelics())
	} else {
		bare = s.equippedWorn(gs, nil)
	}
	s.cantripLife, s.cantripDMG = d.MaxLife-bare.MaxLife, d.DMG-bare.DMG

	wound := live.MaxLife - live.CurrentLife
	d.CurrentLife = min(max(d.MaxLife-wound, 1), d.MaxLife)
	d.Shields, d.Surge = live.Shields, live.Surge
	for i := range d.Relics {
		for _, w := range live.Relics {
			if w.Relic == d.Relics[i].Relic {
				d.Relics[i].Grown = w.Grown
			}
		}
	}
	s.fighter.Duelist = d
}

// heldUsable is whether the carried card in one seat could be spent right now, for the USE tab.
func heldUsable(spendable func(session.Consumable) bool, held []session.Consumable, seat int) bool {
	return spendable != nil && seat >= 0 && seat < len(held) && spendable(held[seat])
}
