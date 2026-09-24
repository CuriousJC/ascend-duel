package screens

// **The crash screen: the last thing a panicked build says.**
//
// **It is a whole screen rather than a dialog** *(2026-09-22)*. Every other box in this game is
// drawn over the scene underneath it, and the scene underneath this one is the scene that has just
// panicked — drawing it again is how one crash becomes two. So the frame stops asking for it and
// asks for this instead, and nothing that was running is run again.
//
// **It draws with the ground, the fonts and two buttons, and nothing else.** No cards, no relics,
// no run, no assets keyed by name: every one of those is a thing that could be the reason the game
// is here. The whole screen is a handful of centered strings over ui.FillGround.
//
// **What it says is what a bug report needs to be able to repeat**: what happened, which run, and
// where the file went. The stack is in the file rather than on the screen — a wall of frames in
// front of somebody who has just lost a climb tells them nothing and reads as blame.
//
// **There is no way back.** A crashed process is a process whose state is not trustworthy, and a
// button returning to the title screen would be an invitation to play on inside it. The way out is
// quitting, which is what ShouldClose does — the same door the window's close button uses, so there
// is still only one way the game ends.

import (
	"image/color"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/curiousjc/ascend-duel/internal/crashlog"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The page.
const (
	crashTitle     = "SOMETHING WENT WRONG"
	crashTitleSize = 44

	// crashApology is the one line addressed to the player. **It says what happened once and does
	// not apologize twice**; everything under it is facts.
	crashApology = "The game hit a fault it could not carry on from."

	crashBodySize  = 19
	crashQuietSize = 15
	crashLineStep  = 30

	// crashLabelGap is how far right of center a value sits, with its label the same distance to
	// the left. **A two-column block rather than sentences**, because what is on this page is meant
	// to be copied into a bug report rather than read aloud.
	crashLabelGap = 24

	crashButtonWidth  = 320
	crashButtonHeight = 80
	crashButtonGap    = 32

	crashQuitLabel   = "QUIT"
	crashRevealLabel = "SHOW REPORT"

	// crashNoFile is what stands where the path would be when the report could not be written. **It
	// says so rather than leaving the line out**: a page that silently omits the file is one that
	// has told the player to send something that is not there.
	crashNoFile = "could not be written"
)

const (
	// crashQuietPct is how far a quiet line is pulled toward the ground, matching the credits page.
	crashQuietPct = 38

	// crashFolderPct is how much of the page's width the folder line may take before it is elided,
	// and crashFolderTop the air between it and the block above it.
	crashFolderPct = 84
	crashFolderTop = 14
)

// crashRevealColor is the second button's face: the flat slate every control that is the program
// rather than the fight takes. The quit is the modal red, because it is the way out and that is
// already what the one red in the game means.
var crashRevealColor = color.RGBA{R: 92, G: 96, B: 108, A: 255}

// CrashScene is the screen a panic ends on.
type CrashScene struct {
	quit   *models.Button
	reveal *models.Button
}

// Init builds the two buttons on first entry and places them every time.
//
// **Two rather than one, and both are real controls** — the controller refactor walks a focus ring
// over whatever is on a screen, and a page with a single button is a page with nothing to walk. See
// CLAUDE.md on what every new screen owes the pad.
func (s *CrashScene) Init(gs *state.GlobalState) {
	if s.quit == nil {
		s.quit = models.NewButton(crashButtonWidth, crashButtonHeight, crashQuitLabel,
			func() { gs.ShouldClose = true })
		s.quit.BaseColor = ui.ModalCloseColor

		s.reveal = models.NewButton(crashButtonWidth, crashButtonHeight, crashRevealLabel,
			func() { s.show(gs) })
		s.reveal.BaseColor = crashRevealColor
	}

	half := (crashButtonWidth*2 + crashButtonGap) / 2
	left := gs.PctX(50) - half
	row := gs.PctY(82)
	s.reveal.ScreenX, s.reveal.ScreenY = left+crashButtonWidth/2, row
	s.quit.ScreenX, s.quit.ScreenY = left+crashButtonWidth+crashButtonGap+crashButtonWidth/2, row

	// **Dead with no file to show.** A store with nowhere to write, or a directory that refused
	// the report, leaves nothing to open — and a button that opened a folder with no report in it
	// would be the page's one actionable thing lying.
	s.reveal.State = models.ButtonStateNormal
	if gs.Crash == nil || gs.Crash.Path == "" {
		s.reveal.State = models.ButtonStateDisabled
	}
}

func (s *CrashScene) Update(gs *state.GlobalState) error {
	systems.UpdateButton(gs, s.reveal)
	systems.UpdateButton(gs, s.quit)
	return nil
}

// show opens the folder the report was written into.
//
// **It reveals rather than sends.** Nothing leaves the machine here — see the plan's sending
// section, which is its own piece of work and is built around a person answering a question every
// time. What this does is the thing a player can do with a report today: find it.
//
// **A failure is noted and nothing else happens.** The game is already on its crash screen; a file
// manager that will not open is not worth a second box about.
func (s *CrashScene) show(gs *state.GlobalState) {
	if gs.Crash == nil || gs.Crash.Path == "" {
		return
	}
	if err := reveal(gs.Crash.Path); err != nil {
		crashlog.Note("could not open the report's folder: %v", err)
	}
}

// reveal asks the desktop to show a file. **One switch rather than three build-tagged files**,
// because this is three command lines rather than three implementations.
func reveal(path string) error {
	switch runtime.GOOS {
	case "windows":
		// **The error is dropped here and only here**: explorer exits non-zero even when it has
		// done exactly what was asked, so reporting it would be a note about a success.
		_ = exec.Command("explorer", "/select,"+path).Run()
		return nil
	case "darwin":
		return exec.Command("open", "-R", path).Run()
	default:
		return exec.Command("xdg-open", path).Run()
	}
}

func (s *CrashScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	ui.FillGround(screen)

	center := float64(gs.PctX(50))
	line := func(str string, size float64, y int, ink color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(center, float64(y))
		op.PrimaryAlign = text.AlignCenter
		op.SecondaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(ink)
		text.Draw(screen, str, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: size}, op)
	}

	quiet := systems.ColorToward(ui.GroundInk, ui.ScreenGround, crashQuietPct)

	line(crashTitle, crashTitleSize, gs.PctY(22), ui.GroundInk)
	line(crashApology, crashBodySize, gs.PctY(31), quiet)

	// The facts, as a two-column block. **Labels quiet and values in the page's ink**, so the thing
	// worth copying is the thing that stands out.
	y := gs.PctY(44)
	for _, f := range crashFacts(gs) {
		label := &text.DrawOptions{}
		label.GeoM.Translate(center-crashLabelGap, float64(y))
		label.PrimaryAlign = text.AlignEnd
		label.SecondaryAlign = text.AlignCenter
		label.ColorScale.ScaleWithColor(quiet)
		text.Draw(screen, f.label,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: crashQuietSize}, label)

		value := &text.DrawOptions{}
		value.GeoM.Translate(center+crashLabelGap, float64(y))
		value.PrimaryAlign = text.AlignStart
		value.SecondaryAlign = text.AlignCenter
		value.ColorScale.ScaleWithColor(ui.GroundInk)
		text.Draw(screen, f.value,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: crashBodySize}, value)

		y += crashLineStep
	}

	// The folder, under the block and across the page. **Not a row in the block**, because it is
	// the one value wide enough to need the whole width — see crashFolder.
	if dir := crashFolder(gs); dir != "" {
		face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: crashQuietSize}
		width := func(str string) float64 { w, _ := text.Measure(str, face, 0); return w }
		line(elideMiddle(dir, float64(gs.PctX(crashFolderPct)), width),
			crashQuietSize, y+crashFolderTop, quiet)
	}

	systems.DrawButton(gs, screen, s.reveal)
	systems.DrawButton(gs, screen, s.quit)
}

// crashFact is one row of the block: what it is, and what it says.
type crashFact struct{ label, value string }

// crashFacts is the block, in the order it is read. **The build is on it** — a bug report that
// cannot name a build is a bug report nobody can place, which is the whole reason main.version
// exists.
func crashFacts(gs *state.GlobalState) []crashFact {
	c := gs.Crash
	if c == nil {
		c = &state.CrashInfo{}
	}
	name := filepath.Base(c.Path)
	if c.Path == "" {
		name = crashNoFile
	}
	return []crashFact{
		{"what happened", c.Panic},
		{"run code", c.Code},
		{"build", gs.Version},
		{"report", name},
	}
}

// crashFolder is the directory the report went into, drawn as its own line under the block.
//
// **The file name is a fact and the folder is where to look**, and they are split because a full
// path is the one value on this page long enough to run off the screen — which is what it did the
// first time this page was looked at, cutting off the end of the thing the page exists to say. The
// name fits the block; the folder gets the width of the page, and is elided from the middle if even
// that is not enough.
func crashFolder(gs *state.GlobalState) string {
	if gs.Crash == nil || gs.Crash.Path == "" {
		return ""
	}
	return filepath.Dir(gs.Crash.Path)
}

// elideMiddle shortens a string until it fits, taking the cut out of the middle.
//
// **The middle, because both ends carry the meaning.** The front of a path says which machine and
// which player and the back says which directory, so a tail cut leaves something that names no
// folder at all.
// **It takes the measurer rather than the face**, so the rule can be checked without a font: a test
// that had to build a GoTextFace to ask whether a cut lands in the middle would be a test about
// type-setting.
func elideMiddle(str string, max float64, width func(string) float64) string {
	if str == "" || width(str) <= max {
		return str
	}
	const ellipsis = "..."
	runes := []rune(str)
	for n := len(runes) - 1; n > len(ellipsis); n-- {
		head := n / 2
		try := string(runes[:head]) + ellipsis + string(runes[len(runes)-(n-head):])
		if width(try) <= max {
			return try
		}
	}
	return ellipsis
}
