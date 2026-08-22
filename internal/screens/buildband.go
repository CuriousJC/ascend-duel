package screens

// **The band that says what you are: your duelist card and the rings on your fingers.**
//
// The combat screen has drawn this since the character block became a card — a duelist card in the
// top-left corner and the worn rings in a row beside it. **The reward screen wants the same thing**
// *(owner's call, 2026-08-22)*: the payout it narrates lands on the purse written on that card, and
// choosing a worm is a choice about a deck you can only judge against the build you are holding.
//
// **It is a free function over the run rather than a method on a scene**, which is what lets a
// second screen draw it. What it deliberately does *not* do is move: the combat screen's own band
// is still its own — it draws a live fighter, mid-fight life, banked AP and an opponent's card at
// the far end, none of which exist here. This is the between-fights view of the same thing, and the
// day the shop wants one too it is one call.
//
// **Rings are laid out by the same functions the combat screen uses** — `ringSlotAt`, `wornRings` —
// so the row cannot drift between the two screens. The pane rectangle is the only thing computed
// here, because there is no enemy card to end it at.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// buildBandRightPct is where the ring row stops. **It mirrors duelistCardLeftPct**, exactly as the
// combat screen's enemy card does — there is no opponent on this screen, so the row simply runs to
// the far margin.
const buildBandRightPct = 99

// buildCardRect is where the duelist card sits: the same corner it occupies in a fight, so the
// player's card does not move between the duel and the screen that follows it.
func buildCardRect(gs *state.GlobalState) image.Rectangle {
	left, top := gs.PctX(duelistCardLeftPct), gs.PctY(topRowTopPct)
	return image.Rect(left, top, left+cards.DuelistStyle.Width, top+cards.DuelistStyle.Height)
}

// buildRingRect is the row's extent, taken off the duelist card beside it for the reason the
// combat screen's is: whichever card moves, the row follows.
func buildRingRect(gs *state.GlobalState) image.Rectangle {
	card := buildCardRect(gs)
	top := card.Min.Y + ringPaneTopDrop
	return image.Rect(card.Max.X+ringPaneGap, top,
		gs.PctX(buildBandRightPct), top+cards.RingStyle.Height)
}

// buildBandBottom is where the band ends, so a screen below it knows what it has left.
func buildBandBottom(gs *state.GlobalState) int {
	return buildCardRect(gs).Max.Y
}

// drawBuildBand puts the whole thing up: the duelist as they came out of the fight, and the rings
// they are wearing.
//
// **Life is the run's `LifeLeft`, not a full bar.** The fight is over and the card still says what
// it cost — a win on nine life reads as one, and it is also the figure the payout was a tenth of.
//
// The AP figure is the duelist's own budget with no bank on it, because a bank is a thing that
// exists inside a round.
func drawBuildBand(gs *state.GlobalState, screen *ebiten.Image, vitae int) {
	if gs.Run == nil {
		return
	}

	if fighter := buildFighter(gs); fighter != nil {
		name := fighter.Name
		if name == "" {
			name = duelistName
		}
		spec := duelistSpec(fighter, name, vitae, gs.Run.LifeLeft(), fighter.ActionPoints())
		if img := cardImage(gs, spec, cards.DuelistStyle); img != nil {
			r := buildCardRect(gs)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
			screen.DrawImage(img, op)
		}
	}

	row := buildRingRect(gs)
	worn := wornRings(gs)
	for i, record := range worn {
		at := ringSlotAt(row, i, len(worn))
		drawRingCard(gs, screen, at, record, true)
	}
}

// buildFighter is the player as a combatant, equipped with what they are wearing.
//
// **It is rebuilt rather than kept.** The fighter that fought is the combat screen's and dies with
// it; what survives is the run, and the run is enough to say what the player *is*. The one thing it
// cannot say is mid-fight life, which is why `LifeLeft` is stored.
func buildFighter(gs *state.GlobalState) *entities.Combatant {
	c := duelistFromRecord(gs, playerRecord)
	if c == nil {
		return nil
	}
	c.Duelist = gs.Run.Equip(c.Duelist)
	return c
}
