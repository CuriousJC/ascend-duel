package screens

// **The Compendium's statistics book: the player's lifetime figures, a section at a time.**
//
// **Three records behind three buttons** — unseeded runs, runs started on a chosen seed, and both
// together. A chosen seed earns nothing, so its figures are kept beside the achievements' rather than
// mixed into them: the unseeded page is the one an achievement's tally could have been earned
// against, the seeded page is still a true account of what was played, and BOTH is the whole of it.
// See profile.Profile.RecordRun.
//
// **Each section is a card**: an oversized playing card's frame with the heading at its top and its
// rows down it, several across the body, scrolled a row of cards at a time. A section longer than a
// card holds goes on over a second card under the same heading.
//
// **What a row says is worked out here, and nothing here is stored.** Every figure is a read of a
// profile.Tally by one of the names in internal/session/stats.go, so a row added here is a row with
// no migration behind it.

import (
	"image"
	"image/color"
	"sort"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The cards. **A card is near the playing card's 5:7**, sized so four stand across the body and two
// rows down it, and the scrollbar is told rows of cards rather than pixels, for the achievements
// page's reason: a page that showed a row and a half would have a bottom row that lies.
const (
	statCardW     = 320
	statCardH     = 436
	statCardGapX  = 20
	statCardGapY  = 20
	statCardPadX  = 18
	statCardHeadH = 52
	statRowHeight = 30

	// statRowSize is a row's type: the small tier, which is what lets four cards stand across.
	statRowSize = systems.TextSmall

	// statRowGap is the least air between a row's label and its figure, and statRowIndent how far
	// in a name on its own line stands.
	statRowGap    = 16
	statRowIndent = 12

	// statCardRows is how many rows stand on one card under its heading.
	statCardRows = (statCardH - statCardHeadH - statCardPadX) / statRowHeight

	// The record buttons, stacked down the left of the body.
	statTabWidth = 260
	statTabGap   = 16
)

// The card's two colors: a ring of the card border's gray on a face a step up from the panel.
var (
	statCardBorder = color.RGBA{R: 128, G: 128, B: 140, A: 255}
	statCardFill   = color.RGBA{R: 40, G: 40, B: 50, A: 255}
)

// statRecord is which of the profile's records the book is showing.
type statRecord int

const (
	statUnseeded statRecord = iota
	statSeeded
	statBoth
)

// statRecords is every record, in the order their buttons stand.
var statRecords = []statRecord{statUnseeded, statSeeded, statBoth}

func (r statRecord) label() string {
	switch r {
	case statSeeded:
		return "SEEDED"
	case statBoth:
		return "BOTH"
	default:
		return "UNSEEDED"
	}
}

// tally is the figures the record reads, the run in progress included.
func (r statRecord) tally(gs *state.GlobalState) profile.Tally {
	switch r {
	case statSeeded:
		return compendiumTally(gs, true)
	case statBoth:
		return everything(gs)
	default:
		return compendiumTally(gs, false)
	}
}

// statCard is one card of the book: a heading and the rows under it.
type statCard struct {
	heading string
	rows    []statRow
}

// statRow is one line of the book: a section heading, or a label and its figure.
//
// An indented row is the line under the one before it, carrying a name too wide to share it.
type statRow struct {
	heading bool
	indent  bool
	label   string
	value   string
}

func statHeading(label string) statRow      { return statRow{heading: true, label: label} }
func statName(name string) statRow          { return statRow{label: name, indent: true} }
func statLine(label, value string) statRow  { return statRow{label: label, value: value} }
func statCount(label string, n int) statRow { return statLine(label, thousands(n)) }

// statsDialog opens the book on the unseeded record.
func statsDialog(gs *state.GlobalState) *compendiumDialog { return statsDialogFor(gs, statUnseeded) }

// statsDialogFor builds the book for one record.
func statsDialogFor(gs *state.GlobalState, rec statRecord) *compendiumDialog {
	return &compendiumDialog{
		title:  "STATISTICS",
		cards:  statCards(statRows(gs, rec.tally(gs), rec == statSeeded)),
		stats:  true,
		record: rec,
	}
}

// statCards deals the rows onto cards, a card per heading, carrying a section too long for one card
// onto the next under the same heading.
func statCards(rows []statRow) []statCard {
	var out []statCard
	for _, row := range rows {
		if row.heading {
			out = append(out, statCard{heading: row.label})
			continue
		}
		if len(out) == 0 || len(out[len(out)-1].rows) == statCardRows {
			heading := ""
			if len(out) > 0 {
				heading = out[len(out)-1].heading
			}
			out = append(out, statCard{heading: heading})
		}
		last := &out[len(out)-1]
		last.rows = append(last.rows, row)
	}
	return out
}

// statRows is every row of the book, in order.
func statRows(gs *state.GlobalState, t profile.Tally, chosen bool) []statRow {
	defeats := t.Total(session.StatRunsDefeat)
	abandons := t.Total(session.StatRunsAbandoned)
	claimed := t.Total(session.StatRunsClaimed)

	rows := []statRow{
		statHeading("RUNS"),
		statCount("Runs finished", defeats+abandons+claimed),
		statCount("Journeys won", t.Total(session.StatVictories)),
		statCount("Victories claimed", claimed),
		statCount("Ended in defeat", defeats),
		statCount("Abandoned", abandons),
		statCount("Furthest realm reached", t.MostOf(session.StatBestRealm)),
		statCount("Most duels won in one run", t.MostOf(session.StatBestRunWins)),
		statCount("Most damage in one run", t.MostOf(session.StatBestRunDealt)),

		statHeading("DUELS"),
		statCount("Duels won", t.Total(session.StatFightsWon)),
		statCount("Duels lost", t.Total(session.StatFightsLost)),
		statCount("Rounds fought", t.Total(session.StatRounds)),
		statLine("Fastest win", fastestWin(t)),
		statCount("Most damage in one duel", t.MostOf(session.StatBestFight)),

		statHeading("COMBAT"),
		statCount("Damage dealt", t.Total(session.StatDealt)),
		statCount("Damage taken", t.Total(session.StatTaken)),
		statCount("Life restored", t.Total(session.StatHealed)),
		statCount("Hits landed", t.Total(session.StatHits)),
		statCount("Hits stopped by shields", t.Total(session.StatBlocked)),
		statCount("Biggest hit", t.MostOf(session.StatBestHit)),
		statCount("Biggest round", t.MostOf(session.StatBestRound)),
		statCount("Cards played", t.Total(session.StatPlayed)),
	}

	// **What was dealt and what was played are two cards**, each the elements and then the forms, a
	// name and one figure a row, because a row carrying both figures is wider than a card.
	var elements []combat.Element
	for _, e := range combat.AllElements {
		if e == combat.Basic && t.Total(session.StatPlayedElement+e.String()) == 0 &&
			t.Total(session.StatDealtElement+e.String()) == 0 {
			continue
		}
		elements = append(elements, e)
	}
	rows = append(rows, statHeading("DAMAGE DEALT"))
	for _, e := range elements {
		rows = append(rows, statCount(titleCase(e.String()), t.Total(session.StatDealtElement+e.String())))
	}
	for _, f := range combat.Forms() {
		rows = append(rows, statCount(titleCase(f.String()), t.Total(session.StatDealtForm+f.String())))
	}
	rows = append(rows, statHeading("CARDS PLAYED"))
	for _, e := range elements {
		rows = append(rows, statCount(titleCase(e.String()), t.Total(session.StatPlayedElement+e.String())))
	}
	for _, f := range combat.Forms() {
		rows = append(rows, statCount(titleCase(f.String()), t.Total(session.StatPlayedForm+f.String())))
	}

	// **Only the rungs that have been formed.** The ladder is discovered rather than given, and a
	// list naming every rung at zero would hand it over.
	rows = append(rows, statHeading("HANDS FORMED"))
	formed := 0
	for _, h := range combat.Hands() {
		if n := t.Total(session.StatHandFormed + h.Key); n > 0 {
			rows = append(rows, statCount(h.Name, n))
			formed++
		}
	}
	if formed == 0 {
		rows = append(rows, statLine("None yet", ""))
	}

	rows = append(rows,
		statHeading("VITAE"),
		statCount("Vitae earned", t.Total(session.StatVitaeEarned)),
		statCount("Vitae spent", t.Total(session.StatVitaeSpent)),
		statCount("Most vitae held at once", t.MostOf(session.StatBestVitaeHeld)),

		statHeading("THE SHOP"),
		statCount("Relics bought", familyTotal(t, session.StatRelicBought)),
		statCount("Boosters opened", familyTotal(t, session.StatBoosterBought)),
		statCount("Potions drunk", familyTotal(t, session.StatPotionDrunk)),
		statCount("Tonics drunk", familyTotal(t, session.StatTonicDrunk)),
		statCount("Essences used", familyTotal(t, session.StatEssenceUsed)),
		statCount("Runes used", familyTotal(t, session.StatRuneUsed)),
		statCount("Stones used", familyTotal(t, session.StatStoneUsed)),
		statCount("Cantrips cast", familyTotal(t, session.StatCantripCast)),

		statHeading("CREATURES"),
		statCount("Creatures beaten", familyTotal(t, session.StatCreatureDefeated)),
	)
	rows = append(rows, creatureLeader(gs, t, "Beaten most often", session.StatCreatureDefeated)...)
	rows = append(rows, creatureLeader(gs, t, "Deadliest", session.StatCreatureKiller)...)

	// **Achievements are the rolled record's alone**, because a chosen seed earns none — a row on
	// the chosen page reading "0 of N" would say the player had failed at something they were never
	// playing for.
	if !chosen {
		rows = append(rows,
			statHeading("ACHIEVEMENTS"),
			statLine("Earned", achievementTally(gs)),
			statCount("Unlocks opened", unlocksOpened(gs)),
		)
	}
	return rows
}

// fastestWin is the fewest rounds a duel has been won in, or a dash for a record that has none.
func fastestWin(t profile.Tally) string {
	n, ok := t.LeastOf(session.StatFastestWin)
	switch {
	case !ok:
		return "-"
	case n == 1:
		return "1 round"
	default:
		return strconv.Itoa(n) + " rounds"
	}
}

// familyTotal adds up every figure filed under one prefix — every relic bought, every potion drunk.
func familyTotal(t profile.Tally, prefix string) int {
	n := 0
	for k, v := range t.Totals {
		if strings.HasPrefix(k, prefix) {
			n += v
		}
	}
	return n
}

// creatureLeader is the creature with the most against one family: the label and its count, then the
// name on a line of its own, since a creature's full name is as wide as a card. A dash when there is
// none. **Ties go to the key that sorts first**, because map order may never decide anything.
func creatureLeader(gs *state.GlobalState, t profile.Tally, label, prefix string) []statRow {
	keys := make([]string, 0)
	for k := range t.Totals {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	best, most := "", 0
	for _, k := range keys {
		if v := t.Totals[k]; v > most {
			best, most = strings.TrimPrefix(k, prefix), v
		}
	}
	if best == "" {
		return []statRow{statLine(label, "-")}
	}
	name := best
	if r, ok := gs.Records[best]; ok {
		name = r.FullName()
	}
	return []statRow{statCount(label, most), statName(name)}
}

// unlocksOpened is how many unlock keys the profile holds.
func unlocksOpened(gs *state.GlobalState) int {
	if gs.Profile == nil {
		return 0
	}
	return len(gs.Profile.Unlocks)
}

// titleCase capitalizes the first letter of a vocabulary word: `fire` as "Fire".
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// statCardGrid is how many cards stand across the body and how many rows of them down it.
func statCardGrid(body image.Rectangle) (cols, rows int) {
	cols = (body.Dx() + statCardGapX) / (statCardW + statCardGapX)
	rows = (body.Dy() + statCardGapY) / (statCardH + statCardGapY)
	return max(cols, 1), max(rows, 1)
}

// placeTabs stands the record buttons down the left of the body, top-aligned with the cards, the
// one latched that is showing.
func (s *CompendiumScene) placeTabs(body image.Rectangle) {
	x := body.Min.X - compendiumScrollGap - statTabWidth/2
	for i, b := range s.recordTabs {
		b.ScreenX = x
		b.ScreenY = body.Min.Y + ui.ButtonSmall/2 + i*(ui.ButtonSmall+statTabGap)
		b.Latched = statRecords[i] == s.open.record
	}
}

// statFrame is the card every section is drawn on, made once.
var statFrame *ebiten.Image

// drawStatCards draws the visible rows of cards.
func (s *CompendiumScene) drawStatCards(gs *state.GlobalState, screen *ebiten.Image, body image.Rectangle) {
	if statFrame == nil {
		statFrame = ebiten.NewImageFromImage(cards.Frame(statCardW, statCardH, statCardBorder, statCardFill))
	}
	heading := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: systems.TextLarge}
	label := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: statRowSize}

	cols, rows := statCardGrid(body)
	used := cols*statCardW + (cols-1)*statCardGapX
	left := body.Min.X + (body.Dx()-used)/2
	for i, card := range s.open.cards {
		row := i/cols - s.scroll.Offset
		if row < 0 || row >= rows {
			continue
		}
		x := left + (i%cols)*(statCardW+statCardGapX)
		y := body.Min.Y + row*(statCardH+statCardGapY)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(statFrame, op)

		head := &text.DrawOptions{}
		head.GeoM.Translate(float64(x+statCardW/2), float64(y+statCardHeadH/2+4))
		head.PrimaryAlign = text.AlignCenter
		head.SecondaryAlign = text.AlignCenter
		head.ColorScale.ScaleWithColor(compendiumQuietInk)
		systems.DrawText(screen, card.heading, heading, head)

		for j, r := range card.rows {
			ry := y + statCardHeadH + j*statRowHeight + statRowHeight/2

			lx := x + statCardPadX
			if r.indent {
				lx += statRowIndent
			}
			lab := &text.DrawOptions{}
			lab.GeoM.Translate(float64(lx), float64(ry))
			lab.SecondaryAlign = text.AlignCenter
			lab.ColorScale.ScaleWithColor(compendiumInk)
			systems.DrawText(screen, r.label, label, lab)

			val := &text.DrawOptions{}
			val.GeoM.Translate(float64(x+statCardW-statCardPadX), float64(ry))
			val.PrimaryAlign = text.AlignEnd
			val.SecondaryAlign = text.AlignCenter
			val.ColorScale.ScaleWithColor(compendiumInk)
			systems.DrawText(screen, r.value, label, val)
		}
	}
}
