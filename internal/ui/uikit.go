package ui

import (
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/pyramid"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The handful of things the shared drawing layer needs that had come to live inside one screen.
//
// **Every one of these was a boundary already broken.** They are helpers, constants and colors
// with nothing about a particular screen in them, and each ended up wherever it was first needed —
// a clamp in the hits file, a blue in the combat scene, a button helper beside the DUEL! button.
// Left there, the shared layer cannot be lifted out without reaching back down into a scene, which
// is the shape a package boundary exists to make impossible.

// DuelistFromRecord resolves a playable duelist. **No sheet to look up** — the character
// block replaced the fighter's sprite, so a duelist record has no picture in it.
func DuelistFromRecord(gs *state.GlobalState, record string) *entities.Combatant {
	return entities.NewDuelistFrom(gs.Duelists[record])
}

// SetEnabled flips a button between disabled and normal without clobbering a hover or
// press it is in the middle of.
func SetEnabled(b *models.Button, enabled bool) {
	if !enabled {
		b.State = models.ButtonStateDisabled
		return
	}
	if b.State == models.ButtonStateDisabled {
		b.State = models.ButtonStateNormal
	}
}

// panelBlue is the stroke around a dialog and the toast's dismiss button. It was the
// action-point bar's own blue until 2026-09-10, when the bar's cells went red and the name
// stopped describing what it does — every remaining caller is a panel edge, so it is named
// for that rather than for the widget it used to belong to.
var panelBlue = color.RGBA{R: 70, G: 130, B: 230, A: 255}

func Abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Clamp01 holds a progress figure inside its range. The flight's own clock runs past the flight so
// the hold can be measured off it, so the traveled fraction has to be clamped rather than trusted.
func Clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// sort, or the row closing up after cards were spent. Shorter than the other three
// journeys because it is the shortest one: a few inches across the row rather than a
// trip across the screen, and a long ease over that distance reads as sluggish rather
// than as deliberate.
func SlideTicks() int { return Beat(1, 2) }

// runeSpec is a rune drawn as a card.
//
// **The picture comes off the record** *(2026-09-12)*, through `data.RuneData.ArtKey`, already
// resolved by the time a `session.Rune` exists. It borrowed the essence's placeholder through one
// constant until then, and the note on that constant said the day runes got art it should be a
// `data/runes.json` field appearing rather than a fallback being unpicked — so the fallback is
// `assets/rune/default-rune.png` now, a seat of the catalog's own.
//
// **The card says nothing at all** *(owner's call, 2026-09-16)*. It carried its authored line
// across the lower half of its picture, and what that cost is the picture: a rune is a full-bleed
// card, so the sentence is a scrim over the one thing on the card worth looking at. The line has
// not gone anywhere — it is in the tooltip, which is where a player who does not recognize a
// picture yet already goes, and where there is room to say the half a face cannot fit. Same call
// the sealed goods took on 2026-09-15 and the stones took with this one.
//
// **The chimera's answer went with it.** Its face printed `COPIES <name>` because its authored line
// cannot say what it would fire; that is now the tooltip's first line — see runeTipLines.
func runeSpec(gs *state.GlobalState, p session.Rune, enabled, selected bool) cards.Spec {
	return cards.Spec{
		Name:     p.Name,
		Form:     cards.FormNone,
		Element:  cards.Basic,
		Art:      Artwork(gs, p.Art),
		Enabled:  enabled,
		Selected: selected,
	}
}

// ShieldPipKey is the shield drawing for one element, neutral for anything that is not one of the
// five. Spelled here rather than exported from internal/cards because the key is a fact about the
// asset family and both packages build it the same way off a form and an element.
func ShieldPipKey(e cards.Element) string {
	switch e {
	case cards.Fire, cards.Ice, cards.Lightning, cards.Earth, cards.Arcane:
		return "formdefend-" + e.String()
	}
	return "formdefend-neutral"
}

// TowerRoomNames is what each of a floor's three fights is called, in order. Indexed by the
// fight's position within its floor, so it must stay fightsPerFloor long — the two are checked
// against each other by TestEveryRoomOnAFloorIsNamed.
var TowerRoomNames = [FightsPerFloor]string{"Outer Room", "Inner Room", "Stairway"}

// TowerFloor is which floor a fight is on, counting from one.
//
// **It is not capped at the tower's eight.** The fight order is every record in the roster —
// 96 of them, scaffolding for a generator that does not exist — so playing far enough reads
// Floor 9 and beyond. A clamp would be a screen quietly disagreeing with the counter it is
// drawing; the honest fix is the tower, not a maximum here.
func TowerFloor(fight int) int { return fight/FightsPerFloor + 1 }

// TowerRoom names which of its floor's fights this is.
func TowerRoom(fight int) string { return TowerRoomNames[fight%FightsPerFloor] }

// BuildFighter is the player as a combatant, equipped with what they are wearing.
//
// **It is rebuilt rather than kept.** The fighter that fought is the combat screen's and dies with
// it; what survives is the run, and the run is enough to say what the player *is*. The one thing it
// cannot say is mid-fight life, which is why `LifeLeft` is stored.
func BuildFighter(gs *state.GlobalState) *entities.Combatant {
	c := DuelistFromRecord(gs, PlayerRecord)
	if c == nil {
		return nil
	}
	c.Duelist = gs.Run.Equip(c.Duelist)
	return c
}

// FightsPerFloor is how many fights a floor holds, and the third of them is its boss.
//
// **It is pyramid.FightsPerFloor rather than a 3 of a screen's own**, because the ascent curve
// that grows an enemy per room reads the same number. Two copies would let the label and the
// difficulty disagree about how deep a floor is.
const FightsPerFloor = pyramid.FightsPerFloor

// dragThreshold is how far the cursor has to travel with the button held before a press counts as
// a drag rather than a click. Without it every click would jitter into a one-pixel reorder and
// selecting a card would be a coin toss.
const dragThreshold = 4

// The slot beside the draw pile: a square sharing the pile's bottom edge. The deck, hands and
// pouch toggles are all sized from it, which is why the measurements are here rather than on the
// screen that happens to draw the pile.
const (
	PileSlotSize = 44

	// **One character on a square**, which is what a control this size can carry — the same
	// reason the sort column is single letters. It is the size the toggles draw their labels at.
	pileSlotTextSize = 30
)
