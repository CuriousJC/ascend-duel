package screens

// **The journey won: the last realm's boss beaten, and what the player does with it.**
//
// A station of the run, stood at once a run at most — straight after the fight that won the journey
// and before that fight's reward. Two answers:
//
//   - **CLAIM VICTORY** ends the run as a win, through the one door every finished run goes through,
//     and lands on the splash with the run's numbers and its code.
//   - **CONTINUE** walks on into the endless journey past the last realm, by way of the reward the
//     boss still owes.
//
// **Either way the victory is already the run's.** It was counted when the boss fell — see
// session/victory.go — so walking on risks nothing that has been won; a run that falls later still
// ends on a splash headed JOURNEY WON.
//
// **Claiming is the red answer and it is on the right**: it commits and it leaves, which is what red
// says everywhere. Continuing is jade, and on the left, as the answer that keeps the run.

import (
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The page's shape.
const (
	victoryTitle     = "VICTORY"
	victoryTitleSize = 72

	victoryButtonWidth = 420
	victoryButtonGap   = 60
)

// victoryLines is what the page says under its heading.
var victoryLines = []string{
	"The last realm has fallen, and the journey is won.",
	"Claim the victory and end the run here,",
	"or walk on into the endless realms beyond.",
	"Either way, this run is a victory.",
}

// VictoryScene is the victory station.
type VictoryScene struct {
	claim, carryOn *models.Button
}

// Init builds the two answers on first entry and positions them every time.
func (s *VictoryScene) Init(gs *state.GlobalState) {
	if s.claim == nil {
		s.carryOn = models.NewButton(victoryButtonWidth, ui.ButtonLarge, "CONTINUE",
			func() { advanceRun(gs) })
		s.carryOn.BaseColor = ui.ButtonJade
		s.carryOn.TextSize = 40

		s.claim = models.NewButton(victoryButtonWidth, ui.ButtonLarge, "CLAIM VICTORY",
			func() { endRun(gs, session.EndedInVictory) })
		s.claim.BaseColor = ui.ButtonRed
		s.claim.TextSize = 40
	}
	half := (victoryButtonWidth + victoryButtonGap) / 2
	s.carryOn.ScreenX, s.carryOn.ScreenY = gs.PctX(50)-half, gs.PctY(74)
	s.claim.ScreenX, s.claim.ScreenY = gs.PctX(50)+half, gs.PctY(74)
}

func (s *VictoryScene) Update(gs *state.GlobalState) error {
	// **No run, no victory.** Nothing reaches here without one; a page offering to end a run that is
	// not there would be two buttons doing nothing.
	if gs.Run == nil {
		gs.ActiveScreen = state.Title
		gs.NewScreen = true
		return nil
	}
	systems.UpdateButton(gs, s.carryOn)
	systems.UpdateButton(gs, s.claim)
	return nil
}

func (s *VictoryScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	ui.FillScreenBackdrop(gs, screen)
	center := float64(gs.PctX(50))

	head := &text.DrawOptions{}
	head.GeoM.Translate(center, float64(gs.PctY(26)))
	head.PrimaryAlign = text.AlignCenter
	head.SecondaryAlign = text.AlignCenter
	head.ColorScale.ScaleWithColor(ui.GroundInk)
	systems.DrawText(screen, victoryTitle,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: victoryTitleSize}, head)

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: systems.TextMedium}
	for i, line := range victoryLines {
		op := &text.DrawOptions{}
		op.GeoM.Translate(center, float64(gs.PctY(42)+i*46))
		op.PrimaryAlign = text.AlignCenter
		op.SecondaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(ui.GroundInk)
		systems.DrawText(screen, line, face, op)
	}

	systems.DrawButton(gs, screen, s.carryOn)
	systems.DrawButton(gs, screen, s.claim)
}
