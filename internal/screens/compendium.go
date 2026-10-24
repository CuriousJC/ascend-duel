package screens

// **The Compendium: the player's book of the game.**
//
// One screen off the title menu holding everything the profile has to say about a player: their
// lifetime statistics, their achievements, and every catalog in the game — the relics, the cantrips,
// the potions and the rest — with what they have found of each and how often they have used it.
//
// **A page of books, and each book opens as a dialog over it** — the modal shape the deck and the
// ledger already have, so the player learns no new kind of panel. Achievements are the one book that
// is a screen of its own, reached from here and coming back here.
//
// **Like Achievements, it is the program's screen rather than a run's.** It never touches
// session.Phase, and it reads the profile — with the run in progress added on, so the page says the
// same thing whether it is opened between climbs or halfway up one. See compendiumTally.
//
// **Nothing here writes anything.** The figures are recorded by internal/session as a run plays and
// handed to the profile once, when it ends — see internal/session/stats.go.

import (
	"image"
	"image/color"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/actions"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The page's shape: a heading, a grid of books, and the way back.
const (
	compendiumTitle     = "COMPENDIUM"
	compendiumTitleSize = 40

	compendiumBookWidth   = 420
	compendiumBookHeight  = ui.ButtonLarge
	compendiumBookColumns = 3
	compendiumBookGapX    = 40
	compendiumBookGapY    = 28

	// The dialog's insides: where the body starts under the panel's title, and the scrollbar down
	// the right. **Nothing stands between a book's title and its body** — a catalog's tally rides on
	// the title and the statistics book's record buttons stand down the left — so the body starts
	// where a title-only panel's does.
	compendiumCatalogTop  = 72
	compendiumBodyBottom  = 30
	compendiumBodyInset   = 24
	compendiumScrollWidth = 14
	compendiumScrollGap   = 24
	compendiumCaptionSize = systems.TextSmall
)

// The inks a dialog's words are written in. **The panel is dark**, so these are light, and a quiet
// line is pulled toward the panel rather than toward black.
var (
	compendiumInk      = color.RGBA{R: 236, G: 236, B: 242, A: 255}
	compendiumQuietInk = color.RGBA{R: 160, G: 160, B: 176, A: 255}
)

// compendiumBook is one entry on the page: what its button says, and what pressing it does.
//
// **A book either opens a dialog or leaves for a screen**, and which is said by which of the two
// fields is set. Achievements is the one that leaves.
type compendiumBook struct {
	label string
	open  func(gs *state.GlobalState) *compendiumDialog
	leave func(gs *state.GlobalState)
}

// compendiumBooks is every book on the page, in the order they are shelved. **One list read by
// Init, Update and Draw**, for TitleScene.menu's reason.
var compendiumBooks = []compendiumBook{
	{label: "STATISTICS", open: statsDialog},
	{label: "ACHIEVEMENTS", leave: actions.OpenAchievements},
	{label: "CREATURES", open: creaturesDialog},
	{label: "REALMS", open: realmsDialog},
	{label: "RELICS", open: relicsDialog},
	{label: "CANTRIPS", open: cantripsDialog},
	{label: "POTIONS", open: potionsDialog},
	{label: "TONICS", open: tonicsDialog},
	{label: "ESSENCES", open: essencesDialog},
	{label: "RUNES", open: runesDialog},
	{label: "STONES", open: stonesDialog},
	{label: "BOOSTERS", open: boostersDialog},
	{label: "TRAITS", open: traitsDialog},
}

// compendiumDialog is one open book. **Either a grid of cards or a column of figures**, and which
// is said by which of the two lists is filled — the statistics book is rows, every catalog is cards.
type compendiumDialog struct {
	// title is the book's name, and tally the "(N/M)" found count written after it on the one line.
	title   string
	tally   string
	entries []compendiumEntry
	cards   []statCard

	// pic and captionH are one catalog entry's picture and the tag lines under it: the reduced card
	// for every book but the realms, whose rooms are landscape. **A count is never under a card**;
	// it is the last line of the card's tooltip, so a page holds another row of cards.
	pic      image.Point
	captionH int

	// facets are the book's filter chips, and shown the indices of the entries they let through, in
	// book order. **A book with no facets shows everything** — see compendium_filter.go.
	facets []*compendiumFacet
	shown  []int

	// perCol is how many chips stand down a column, and refiltered that a chip was pressed since the
	// scene last looked, so the book goes back to its top.
	perCol     int
	refiltered bool

	// stats is the statistics book's: it is rebuilt when its buttons move between the records, so it
	// remembers which it is showing.
	stats  bool
	record statRecord
}

// CompendiumScene is the Compendium screen.
type CompendiumScene struct {
	back  *models.Button
	books []*models.Button

	// The open book, its X, its scrollbar and its tooltip. **nil is the page with no book open.**
	open   *compendiumDialog
	closer ui.ModalCloser
	scroll *models.Scrollbar
	tip    models.Tooltip

	// peek is the found creature under the cursor, drawn large beside its deck, and peekSeat the
	// card it is about. **Re-asked every frame**, like the tooltip, so it is nil the moment the
	// cursor leaves.
	peek     *creaturePeek
	peekSeat image.Rectangle

	// The statistics book's record buttons, one per statRecord in statRecords' order.
	recordTabs []*models.Button
}

// Init builds the widgets on first entry and positions them every time.
func (s *CompendiumScene) Init(gs *state.GlobalState) {
	if s.back == nil {
		s.back = models.NewButton(320, ui.ButtonLarge, "BACK", func() { s.leave(gs) })
		for _, b := range compendiumBooks {
			book := b
			btn := models.NewButton(compendiumBookWidth, compendiumBookHeight, book.label,
				func() { s.openBook(gs, book) })
			btn.TextSize = 36
			s.books = append(s.books, btn)
		}
		s.scroll = models.NewScrollbar(compendiumScrollWidth, 100)
		for _, r := range statRecords {
			rec := r
			btn := models.NewButton(statTabWidth, ui.ButtonSmall, rec.label(), func() { s.showStats(gs, rec) })
			btn.TextSize = 26
			s.recordTabs = append(s.recordTabs, btn)
		}
	}
	s.back.ScreenX, s.back.ScreenY = gs.PctX(50), gs.PctY(88)

	rows := (len(s.books) + compendiumBookColumns - 1) / compendiumBookColumns
	gridW := compendiumBookColumns*compendiumBookWidth + (compendiumBookColumns-1)*compendiumBookGapX
	gridH := rows*compendiumBookHeight + (rows-1)*compendiumBookGapY
	left := gs.PctX(50) - gridW/2 + compendiumBookWidth/2
	top := gs.PctY(52) - gridH/2 + compendiumBookHeight/2
	for i, b := range s.books {
		col, row := i%compendiumBookColumns, i/compendiumBookColumns
		b.ScreenX = left + col*(compendiumBookWidth+compendiumBookGapX)
		b.ScreenY = top + row*(compendiumBookHeight+compendiumBookGapY)
	}

	// **Closed on every entry**: a page re-opened with a book already up is a dialog nobody asked
	// for. Coming back from Achievements lands on the shelf.
	s.open = nil
	s.peek = nil
	s.tip = models.Tooltip{DwellTicks: ui.TipDwell()}
}

// openBook does whatever a book's button does.
func (s *CompendiumScene) openBook(gs *state.GlobalState, b compendiumBook) {
	if b.leave != nil {
		b.leave(gs)
		return
	}
	s.open = b.open(gs)
	s.scroll.Offset = 0
}

// showStats moves the statistics book between its records.
func (s *CompendiumScene) showStats(gs *state.GlobalState, rec statRecord) {
	if s.open == nil || !s.open.stats || s.open.record == rec {
		return
	}
	s.open = statsDialogFor(gs, rec)
	s.scroll.Offset = 0
}

func (s *CompendiumScene) Update(gs *state.GlobalState) error {
	if s.open == nil {
		for _, b := range s.books {
			systems.UpdateButton(gs, b)
		}
		systems.UpdateButton(gs, s.back)
		systems.UpdateTooltip(gs, &s.tip)
		return nil
	}

	gs.ModalOpen = true
	s.peek = nil
	if s.closer.Update(gs) {
		s.open = nil
		systems.UpdateTooltip(gs, &s.tip)
		return nil
	}

	body := compendiumBody(gs, s.open)
	track := image.Rect(body.Max.X+compendiumScrollGap, body.Min.Y,
		body.Max.X+compendiumScrollGap+compendiumScrollWidth, body.Max.Y)
	s.scroll.Width, s.scroll.Height = track.Dx(), track.Dy()
	s.scroll.ScreenX, s.scroll.ScreenY = track.Min.X+track.Dx()/2, track.Min.Y+track.Dy()/2

	if s.open.stats {
		s.placeTabs(body)
		for _, b := range s.recordTabs {
			systems.UpdateButton(gs, b)
		}
		cols, rows := statCardGrid(body)
		s.scroll.Total, s.scroll.Visible = (len(s.open.cards)+cols-1)/cols, rows
	} else {
		s.updateChips(gs)
		cols, rows := s.open.catalogGrid(body)
		s.scroll.Total = (len(s.open.shown) + cols - 1) / cols
		s.scroll.Visible = rows
		s.pointAtEntry(gs, body)
	}
	systems.UpdateScrollbar(gs, s.scroll)
	systems.UpdateTooltip(gs, &s.tip)
	return nil
}

func (s *CompendiumScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	ui.FillScreenBackdrop(gs, screen)

	heading := &text.DrawOptions{}
	heading.GeoM.Translate(float64(gs.PctX(50)), float64(gs.PctY(14)))
	heading.PrimaryAlign = text.AlignCenter
	heading.SecondaryAlign = text.AlignCenter
	heading.ColorScale.ScaleWithColor(ui.GroundInk)
	systems.DrawText(screen, compendiumTitle,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: compendiumTitleSize}, heading)

	for _, b := range s.books {
		systems.DrawButton(gs, screen, b)
	}
	systems.DrawButton(gs, screen, s.back)

	if s.open != nil {
		s.drawDialog(gs, screen)
	}
	s.drawPeek(gs, screen)
	systems.DrawTooltip(gs, screen, &s.tip)
}

// drawDialog puts the open book up: the panel, its title and tally, its body, its scrollbar and its X.
func (s *CompendiumScene) drawDialog(gs *state.GlobalState, screen *ebiten.Image) {
	title := s.open.title
	if s.open.tally != "" {
		title += " " + s.open.tally
	}
	ui.DrawModalFrame(gs, screen, ui.ModalHead{Title: title})

	body := compendiumBody(gs, s.open)
	if s.open.stats {
		for _, b := range s.recordTabs {
			systems.DrawButton(gs, screen, b)
		}
		s.drawStatCards(gs, screen, body)
	} else {
		s.drawChips(gs, screen)
		s.drawCatalog(gs, screen, body)
	}
	systems.DrawScrollbar(gs, screen, s.scroll)
	s.closer.Draw(gs, screen)
}

// compendiumBody is the rectangle a book's contents are drawn in: inside the panel, under its title,
// between any filter chips or record buttons, and clear of the scrollbar down the right.
func compendiumBody(gs *state.GlobalState, d *compendiumDialog) image.Rectangle {
	r := ui.ModalPanelRect(gs)
	left, right := d.chipSides()
	if d.stats {
		left += statTabWidth + compendiumScrollGap
	}
	return image.Rect(
		r.Min.X+compendiumBodyInset+left,
		r.Min.Y+compendiumCatalogTop,
		r.Max.X-compendiumBodyInset-compendiumScrollGap-compendiumScrollWidth-right,
		r.Max.Y-compendiumBodyBottom,
	)
}

// leave goes back to whichever screen opened this one, on the terms every menu screen's Back does.
func (s *CompendiumScene) leave(gs *state.GlobalState) {
	gs.PopReturn()
}

// compendiumTally is what one of the profile's two records says, with the run in progress added on
// if it is the same kind of run.
//
// **The run in progress is added rather than left out**, because a player who opens the page halfway
// up a climb and finds the relic they bought an hour ago still marked unfound has been shown a page
// that is wrong. The profile is only handed a run when it ends; this is the page catching up.
func compendiumTally(gs *state.GlobalState, chosen bool) profile.Tally {
	var out profile.Tally
	if gs.Profile != nil {
		if chosen {
			out = out.Plus(gs.Profile.ChosenStats)
		} else {
			out = out.Plus(gs.Profile.Stats)
		}
	}
	if gs.Run != nil && gs.Run.SeedChosen() == chosen {
		out = out.Plus(gs.Run.Stats())
	}
	return out
}

// everything is both records and the run in progress together: what a catalog reads.
//
// **A catalog does not split rolled from chosen** — whether a player has ever held a relic is a fact
// about what they have seen, and a chosen seed shows them the same relics a rolled one does. The
// split is the statistics book's, where the figures sit beside the achievements.
func everything(gs *state.GlobalState) profile.Tally {
	return compendiumTally(gs, false).Plus(compendiumTally(gs, true))
}

// thousands writes a figure with a comma every three digits, which is how a lifetime total stays
// readable once it is in the tens of thousands.
func thousands(n int) string {
	s := strconv.Itoa(n)
	neg := n < 0
	if neg {
		s = s[1:]
	}
	out := make([]byte, 0, len(s)+len(s)/3)
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}
