package screens

// **The splash a finished run ends on: what the climb came to, and the code that would deal it
// again.**
//
// Before this, a death was an **End Run** button and then the title screen — the whole run gone with
// nothing said about it. That is the one moment a player wants the numbers, and it was also the only
// moment the run code was reachable at all: it went to the log at launch and nowhere a player could
// see. *(owner's call, 2026-09-03)*
//
// **It is the one screen that draws something no longer in the game.** `gs.Run` is nil by the time
// this is up — a summary that held the `Session` open would be a finished run that is still
// resumable — so what it reads is a `session.RunSummary` on the global state, which is plain
// numbers and a string. See screens.endRun, which takes the summary before it destroys anything.
//
// **The seed is the loudest thing on the page**, in its own framed box under the totals. Everything
// else is a fact about a run that is over; the code is the one thing that is still useful
// afterwards, and it is six characters that have to be transcribed by eye — Crockford base32 exactly
// so that reading it off a screen cannot land on a different valid run. See internal/seeds.
//
// **There is no Retry and no "run it again" button**, and there should not be. The seed being on
// screen is what makes the run repeatable; a button that dealt it again would be a retry with a
// longer name.

import (
	"image/color"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The page's shape.
const (
	runOverTitleSize = 44
	runOverHowSize   = 20

	// The totals block: a two-column list, labels right-aligned into the middle and figures left of
	// it. **Aligned on the gutter rather than centred as lines**, because a column of figures is
	// read by scanning down it and centred rows put every number in a different place.
	runOverRowHeight  = 40
	runOverLabelSize  = 20
	runOverFigureSize = 26
	runOverGutter     = 24

	// The seed box.
	runOverSeedBoxW    = 340
	runOverSeedBoxH    = 96
	runOverSeedSize    = 46
	runOverSeedCapSize = 15
	runOverSeedCaption = "run code"
)

// The two words at the top, by how the run ended. **The heading says what happened and the line
// under it says it in a sentence**, because "Defeated" alone on a screen the player did not ask for
// reads as an error message.
var runOverWords = map[string]struct{ title, how string }{
	session.EndedInDefeat: {
		"Your climb ends here",
		"the duelist has fallen",
	},
	session.EndedByChoice: {
		"Run abandoned",
		"you walked away from the tower",
	},
}

// runOverSeedFill is the seed box's face. **A cool slate against the cream ground**, which is the
// same "this is the program, not the fight" colour the chrome and the sliders take — the code is a
// fact about the software rather than about the duel.
var (
	runOverSeedFill = color.RGBA{R: 58, G: 62, B: 74, A: 255}
	runOverSeedInk  = color.RGBA{R: 240, G: 238, B: 232, A: 255}
	runOverSeedCap  = color.RGBA{R: 158, G: 160, B: 172, A: 255}
)

// RunOverScene is the end-of-run splash.
type RunOverScene struct {
	back *models.Button
}

// Init builds the one button on first entry and positions it every time. See TitleScene.Init for
// why positioning is not done in Draw.
func (s *RunOverScene) Init(gs *state.GlobalState) {
	if s.back == nil {
		s.back = models.NewButton(260, 62, "Back to Title", func() { s.leave(gs) })
		s.back.TextSize = 22
	}
	s.back.ScreenX, s.back.ScreenY = gs.PctX(50), gs.PctY(91)
}

func (s *RunOverScene) Update(gs *state.GlobalState) error {
	// **A screen with nothing to say leaves by itself.** Nothing reaches here without a summary
	// today; if something ever does, a blank page with a button on it is worse than the title.
	if gs.Summary == nil {
		s.leave(gs)
		return nil
	}
	systems.UpdateButton(gs, s.back)
	return nil
}

func (s *RunOverScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	screen.Fill(screenGround)
	if gs.Summary == nil {
		return
	}
	sum := *gs.Summary
	centre := float64(gs.PctX(50))

	words, ok := runOverWords[sum.Ended]
	if !ok {
		// An ending this build does not have a phrase for still gets a page. **The summary is the
		// point and the wording is decoration**, so an unknown value loses the sentence rather than
		// the numbers.
		words.title = "Run over"
	}

	title := &text.DrawOptions{}
	title.GeoM.Translate(centre, float64(gs.PctY(13)))
	title.PrimaryAlign = text.AlignCenter
	title.SecondaryAlign = text.AlignCenter
	title.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, words.title,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: runOverTitleSize}, title)

	if words.how != "" {
		how := &text.DrawOptions{}
		how.GeoM.Translate(centre, float64(gs.PctY(19)))
		how.PrimaryAlign = text.AlignCenter
		how.SecondaryAlign = text.AlignCenter
		how.ColorScale.ScaleWithColor(systems.ColorToward(groundInk, screenGround, 38))
		text.Draw(screen, words.how,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: runOverHowSize}, how)
	}

	s.drawTotals(gs, screen, sum)
	s.drawSeed(gs, screen, sum.Seed)

	systems.DrawButton(gs, screen, s.back)
}

// drawTotals is the two-column list. **The four the run is judged on come first** — how far, how
// many, how hard — and the two that are context come after, quieter.
func (s *RunOverScene) drawTotals(gs *state.GlobalState, screen *ebiten.Image, sum session.RunSummary) {
	rows := []struct {
		label  string
		figure string
		quiet  bool
	}{
		{"Floor reached", strconv.Itoa(sum.Floor), false},
		{"Enemies defeated", strconv.Itoa(sum.Defeated), false},
		{"Damage dealt", strconv.Itoa(sum.Dealt), false},
		{"Rooms entered", strconv.Itoa(sum.Rooms), true},
		{"Rounds fought", strconv.Itoa(sum.Rounds), true},
		{"Vitae unspent", strconv.Itoa(sum.Vitae), true},
	}

	gutter := gs.PctX(50)
	y := gs.PctY(28)

	for _, r := range rows {
		ink := groundInk
		if r.quiet {
			ink = systems.ColorToward(groundInk, screenGround, 32)
		}

		// The label ends at the gutter, the figure starts after it. One axis, so the eye reads the
		// figures as a column.
		label := &text.DrawOptions{}
		label.GeoM.Translate(float64(gutter-runOverGutter), float64(y))
		label.PrimaryAlign = text.AlignEnd
		label.SecondaryAlign = text.AlignCenter
		label.ColorScale.ScaleWithColor(systems.ColorToward(ink, screenGround, 22))
		text.Draw(screen, r.label,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: runOverLabelSize}, label)

		figure := &text.DrawOptions{}
		figure.GeoM.Translate(float64(gutter+runOverGutter), float64(y))
		figure.SecondaryAlign = text.AlignCenter
		figure.ColorScale.ScaleWithColor(ink)
		text.Draw(screen, r.figure,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: runOverFigureSize}, figure)

		y += runOverRowHeight
	}
}

// drawSeed is the run code, in its own box, at the size the rest of the page is not.
//
// **It is the only thing on this screen worth writing down**, which is the whole argument for
// giving it a box: everything else is a number about a run that is finished, and this is what makes
// the run repeatable. See internal/seeds on why the alphabet is what it is.
func (s *RunOverScene) drawSeed(gs *state.GlobalState, screen *ebiten.Image, code string) {
	left := gs.PctX(50) - runOverSeedBoxW/2
	top := gs.PctY(70)

	systems.BevelRect(screen, left, top, runOverSeedBoxW, runOverSeedBoxH,
		systems.PaneBevelWidth, runOverSeedFill, false)

	centre := float64(left + runOverSeedBoxW/2)

	cap := &text.DrawOptions{}
	cap.GeoM.Translate(centre, float64(top+22))
	cap.PrimaryAlign = text.AlignCenter
	cap.SecondaryAlign = text.AlignCenter
	cap.ColorScale.ScaleWithColor(runOverSeedCap)
	text.Draw(screen, runOverSeedCaption,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: runOverSeedCapSize}, cap)

	seed := &text.DrawOptions{}
	seed.GeoM.Translate(centre, float64(top+62))
	seed.PrimaryAlign = text.AlignCenter
	seed.SecondaryAlign = text.AlignCenter
	seed.ColorScale.ScaleWithColor(runOverSeedInk)
	text.Draw(screen, code,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: runOverSeedSize}, seed)
}

// leave clears the summary and goes to the title.
//
// **The summary is dropped on the way out**, rather than left standing. It describes a run that no
// longer exists, and a stale one still on the state is a page that could be reached again later
// showing numbers from two runs ago.
//
// **It goes to the title outright rather than reading ReturnScreen.** Every other screen that can be
// entered from anywhere puts the player back where they were; there is nowhere to put them back to
// here, because the screen they came from was drawing a run that has ended.
func (s *RunOverScene) leave(gs *state.GlobalState) {
	gs.Summary = nil
	gs.ActiveScreen = state.Title
	gs.NewScreen = true
}
