package screens

// **The band that says what you are: your duelist card and the relics on your fingers.**
//
// The combat screen has drawn this since the character block became a card — a duelist card in the
// top-left corner and the worn relics in a row beside it. **The reward screen wants the same thing**
// *(owner's call, 2026-08-22)*: the payout it narrates lands on the purse written on that card, and
// choosing an essence is a choice about a deck you can only judge against the build you are holding.
//
// **The shop, a sealed good and the portal draw it too**, and **The Prismatic stands in the
// opponent's corner** on all of them *(owner's call, 2026-10-04)*, so the panes span the gap between
// two corner cards exactly as they do in a fight. The shop draws its relic row itself, because its
// row moves when a relic is bought or sold.
//
// **It is a free function over the run rather than a method on a scene**, which is what lets every
// screen draw it. The combat screen draws its own live fighter, mid-fight life, standing shields and
// the creature — the between-fights view is this one — but **the input is shared everywhere**: see
// band.go.
//
// **Relics are laid out by the same functions the combat screen uses** — `relicSlotAt`,
// `wornRelics`, `buildTopRowPanes` — so the row cannot drift between screens.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// buildCardRect is where the duelist card sits: the same corner it occupies in a fight, so the
// player's card does not move between the duel and the screen that follows it.
func buildCardRect(gs *state.GlobalState) image.Rectangle { return ui.DuelistCardRect(gs) }

// buildOpponentRect is the far corner: the creature in a fight, and The Prismatic on every other
// screen that shows the band. **The same seat everywhere** *(owner's call, 2026-10-04)*, so the two
// panes between the cards are the same width on every screen and nothing in the band moves when a
// fight begins or ends.
func buildOpponentRect(gs *state.GlobalState) image.Rectangle { return ui.EnemyCardRect(gs) }

// buildRelicRect is the row's extent, taken off the duelist card beside it for the reason the
// combat screen's is: whichever card moves, the row follows.
func buildRelicRect(gs *state.GlobalState) image.Rectangle {
	relics, _ := buildTopRowPanes(gs)
	return relics
}

// buildConsumableRect is the row's other half: the runes the run is carrying, in the pane the
// combat screen draws in the same place. See consumables.go.
func buildConsumableRect(gs *state.GlobalState) image.Rectangle {
	_, consumables := buildTopRowPanes(gs)
	return consumables
}

// buildTopRowPanes is the split, over the span between the two corner cards: **the combat screen's
// own span**, so every screen showing the band lays the relics and the consumables out the same.
func buildTopRowPanes(gs *state.GlobalState) (relics, consumables image.Rectangle) {
	left, right := relicRowSpan(gs)
	return topRowPanes(left, right, buildCardRect(gs).Min.Y+relicPaneTopDrop)
}

// buildBandBottom is where the band ends, so a screen below it knows what it has left.
//
// **It is the lower of the two, and that is not pedantry** *(2026-09-05)*. The relic row is dropped
// relicPaneTopDrop below the duelist card's top and is the same height, so the row ends that many
// pixels *under* the card — and everything that placed itself below the band was reading the card
// alone and landing inside the relics. That was invisible while both were 224 tall on a 960-high
// screen and the numbers under it were written by eye; the cards grew a quarter on 2026-09-04 and
// the reward screen's narration ended up struck through by a worn relic.
func buildBandBottom(gs *state.GlobalState) int {
	card := buildCardRect(gs).Max.Y
	if relics := buildRelicRect(gs).Max.Y; relics > card {
		return relics
	}
	return card
}

// drawBuildBand puts the whole thing up: the duelist as they came out of the fight, and the relics
// they are wearing.
//
// **Life is the run's `LifeLeft`, not a full bar.** The fight is over and the card still says what
// it cost — a win on nine life reads as one, and it is also the figure the payout was cut in thirds by.
//
// The AP figure is the duelist's own budget, which is now simply the stat — nothing adds to it any
// more. No shields either: nothing is standing between fights.
func drawBuildBand(gs *state.GlobalState, screen *ebiten.Image, vitae int, band *bandControls, raise bool) {
	drawBuildCard(gs, screen, vitae)
	drawBuildRelics(gs, screen, &band.relicDrag)
	// **nil: a rune is carried on these screens, not spent.** The pane draws the same two seats
	// and the same cards, dim, and the tooltip still explains them. See canSpend.
	drawConsumablePane(gs, screen, buildConsumableRect(gs), nil, band.heldSkip, raise)
	drawGuideCard(gs, screen)
}

// drawBandOverlay is what the band draws over everything below it: the card riding the cursor and
// the tabs under whatever is armed. Called after the screen's own content, before its tooltip.
func drawBandOverlay(gs *state.GlobalState, screen *ebiten.Image, band *bandControls) {
	drawDraggedRelic(gs, screen, &band.relicDrag, runCounters(gs))
	band.drawHeldGhost(gs, screen, nil)
	band.sale.drawTab(gs, screen)
}

// drawGuideCard puts The Prismatic in the opponent's corner.
func drawGuideCard(gs *state.GlobalState, screen *ebiten.Image) {
	ui.BlitCard(gs, screen, buildOpponentRect(gs).Min, ui.GuideSpec(gs), cards.GuideStyle)
}

// hoverGuideCard explains The Prismatic, and reports whether the cursor was on it.
func hoverGuideCard(gs *state.GlobalState, at image.Point, tip *models.Tooltip) bool {
	seat := buildOpponentRect(gs)
	if !at.In(seat) {
		return false
	}
	title, lines := ui.GuideTip()
	tip.Point(seat, ui.TipLine(title), ui.TipLines(lines))
	return true
}

// drawBuildCard is the duelist half of the band on its own.
//
// **It is split out for the shop** *(2026-08-22)*, which draws the relic row itself: a relic there is
// a thing you can sell, so it carries a sell tab when armed and it moves when the row re-centers. The
// row's *geometry* is still `buildRelicRect` and `relicSlotAt`, so the two screens cannot disagree
// about where a finger is — only about what is drawn on it.
func drawBuildCard(gs *state.GlobalState, screen *ebiten.Image, vitae int) {
	if gs.Run == nil {
		return
	}

	if fighter := ui.BuildFighter(gs); fighter != nil {
		name := fighter.Name
		if name == "" {
			name = ui.DuelistName
		}
		// **The life is the ceiling less the wound, not the figure the last fight ended on**
		// *(2026-09-06)*. It was `LifeLeft`, and the two agree the moment a fight is won — WonFight
		// sets the wound from exactly that subtraction — so nothing looked wrong until something
		// changed one of them *between* fights. Two things now do: a Salve takes the wound down and
		// a portal-room win clears it outright, and both left this card drawing the number the player
		// walked out of the last room with.
		//
		// **The ceiling is the fighter's**, so it already carries the potions, the boss bonus and
		// every relic — see Session.Equip.
		life := gs.Run.LifeAtFightStart(fighter.MaxLife)

		// No shields: this is the build band between fights, where nothing has been raised and
		// nothing is standing.
		spec := ui.DuelistSpec(gs, fighter, name, fighter.DMG, vitae, life, fighter.MaxLife,
			fighter.ActionPoints(), fightIndex(gs.Run), 0)
		if img := ui.CardImage(gs, spec, cards.DuelistStyle); img != nil {
			r := buildCardRect(gs)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
			screen.DrawImage(img, op)
		}
	}
}

// drawBuildRelics is the row of worn relics: what is on your fingers, in firing order.
//
// **`drag` is the press in progress over it**, and it may be nil for a caller that does not let the
// row be reordered. Nothing passes nil today; the parameter is there so that a screen putting the
// band up to be *read* does not have to invent a controller nobody drives.
func drawBuildRelics(gs *state.GlobalState, screen *ebiten.Image, drag *ui.CardDrag) {
	if gs.Run == nil {
		return
	}

	row := buildRelicRect(gs)
	drawRelicPaneFrame(gs, screen, row)
	worn := wornRelics(gs)
	counters := runCounters(gs)
	for i, record := range worn {
		// The seat a dragged relic left stays empty; see the combat screen's row.
		if drag != nil && drag.Dragging() && i == drag.Origin() {
			continue
		}
		at := relicSlotAt(row, i, len(worn))
		ui.DrawFloatingCard(gs, screen, at, i,
			ui.RelicSpec(gs, record, counters[record.RelicRecord], true, false), cards.RelicStyle)
	}
	// The relic riding the cursor is the band's overlay, drawn over everything — see
	// drawBandOverlay.
}

// buildRelicRow is the band's row, addressed by the shared drag: the reward screen's and the shop's.
//
// **There is no live duelist on either screen**, so the run is the only copy of the row and
// moveWornRelic is the whole move. The combat screen's is the one with more to do — see
// CombatScene.moveRelic.
func buildRelicRow(gs *state.GlobalState, click func(i int)) relicRow {
	return relicRow{
		rect:  buildRelicRect(gs),
		worn:  len(wornRelics(gs)),
		click: click,
		move:  func(from, to int) { moveWornRelic(gs, from, to) },
	}
}

// hoverBuildRelics explains whichever worn relic the cursor is resting on, and reports whether it
// found one.
//
// **The band draws relics on three screens and only one of them explained them** *(2026-08-22)*.
// The combat screen has `hoverRelics`, the shop grew its own loop, and the reward screen had
// neither — so the row a player reads their build off went silent on exactly the screen where they
// are choosing what to do to that build. It is one function now, over the same geometry the row is
// drawn with, so a fourth screen putting the band up cannot forget again.
//
// **The tooltip says where a relic sits in the firing order**, which is what `relicTip` is for: worn
// order is firing order, and the row is the only place that fact is visible.
func hoverBuildRelics(gs *state.GlobalState, at image.Point, tip *models.Tooltip) bool {
	if gs.Run == nil {
		return false
	}

	// **The consumables pane is asked first and through the same door** *(2026-09-06)*. It is the
	// other half of the row this function already owns, and a second call at every site is the
	// forgetting hoverBuildRelics exists to prevent.
	// **No hand to clamp against**: an essence carried past a between-fights screen is spent in a
	// fight that has not been dealt, so what the pane can say is what the essence will do.
	if hoverConsumables(gs, buildConsumableRect(gs), at, tip, runEssenceTargets(gs)) {
		return true
	}
	if hoverGuideCard(gs, at, tip) {
		return true
	}

	row := buildRelicRect(gs)
	worn := gs.Run.Worn()

	// **ui.HoveredSeat, like every other row** — the band's fingers overlap once a run wears enough
	// of them, and the relic on top is the last drawn.
	i := ui.HoveredSeat(at, len(worn), func(i int) image.Rectangle {
		return relicSlotRect(row, i, len(worn))
	})
	if i < 0 {
		return false
	}
	record, ok := gs.Relics[worn[i]]
	if !ok {
		return false
	}
	seat := relicSlotRect(row, i, len(worn))
	title, lines := ui.RelicTip(record)
	// **To the left of the card, never under it** *(owner's call)*: a panel under the top row
	// covers the pane's count and the tab a click hangs there. PointLeft flips right at the
	// screen's edge.
	tip.PointLeft(seat, ui.TipLine(title), ui.TipLines(lines))
	return true
}

// relicSlotRect is one finger as a rectangle. `relicSlotAt` answers with the point a card is drawn
// from; **anything hit-testing the row needs the same seat as an area**, and deriving it twice is
// the drawn-here-clicked-there bug every other row in this game is shaped to avoid.
func relicSlotRect(row image.Rectangle, i, worn int) image.Rectangle {
	at := relicSlotAt(row, i, worn)
	return image.Rect(at.X, at.Y, at.X+cards.RelicStyle.Width, at.Y+cards.RelicStyle.Height)
}
