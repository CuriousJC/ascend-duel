package screens

import (
	"github.com/curiousjc/ascend-duel/internal/actions"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"

	"image/color"
)

// TitleScene is the front screen: the logo and the three menu buttons.
type TitleScene struct {
	combatButton   *models.Button
	settingsButton *models.Button
	exitButton     *models.Button
}

// Init builds the buttons on first entry and positions them every time.
//
// Positioning belongs here rather than in Draw: Update hit-tests against ScreenX/Y,
// so a Draw-time assignment leaves the first frame testing against zeroes and goes
// stale any frame Ebiten chooses to skip Draw. The internal resolution is fixed, so
// these coordinates only need computing once per visit.
func (s *TitleScene) Init(gs *state.GlobalState) {
	if s.combatButton == nil {
		s.combatButton = models.NewButton(275, 100, "Combat", func() { actions.GoToCombat(gs) })
		s.settingsButton = models.NewButton(275, 100, "Settings", func() { actions.OpenSettings(gs) })
		s.exitButton = models.NewButton(275, 100, "Exit", func() { actions.QuitGame(gs) })
	}

	s.combatButton.ScreenX = gs.HalfwayX
	s.combatButton.ScreenY = gs.FirstThirdY

	s.settingsButton.ScreenX = gs.HalfwayX
	s.settingsButton.ScreenY = gs.FirstThirdY + 150

	s.exitButton.ScreenX = gs.HalfwayX
	s.exitButton.ScreenY = gs.FirstThirdY + 300
}

func (s *TitleScene) Update(gs *state.GlobalState) error {
	systems.UpdateButton(gs, s.combatButton)
	systems.UpdateButton(gs, s.settingsButton)
	systems.UpdateButton(gs, s.exitButton)
	return nil
}

func (s *TitleScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 109, G: 141, B: 138, A: 255})

	//TITLE
	//
	var title *ebiten.Image
	if gs.CountSecond > 9 && gs.CountSecond%10 == 0 {
		title = gs.Assets["titleEaster_png"]
	} else {
		title = gs.Assets["title_png"]
	}
	bounds := title.Bounds()
	imageCenterX := float64(bounds.Dx()) / 2
	imageCenterY := float64(bounds.Dy()) / 2
	op := &colorm.DrawImageOptions{}
	scaleFactor := 0.75
	op.GeoM.Scale(scaleFactor, scaleFactor)
	op.GeoM.Translate(float64(gs.HalfwayX)-imageCenterX*scaleFactor, 150-imageCenterY*scaleFactor)
	hue := float64(1)
	saturation := float64(1)
	value := float64(1)
	var c colorm.ColorM
	c.ChangeHSV(hue, saturation, value)
	colorm.DrawImage(screen, title, c, op)

	//BUTTONS
	//
	// Positions are set in Init; Draw only draws.
	systems.DrawButton(gs, screen, s.combatButton)
	systems.DrawButton(gs, screen, s.settingsButton)
	systems.DrawButton(gs, screen, s.exitButton)
}
