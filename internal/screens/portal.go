package screens

// **The portal: which realm comes next.**
//
// A portal room's boss opens two portals, and this is the station the run stands at after the shop
// that follows it. Each portal is a gate with a button in the middle of it, and under the gate a
// panel — the realm's name, what the motif is, what the element does to it. Taking one is the only
// way on.
//
// **The way through is a swirl, not a button**: a round, rim-faded picture turning in place over the
// heart of each painted portal, faster under the cursor. See models.Swirl. It carries no word, so the
// turning says it can be pressed and the tooltip names the realm it leads to.
//
// **The gates are painted, not drawn.** The screen's backdrop, `screen-portal` in
// `data/screen_art.json`, carries a rainbow portal in each gate's rectangle, and the button stands
// at its heart. The gates are placed where the picture put its portals, so the picture leads and
// the layout follows it. `TestThePortalBackdropStandsWhereTheGatesAre` holds the record to portalGateRect. See
// session.Session.TakePortal for what a pick does and internal/journey for where the offers come
// from.
//
// **There is no Back and no Skip.** The shop is done and the next realm has to be one of the two;
// a screen that could be left without choosing would be a realm chosen by default.
//
// **Every word on it comes off the motif record** — `Name`, `Text` and `ElementText` — so the
// portal says what the catalog says, and an unwritten line shows as TBD rather than as a panel
// that looks finished.

import (
	"bytes"
	"image"
	_ "image/png"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/journal"
	"github.com/curiousjc/ascend-duel/internal/journey"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The page's shape.
const (
	portalHeadSize = 44
	portalSubSize  = systems.TextMedium

	// The heading and the realm line stand between the two gates, on the buttons' own line, because
	// the build band has the top of the screen.
	portalSubDrop = 46

	portalPanelW = 700
	portalPanelH = 320
	portalPad    = 40

	// The gates are where the backdrop painted its portals, not where a layout would put them: the
	// picture is a fixed 1920x1080 and the button has to stand at the heart of the swirl it was
	// given. portalGateMids are the two centers, measured off assets/screen/screen-portal.jpg.
	portalGateW   = 392
	portalGateH   = 560
	portalGateTop = 140
	portalGateGap = 15 // between the foot of the gate and the top of its panel

	portalTitleSize = 34
	portalProseSize = systems.TextMedium
	portalLinePitch = 31
	portalParaGap   = 18

	// portalSwirlRadius is how big the turning swirl is drawn over each portal's heart.
	portalSwirlRadius = 140
)

// portalSwirlArt is the picture every swirl turns, by its asset key.
const portalSwirlArt = "portal-swirl"

// portalSwirlImage is that picture decoded, once.
var portalSwirlImage *ebiten.Image

// portalSwirl is the swirl's picture, decoded on first use. nil when it is not embedded, which draws
// nothing and leaves the painted portal to stand on its own.
func portalSwirl(gs *state.GlobalState) *ebiten.Image {
	if portalSwirlImage == nil {
		raw := gs.ImageData[portalSwirlArt]
		if len(raw) == 0 {
			return nil
		}
		decoded, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil
		}
		portalSwirlImage = ebiten.NewImageFromImage(decoded)
	}
	return portalSwirlImage
}

// portalGateMids is the middle of each gate, which its button and its panel stand on. **A
// regenerated picture means re-measuring these**, and TestThePortalBackdropStandsWhereTheGatesAre
// is what fails when the record and this disagree.
var portalGateMids = [data.PortalOffers]int{532, 1438}

// portalPanel is a panel's face: the same dark surface and prose ink as the tooltips and Bob's
// bubble, because it is the same object — a box of type over the light table.
var (
	portalPanel = opaque(systems.PanelSurface)
	portalProse = systems.PanelSpeech
)

// PortalScene is the choice of realm.
//
// **The two swirls are the only way off it, and the frame's own controls stand down** — the cog,
// the ledger and the gallery; see chromeShowing. What stays is what the choice is made against:
// the build band, with its relics and consumables explained on hover, and the deck pile, whose
// panel opens over the screen like everywhere else.
type PortalScene struct {
	take [data.PortalOffers]*models.Swirl

	// deck is the panel over the whole deck, opened by clicking the pile.
	deck ui.DeckToggle

	// band is the top third's input, the same on every screen that shows it — see band.go.
	band bandControls

	tip models.Tooltip
}

// Init builds the swirls on first entry and positions them every time.
func (s *PortalScene) Init(gs *state.GlobalState) {
	for i := range s.take {
		if s.take[i] == nil {
			i := i
			s.take[i] = &models.Swirl{Radius: portalSwirlRadius, OnClick: func() { s.enter(gs, i) }}
		}
		g := portalGateRect(gs, i)
		s.take[i].ScreenX = (g.Min.X + g.Max.X) / 2
		s.take[i].ScreenY = (g.Min.Y + g.Max.Y) / 2
	}
	s.deck.InitAsPile()
	s.band.init()
	s.tip = models.Tooltip{DwellTicks: ui.TipDwell()}
}

func (s *PortalScene) Update(gs *state.GlobalState) error {
	// **A portal with nothing open walks on by itself.** Nothing reaches here without one today —
	// the loop walks past the station when PortalDue is false — but a blank page with no way off it
	// would be worse than the next fight.
	if gs.Run == nil || !gs.Run.PortalDue() {
		advanceRun(gs)
		return nil
	}

	// **The deck panel runs first and swallows the frame**, the shop's own order: while it is up
	// the swirls underneath are dead.
	if s.deck.Update(gs, ui.OwnedContents(gs)) {
		return nil
	}
	if gs.CursorAllowed() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) &&
		image.Pt(gs.MouseX, gs.MouseY).In(deckPileBounds(gs)) {
		s.deck.Toggle()
		s.tip.Forget()
		return nil
	}

	if s.band.update(gs, bandHooks{live: true, forget: s.tip.Forget}) {
		return nil
	}

	offers := gs.Run.PortalOffers()
	for i := range min(len(offers), len(s.take)) {
		systems.UpdateSwirl(gs, s.take[i])
	}

	if gs.CursorAllowed() {
		at := image.Pt(gs.MouseX, gs.MouseY)
		if !hoverBuildRelics(gs, at, &s.tip) {
			for i := range min(len(offers), len(s.take)) {
				if systems.SwirlCovers(s.take[i], at) {
					s.tip.Point(systems.SwirlBounds(s.take[i]),
						ui.TipLine("ENTER "+strings.ToUpper(s.realmName(gs, offers[i]))), nil)
				}
			}
		}
	}
	systems.UpdateTooltip(gs, &s.tip)
	return nil
}

func (s *PortalScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	ui.FillScreenBackdrop(gs, screen)
	if gs.Run == nil {
		return
	}

	drawBuildBand(gs, screen, gs.Run.Vitae(), &s.band, s.tip.Showing())
	drawDeckPile(gs, screen)

	center := float64(portalGateMids[0]+portalGateMids[len(portalGateMids)-1]) / 2
	line := float64(portalGateTop + portalGateH/2)

	head := &text.DrawOptions{}
	head.GeoM.Translate(center, line)
	head.PrimaryAlign = text.AlignCenter
	head.SecondaryAlign = text.AlignCenter
	head.ColorScale.ScaleWithColor(ui.GroundInk)
	systems.DrawText(screen, "CHOOSE YOUR REALM",
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: portalHeadSize}, head)

	sub := &text.DrawOptions{}
	sub.GeoM.Translate(center, line+portalSubDrop)
	sub.PrimaryAlign = text.AlignCenter
	sub.SecondaryAlign = text.AlignCenter
	sub.ColorScale.ScaleWithColor(systems.ColorToward(ui.GroundInk, ui.ScreenGround, 30))
	systems.DrawText(screen, "REALM "+strconv.Itoa(gs.Run.Realm()),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: portalSubSize}, sub)

	offers := gs.Run.PortalOffers()
	for i := range min(len(offers), len(s.take)) {
		s.drawPanel(gs, screen, i, offers[i])
		systems.DrawSwirl(screen, s.take[i], portalSwirl(gs))
	}
	drawBandOverlay(gs, screen, &s.band)
	systems.DrawTooltip(gs, screen, &s.tip)

	// Last, and over everything: the panel covers the screen.
	s.deck.Draw(gs, screen, ui.OwnedContents(gs))
}

// drawPanel is one portal: the realm's name with its element in the element's color, then what the
// motif is, then what the element does to it.
func (s *PortalScene) drawPanel(gs *state.GlobalState, screen *ebiten.Image, i int, offer journey.Realm) {
	r := portalPanelRect(gs, i)
	systems.BevelRect(screen, r.x, r.y, portalPanelW, portalPanelH, systems.PaneBevelWidth, portalPanel, false)

	motif := gs.Motifs[offer.Motif]
	name := s.realmName(gs, offer)

	x, y := r.x+portalPad, r.y+portalPad+portalTitleSize/2
	width := float64(portalPanelW - 2*portalPad)

	title := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: portalTitleSize}
	for _, line := range systems.WrapLine(ui.TipLine(strings.ToUpper(name)), title, width) {
		systems.DrawLine(screen, line, title, x, y, portalProse)
		y += portalTitleSize + 6
	}
	y += portalParaGap

	prose := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: portalProseSize}
	about, inElement := motif.PortalLines(offer.Element)
	for _, para := range []string{about, inElement} {
		for _, line := range systems.WrapLine(ui.TipLine(para), prose, width) {
			systems.DrawLine(screen, line, prose, x, y, portalProse)
			y += portalLinePitch
		}
		y += portalParaGap
	}
}

// realmName is what a portal leads to, in words: the element and the motif's name.
func (s *PortalScene) realmName(gs *state.GlobalState, offer journey.Realm) string {
	name := gs.Motifs[offer.Motif].Name
	if name == "" {
		name = offer.Motif
	}
	return offer.Element + " " + name
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

// portalGateRect is where portal i's gate is painted, and the button stands at its center.
func portalGateRect(gs *state.GlobalState, i int) image.Rectangle {
	mid := portalGateMids[i]
	return image.Rect(mid-portalGateW/2, portalGateTop, mid+portalGateW/2, portalGateTop+portalGateH)
}

// portalPanelRect is the top-left of portal i's panel, under its gate.
func portalPanelRect(gs *state.GlobalState, i int) struct{ x, y int } {
	return struct{ x, y int }{portalGateMids[i] - portalPanelW/2, portalGateRect(gs, i).Max.Y + portalGateGap}
}
