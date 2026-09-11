package screens

import (
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// The table everything on a between-fight screen is drawn on.
//
// **These live here rather than on the combat screen because every scene stands on them.**
// They were declared in combat.go, which meant the post-battle screen — and the shop and the
// room choice after it — had to reach into the combat screen's own file to find out what
// colour the game is. The ground is not the combat screen's; it is the game's.
//
// Anything drawn on a surface of its own — a card, a panel, a button — takes that surface's
// own colours instead. These two are only for what is painted straight onto the table.

// screenGround is what every screen is painted on. It went cream on 2026-08-14 from the
// {50,50,50} dark grey the combat screen had been since it existed, and **it went a light slate
// blue on 2026-09-07** *(owner's call)*.
//
// **Its lightness is the load-bearing part, not its hue.** Everything written straight onto the
// table takes `groundInk`, which is near black, and everything dimmed toward the table goes
// through `systems.ColorToward(x, screenGround, pct)` — both of which are correct only on a
// *light* ground. That was the whole cost of the 2026-08-14 change and it is documented in
// CLAUDE.md: `ColorAtStrength` scales toward black and therefore makes things *louder* on a light
// surface, which is why `ColorToward` exists. So this blue was chosen at the cream's lightness
// rather than at a hue that read well on its own, and **a darker blue is not a colour change —
// it is a re-tune of every figure on the table.**
//
// **It went bluer twice on the same day** *(owner's call)*, from {168,188,212} through
// {150,185,228} to this: the red came down and the blue went up, widening the gap between the
// channels from 44 points to 102, which is what actually reads as blue rather than as grey with
// an opinion. **The second step also took real lightness with it** — about nine percent against
// the cream this replaced — so the paragraph above is no longer describing a swap made at
// constant lightness. It is still a light ground and `ColorToward` is still the right tool on it,
// but the margin that made that obviously true is smaller than it was, and the next step down is
// the one that stops being a colour change.
//
// **It is deeper than the cards stand on** — `cards.Surface` is {240,239,234} — because a card, a
// panel and the table cannot all be the same near-white or the objects stop having edges. The
// separation used to come from warmth, the ground being the yellowest of the three; it now comes
// from hue outright, which is a wider gap than the cream ever had.
//
// **It is a single colour even though the screen is painted with a gradient.** Everything that
// dims toward the ground needs one answer to "what colour is the table", and a per-pixel one
// would make a figure's dimming depend on where on the screen it happened to be drawn. So this is
// the gradient's midpoint and the two ends are derived from it — see groundTop and groundBottom.
var screenGround = color.RGBA{R: 126, G: 172, B: 228, A: 255}

// groundLift and groundSink are how far the background gradient moves either side of
// screenGround, in percent. They are separate numbers because the gradient is deliberately
// **not** symmetric *(owner's call, 2026-09-07)*: it started at six percent each way and read as
// almost nothing, and what was wanted from a stronger one was a *darker edge* rather than a
// brighter middle — a table whose far end falls away, not a screen with a light shining on it.
//
// **It grew three times on the same day**, from six each way, to 5/20, to 12/30, to 12/38, to this — the first two both
// read as nothing at all on a 1080-tall screen, which is the honest answer to how little a
// percent of lightness is worth spread over that distance. The lift is still the smaller of the
// two, so the weight of the sweep is in the dark end.
//
// **The lift stays small and the sink is the one that grew.** Climbing toward white is where a
// gradient stops looking like light and starts looking like fog, and the top of the screen is
// where the enemy card and the fight log sit — a lighter ground under an off-white card is the
// one place the two surfaces stop having an edge.
//
// **Two costs, and both are now real rather than theoretical.** Everything dimmed toward the
// ground reads `screenGround` alone, so a figure dimmed at the very bottom of the screen sits on
// a surface nearly a third darker than the colour it was dimmed toward; and `groundInk` is near
// black, so the bottom band is where text-on-table contrast is thinnest. Both are bounded by the
// sink and nothing else. **Past about a third the ink has to move too**, and the dim would have
// to be computed per row — which would make a figure's colour depend on where it happened to be
// drawn, the thing groundAtRow's neighbours exist to avoid.
const (
	groundLift = 12
	groundSink = 44
)

// The two ends of the background gradient, derived from screenGround so that changing the one
// colour moves the whole screen.
//
// **Lighter at the top and darker at the bottom**, which is the direction `systems.BevelEdges`
// already lights every card, button and panel from. A screen lit from below with objects on it
// lit from above is the kind of disagreement nobody can name and everybody can see.
var (
	groundTop    = systems.ColorToward(screenGround, color.RGBA{R: 255, G: 255, B: 255, A: 255}, groundLift)
	groundBottom = systems.ColorAtStrength(screenGround, 100-groundSink)
)

// groundStrip is the cached gradient: one pixel wide and as tall as the screen, stretched across
// the width when it is drawn.
//
// **One pixel wide because the gradient is vertical**, so every column is identical and painting
// 1920 of them would be 1920 times the work for the same picture. Ebitengine's default filter is
// nearest, so the stretch is an exact repeat rather than a resample.
//
// **Rebuilt only when the height changes**, which in practice is once: the game runs at a fixed
// 1920x1080 internal resolution. The check is there rather than an assumption because `Layout`
// owns that number and this file should not have an opinion about it.
var (
	groundStrip *ebiten.Image
	groundAt    int
)

// fillGround paints the background of a screen. It replaces `screen.Fill(screenGround)`, which is
// what every scene did until the gradient landed.
//
// **Every scene calls this rather than filling its own colour.** The ground is the game's, not any
// one screen's — the same argument that moved these colours out of combat.go in the first place —
// and a scene painting its own would be the one screen that did not follow when the colour moved.
func fillGround(screen *ebiten.Image) {
	h := screen.Bounds().Dy()
	if h <= 0 {
		return
	}
	if groundStrip == nil || groundAt != h {
		strip := ebiten.NewImage(1, h)
		for y := 0; y < h; y++ {
			strip.Set(0, y, groundAtRow(y, h))
		}
		groundStrip, groundAt = strip, h
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(screen.Bounds().Dx()), 1)
	screen.DrawImage(groundStrip, op)
}

// groundAtRow is the gradient's colour on one row of a screen h tall.
//
// **A per-channel interpolation rather than `systems.ColorToward`**, which is the one place in
// this codebase that rule is deliberately not followed. ColorToward takes a whole-number percent,
// so over 1080 rows it produces about a hundred steps — ten-pixel bands, which on a gradient this
// subtle is the only thing anybody would see. Interpolating the channels at full precision is
// safe here for the reason it is not safe generally: the two ends are the same hue at two
// lightnesses, so there is no hue for a straight lerp to drift off.
func groundAtRow(y, h int) color.RGBA {
	if h <= 1 {
		return screenGround
	}
	mix := func(a, b uint8) uint8 {
		return uint8(int(a) + (int(b)-int(a))*y/(h-1))
	}
	return color.RGBA{
		R: mix(groundTop.R, groundBottom.R),
		G: mix(groundTop.G, groundBottom.G),
		B: mix(groundTop.B, groundBottom.B),
		A: 255,
	}
}

// groundInk is for text written straight onto the ground rather than onto a card, a pane or a
// button — the action-point figure, the draw pile's count, the relic row's fraction, the
// post-battle screen's heading. Near black and slightly warm.
//
// **It stayed warm when the ground went blue** *(2026-09-07)*, which is deliberate rather than an
// oversight: it is the same near-black the cards' own text uses, so a figure on the table and a
// figure on a card read as the same ink rather than as two blacks that not quite match. What it
// must stay is *near-black* — see screenGround, where the lightness of the table is the thing
// holding this up.
//
// Anything drawn on a surface of its own takes that surface's ink instead; this is only for what
// has nothing behind it.
var groundInk = color.RGBA{R: 44, G: 40, B: 34, A: 255}

// vitaeInk is the crimson vitae is written in, **everywhere it is written** *(owner's call,
// 2026-08-22)*: the purse on the duelist card, and the word itself in the reward screen's prose.
//
// **One colour for one thing.** Vitae is the run's only currency and it is the only red on the
// table, so a figure in this colour says "money" before it is read. It is louder against the blue
// ground than it was against the cream, red and blue being opposite ends of the wheel where red
// and cream were neighbours — which is a gain for a figure whose whole job is to be spotted. It is deliberately not
// `lifeColor` — life is a bar and a fraction on a card, and the two reds never share a surface.
var vitaeInk = color.RGBA{R: 168, G: 26, B: 42, A: 255}
