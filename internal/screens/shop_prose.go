package screens

// **The shopkeeper speaks.** Two sentences typed out on arrival, before anything is for sale.
//
// The shop had a title and a hint line from the day it landed and **neither had ever been seen**:
// both were drawn at the reward screen's offsets, 262 and 300, and the shelf row started at y=211
// and was drawn over the top of them. So the one place the five-finger cap was explained was
// painted out by the cards it was explaining.
//
// **The fix is not to move the title back** *(owner's call, 2026-08-22)*. The reward screen
// narrates its payout, and a shop that opened with a heading reading "The shop" would be the one
// between-fights screen with no voice in it. So the room introduces itself the way the reward
// screen's does, on the same typewriter, and the hint keeps the line under it.
//
// **Nothing here pays.** `proseLine.pays` is what makes the reward screen's figures land in the
// purse; these lines are flavour and leave it alone, which is why the typewriter is reused rather
// than copied — the claims are a field, not a behaviour.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// shopkeeperLines is the greeting *(owner's wording, 2026-08-22)*.
//
// **The rasp is one run in the ring pink**, the colour this game already spends on rings and on
// the chrome around them — so the sentence that is the creature talking is visibly not the
// sentence that is the room being described.
func shopkeeperLines() []proseLine {
	return []proseLine{
		{runs: []proseRun{{text: "A hooded creature appears from the shadow and you see the glint of gold in his hands."}}},
		{runs: []proseRun{{text: `"Vitae for jewelry, duelist?"`, ink: handNameInk}, {text: " it rasps."}}},
	}
}

// shopProseLineAt is the middle of one narrated line. **Nothing flies out of it today** — the
// typewriter asks for it so a line that pays can, and no line here does — but it is derived rather
// than passed as a zero, so the day the shopkeeper hands something over it already works.
func shopProseLineAt(gs *state.GlobalState, i int) image.Point {
	return image.Pt(gs.PctX(50), shopProseTop+i*proseLineGap)
}

// drawProse writes whatever the shopkeeper has said so far.
func (s *ShopScene) drawProse(gs *state.GlobalState, screen *ebiten.Image, face *text.GoTextFace) {
	for i, line := range s.prose.lines {
		runs, on := s.prose.visible(i)
		if !on {
			break
		}
		drawProseLine(screen, face, line.plain(), runs, gs.PctX(50), shopProseTop+i*proseLineGap)
	}
}
