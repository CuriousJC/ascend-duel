package screens

// The potions pane, and the brand seat beside it.
//
// **A potion is the one thing in the shop that changes the duelist** *(owner's call, 2026-09-06)*.
// A ring is worn, a stone raises a rung, a worm eats a card, a parasite waits in a bucket; a potion
// is drunk on the spot and moves one of the three figures the fighter is made of. See
// `internal/session/potion.go`, which owns what each one does and refuses a record the rules cannot
// apply.
//
// **The whole catalogue is on the shelf every visit, and it does not reroll** *(owner's call)*.
// There is no draw here and so no stream: three vessels, always the same three, in the order
// `data/potions.json` writes them. A reroll would be asking to be offered the thing that is already
// being offered.
//
// **A potion is bought with one click and no confirm**, the shelf's rule rather than the worn row's:
// the price is on the card, the purse cannot go into debt, and there is nothing a confirmation
// would protect. What it does is immediate and visible on the duelist card two hundred pixels
// above — a Salve moves the life fraction, a Draught the DMG line, a Tonic both.
//
// **The brand is a placeholder and cannot be clicked** *(owner's call, 2026-09-06)*. The seat, the
// pane and the layout land now; the mechanic is still only MECHANICS.md §Brands, and a card that
// took vitae and recorded something nothing reads would be worse than one that cannot be bought.
// It draws dim, it says what a brand is, and its price is written so the pane can be judged at the
// size it will actually be.

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// shopPotions is the catalogue as the shelf stands it. **Asked of the run's package rather than
// held on the scene**: every visit offers all three, so there is nothing to deal and nothing to
// remember between visits.
func shopPotions() []session.Potion { return session.Potions() }

// The brand's placeholder face. **Its own words rather than a real record**, because there is no
// `brands.json` — see MECHANICS.md §Brands, which is the whole of what exists.
const (
	brandName  = "SIXTH FINGER"
	brandLine  = "WEAR SIX RINGS\nnever comes off"
	brandPrice = 12

	// brandArtKey is the placeholder picture: the ring catalogue's own fallback. **A ring's
	// default rather than a drawing of its own**, on the argument the bucket already borrowed the
	// worm's: a placeholder something else already wears is better than a blank face, and it will
	// be replaced by a key on a record the day brands become one.
	brandArtKey = "defaultring_png"
)

// potionSeat and brandSeat are where a card in either pane is drawn and clicked.
func potionSeat(gs *state.GlobalState, i int) image.Rectangle {
	return shopSeatRect(gs, shopPanePotions, i)
}

func brandSeat(gs *state.GlobalState) image.Rectangle {
	return shopSeatRect(gs, shopPaneBrand, 0)
}

// drawPotions puts the three vessels up with their prices under them, on the shelf's terms: one
// that cannot be afforded is dimmed rather than hidden, so the pane still says what was offered.
func (s *ShopScene) drawPotions(gs *state.GlobalState, screen *ebiten.Image) {
	drawShopPaneBack(gs, screen, shopPanePotions)

	for i, p := range shopPotions() {
		at := potionSeat(gs, i)
		if s.drunk[p.Record] {
			continue
		}

		lit := gs.Run != nil && gs.Run.CanDrink(p.Record)
		blitCard(gs, screen, at.Min, potionSpec(gs, p, lit), cards.WormStyle)
		s.figure(gs, screen, at, fmt.Sprintf("%d vitae", p.Price), lit)
	}
}

// drawBrand puts the one seat up. **Always dim**, because nothing can be done with it yet and a lit
// card is a card that says click me.
func (s *ShopScene) drawBrand(gs *state.GlobalState, screen *ebiten.Image) {
	drawShopPaneBack(gs, screen, shopPaneBrand)

	at := brandSeat(gs)
	blitCard(gs, screen, at.Min, brandSpec(gs), cards.WormStyle)
	s.figure(gs, screen, at, fmt.Sprintf("%d vitae", brandPrice), false)
}

// potionSpec is a potion drawn as a card.
//
// **Basic, not an element.** A potion changes the duelist rather than a card, and the duelist is not
// a colour — so its border is the mid grey `cards.BorderOf` gives `basic`, exactly as a stone's and
// a sealed good's are. The hue wheel is full and a fourth kind of shelf card cannot have one.
func potionSpec(gs *state.GlobalState, p session.Potion, enabled bool) cards.Spec {
	return cards.Spec{
		Name:       p.Name,
		Form:       cards.FormNone,
		Cost:       0,
		Element:    artFor(combat.Basic),
		Art:        artwork(gs, brandArtKey),
		Text:       p.Text,
		Highlights: cards.ElementHighlights(p.Text),
		Enabled:    enabled,
	}
}

// brandSpec is the placeholder brand as a card. It says what a brand *is* — the container/contents
// axis MECHANICS.md draws — rather than naming a rule the game can resolve, because it cannot
// resolve one yet.
func brandSpec(gs *state.GlobalState) cards.Spec {
	return cards.Spec{
		Name:       brandName,
		Form:       cards.FormNone,
		Cost:       0,
		Element:    artFor(combat.Basic),
		Art:        artwork(gs, brandArtKey),
		Text:       brandLine,
		Highlights: cards.ElementHighlights(brandLine),
		Enabled:    false,
	}
}

// drinkPotion pays for one and applies it.
//
// **The purse moves inside `Drink`**, which is the one place a refusal happens — so a potion the
// run could not afford changes nothing at all, and the seat it was clicked in stays full.
func (s *ShopScene) drinkPotion(gs *state.GlobalState, key string) {
	if gs.Run == nil || s.drunk[key] || !gs.Run.CanDrink(key) {
		return
	}
	if !gs.Run.Drink(key) {
		return
	}

	// **The seat empties for the rest of the visit**, the sealed goods' rule rather than the ring
	// shelf's: a potion is not taken off a shelf of one-offs, but three of one bottle bought in a
	// row would be a shop that sells the same thing until the purse is empty.
	s.drunk[key] = true
	s.tip.Forget()
}

// potionTip is what resting on one says. **It explains what the figure does**, since a player
// meeting the shelf on floor one has never seen a potion and the face has room for two short lines.
func potionTip(p session.Potion) (string, []string) {
	var what string
	switch p.Effect {
	case session.PotionHeal:
		what = fmt.Sprintf("heals %d of the wound you are carrying", p.Amount)
	case session.PotionDMG:
		what = fmt.Sprintf("adds %d to your DMG for the rest of the run", p.Amount)
	default:
		what = fmt.Sprintf("raises your max life by %d for the rest of the run", p.Amount)
	}
	return p.Name, []string{what, "drunk on the spot", fmt.Sprintf("%d vitae", p.Price)}
}

// brandTip says what the seat is for, and that it is not ready. **It says so plainly** rather than
// leaving a dim card the player keeps clicking.
func brandTip() (string, []string) {
	return brandName, []string{
		"a brand alters the duelist, not the cards",
		"and never comes off",
		"not yet buyable",
	}
}
