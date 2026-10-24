package screens

// **The achievements screen: what has been earned, and what has not.**
//
// It is one of the screens reachable from the title menu rather than from the run loop, and it is
// the program's rather than a journey's — like the settings screen, it never touches `session.Phase`
// and puts the player back where they came from.
//
// **It shows locked entries as well as earned ones**, grayed, with the name and the line still
// legible. An achievements page that listed only what you already have is a page that says nothing
// on the day a player most wants to read it. A spoiler is a record with `Hidden` set, which keeps
// its row and gives up its words and its picture until it is earned.
//
// **The catalog moved to `data/achievements.json` on 2026-09-06**, which is exactly the move the
// old note in this file said it would make once there were enough of these to scroll. Eleven records
// carrying trigger grammars is well past that line, and the argument that a name and a sentence do
// not earn a loader stopped holding the moment a record had to say *what earns it*. `internal/achieve`
// is the loader; this file draws what it hands over and decides nothing.
//
// **The key is the disk contract and the name is not.** A record's `APIName` may never
// change once shipped; its `DisplayName` can be reworded any afternoon.
//
// **It scrolls**, on `models.Scrollbar`, a drag and a pair of arrows because the input vocabulary has
// no wheel. **The entries stand three to a row**, a fixed height, so the bar counts rows and the page
// cannot land half a line off.
//
// **The icon is the whole of "earned"**: the achieved picture in color on an earned entry and in
// grayscale on a locked one, so the entry carries no second mark saying the same thing.

import (
	"image"
	"image/color"
	"strconv"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/achieve"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The page's shape. A heading carrying the tally, a grid of entries under it, and the way back.
const (
	achievementsTitle     = "ACHIEVEMENTS"
	achievementsTitleSize = 40

	// achievementRowHeight is the band one entry takes, and achievementRowGap the air between two
	// rows. An entry carries a name and up to two lines of description, so it is deeper than a
	// button. achievementColumns stand across the page, achievementColumnGap apart.
	achievementRowHeight = 106
	achievementRowGap    = 12
	achievementRowWidth  = 590
	achievementColumns   = 3
	achievementColumnGap = 20

	// achievementLineTop is where an entry's description starts, and achievementLineStep the
	// distance between its two lines.
	achievementLineTop  = 46
	achievementLineStep = 26

	// achievementNameTop is the middle of the name's line, which the progress figure shares.
	achievementNameTop = 24

	achievementNameSize = systems.TextLarge
	achievementLineSize = systems.TextMedium

	// achievementRowInset is the margin at either end of a row: the icon sits that far in from the
	// left edge, and the progress figure that far in from the right.
	achievementRowInset = 20

	// achievementIconSize is the icon's side on the page — **Steam's own display size**, so the icon
	// a player reads here is the one Steam's pop-up will show — and achievementTextLeft is where the
	// words start, clear of it.
	achievementIconSize = 64
	achievementTextLeft = achievementRowInset/2 + achievementIconSize + achievementRowInset

	// achievementTextWidth is how wide an entry's words may run: from achievementTextLeft to the
	// right-hand margin.
	achievementTextWidth = achievementRowWidth - achievementTextLeft - achievementRowInset

	// The rows are cut off at the top of the Back button. **A count of rows rather than a pixel
	// height**, because that is what the scrollbar is told and a page that could show two and a
	// half rows would be a page whose bottom row is a lie.
	achievementsVisible = 6

	// The scrollbar's track, to the right of the rows with a gap between.
	achievementScrollWidth = 14
	achievementScrollGap   = 18
)

// The two grounds a row is drawn on, earned and not, and the light inks written on them.
//
// **A row is a dark card with white type**, for the contrast the light ground cannot give small
// type. Earned is a step lighter than locked and its words are brighter: the distinction is carried
// by *weight* rather than by hue, per the color rule in CLAUDE.md — the wheel belongs to the
// elements and an achievements page has no business claiming one.
var (
	achievementEarnedFill = color.RGBA{R: 52, G: 50, B: 62, A: 255}
	achievementLockedFill = color.RGBA{R: 34, G: 33, B: 42, A: 255}

	achievementNameInk       = color.RGBA{R: 246, G: 246, B: 250, A: 255}
	achievementLineInk       = color.RGBA{R: 214, G: 214, B: 224, A: 255}
	achievementLockedNameInk = color.RGBA{R: 186, G: 186, B: 198, A: 255}
	achievementLockedLineInk = color.RGBA{R: 156, G: 156, B: 170, A: 255}
)

// What a hidden achievement's row says while it is locked.
const (
	hiddenAchievementName = "HIDDEN ACHIEVEMENT"
	hiddenAchievementLine = "Earn it to find out what it is."
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
		s.back = models.NewButton(320, ui.ButtonLarge, "BACK", func() { s.leave(gs) })
	}
	s.back.ScreenX, s.back.ScreenY = gs.PctX(50), gs.PctY(88)

	if s.scroll == nil {
		s.scroll = models.NewScrollbar(achievementScrollWidth, achievementsColumnHeight())
		// The page is drawn on the light ground, so the track is dimmed toward that rather than
		// toward black — the same reason models.Slider.Ink exists. See systems.ColorToward.
		s.scroll.Ground = ui.ScreenGround
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
	s.scroll.Total = (len(achieve.Loaded().All()) + achievementColumns - 1) / achievementColumns
	s.scroll.Visible = achievementsVisible
	systems.UpdateScrollbar(gs, s.scroll)
	return nil
}

func (s *AchievementsScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	ui.FillScreenBackdrop(gs, screen)

	heading := &text.DrawOptions{}
	heading.GeoM.Translate(float64(gs.PctX(50)), float64(gs.PctY(14)))
	heading.PrimaryAlign = text.AlignCenter
	heading.SecondaryAlign = text.AlignCenter
	heading.ColorScale.ScaleWithColor(ui.GroundInk)
	// **The tally rides on the heading**, because the whole reason to open this page is to find out
	// how far along it you are.
	got, total := achievementCount(gs)
	systems.DrawText(screen, achievementsTitle+" ("+strconv.Itoa(got)+"/"+strconv.Itoa(total)+")",
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: achievementsTitleSize}, heading)

	left := achievementsLeft(gs)
	top := achievementsTop(gs)

	all := achieve.Loaded().All()
	first := s.scroll.Offset * achievementColumns
	for i := 0; i < achievementsVisible*achievementColumns && first+i < len(all); i++ {
		a := all[first+i]
		x := left + (i%achievementColumns)*(achievementRowWidth+achievementColumnGap)
		y := top + (i/achievementColumns)*(achievementRowHeight+achievementRowGap)
		s.drawRow(gs, screen, a, x, y, earned(gs, a.APIName))
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
	nameInk, lineInk := achievementNameInk, achievementLineInk
	if !got {
		nameInk, lineInk = achievementLockedNameInk, achievementLockedLineInk
	}

	// **A hidden achievement says nothing about itself until it is earned** — Steam's meaning of the
	// flag. It keeps its row, so the tally and the scrollbar still count it; what it gives up is its
	// name, its description and its icon, all three of which would give it away.
	shown := a
	if a.Hidden && !got {
		shown.DisplayName = hiddenAchievementName
		shown.Description = hiddenAchievementLine
		shown.AchievedIconKey = data.AchievementIconPrefix + data.DefaultAchievementIcon
	}

	// **The icon is the Steam pair**: the achieved picture on an earned row and the same picture in
	// grayscale on a locked one — see systems.ArtMarkGray.
	icon := systems.ArtMarkGrayImage(shown.AchievedIconKey, achievementIconSize, achievementIconSize)
	if got {
		icon = systems.ArtMarkImage(shown.AchievedIconKey, achievementIconSize, achievementIconSize)
	}
	if icon != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x+achievementRowInset/2),
			float64(y+(achievementRowHeight-achievementIconSize)/2))
		screen.DrawImage(icon, op)
	}

	name := &text.DrawOptions{}
	name.GeoM.Translate(float64(x+achievementTextLeft), float64(y+achievementNameTop))
	name.SecondaryAlign = text.AlignCenter
	name.ColorScale.ScaleWithColor(nameInk)
	systems.DrawText(screen, shown.DisplayName,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: achievementNameSize}, name)

	lineFace := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: achievementLineSize}
	for i, l := range achievementLines(shown.Description, lineFace) {
		systems.DrawLine(screen, l, lineFace, x+achievementTextLeft,
			y+achievementLineTop+i*achievementLineStep, lineInk)
	}

	// **Progress, but only where there is any.** A tally has a fraction and a moment does not, so a
	// row that is either done or not says nothing here rather than saying "0 / 1" — see
	// achieve.Achievement.Progress, which is the one place that decision is made.
	//
	// **It is drawn on a locked row and not on an earned one**, where a finished tally would read
	// "300 / 300" beside an icon already in color. It stands on the name's line, the description
	// below being free to wrap under it.
	if !got && !a.Hidden && gs.Profile != nil {
		if p := a.Progress(gs.Profile.Counters); p != "" {
			prog := &text.DrawOptions{}
			prog.GeoM.Translate(
				float64(x+achievementRowWidth-achievementRowInset), float64(y+achievementNameTop))
			prog.PrimaryAlign = text.AlignEnd
			prog.SecondaryAlign = text.AlignCenter
			prog.ColorScale.ScaleWithColor(lineInk)
			systems.DrawText(screen, p,
				&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: achievementLineSize}, prog)
		}
	}

}

// achievementLines is a description wrapped to the entry's width.
func achievementLines(description string, face *text.GoTextFace) []models.TipLine {
	return systems.WrapLine(models.TipLine{{Text: description}}, face, achievementTextWidth)
}

// achievementsTop is where the first row sits, and achievementsColumnHeight how deep the whole
// column is. **One is derived from the other**, so the scrollbar's track and the rows it scrolls
// cannot drift apart.
func achievementsTop(gs *state.GlobalState) int { return gs.PctY(19) }

// achievementsGridWidth is how wide the grid of entries is, and achievementsLeft where it starts:
// centered on the page.
const achievementsGridWidth = achievementColumns*achievementRowWidth +
	(achievementColumns-1)*achievementColumnGap

func achievementsLeft(gs *state.GlobalState) int { return gs.PctX(50) - achievementsGridWidth/2 }

func achievementsColumnHeight() int {
	return achievementsVisible*(achievementRowHeight+achievementRowGap) - achievementRowGap
}

// achievementsScrollRect is the bar's track: beside the grid, the full depth of the column.
func achievementsScrollRect(gs *state.GlobalState) image.Rectangle {
	left := achievementsLeft(gs) + achievementsGridWidth + achievementScrollGap
	top := achievementsTop(gs)
	return image.Rect(left, top, left+achievementScrollWidth, top+achievementsColumnHeight())
}

// earned reports whether the profile holds an entry. **A missing profile reads as nothing earned**
// rather than as an error: a machine that could not read its profile still gets to look at the page.
// A revealing scenario or the reveal dev tool reads its share as earned — see reveals; nothing is written.
func earned(gs *state.GlobalState, key string) bool {
	if reveals(gs, key) {
		return true
	}
	return gs.Profile != nil && gs.Profile.Has(key)
}

// achievementCount is how many entries are earned, out of how many there are.
func achievementCount(gs *state.GlobalState) (got, total int) {
	all := achieve.Loaded().All()
	for _, a := range all {
		if earned(gs, a.APIName) {
			got++
		}
	}
	return got, len(all)
}

// achievementTally is the count written "N of M", as the statistics book's row reads it.
func achievementTally(gs *state.GlobalState) string {
	got, total := achievementCount(gs)
	return strconv.Itoa(got) + " of " + strconv.Itoa(total)
}

// leave goes back to whichever screen opened this one, on the same terms the settings screen's Back
// does — including the title-screen fallback, which is what an ActiveScreen of zero already means.
func (s *AchievementsScene) leave(gs *state.GlobalState) {
	gs.PopReturn()
}
