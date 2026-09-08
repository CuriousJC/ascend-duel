package screens

// **The achievements screen: what has been earned, and what has not.**
//
// It is one of the screens reachable from the title menu rather than from the run loop, and it is
// the program's rather than a climb's — like the settings screen, it never touches `session.Phase`
// and puts the player back where they came from.
//
// **It shows locked entries as well as earned ones**, greyed, with the name and the line still
// legible. An achievements page that listed only what you already have is a page that says nothing
// on the day a player most wants to read it. Nothing here is a spoiler yet; the day one is, that
// entry gets a hidden flag rather than the page getting a policy.
//
// **The catalogue moved to `data/achievements.json` on 2026-09-06**, which is exactly the move the
// old note in this file said it would make once there were enough of these to scroll. Eleven records
// carrying trigger grammars is well past that line, and the argument that a name and a sentence do
// not earn a loader stopped holding the moment a record had to say *what earns it*. `internal/achieve`
// is the loader; this file draws what it hands over and decides nothing.
//
// **The key is the disk contract and the name is not.** A record's `AchievementRecord` may never
// change once shipped; its `Name` can be reworded any afternoon.
//
// **It scrolls now**, on `models.Scrollbar` — the third widget in the game, a drag rather than a
// wheel because the input vocabulary has no wheel. The rows are a fixed height, so the bar counts
// rows and the page cannot land half a line off.

import (
	"image"
	"image/color"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/achieve"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

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

	// achievementTallySize is the "1 of 11" under the heading.
	achievementTallySize = 18

	// The rows are cut off at the top of the Back button. **A count of rows rather than a pixel
	// height**, because that is what the scrollbar is told and a page that could show two and a
	// half rows would be a page whose bottom row is a lie.
	achievementsVisible = 6

	// The scrollbar's track, to the right of the rows with a gap between.
	achievementScrollWidth = 14
	achievementScrollGap   = 18
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
	back   *models.Button
	scroll *models.Scrollbar
}

// Init builds the widgets on first entry and positions them every time. See TitleScene.Init for
// why positioning is not done in Draw.
func (s *AchievementsScene) Init(gs *state.GlobalState) {
	if s.back == nil {
		s.back = models.NewButton(320, 80, "BACK", func() { s.leave(gs) })
	}
	s.back.ScreenX, s.back.ScreenY = gs.PctX(50), gs.PctY(88)

	if s.scroll == nil {
		s.scroll = models.NewScrollbar(achievementScrollWidth, achievementsColumnHeight())
		// The page is drawn on the light ground, so the track is dimmed toward that rather than
		// toward black — the same reason models.Slider.Ink exists. See systems.ColorToward.
		s.scroll.Ground = screenGround
	}

	// **Back to the top on every entry.** A page re-opened where it was left is a page whose first
	// row is missing for no reason the player can see.
	s.scroll.Offset = 0
}

func (s *AchievementsScene) Update(gs *state.GlobalState) error {
	systems.UpdateButton(gs, s.back)

	track := achievementsScrollRect(gs)
	s.scroll.Width, s.scroll.Height = track.Dx(), track.Dy()
	s.scroll.ScreenX = track.Min.X + track.Dx()/2
	s.scroll.ScreenY = track.Min.Y + track.Dy()/2
	s.scroll.Total = len(achieve.Loaded().All())
	s.scroll.Visible = achievementsVisible
	systems.UpdateScrollbar(gs, s.scroll)
	return nil
}

func (s *AchievementsScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	fillGround(screen)

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
	top := achievementsTop(gs)

	all := achieve.Loaded().All()
	offset := s.scroll.Offset
	for i := 0; i < achievementsVisible && offset+i < len(all); i++ {
		a := all[offset+i]
		y := top + i*(achievementRowHeight+achievementRowGap)
		s.drawRow(gs, screen, a, left, y, earned(gs, a.Key))
	}

	systems.DrawScrollbar(gs, screen, s.scroll)
	systems.DrawButton(gs, screen, s.back)
}

// drawRow puts one entry on the page. **A locked row is the same row with its ink pulled toward the
// ground** rather than a different layout — the shape of the page must not change as it fills up.
func (s *AchievementsScene) drawRow(gs *state.GlobalState, screen *ebiten.Image,
	a achieve.Achievement, x, y int, got bool) {

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
	text.Draw(screen, a.Name,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: achievementNameSize}, name)

	line := &text.DrawOptions{}
	line.GeoM.Translate(float64(x+achievementRowInset), float64(y+54))
	line.SecondaryAlign = text.AlignCenter
	line.ColorScale.ScaleWithColor(lineInk)
	text.Draw(screen, a.How,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: achievementLineSize}, line)

	// **Progress, but only where there is any.** A tally has a fraction and a moment does not, so a
	// row that is either done or not says nothing here rather than saying "0 / 1" — see
	// achieve.Achievement.Progress, which is the one place that decision is made.
	//
	// **It is drawn on a locked row and not on an earned one.** A finished tally would read
	// "300 / 300" next to a tick, which is the same fact twice.
	if !got && gs.Profile != nil {
		if p := a.Progress(gs.Profile.Counters); p != "" {
			prog := &text.DrawOptions{}
			prog.GeoM.Translate(
				float64(x+achievementRowWidth-achievementRowInset), float64(y+54))
			prog.PrimaryAlign = text.AlignEnd
			prog.SecondaryAlign = text.AlignCenter
			prog.ColorScale.ScaleWithColor(lineInk)
			text.Draw(screen, p,
				&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: achievementLineSize}, prog)
		}
	}

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

// achievementsTop is where the first row sits, and achievementsColumnHeight how deep the whole
// column is. **One is derived from the other**, so the scrollbar's track and the rows it scrolls
// cannot drift apart.
func achievementsTop(gs *state.GlobalState) int { return gs.PctY(28) }

func achievementsColumnHeight() int {
	return achievementsVisible*(achievementRowHeight+achievementRowGap) - achievementRowGap
}

// achievementsScrollRect is the bar's track: beside the rows, the full depth of the column.
func achievementsScrollRect(gs *state.GlobalState) image.Rectangle {
	left := gs.PctX(50) + achievementRowWidth/2 + achievementScrollGap
	top := achievementsTop(gs)
	return image.Rect(left, top, left+achievementScrollWidth, top+achievementsColumnHeight())
}

// earned reports whether the profile holds an entry. **A missing profile reads as nothing earned**
// rather than as an error: a machine that could not read its profile still gets to look at the page.
func earned(gs *state.GlobalState, key string) bool {
	return gs.Profile != nil && gs.Profile.Has(key)
}

// achievementTally is the "N of M" line under the heading.
func achievementTally(gs *state.GlobalState) string {
	all := achieve.Loaded().All()
	got := 0
	for _, a := range all {
		if earned(gs, a.Key) {
			got++
		}
	}
	return strconv.Itoa(got) + " of " + strconv.Itoa(len(all))
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
