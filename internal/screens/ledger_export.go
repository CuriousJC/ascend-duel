package screens

// **The ledger's one other control: writing the account out to a file.**
//
// The panel is the run's account on screen and it is gone with the run — a defeat clears the
// snapshot, and nothing survives a climb that can be read afterwards or pasted into a bug report.
// This writes what the panel is showing to a JSON file beside the profile, named for the run code
// and the moment it was taken. See session/export.go for what goes in it and profile/store.go for
// where it lands.
//
// **It is on the ledger rather than on the settings screen**, because it is a thing done to the
// record the player is looking at: the button is where the account is. It sits in the panel's own
// title band, mirroring the closing X in the opposite corner, so the panel keeps one control at
// each end and no row loses space to it.
//
// **A write is reported where it happened.** The button says what it did on the panel beside it —
// the file name on a success, the reason on a failure — rather than logging somewhere the player
// cannot see, which for a machine that cannot write to its config directory is the difference
// between a control that did nothing and a control that is broken.

import (
	"image"
	"path/filepath"
	"time"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	// The button carries a word rather than a letter: it stands in a whole title band with room
	// for one, and a single `E` on a panel nobody has pressed it on before says nothing. The
	// height is the X's, so the two controls in the band are one size.
	ledgerExportLabel = "EXPORT"
	ledgerExportWidth = 150
	ledgerExportText  = 20

	// ledgerNoteSize is the line reporting what the last press did, written beside the button and
	// small enough that a file name fits without reaching the centered title.
	ledgerNoteSize = 16

	// ledgerNoteGap is the air between the button and that line.
	ledgerNoteGap = 12
)

// ledgerExporter is the button and what the last press of it came to.
//
// **The note is kept on the panel rather than timed out**, because the thing a player does next
// with an exported file is go and find it, and a message that has already gone is one they have to
// press the button a second time to read.
type ledgerExporter struct {
	button  *models.Button
	pressed bool

	// note is what to say about the last press, and failed says whether it went wrong — which is
	// the ink, and is a flag rather than a stored color for the ledger's own reason: a color is a
	// palette decision and the panel owns its palette.
	note   string
	failed bool
}

// update runs the button and writes the file if it was pressed.
func (e *ledgerExporter) update(gs *state.GlobalState, r image.Rectangle) {
	if e.button == nil {
		e.button = models.NewButton(ledgerExportWidth, ui.ModalCloseSize, ledgerExportLabel,
			func() { e.pressed = true })
		e.button.BaseColor = ledgerPane.Color
		e.button.TextSize = ledgerExportText
	}
	e.button.ScreenX = r.Min.X + ui.ModalCloseInset + ledgerExportWidth/2
	e.button.ScreenY = r.Min.Y + ui.ModalCloseInset + ui.ModalCloseSize/2

	// **A run with nothing in it is a dead control, not a file saying nothing.** The panel is
	// already drawing its own "nothing has happened yet"; an export of it would be a file the
	// player went and found in order to read the same sentence.
	e.button.State = models.ButtonStateNormal
	if !e.exportable(gs) {
		e.button.State = models.ButtonStateDisabled
	}

	e.pressed = false
	systems.UpdateButton(gs, e.button)
	if e.pressed {
		e.export(gs)
	}
}

// exportable reports whether there is an account worth writing out.
func (e *ledgerExporter) exportable(gs *state.GlobalState) bool {
	return gs.Run != nil && len(gs.Run.LedgerFights()) > 0
}

// export writes the file and records what happened.
//
// **The clock is read here and handed down.** `internal/session` reads no wall clock — see the
// determinism rules — so the one call to time.Now is in the screen that pressed the button, which
// is also what lets the export be tested against a fixed moment.
func (e *ledgerExporter) export(gs *state.GlobalState) {
	if !e.exportable(gs) {
		return
	}
	at := time.Now()
	code := seeds.Code(gs.RunSeed)

	path, err := gs.Store.WriteExport(session.LedgerExportName(code, at), gs.Run.ExportLedger(code, at, ui.LedgerLines))
	if err != nil {
		e.note, e.failed = err.Error(), true
		return
	}
	e.note, e.failed = "Saved "+filepath.Base(path), false
}

// draw puts the button and its note in the panel's title band.
func (e *ledgerExporter) draw(gs *state.GlobalState, screen *ebiten.Image, r image.Rectangle) {
	if e.button != nil {
		systems.DrawButton(gs, screen, e.button)
	}
	if e.note == "" {
		return
	}

	ink := ledgerPane.Ink
	if e.failed {
		ink = ui.ModalCloseColor
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(
		float64(r.Min.X+ui.ModalCloseInset+ledgerExportWidth+ledgerNoteGap),
		float64(r.Min.Y+ui.ModalCloseInset+(ui.ModalCloseSize-ledgerNoteSize)/2),
	)
	op.ColorScale.ScaleWithColor(ink)
	text.Draw(screen, e.note, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: ledgerNoteSize}, op)
}
