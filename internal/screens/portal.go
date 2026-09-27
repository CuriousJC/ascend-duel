package screens

// **The portal: which realm comes next.**
//
// A portal room's boss opens two portals, and this is the station the run stands at after the shop
// that follows it. Each portal is a panel — the realm's name, what the motif is, what the element
// does to it — with a button under it, and taking one is the only way on. See
// session.Session.TakePortal for what a pick does and internal/pyramid for where the offers come
// from.
//
// **There is no Back and no Skip.** The shop is done and the next floor has to be one of the two;
// a screen that could be left without choosing would be a floor chosen by default.
//
// **Every word on it comes off the motif record** — `Name`, `Text` and `ElementText` — so the
// portal says what the catalog says, and an unwritten line shows as TBD rather than as a panel
// that looks finished.

import (
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/journal"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/pyramid"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The page's shape.
const (
	portalHeadSize = 44
	portalSubSize  = 22

	// The two panels stand side by side, centered, with the button inside the foot of each.
	portalPanelW   = 700
	portalPanelH   = 620
	portalPanelGap = 80
	portalPanelTop = 22 // percent of the screen height
	portalPad      = 40

	portalTitleSize = 34
	portalProseSize = 22
	portalLinePitch = 31
	portalParaGap   = 18

	portalButtonW = 260
	portalButtonH = 72
)

// portalPanel is a panel's face: the same dark surface and prose ink as the tooltips and Bob's
// bubble, because it is the same object — a box of type over the light table.
var (
	portalPanel = opaque(systems.PanelSurface)
	portalProse = systems.PanelSpeech
)

// PortalScene is the choice of realm.
type PortalScene struct {
	take [data.PortalOffers]*models.Button
}

// Init builds the buttons on first entry and positions them every time.
func (s *PortalScene) Init(gs *state.GlobalState) {
	for i := range s.take {
		if s.take[i] == nil {
			i := i
			s.take[i] = models.NewButton(portalButtonW, portalButtonH, "ENTER", func() { s.enter(gs, i) })
			s.take[i].TextSize = 32
		}
		r := portalPanelRect(gs, i)
		s.take[i].ScreenX = r.x + portalPanelW/2
		s.take[i].ScreenY = r.y + portalPanelH - portalPad - portalButtonH/2
	}
}

func (s *PortalScene) Update(gs *state.GlobalState) error {
	// **A portal with nothing open walks on by itself.** Nothing reaches here without one today —
	// the loop walks past the station when PortalDue is false — but a blank page with no way off it
	// would be worse than the next fight.
	if gs.Run == nil || !gs.Run.PortalDue() {
		advanceRun(gs)
		return nil
	}
	for i := range min(len(gs.Run.PortalOffers()), len(s.take)) {
		systems.UpdateButton(gs, s.take[i])
	}
	return nil
}

func (s *PortalScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	ui.FillGround(screen)
	if gs.Run == nil {
		return
	}
	center := float64(gs.PctX(50))

	head := &text.DrawOptions{}
	head.GeoM.Translate(center, float64(gs.PctY(9)))
	head.PrimaryAlign = text.AlignCenter
	head.SecondaryAlign = text.AlignCenter
	head.ColorScale.ScaleWithColor(ui.GroundInk)
	text.Draw(screen, "CHOOSE YOUR REALM",
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: portalHeadSize}, head)

	sub := &text.DrawOptions{}
	sub.GeoM.Translate(center, float64(gs.PctY(15)))
	sub.PrimaryAlign = text.AlignCenter
	sub.SecondaryAlign = text.AlignCenter
	sub.ColorScale.ScaleWithColor(systems.ColorToward(ui.GroundInk, ui.ScreenGround, 30))
	text.Draw(screen, "FLOOR "+strconv.Itoa(gs.Run.Floor()),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: portalSubSize}, sub)

	offers := gs.Run.PortalOffers()
	for i := range min(len(offers), len(s.take)) {
		s.drawPanel(gs, screen, i, offers[i])
		systems.DrawButton(gs, screen, s.take[i])
	}
}

// drawPanel is one portal: the realm's name with its element in the element's color, then what the
// motif is, then what the element does to it.
func (s *PortalScene) drawPanel(gs *state.GlobalState, screen *ebiten.Image, i int, offer pyramid.Floor) {
	r := portalPanelRect(gs, i)
	systems.BevelRect(screen, r.x, r.y, portalPanelW, portalPanelH, systems.PaneBevelWidth, portalPanel, false)

	motif := gs.Motifs[offer.Motif]
	name := motif.Name
	if name == "" {
		name = offer.Motif
	}

	x, y := r.x+portalPad, r.y+portalPad+portalTitleSize/2
	width := float64(portalPanelW - 2*portalPad)

	title := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: portalTitleSize}
	for _, line := range systems.WrapRuns(ui.TipLine(strings.ToUpper(offer.Element+" "+name)), title, width) {
		systems.DrawRuns(screen, line, title, x, y, portalProse)
		y += portalTitleSize + 6
	}
	y += portalParaGap

	prose := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: portalProseSize}
	about, inElement := motif.PortalLines(offer.Element)
	for _, para := range []string{about, inElement} {
		for _, line := range systems.WrapRuns(ui.TipLine(para), prose, width) {
			systems.DrawRuns(screen, line, prose, x, y, portalProse)
			y += portalLinePitch
		}
		y += portalParaGap
	}
}

// enter walks the run through a portal and on to the next fight.
func (s *PortalScene) enter(gs *state.GlobalState, i int) {
	if gs.Run == nil {
		return
	}
	offers := gs.Run.PortalOffers()
	key, err := gs.Run.TakePortal(i)
	if err != nil {
		return
	}
	gs.Journal.Write(journal.Record{
		Kind:   journal.KindPortal,
		Key:    key,
		Action: offers[i].Element,
		Seat:   i,
		Fight:  gs.Run.Fight(),
	})
	advanceRun(gs)
}

// portalPanelRect is the top-left of portal i's panel.
func portalPanelRect(gs *state.GlobalState, i int) struct{ x, y int } {
	total := data.PortalOffers*portalPanelW + (data.PortalOffers-1)*portalPanelGap
	left := gs.PctX(50) - total/2
	return struct{ x, y int }{left + i*(portalPanelW+portalPanelGap), gs.PctY(portalPanelTop)}
}
