package screens

// **The credits screen: who made this, what it is built on, and what it is licensed as.**
//
// It was a stub for months — registered in the scene map, drawing nothing, reachable from nowhere.
// It is real as of 2026-09-03 *(owner's call)* and reachable from the title menu.
//
// **The licensing block is not decoration.** The project is source-available and intended to be
// sold, the art is bundle art whose licence permits shipping it inside a game, and the score is
// synthesised rather than recorded precisely so there is no provenance question — see CLAUDE.md.
// A game that ships without naming any of that is a game with an attribution problem the day it
// goes on sale, so the page exists partly to be *correct* and only partly to be read.
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

	// creditsQuiet is a line that has to be there and does not have to be read: the licence
	// wording, the attribution small print.
	creditsQuiet

	// creditsGap is a blank line. **A kind rather than an empty string**, so a gap is deliberate
	// rather than a line somebody forgot to fill in.
	creditsGap
)

// The page's shape.
const (
	creditsTitle     = "CREDITS"
	creditsTitleSize = 40

	creditsHeadingSize = 22
	creditsBodySize    = 18
	creditsQuietSize   = 15

	// creditsLineHeight is the pitch of an ordinary line, and creditsHeadingTop the extra air a
	// heading gets above it. Headings are separated by space rather than by a rule, because the
	// page is short and four rules on it would read as a table.
	creditsLineHeight = 26
	creditsHeadingTop = 18
	creditsGapHeight  = 12
)

// credits is the page. **The version is not in here** — it is drawn separately at the bottom, from
// `gs.Version`, because it is a fact about the build rather than a line somebody wrote.
var credits = []creditsLine{
	{"Ascending Duel", creditsHeading},
	{"a roguelike duel up a tower", creditsQuiet},
	{"", creditsGap},

	{"MADE BY", creditsHeading},
	{"Justin Crosby  ·  CuriousJC", creditsBody},
	{"KingSherman1820", creditsBody},
	{"", creditsGap},

	{"BUILT WITH", creditsHeading},
	{"Ebitengine  ·  Apache-2.0", creditsBody},
	{"Oto  ·  Apache-2.0", creditsBody},
	{"", creditsGap},

	{"ART AND SOUND", creditsHeading},
	{"Creature and boss portraits by PVGames", creditsBody},
	{"Interface art and glyphs generated in-engine", creditsQuiet},
	{"Score synthesised from MIDI in-engine", creditsQuiet},
	{"", creditsGap},

	{"LICENCE", creditsHeading},
	{"PolyForm Noncommercial 1.0.0", creditsBody},
	{"Source-available. Streaming and video of gameplay are permitted,", creditsQuiet},
	{"monetised or not. See LICENSE for the full terms.", creditsQuiet},
}

// creditsQuietInk is how far a quiet line is pulled toward the ground it is written on.
const creditsQuietPct = 38

// creditsVersionColor is the build string at the foot of the page — dimmer than a quiet line,
// because it is a thing to be found rather than read.
var creditsVersionColor = color.RGBA{R: 120, G: 108, B: 90, A: 255}

// CreditsScene is the credits screen.
type CreditsScene struct {
	back *models.Button
}

// Init builds the one button on first entry and positions it every time. See TitleScene.Init for
// why positioning is not done in Draw.
func (s *CreditsScene) Init(gs *state.GlobalState) {
	if s.back == nil {
		s.back = models.NewButton(320, 80, "BACK", func() { s.leave(gs) })
	}
	s.back.ScreenX, s.back.ScreenY = gs.PctX(50), gs.PctY(91)
}

func (s *CreditsScene) Update(gs *state.GlobalState) error {
	systems.UpdateButton(gs, s.back)
	return nil
}

func (s *CreditsScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	screen.Fill(screenGround)

	heading := &text.DrawOptions{}
	heading.GeoM.Translate(float64(gs.PctX(50)), float64(gs.PctY(9)))
	heading.PrimaryAlign = text.AlignCenter
	heading.SecondaryAlign = text.AlignCenter
	heading.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, creditsTitle,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: creditsTitleSize}, heading)

	// **Everything is centred on one axis**, which is what makes a page of unequal-length lines
	// read as a document rather than as a list.
	centre := float64(gs.PctX(50))
	y := gs.PctY(17)

	for _, l := range credits {
		if l.kind == creditsGap {
			y += creditsGapHeight
			continue
		}
		if l.kind == creditsHeading {
			y += creditsHeadingTop
		}

		op := &text.DrawOptions{}
		op.GeoM.Translate(centre, float64(y))
		op.PrimaryAlign = text.AlignCenter
		op.SecondaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(creditsInk(l.kind))
		text.Draw(screen, l.text,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: creditsSize(l.kind)}, op)

		y += creditsLineHeight
	}

	// The build, at the foot of the page. **The one line here that is not authored** — it comes off
	// the linker, and it is the thing that makes a bug report able to name a build.
	version := &text.DrawOptions{}
	version.GeoM.Translate(centre, float64(gs.PctY(84)))
	version.PrimaryAlign = text.AlignCenter
	version.SecondaryAlign = text.AlignCenter
	version.ColorScale.ScaleWithColor(creditsVersionColor)
	text.Draw(screen, gs.Version,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}, version)

	systems.DrawButton(gs, screen, s.back)
}

// creditsSize is the point size a kind is set at.
func creditsSize(k creditsKind) float64 {
	switch k {
	case creditsHeading:
		return creditsHeadingSize
	case creditsQuiet:
		return creditsQuietSize
	default:
		return creditsBodySize
	}
}

// creditsInk is the colour a kind is set in. **ColorToward rather than ColorAtStrength**, because
// this page is drawn on the cream ground — scaling toward black there makes a line louder, not
// quieter. See the colour rule in CLAUDE.md.
func creditsInk(k creditsKind) color.Color {
	if k == creditsQuiet {
		return systems.ColorToward(groundInk, screenGround, creditsQuietPct)
	}
	return groundInk
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
