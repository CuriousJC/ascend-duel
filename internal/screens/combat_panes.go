package screens

// The two read-only panes — Action Flow and Resolution — their placements and colours, the
// row model they are both drawn from, and the prose that turns an event into a sentence.
//
// Split out of combat.go on 2026-08-07. **The prose lives here and not in internal/combat**
// on purpose: the rules package names actions, it does not describe them. Everything in this
// file is presentation over a log the engine has already finished deciding, which is what
// makes it structurally impossible for a pane to disagree with the round it reports.

import (
	"fmt"
	"image"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
)

// The Resolution pane's vertical band. It reaches higher than it did — the heading text
// that used to sit at the top of the screen is gone — and stops well short of the bottom,
// because the hand took the lower third when the cards turned portrait and the caption box
// sits between the two.
const (
	paneTopPct    = 12
	paneBottomPct = 46

	paneTitleInset = 10 // gap from the pane's top edge to its title
	paneFirstRow   = 45 // gap from the top edge to the first action row
	paneRowHeight  = 30
	paneRowInset   = 10 // gap from the pane's left edge to a row's swatch
	swatchSize     = 16
	swatchGap      = 6 // gap between a swatch and its label

	// The Resolution pane's rows are sentences rather than card names, and there are more of
	// them — a busy round merges to a dozen lines where the flow pane draws at most ten. A
	// tighter pitch is what fits them in the same band without the pane having to grow into
	// the caption's slot.
	paneTextRowHeight = 22
)

// panePlacement is one pane's horizontal slot, label and identifying colour. The
// colours are loud on purpose — these are placeholders for finding the layout, not a
// palette anyone has chosen yet.
type panePlacement struct {
	leftPct, rightPct int
	title             string
	color             color.RGBA

	// **A pane carries its own surface and its own ink**, rather than deriving both from one
	// colour. Resolution went off-white on 2026-08-07 because coloured verb chips on a dim
	// plum ground were hard to read — three saturated colours competing with a fourth behind
	// them. A light ground makes the chips the only saturated thing in the pane.
	//
	// This is the same exception glyphs are documented under in `CLAUDE.md`: the one-colour
	// rule governs how a widget responds to hover, press and disable, and it cannot describe
	// a surface and the thing sitting on it at once. `color` still drives the border and is
	// what the pane is "named", so the scale-don't-add rule keeps working for state.
	fill   color.RGBA // the pane's ground
	ink    color.RGBA // text drawn on that ground
	nowInk color.RGBA // text of the row playback is on: coloured, bold and underlined

	// rowHeight is the pitch this pane draws its rows at. Carried on the placement rather
	// than being one constant because the two panes hold different things: card names, and
	// sentences about what those cards did.
	rowHeight int
}

// **Two panes, and they answer different questions at different times** *(2026-08-07)*.
//
//   - **Action Flow** is what you *queued*, in play order. It is live while you are planning,
//     before anything has happened — a prediction, and the thing drag-to-reorder edits.
//   - **Resolution** is what actually *happened*. It is empty until DUEL! is pressed and fills
//     as the round plays back — a record.
//
// Showing the round twice is only worth the space because of that split. It also retired the
// open question of how one pane could be both: the flow pane never learned to bracket a combo
// across non-adjacent rows, and no longer has to, because Resolution says it in words.
//
// The narrow column and the wide one are **not** interchangeable. Flow rows are short labels
// (`Strike`, `??? (attack)`) and fit the 15–39% column the Actions pane vacated; Resolution
// rows are sentences and keep the wide middle slot, which is also what the pane billed as the
// centrepiece should have.
var (
	// Action Flow keeps the dark ground it has always had. It is not drawn today, and it holds
	// card names rather than sentences with chips in them, so it has none of the problem that
	// moved Resolution to a light one. **If it comes back beside Resolution the two will want
	// deciding together** — one light pane and one dark one side by side is not a scheme.
	actionFlowPane = panePlacement{
		leftPct: 15, rightPct: 39,
		title:     "Action Flow",
		color:     paneEdge,
		fill:      systems.ColorAtStrength(paneEdge, 25),
		ink:       color.RGBA{R: 245, G: 245, B: 245, A: 255},
		nowInk:    color.RGBA{R: 255, G: 158, B: 205, A: 255},
		rowHeight: paneRowHeight,
	}
	resolutionPane = panePlacement{
		leftPct: 15, rightPct: 78,
		title:     "Resolution",
		color:     paneEdge,
		fill:      color.RGBA{R: 234, G: 230, B: 224, A: 255},
		ink:       color.RGBA{R: 34, G: 32, B: 38, A: 255},
		nowInk:    color.RGBA{R: 178, G: 22, B: 106, A: 255},
		rowHeight: paneTextRowHeight,
	}
)

// paneEdge is the pink both panes are bordered and named in. Still a placeholder palette.
var paneEdge = color.RGBA{R: 235, G: 105, B: 170, A: 255}

// comboSwatch marks a line that is not one side acting but something the round did — a combo
// forming. **It is the yellow the enemy used to be**, freed when the opponent went grey on
// 2026-08-07: a combo is the loudest thing that can happen in a round and had been sharing a
// hue with every enemy action on screen.
//
// Darker than a screen yellow because it sits on a light pane now — the same figure that read
// as amber on plum reads as washed-out cream on off-white.
var comboSwatch = color.RGBA{R: 198, G: 142, B: 16, A: 255}

// The two sides' colours: **green is you, grey is them.**
//
// The opponent was yellow until 2026-08-07 and went grey to give the yellow to `comboSwatch` —
// a combo is the loudest thing that can happen in a round and was sharing a hue with every
// enemy action on screen. Grey is also the right *rank* for the opponent: their rows are
// context for yours, and a saturated colour was claiming more attention than they earn.
//
// **It settles a collision recorded as open in `MECHANICS.md`**, where lightning's yellow card
// surface ran into `enemySwatch`. Green and the player still collide with poison/earth, so the
// element scheme is only half-untangled.
var (
	playerSwatch = color.RGBA{R: 46, G: 150, B: 70, A: 255}
	enemySwatch  = color.RGBA{R: 108, G: 110, B: 122, A: 255}
)

// paneRow is one line in a pane: a label, optionally preceded by a colour swatch
// saying whose action it is. A zero-alpha swatch means the row has none, in which case
// the label is centred instead of sitting in a column beside the squares.
type paneRow struct {
	// A row is drawn as three runs, so the verb in the middle can be coloured, bolded and
	// underlined while the words either side of it are not. Rows that are not a sentence —
	// a card name in Action Flow, a placeholder — put everything in prefix and leave the
	// other two empty, which is why prefix rather than verb is the one that always has to
	// be set.
	prefix, verb, suffix string

	// verbInk is the colour the verb itself is written in. **Zero alpha means "the row's own
	// ink"**, the same convention Button.BaseColor uses, and it is what the neutral category
	// takes — see verbInkFor. Storing a colour rather than a category keeps drawPane from
	// having to know anything about combat.
	verbInk color.RGBA

	swatch color.RGBA

	// highlighted marks the row as the one happening right now, drawn lit against the
	// dim pane behind it.
	highlighted bool
}

// drawActionFlow shows the two queues merged into play order: the plan, not the outcome.
func (s *CombatScene) drawActionFlow(gs *state.GlobalState, screen *ebiten.Image) {
	s.drawPane(gs, screen, actionFlowPane,
		s.actionFlowRows(s.fighterActions, s.enemyActions, s.concealEnemy(gs)))
}

// drawResolution shows what the round actually did, accumulating as it plays back.
func (s *CombatScene) drawResolution(gs *state.GlobalState, screen *ebiten.Image) {
	s.drawPane(gs, screen, resolutionPane, s.resolutionLines(gs))
}

// resolutionCapacity is how many lines fit between the pane's first row and its bottom edge.
// Derived rather than written down, so changing the band or the pitch cannot leave a constant
// behind claiming a capacity the pane does not have.
func resolutionCapacity(gs *state.GlobalState) int {
	h := gs.PctY(paneBottomPct) - gs.PctY(paneTopPct)
	n := (h - paneFirstRow) / paneTextRowHeight
	if n < 1 {
		return 1
	}
	return n
}

// resolutionLines turns the event log into one line per slot, an action and what it did,
// **built only from events playback has already reached**. That is what makes the pane a
// record rather than a spoiler: it says exactly as much as the player has been shown.
//
// **One line per slot, not one per event.** A busy round is 25-30 events and the pane holds a
// dozen lines, so drawing the log verbatim would need either a scrollback — and there is no
// scroll gesture in the input vocabulary, no wheel convention and no keyboard — or a pane that
// only ever showed the tail, which is the opposite of being able to read the round afterwards.
// Merging an action with its outcome is presentation of events the engine already decided; it
// computes nothing, so the pane still cannot disagree with the round.
//
// Combos and staggers get lines of their own. They are not something a card did, they are
// something that happened *to* the round, and folding a combo into the line of the card that
// happened to start it would bury the one thing this pane was added to show.
func (s *CombatScene) resolutionLines(gs *state.GlobalState) []paneRow {
	end := s.cursor + 1
	if end > len(s.log) {
		end = len(s.log)
	}
	if end <= 0 {
		return []paneRow{{prefix: "(press DUEL!)"}}
	}

	var rows []paneRow

	// cur is the line the next outcome attaches to, or -1 when the last thing appended was
	// an announcement rather than an action.
	cur := -1
	outcomes := 0

	// Outcomes are appended to the tail of the sentence, after the verb, so the coloured verb
	// never moves as a line grows.
	attach := func(what string) {
		if cur < 0 {
			return
		}
		sep := " - "
		if outcomes > 0 {
			sep = ", "
		}
		rows[cur].suffix += sep + what
		outcomes++
	}

	// act opens a line in the form "<who> <verb> <what>", with the verb carrying its
	// category's colour. See actionPhrase.
	act := func(side combat.Side, a combat.ActionKind) {
		rows = append(rows, paneRow{
			prefix:  s.sideName(side) + " ",
			verb:    verbFor(a.Category()),
			suffix:  " " + actionPhrase(a),
			verbInk: verbInkFor(a.Category()),
			swatch:  swatchFor(side),
		})
		cur = len(rows) - 1
		outcomes = 0
	}

	announce := func(label string, swatch color.RGBA) {
		rows = append(rows, paneRow{prefix: label, swatch: swatch})
		cur = -1
	}

	// Which defence stopped the last blow, so the damage that follows can be narrated as a
	// riposte's counter or a mirror's reflection. The engine already distinguishes them on the
	// KindNegated event; this only remembers it for the line after.
	lastNegation := combat.ActionKind(-1)

	for _, e := range s.log[:end] {
		switch e.Kind {
		case combat.KindRoundStart, combat.KindRoundEnd:
			// The pane holds one round, so saying which round it is would be a line spent on
			// something the caption and the character block both already carry.

		case combat.KindAction:
			act(e.Side, e.Action)

		case combat.KindStaggered:
			announce(fmt.Sprintf("%s is staggered - %v is lost", s.sideName(e.Side), e.Action),
				swatchFor(e.Side))

		case combat.KindCombo:
			name := "combo"
			if c, ok := combat.ComboByID(e.Combo); ok {
				name = c.Name
			}
			announce(fmt.Sprintf("COMBO!  %s lands a %s", s.sideName(e.Side), name), comboSwatch)

		case combat.KindGathered:
			attach(fmt.Sprintf("+%d AP", e.Amount))

		case combat.KindGuarded:
			attach("guarded")

		case combat.KindBraced:
			attach("braced")

		case combat.KindStripped:
			// Nothing was stopped, so this must not read like a negation. It is the Feint
			// doing something *to* the defence rather than the defence doing its job.
			attach(fmt.Sprintf("strips their %v", lower(e.Action.String())))

		case combat.KindNegated:
			lastNegation = e.Action
			attach(fmt.Sprintf("stopped by a %v", lower(e.Action.String())))

		case combat.KindDamage:
			// A Riposte's counter and a Mirror's reflection both belong to the *defender*, so
			// they land on the attacker's line as something done back rather than as a hit of
			// their own. Which of the two it was decides the verb: a riposte hits back, a
			// mirror returns the blow it just refused.
			switch {
			case cur >= 0 && rows[cur].swatch != swatchFor(e.Side) && lastNegation == combat.Mirror:
				attach(fmt.Sprintf("reflects %d", e.Amount))
			case cur >= 0 && rows[cur].swatch != swatchFor(e.Side):
				attach(fmt.Sprintf("hits back for %d", e.Amount))
			default:
				attach(fmt.Sprintf("%d damage", e.Amount))
			}

		case combat.KindDefeated:
			announce(fmt.Sprintf("%s falls", s.sideName(e.Target)), swatchFor(e.Target))
		}
	}

	// Never silently drop lines. The deck overlay draws "+N more not shown" for the same
	// reason: a panel that quietly hides part of what it claims to show is a picture that
	// lies, and here it would be lying about the round the player just watched.
	if cap := resolutionCapacity(gs); len(rows) > cap {
		cut := len(rows) - cap + 1
		rows = append([]paneRow{{prefix: fmt.Sprintf("... %d earlier", cut)}}, rows[cut:]...)
	}

	// The newest line is the one playback is on, which ties this pane to the lit row in
	// Action Flow — the same moment, told two ways.
	if s.cursor < len(s.log) && len(rows) > 0 {
		rows[len(rows)-1].highlighted = true
	}
	return rows
}

// swatchFor is a side's colour: green is you, yellow is them.
func swatchFor(side combat.Side) color.RGBA {
	if side == combat.SideB {
		return enemySwatch
	}
	return playerSwatch
}

// **The Resolution pane writes sentences, and this is where the English lives.**
//
// A line is `<who> <verb> <phrase>`: "Duelist attacks with a heavy strike". The verb comes
// from the action's category and the phrase from the card, which is why the two are separate
// tables rather than one string per card — the verb has to be its own run so it can be drawn
// on a coloured background, and it would otherwise have to be sliced back out of a sentence.
//
// **The prose is here and not in `internal/combat`.** The rules package names actions; it does
// not describe them. A card renamed changes `String()`; a card that reads badly in a sentence
// changes only this file.
var actionPhrases = map[combat.ActionKind]string{
	combat.Gather:  "and gathers their strength",
	combat.Sift:    "and sifts through their options",
	combat.Guard:   "with a guard",
	combat.Ritual:  "with a long ritual",
	combat.Jab:     "with a jab",
	combat.Strike:  "with a strike",
	combat.Feint:   "with a feint",
	combat.Heavy:   "with a heavy strike",
	combat.Brace:   "and braces",
	combat.Dodge:   "with a dodge",
	combat.Riposte: "with a riposte",
	combat.Mirror:  "behind a mirror",
}

// actionPhrase is what follows the verb. A card with no phrase falls back to naming itself
// rather than producing a sentence with a hole in it — a new card reads awkwardly until it is
// given a line here, which is a better failure than reading as though nothing happened.
func actionPhrase(a combat.ActionKind) string {
	if p, ok := actionPhrases[a]; ok {
		return p
	}
	return "with " + lower(a.String())
}

// verbFor is the verb a category is spoken with.
func verbFor(c combat.Category) string {
	switch c {
	case combat.CategoryPrepare:
		return "prepares"
	case combat.CategoryDefend:
		return "defends"
	default:
		return "attacks"
	}
}

// The colour the verb is *written* in. **Red for attack, blue for defend, the row's own ink for
// prepare** — the category made loud enough to scan a round by, without reading it.
//
// **The verb was a filled chip until 2026-08-08 and is now the word itself**, coloured, bolded
// and underlined. The chip was a saturated block in a pane that already carries a swatch and a
// sentence, and it drew the eye to a rectangle rather than to the word inside it. Marking the
// word spends the same signal on the thing being read, which is the reasoning that already
// retired the full-width highlight bar a day earlier — this is the same mistake one scale
// smaller.
//
// **Prepare returns zero alpha and inherits the row's ink, deliberately.** As a chip it had to
// name a near-white ground *and* a near-black foreground, because white-on-white is invisible.
// With no ground to sit on there is nothing for a pale colour to be legible against, and the
// pane's own ink is already the colour that reads on that pane whether it is the plum one or the
// off-white one. So prepare is the category with no hue — which is the right rank for it, since
// it is the one that does nothing to the opponent — and it is still marked as a verb by the bold
// and the underline that every verb gets.
func verbInkFor(c combat.Category) color.RGBA {
	switch c {
	case combat.CategoryPrepare:
		return color.RGBA{}
	case combat.CategoryDefend:
		return color.RGBA{R: 52, G: 104, B: 196, A: 255}
	default:
		return color.RGBA{R: 186, G: 52, B: 52, A: 255}
	}
}

// lower is strings.ToLower under a shorter name, used only to drop a card name into the middle
// of a sentence.
func lower(s string) string { return strings.ToLower(s) }

// duelistName is what the player's combatant is called on screen. The data record is still
// keyed `Fighter1` in combatants.json — this is the label, not the identifier, and renaming
// the key would mean renaming it in the roster, the balance tool and the tests for no gain.
const duelistName = "Duelist"

// sideName is who a Resolution line belongs to, written out beside the swatch that already
// says it in colour. **Saying it twice is deliberate**: the colours carry the pattern at a
// glance, but a line that begins "Strike" reads as an instruction rather than a report, and
// with both sides' actions in one list the reader has to hold which colour is which. The name
// makes each line stand on its own.
func (s *CombatScene) sideName(side combat.Side) string {
	if side == combat.SideB {
		return enemyRoster[s.fightIndex%len(enemyRoster)]
	}
	return duelistName
}

// concealEnemy reports whether the opponent's queued actions should be hidden from the
// player. True while planning, false once the round is playing back — an action that has
// happened is not a secret — and always false with DebugGameplay on.
//
// What concealment hides is *what* the enemy queued, not *how many* actions it queued:
// a concealed queue still occupies its real number of rows in both panes. That leaks the
// opponent's action-point spend, which against a greedy planner is most of the tell. It
// is deliberate rather than overlooked: collapsing the rows would hide the spend but would
// also destroy the Resolution pane's account of who acts when, and that alternation is a
// rule the player is meant to read and eventually manipulate. Revisit alongside the wider
// hidden-information decision — see TODO.md.
func (s *CombatScene) concealEnemy(gs *state.GlobalState) bool {
	return !gs.DebugGameplay && s.planning()
}

// drawPaneFrame draws a column's fill, border and title, and reports its rectangle.
// Split out because the card panes fill themselves rather than drawing text rows.
func (s *CombatScene) drawPaneFrame(gs *state.GlobalState, screen *ebiten.Image, p panePlacement) (x, y, w, h float32) {
	r := image.Rect(
		gs.PctX(p.leftPct), gs.PctY(paneTopPct),
		gs.PctX(p.rightPct), gs.PctY(paneBottomPct),
	)

	// Not drawBox: a pane names its own ground and its own ink, where drawBox derives a dim
	// fill from one colour. drawBox still serves the caption and the character strip, which
	// have no text on a light ground to worry about.
	x, y = float32(r.Min.X), float32(r.Min.Y)
	w, h = float32(r.Dx()), float32(r.Dy())

	vector.DrawFilledRect(screen, x, y, w, h, p.fill, false)
	vector.StrokeRect(screen, x, y, w, h, 2, p.color, false)

	if p.title != "" {
		titleOp := &text.DrawOptions{}
		titleOp.GeoM.Translate(float64(x+w/2), float64(y+paneTitleInset))
		titleOp.PrimaryAlign = text.AlignCenter
		titleOp.ColorScale.ScaleWithColor(p.ink)
		text.Draw(screen, p.title, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, titleOp)
	}

	return x, y, w, h
}

// drawPane draws a read-only column: the frame, then a row per action.
func (s *CombatScene) drawPane(gs *state.GlobalState, screen *ebiten.Image, p panePlacement, rows []paneRow) {
	x, y, w, _ := s.drawPaneFrame(gs, screen, p)

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}

	// **The highlight is centred on the text, not offset from the row's top by a constant.**
	// It used to be drawn at rowY-4 with height rowHeight-2, numbers picked by eye against a
	// single 30px pitch. When the Resolution pane arrived at 22 the bar came out 20 tall
	// against a ~19px line sitting 4px lower, so it clipped the text and the swatch along its
	// bottom edge. Measuring the line and centring on it works at any pitch, which is the
	// point — the pane's pitch is now a property of the placement and free to change again.
	_, lineHeight := text.Measure("Ag", face, 0)

	for i, row := range rows {
		rowY := y + paneFirstRow + float32(i*p.rowHeight)
		rowOp := &text.DrawOptions{}

		// **The row playback is on is set in the text itself — coloured, bold and underlined —
		// rather than sat on a lit bar** *(changed 2026-08-07)*. A full-width bar was a fourth
		// saturated block in a pane that already carries a swatch, a verb chip and a sentence,
		// and on a light ground it had to be pale enough to read through, which left it
		// shouting and saying little. Marking the words is the same signal spent on the thing
		// the reader is actually looking at.
		//
		// Bold is faux — the same run drawn again a pixel right. `text/v2` has no synthetic
		// weight and kubasta ships one, so this is the only way to get one without a second
		// font file. At a pixel font's sizes it is exactly what a bold face would do anyway.
		ink := p.ink
		if row.highlighted {
			ink = p.nowInk
		}

		// A row with no verb is a single centred or left-aligned run and keeps the old path.
		// One with a verb has to be laid out left to right so the chip can be measured into
		// place, which rules out centring it — a sentence in a list wants a common left edge
		// anyway.
		if row.swatch.A == 0 && row.verb == "" {
			rowOp.GeoM.Translate(float64(x+w/2), float64(rowY))
			rowOp.PrimaryAlign = text.AlignCenter
			rowOp.ColorScale.ScaleWithColor(ink)
			text.Draw(screen, row.prefix, face, rowOp)
			continue
		}

		textX := x + paneRowInset
		if row.swatch.A != 0 {
			// A swatch turns the row into a column: square on the left, the line beside it,
			// so the squares line up down the pane and the alternation is readable as a
			// pattern rather than as text.
			//
			// **Idle swatches fade toward the pane's own ground**, so the lit one is the
			// strongest thing in the pane whether that ground is dark or light. Scaling
			// toward black — which is what dimming used to mean here — made idle rows *more*
			// contrasty than the lit one the moment Resolution went off-white. See
			// systems.ColorToward.
			swatch := row.swatch
			if !row.highlighted {
				swatch = systems.ColorToward(swatch, p.fill, 45)
			}
			// Centred on the line for the same reason the bar is, so the squares sit level
			// with the text they belong to whatever pitch the pane draws at.
			swatchTop := rowY + float32(lineHeight)/2 - swatchSize/2
			vector.DrawFilledRect(screen, x+paneRowInset, swatchTop, swatchSize, swatchSize, swatch, false)
			textX = x + paneRowInset + swatchSize + swatchGap
		}

		// Three runs, measured one after the next. The verb is written in its category's own
		// colour — red for attack, blue for defend, the row's ink for prepare — so a round can
		// be scanned for what *kind* of thing happened before any of it is read.
		cursorX := float64(textX)
		draw := func(str string, tint color.RGBA, bold bool) {
			if str == "" {
				return
			}
			at := func(dx float64) {
				op := &text.DrawOptions{}
				op.GeoM.Translate(cursorX+dx, float64(rowY))
				op.ColorScale.ScaleWithColor(tint)
				text.Draw(screen, str, face, op)
			}
			at(0)
			if bold {
				at(1) // faux bold
			}

			// Advance by the *unbolded* width, so the second pass thickens the strokes without
			// walking the runs after it out of place.
			wRun, _ := text.Measure(str, face, 0)
			cursorX += wRun
		}

		draw(row.prefix, ink, row.highlighted)
		if row.verb != "" {
			// **The verb is always bold and always underlined, on every row.** That is what makes
			// it read as the verb rather than as a word that happens to be coloured — one mark
			// would be ambiguous against a pane that also uses colour for the side and for the
			// live row, and three together are unmistakable at a glance.
			verbInk := row.verbInk
			if verbInk.A == 0 {
				verbInk = ink
			}

			verbLeft := float32(cursorX)
			wVerb, _ := text.Measure(row.verb, face, 0)
			draw(row.verb, verbInk, true)

			// **Flush with the bottom of the measured line box**, not a constant above it. The
			// underline used to sit under a chip whose height was fixed at 18 against a 22px
			// pitch; with no chip the only thing it can be positioned against is the text, and
			// text.Measure already reports the full line including descent. That is what keeps
			// it clear of the `p` in "prepares" — a rule three pixels up from the baseline
			// struck straight through it — and what lets either pane's pitch change again.
			vector.DrawFilledRect(screen,
				verbLeft, rowY+float32(lineHeight)-underlineHeight,
				float32(wVerb), underlineHeight,
				verbInk, false)
		}
		draw(row.suffix, ink, row.highlighted)
	}
}

const (
	// underlineHeight is how thick the verb's underline is. Two pixels rather than one: at
	// kubasta's weight a single pixel reads as an artefact of the font rather than a mark.
	underlineHeight = 2
)

// actionFlowRows lays the two queued sets out in play order, and marks the row for the action
// currently playing back. Each row is swatched in its side's colour, so who-acts-when reads as
// a pattern of squares before any of the labels are read.
//
// Whichever set is longer keeps going alone once the other runs out — a faster duelist
// buys more actions, and the tail is exactly where that advantage shows.
//
// This layout is the order combat.ResolveRound actually plays, so the highlight walks
// straight down the pane. Keep the two in step: the pane is the player's model of the
// round, and effects that reorder resolution will have to move both.
// concealEnemy replaces the opponent's labels with placeholders while leaving their rows
// in place, so the interleaving still reads correctly and only the content is withheld.
//
// This function needs no change when phase-based resolution lands — it draws whatever
// ResolutionOrder returns and never works the order out for itself, which is the whole
// point of that split.
//
// **It no longer has to draw a combo spanning non-adjacent slots**, which was an open problem
// for as long as this was the only pane: one row per slot with a single walking highlight has
// no way to say "these together did a thing". The Resolution pane says it in words instead.
// The same goes for a slot a stagger deleted — this pane still draws it as a row, and the
// other one is where it is reported as lost.
func (s *CombatScene) actionFlowRows(fighter, enemy []combat.ActionKind, concealEnemy bool) []paneRow {
	order := combat.ResolutionOrder(fighter, enemy)
	if len(order) == 0 {
		return []paneRow{{prefix: "(empty)"}}
	}

	playingSlot, playing := s.currentSlot()

	rows := make([]paneRow, 0, len(order))
	for i, slot := range order {
		label, swatch := slot.Action.String(), playerSwatch
		if slot.Side == combat.SideB {
			swatch = enemySwatch
			if concealEnemy {
				label = concealedLabel(slot.Action)
			}
		}

		rows = append(rows, paneRow{
			prefix:      label,
			swatch:      swatch,
			highlighted: playing && i == playingSlot,
		})
	}
	return rows
}

// concealedLabel is what a hidden action shows instead of its name. The category is
// deliberately not hidden: it is what decides where the action sits in the order, so
// withholding it would make the Resolution pane unreadable rather than merely uncertain —
// the player could not tell why the rows are arranged as they are. It replaced the
// initiative number in exactly that job when initiative was removed.
//
// This is the first cut at graded reveal rather than the finished scheme. What else
// leaks per action — whether it damages, whether it applies a status — is still open;
// see TODO.md.
func concealedLabel(a combat.ActionKind) string {
	return fmt.Sprintf("??? (%s)", a.Category())
}
