package screens

// **The credits screen: who made this, what it is built on, and what it is licensed as.**
//
// It was a stub for months — registered in the scene map, drawing nothing, reachable from nowhere.
// It is real as of 2026-09-03 *(owner's call)* and reachable from the title menu.
//
// **The LICENSE block is not decoration.** It is what the project is source-available *under*,
// and a game that ships without naming its own terms is a game nobody can tell what they may do
// with. The page exists partly to be *correct* and only partly to be read.
//
// **It is a static page, on purpose.** A scrolling crawl is the obvious thing to reach for and it
// would be the wrong one here: there is no keyboard to skip it with, the input vocabulary has no
// wheel, and a page short enough to fit the screen is a page nobody has to wait for. If it ever
// outgrows the screen it takes a `models.Scrollbar`, which already exists.

import (
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// creditsLine is one line of the page: its words, and how it is set.
//
// **A kind rather than a size**, so the page is written as a document and the type scale lives in
// one place. A line that named its own point size would be a page that drifts out of alignment one
// edit at a time.
type creditsLine struct {
	text string
	kind creditsKind
}

type creditsKind int

const (
	// creditsHeading is a section title — small, and set apart by the air above it.
	creditsHeading creditsKind = iota

	// creditsBody is an ordinary line.
	creditsBody

	// creditsGap is a blank line. **A kind rather than an empty string**, so a gap is deliberate
	// rather than a line somebody forgot to fill in.
	creditsGap
)

// The page's shape.
const (
	creditsTitle     = "CREDITS"
	creditsTitleSize = 56

	creditsHeadingSize = 32
	creditsBodySize    = systems.TextLarge

	// creditsLineHeight is the pitch of an ordinary line, and creditsHeadingTop the extra air a
	// heading gets above it. Headings are separated by space rather than by a rule, because the
	// page is short and four rules on it would read as a table.
	creditsLineHeight = 34
	creditsHeadingTop = 22
	creditsGapHeight  = 14
)

// credits is the page. **The version is not in here** — it is drawn separately at the bottom, from
// `gs.Version`, because it is a fact about the build rather than a line somebody wrote.
var credits = []creditsLine{
	{"MADE BY", creditsHeading},
	{"Ambivalent Systems", creditsBody},
	{"", creditsGap},

	{"BUILT WITH", creditsHeading},
	{"Ebitengine  ·  Apache-2.0", creditsBody},
	{"Oto  ·  Apache-2.0", creditsBody},
	{"", creditsGap},

	{"ART", creditsHeading},
	{"Art generated with iteratively revised prompts using ChatGPT.", creditsBody},
	{"", creditsGap},

	{"SOUND", creditsHeading},
	{"TBD", creditsBody},
	{"", creditsGap},

	{"LICENSE", creditsHeading},
	{"PolyForm Noncommercial 1.0.0", creditsBody},
}

// The page's inks. **Light rather than `ui.GroundInk`**, because the page stands on the gray fog
// backdrop rather than the light ground, and near-black type on mid-gray fog does not read. A light
// ink multiplies the prose sheet, so the letters keep their dark outline against the fog.
var (
	creditsTextInk = color.RGBA{R: 240, G: 238, B: 232, A: 255}

	// creditsVersionColor is the build string at the foot of the page — dimmer than the page,
	// because it is a thing to be found rather than read.
	creditsVersionColor = color.RGBA{R: 180, G: 182, B: 192, A: 255}
)

// CreditsScene is the credits screen.
type CreditsScene struct {
	back *models.Button
}

// Init builds the one button on first entry and positions it every time. See TitleScene.Init for
// why positioning is not done in Draw.
func (s *CreditsScene) Init(gs *state.GlobalState) {
	if s.back == nil {
		s.back = models.NewButton(320, ui.ButtonLarge, "BACK", func() { s.leave(gs) })
	}
	s.back.ScreenX, s.back.ScreenY = gs.PctX(50), gs.PctY(91)
}

func (s *CreditsScene) Update(gs *state.GlobalState) error {
	systems.UpdateButton(gs, s.back)
	return nil
}

func (s *CreditsScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	ui.FillScreenBackdrop(gs, screen)

	heading := &text.DrawOptions{}
	heading.GeoM.Translate(float64(gs.PctX(50)), float64(gs.PctY(9)))
	heading.PrimaryAlign = text.AlignCenter
	heading.SecondaryAlign = text.AlignCenter
	heading.ColorScale.ScaleWithColor(creditsTextInk)
	systems.DrawText(screen, creditsTitle,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: creditsTitleSize}, heading)

	// **Everything is centered on one axis**, which is what makes a page of unequal-length lines
	// read as a document rather than as a list.
	center := float64(gs.PctX(50))
	y := gs.PctY(16)

	for _, l := range credits {
		if l.kind == creditsGap {
			y += creditsGapHeight
			continue
		}
		if l.kind == creditsHeading {
			y += creditsHeadingTop
		}

		op := &text.DrawOptions{}
		op.GeoM.Translate(center, float64(y))
		op.PrimaryAlign = text.AlignCenter
		op.SecondaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(creditsInk(l.kind))
		systems.DrawText(screen, l.text,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: creditsSize(l.kind)}, op)

		y += creditsLineHeight
	}

	// The build, at the foot of the page. **The one line here that is not authored** — it comes off
	// the linker, and it is the thing that makes a bug report able to name a build.
	version := &text.DrawOptions{}
	version.GeoM.Translate(center, float64(gs.PctY(84)))
	version.PrimaryAlign = text.AlignCenter
	version.SecondaryAlign = text.AlignCenter
	version.ColorScale.ScaleWithColor(creditsVersionColor)
	systems.DrawText(screen, gs.Version,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: systems.TextMedium}, version)

	systems.DrawButton(gs, screen, s.back)
}

// creditsSize is the point size a kind is set at.
func creditsSize(k creditsKind) float64 {
	switch k {
	case creditsHeading:
		return creditsHeadingSize
	default:
		return creditsBodySize
	}
}

// creditsInk is the color a kind is set in.
func creditsInk(k creditsKind) color.Color {
	return creditsTextInk
}

// leave goes back to whichever screen opened this one, on the same terms the settings and
// achievements screens do.
func (s *CreditsScene) leave(gs *state.GlobalState) {
	back := gs.ReturnScreen
	if back == state.Credits {
		back = state.Title
	}
	gs.ActiveScreen = back
	gs.NewScreen = true
}
