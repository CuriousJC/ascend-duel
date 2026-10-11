package screens

// **The Compendium's statistics book: the player's lifetime figures, a section at a time.**
//
// **Two records behind two tabs** — rolled runs, and runs started on a chosen seed. A chosen seed
// earns nothing, so its figures are kept beside the achievements' rather than mixed into them: the
// rolled page is the one an achievement's tally could have been earned against, and the chosen page
// is still a true account of what was played. See profile.Profile.RecordRun.
//
// **What a row says is worked out here, and nothing here is stored.** Every figure is a read of a
// profile.Tally by one of the names in internal/session/stats.go, so a row added here is a row with
// no migration behind it.

import (
	"image"
	"sort"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The column. **A count of rows rather than a pixel height**, for the achievements page's reason:
// the scrollbar is told rows, and a page that showed a row and a half would have a bottom row that
// lies.
const (
	statRowHeight  = 40
	statColumnW    = 940
	statHeadingGap = 10
	statTabGap     = 24
)

// statRow is one line of the book: a section heading, or a label and its figure.
type statRow struct {
	heading bool
	label   string
	value   string
}

func statHeading(label string) statRow      { return statRow{heading: true, label: label} }
func statLine(label, value string) statRow  { return statRow{label: label, value: value} }
func statCount(label string, n int) statRow { return statLine(label, thousands(n)) }

// statsDialog opens the book on the rolled record.
func statsDialog(gs *state.GlobalState) *compendiumDialog { return statsDialogFor(gs, false) }

// statsDialogFor builds the book for one of the two records.
func statsDialogFor(gs *state.GlobalState, chosen bool) *compendiumDialog {
	return &compendiumDialog{
		title:  "STATISTICS",
		rows:   statRows(gs, compendiumTally(gs, chosen), chosen),
		stats:  true,
		chosen: chosen,
	}
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

	rows = append(rows, statHeading("BY ELEMENT"))
	for _, e := range combat.AllElements {
		played := t.Total(session.StatPlayedElement + e.String())
		dealt := t.Total(session.StatDealtElement + e.String())
		if e == combat.Basic && played == 0 && dealt == 0 {
			continue
		}
		rows = append(rows, statLine(titleCase(e.String()), splitFigure(dealt, played)))
	}

	rows = append(rows, statHeading("BY FORM"))
	for _, f := range combat.Forms() {
		played := t.Total(session.StatPlayedForm + f.String())
		dealt := t.Total(session.StatDealtForm + f.String())
		rows = append(rows, statLine(titleCase(f.String()), splitFigure(dealt, played)))
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

		statHeading("THE SHOP AND THE PACK"),
		statCount("Relics bought", familyTotal(t, session.StatRelicBought)),
		statCount("Sealed goods opened", familyTotal(t, session.StatGoodBought)),
		statCount("Potions drunk", familyTotal(t, session.StatPotionDrunk)),
		statCount("Tonics drunk", familyTotal(t, session.StatTonicDrunk)),
		statCount("Essences used", familyTotal(t, session.StatEssenceUsed)),
		statCount("Runes used", familyTotal(t, session.StatRuneUsed)),
		statCount("Stones used", familyTotal(t, session.StatStoneUsed)),
		statCount("Cantrips cast", familyTotal(t, session.StatCantripCast)),

		statHeading("CREATURES"),
		statCount("Creatures beaten", familyTotal(t, session.StatCreatureDefeated)),
		statLine("Beaten most often", creatureLeader(gs, t, session.StatCreatureDefeated)),
		statLine("Deadliest", creatureLeader(gs, t, session.StatCreatureKiller)),
	)

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

// splitFigure is an element's or a form's row: what it dealt, and how many cards of it were played.
func splitFigure(dealt, played int) string {
	return thousands(dealt) + " dealt / " + thousands(played) + " cards"
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

// creatureLeader is the creature with the most against one family, by name, or a dash. **Ties go to
// the key that sorts first**, because map order may never decide anything.
func creatureLeader(gs *state.GlobalState, t profile.Tally, prefix string) string {
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
		return "-"
	}
	name := best
	if r, ok := gs.Records[best]; ok {
		name = r.FullName()
	}
	return name + " (" + thousands(most) + ")"
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

// statRowsVisible is how many rows the body holds.
func statRowsVisible(body image.Rectangle) int {
	return max(body.Dy()/statRowHeight, 1)
}

// placeTabs puts the two record tabs on the line under the book's title, the one latched that is
// showing.
func (s *CompendiumScene) placeTabs(gs *state.GlobalState) {
	r := ui.ModalPanelRect(gs)
	y := r.Min.Y + compendiumSubtitleTop
	cx := r.Min.X + r.Dx()/2
	s.rolledTab.ScreenX, s.rolledTab.ScreenY = cx-s.rolledTab.Width/2-statTabGap/2, y
	s.chosenTab.ScreenX, s.chosenTab.ScreenY = cx+s.chosenTab.Width/2+statTabGap/2, y
	s.rolledTab.Latched = !s.open.chosen
	s.chosenTab.Latched = s.open.chosen
}

// drawStatRows draws the visible rows of the book.
func (s *CompendiumScene) drawStatRows(gs *state.GlobalState, screen *ebiten.Image, body image.Rectangle) {
	label := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: systems.TextMedium}
	left := body.Min.X + (body.Dx()-statColumnW)/2
	right := left + statColumnW

	visible := statRowsVisible(body)
	for i := 0; i < visible && s.scroll.Offset+i < len(s.open.rows); i++ {
		row := s.open.rows[s.scroll.Offset+i]
		y := body.Min.Y + i*statRowHeight + statRowHeight/2

		op := &text.DrawOptions{}
		op.SecondaryAlign = text.AlignCenter
		if row.heading {
			op.GeoM.Translate(float64(left), float64(y+statHeadingGap/2))
			op.ColorScale.ScaleWithColor(compendiumQuietInk)
			systems.DrawText(screen, row.label, label, op)
			continue
		}
		op.GeoM.Translate(float64(left+24), float64(y))
		op.ColorScale.ScaleWithColor(compendiumInk)
		systems.DrawText(screen, row.label, label, op)

		val := &text.DrawOptions{}
		val.GeoM.Translate(float64(right), float64(y))
		val.PrimaryAlign = text.AlignEnd
		val.SecondaryAlign = text.AlignCenter
		val.ColorScale.ScaleWithColor(compendiumInk)
		systems.DrawText(screen, row.value, label, val)
	}
}
