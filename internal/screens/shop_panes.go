package screens

// The shop's four panes, and the one rhythm they pack at.
//
// **The shelf is four panes on one line since 2026-09-06** *(owner's call)*: the sealed packs, the
// relics, the potions and the brand, left to right, each standing on the pane the top row already
// draws — a flat fill one step off the ground, eight pixels of padding, no border and no title. It
// was one flat row of six seats told apart by the labels under them, and the panes are what let a
// fourth kind of thing join the shelf without a fifth label to read.
//
// **There are no labels at all now** *(owner's call)*. The cards say what they are: a relic borders
// pink, a pack is a crate and says what is inside it, a potion is a flask, and the brand stands
// alone. A caption under each pane would be the fourth thing on screen naming what the picture
// already names.
//
// **The row overlaps, and that is the price of one line.** Nine full-size cards are 1827 pixels of
// a 1920-wide screen before a pane's padding or a gutter is paid for, so a row that did not overlap
// could not exist — see shopPitch, which solves for the pitch that makes all four panes full at
// once, exactly as topRowPitch does for the two panes above. About nineteen pixels of each card is
// covered by its neighbour, against the thirty-four the hand of eight has always overlapped by.
//
// **One pitch across all four panes**, for the reason the top row has one: two rhythms on one line
// read as two rows that happen to be level with each other.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// shopPane is which of the four a seat belongs to, in the order they stand.
type shopPane int

const (
	shopPanePacks shopPane = iota
	shopPaneRelics
	shopPanePotions
	shopPaneBrand
)

// shopPaneOrder is the four in standing order, for anything that walks them.
func shopPaneOrder() []shopPane {
	return []shopPane{shopPanePacks, shopPaneRelics, shopPanePotions, shopPaneBrand}
}

const (
	// shopPaneMargin is the bare ground kept at each end of the row. It is what the pitch is
	// solved against, so widening it narrows every pane rather than pushing one off the screen.
	shopPaneMargin = 24

	// shopPaneGutter is the bare ground between two panes' backings. **Wider than the padding
	// inside one**, which is what makes the four read as four rather than as one long tray.
	shopPaneGutter = 24

	// shopPaneTopPct is where the cards stand. Under the narration and the hint, and far enough
	// above the Leave button for a price and a reroll button to hang beneath them.
	shopPaneTopPct = 46

	// shopRerollGap is how far a reroll button sits under the price figures of the pane it
	// belongs to.
	shopRerollGap = 12

	// The reroll button itself. **Narrower than the pane it hangs under**, so it reads as
	// attached to that pane rather than as a row of its own — the sell tab's rule.
	shopRerollWidth  = 240
	shopRerollHeight = 40
	shopRerollText   = 20
)

// shopPaneSeats is how many cards a pane holds, full or empty.
//
// **A pane is its seats whether or not anything is standing in them**, exactly as the consumables
// pane above is two: a bought relic leaves an empty seat and the row does not close up, so nothing
// moves under the hand of a player who is still reading it.
func shopPaneSeats(p shopPane) int {
	switch p {
	case shopPanePacks:
		return packsOffered
	case shopPaneRelics:
		return shelfSize
	case shopPanePotions:
		return len(shopPotions())
	default:
		return 1
	}
}

// shopSeatTotal is every seat on the shelf.
func shopSeatTotal() int {
	n := 0
	for _, p := range shopPaneOrder() {
		n += shopPaneSeats(p)
	}
	return n
}

// shopPitch is the one pitch the whole row packs at.
//
// **Solved rather than chosen**, which is the arithmetic topRowPitch already documents: the row is
// every seat's card plus a pitch for every seat after the first in its own pane, plus each pane's
// padding and the gutters between them. Rearranged for the pitch that makes all four panes exactly
// fill the width they are given.
//
//	usable = width - 2*margin - panes*2*pad - (panes-1)*gutter
//	pitch  = (usable - panes*cardWidth) / (seats - panes)
//
// **Capped at the row's own comfortable gap**, like every other row in the game: a shelf that
// spread to whatever it was given would stop reading as one shelf. The cap does not bite today —
// nine seats solve to about 184 against a 200-pixel card — and it is what keeps the row honest if a
// pane is ever taken off it.
func shopPitch(gs *state.GlobalState) int {
	n := len(shopPaneOrder())
	usable := gs.ScreenWidth - 2*shopPaneMargin - n*2*relicPaneBackPad - (n-1)*shopPaneGutter

	steps := shopSeatTotal() - n
	if steps <= 0 {
		return cardWidth + relicSlotMaxGap
	}

	pitch := (usable - n*cardWidth) / steps
	if max := cardWidth + relicSlotMaxGap; pitch > max {
		return max
	}
	return pitch
}

// shopPaneWidth is how wide one pane's cards run, before its padding.
func shopPaneWidth(gs *state.GlobalState, p shopPane) int {
	return (shopPaneSeats(p)-1)*shopPitch(gs) + cardWidth
}

// shopPaneRect is one pane's card extent — what the seats are cut out of, and what the backing is
// derived from. **The cards and not the backing**, which is the distinction relicPaneRect draws and
// for the same reason: growing the padding must not be able to move a card.
func shopPaneRect(gs *state.GlobalState, p shopPane) image.Rectangle {
	top := gs.PctY(shopPaneTopPct)

	left := shopPaneMargin + relicPaneBackPad
	for _, before := range shopPaneOrder() {
		if before == p {
			break
		}
		left += shopPaneWidth(gs, before) + 2*relicPaneBackPad + shopPaneGutter
	}

	return image.Rect(left, top, left+shopPaneWidth(gs, p), top+cardHeight)
}

// shopPaneBackRect is the surface a pane's cards stand on: its extent, grown by the padding every
// pane in the game uses.
func shopPaneBackRect(gs *state.GlobalState, p shopPane) image.Rectangle {
	return shopPaneRect(gs, p).Inset(-relicPaneBackPad)
}

// shopSeatRect is where one card in a pane is drawn, and the rectangle it is clicked in. **One
// function for both**, the rule every row in this game follows.
func shopSeatRect(gs *state.GlobalState, p shopPane, i int) image.Rectangle {
	r := shopPaneRect(gs, p)
	left := r.Min.X + i*shopPitch(gs)
	return image.Rect(left, r.Min.Y, left+cardWidth, r.Max.Y)
}

// shopFigureBottom is where the prices under a pane stop, and what anything hanging below them has
// to clear. The figures are written under the *backing* rather than under the cards, so a pane's
// own edge is not struck through by its prices.
func shopFigureBottom(gs *state.GlobalState) int {
	return shopPaneBackRect(gs, shopPaneRelics).Max.Y + shopFigureGap + shopFigureSize
}

// shopRerollRect is where a pane's reroll button hangs: centred under it, below the prices.
func shopRerollRect(gs *state.GlobalState, p shopPane) image.Rectangle {
	back := shopPaneBackRect(gs, p)
	cx := (back.Min.X + back.Max.X) / 2
	top := shopFigureBottom(gs) + shopRerollGap
	return image.Rect(cx-shopRerollWidth/2, top, cx+shopRerollWidth/2, top+shopRerollHeight)
}

// drawShopPaneBack paints one pane's surface.
//
// **Flat, and the same colour the worn relics stand on.** It covers nothing, so there is no lit edge
// to say it is in front of anything — the argument drawRelicPane already makes, and the reason the
// bevelled cards standing on it are what gets read.
func drawShopPaneBack(gs *state.GlobalState, screen *ebiten.Image, p shopPane) {
	back := shopPaneBackRect(gs, p)
	vector.DrawFilledRect(screen,
		float32(back.Min.X), float32(back.Min.Y), float32(back.Dx()), float32(back.Dy()),
		relicPaneBackColor, false)
}
