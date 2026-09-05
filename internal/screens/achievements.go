package screens

// **The achievements screen: what has been earned, and what has not.**
//
// It is the second screen reachable from the title menu rather than from the run loop, and it is
// the program's rather than a climb's — like the settings screen, it never touches `session.Phase`
// and puts the player back where they came from.
//
// **It shows locked entries as well as earned ones**, greyed, with the name and the line still
// legible. An achievements page that listed only what you already have is a page that says nothing
// on the day a player most wants to read it. Nothing here is a spoiler yet; the day one is, that
// entry gets a hidden flag rather than the page getting a policy.
//
// **The catalogue is a table in this file rather than a file in `data/`** *(2026-09-03)*, and that
// is a decision about size rather than about where catalogues live. There is one achievement. A
// JSON file plus a loader plus its validation, for one record whose only two fields are a name and
// a sentence, would be ceremony that makes the thing harder to read rather than easier — and the
// half that genuinely *is* a contract, the key written to disk, already lives in `internal/profile`
// where it belongs. **When there are enough of these to scroll, it moves to `data/achievements.json`
// and gets the loader**; see the `data` skill for what that costs.
//
// **The key is the disk contract and the name is not.** `profile.AchievementFirstSteps` may never
// change; "First Steps" can be reworded any afternoon.

import (
	"image/color"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// achievement is one row: the key on disk, and the two strings shown to a player.
type achievement struct {
	key  string
	name string
	line string
}

// achievements is the catalogue, in the order the page lists them. **Order is authored** rather
// than sorted by key or by whether it is earned — a page that reshuffled as you played would be a
// page you have to re-read each visit.
var achievements = []achievement{
	{
		key:  profile.AchievementFirstSteps,
		name: "FIRST STEPS",
		line: "Defeat your first enemy.",
	},
}

// The page's shape. A heading, a column of rows under it, and the way back.
const (
	achievementsTitle     = "ACHIEVEMENTS"
	achievementsTitleSize = 40

	// achievementRowHeight is the band one entry takes, and achievementRowGap the air between two.
	// A row carries two lines of text, so it is deeper than a button.
	achievementRowHeight = 78
	achievementRowGap    = 14
	achievementRowWidth  = 640

	achievementNameSize = 24
	achievementLineSize = 17

	// achievementRowInset is how far in from the row's left edge the words start.
	achievementRowInset = 20

	// achievementTallySize is the "1 of 1" under the heading.
	achievementTallySize = 18
)

// The two grounds a row is drawn on: earned, and not.
//
// **Earned is a warmer, lighter card and locked is nearly the screen itself.** The distinction is
// carried by *weight* rather than by hue, per the colour rule in CLAUDE.md — the wheel belongs to
// the elements and an achievements page has no business claiming one.
var (
	achievementEarnedFill = color.RGBA{R: 244, G: 234, B: 214, A: 255}
	achievementLockedFill = color.RGBA{R: 216, G: 200, B: 172, A: 255}
)

// AchievementsScene is the achievements screen.
type AchievementsScene struct {
	back *models.Button
}

// Init builds the one button on first entry and positions it every time. See TitleScene.Init for
// why positioning is not done in Draw.
func (s *AchievementsScene) Init(gs *state.GlobalState) {
	if s.back == nil {
		s.back = models.NewButton(320, 80, "BACK", func() { s.leave(gs) })
	}
	s.back.ScreenX, s.back.ScreenY = gs.PctX(50), gs.PctY(88)
}

func (s *AchievementsScene) Update(gs *state.GlobalState) error {
	systems.UpdateButton(gs, s.back)
	return nil
}

func (s *AchievementsScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	screen.Fill(screenGround)

	heading := &text.DrawOptions{}
	heading.GeoM.Translate(float64(gs.PctX(50)), float64(gs.PctY(14)))
	heading.PrimaryAlign = text.AlignCenter
	heading.SecondaryAlign = text.AlignCenter
	heading.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, achievementsTitle,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: achievementsTitleSize}, heading)

	// **The tally is the one number worth having at the top**, because the whole reason to open
	// this page is to find out how far along it you are.
	tally := &text.DrawOptions{}
	tally.GeoM.Translate(float64(gs.PctX(50)), float64(gs.PctY(20)))
	tally.PrimaryAlign = text.AlignCenter
	tally.SecondaryAlign = text.AlignCenter
	tally.ColorScale.ScaleWithColor(systems.ColorToward(groundInk, screenGround, 35))
	text.Draw(screen, achievementTally(gs),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: achievementTallySize}, tally)

	left := gs.PctX(50) - achievementRowWidth/2
	top := gs.PctY(28)

	for i, a := range achievements {
		y := top + i*(achievementRowHeight+achievementRowGap)
		s.drawRow(gs, screen, a, left, y, earned(gs, a))
	}

	systems.DrawButton(gs, screen, s.back)
}

// drawRow puts one entry on the page. **A locked row is the same row with its ink pulled toward the
// ground** rather than a different layout — the shape of the page must not change as it fills up.
func (s *AchievementsScene) drawRow(gs *state.GlobalState, screen *ebiten.Image,
	a achievement, x, y int, got bool) {

	fill := achievementLockedFill
	if got {
		fill = achievementEarnedFill
	}

	// Bevelled like every other surface in the game, and **raised whether or not it is earned**:
	// the row is a card on the table either way, and a sunken one would mean "pushed in".
	systems.BevelRect(screen, x, y, achievementRowWidth, achievementRowHeight,
		systems.PaneBevelWidth, fill, false)

	// How far the words are pulled toward the ground. An earned row is at full strength; a locked
	// one is quiet but still readable, which is the whole point of listing it.
	nameInk, lineInk := groundInk, systems.ColorToward(groundInk, fill, 30)
	if !got {
		nameInk = systems.ColorToward(groundInk, fill, 45)
		lineInk = systems.ColorToward(groundInk, fill, 60)
	}

	name := &text.DrawOptions{}
	name.GeoM.Translate(float64(x+achievementRowInset), float64(y+22))
	name.SecondaryAlign = text.AlignCenter
	name.ColorScale.ScaleWithColor(nameInk)
	text.Draw(screen, a.name,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: achievementNameSize}, name)

	line := &text.DrawOptions{}
	line.GeoM.Translate(float64(x+achievementRowInset), float64(y+54))
	line.SecondaryAlign = text.AlignCenter
	line.ColorScale.ScaleWithColor(lineInk)
	text.Draw(screen, a.line,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: achievementLineSize}, line)

	// **A tick on an earned row, and nothing on a locked one.** A word would need a second column
	// and a padlock would need art nobody has drawn; the mark is two strokes, which is the least
	// that can say "this one".
	if got {
		s.drawTick(screen, x+achievementRowWidth-46, y+achievementRowHeight/2)
	}
}

// drawTick is the earned mark: two strokes, drawn rather than glyphed.
//
// **Not a GlyphKind**, because the glyph enum is append-only and keyed by ordinal into a cache, and
// a shape this small and this specific to one page does not earn a permanent seat in it. See
// CLAUDE.md on what a generated glyph costs.
func (s *AchievementsScene) drawTick(screen *ebiten.Image, cx, cy int) {
	const w = 3
	x, y := float32(cx), float32(cy)
	vector.StrokeLine(screen, x-11, y, x-3, y+9, w, groundInk, true)
	vector.StrokeLine(screen, x-3, y+9, x+12, y-10, w, groundInk, true)
}

// earned reports whether the profile holds an entry. **A missing profile reads as nothing earned**
// rather than as an error: a machine that could not read its profile still gets to look at the page.
func earned(gs *state.GlobalState, a achievement) bool {
	return gs.Profile != nil && gs.Profile.Has(a.key)
}

// achievementTally is the "N of M" line under the heading.
func achievementTally(gs *state.GlobalState) string {
	got := 0
	for _, a := range achievements {
		if earned(gs, a) {
			got++
		}
	}
	return strconv.Itoa(got) + " of " + strconv.Itoa(len(achievements))
}

// leave goes back to whichever screen opened this one, on the same terms the settings screen's Back
// does — including the title-screen fallback, which is what an ActiveScreen of zero already means.
func (s *AchievementsScene) leave(gs *state.GlobalState) {
	back := gs.ReturnScreen
	if back == state.Achievements {
		back = state.Title
	}
	gs.ActiveScreen = back
	gs.NewScreen = true
}
