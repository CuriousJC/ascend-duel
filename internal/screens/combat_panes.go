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
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
)

// The band a full-height pane occupies. **Resolution left it on 2026-08-11** and moved down
// to the strip above the hand — see the feed constants below — so today this describes only
// Action Flow, which is not drawn. The space between 12% and 46% is deliberately empty and
// spoken for.
const (
	paneTopPct    = 12
	paneBottomPct = 46

	paneTitleInset = 10 // gap from the pane's top edge to its title
	paneFirstRow   = 45 // gap from the top edge to the first action row
	paneRowHeight  = 30
	paneRowInset   = 10 // gap from the pane's left edge to a row's swatch
	swatchSize     = 16
	swatchGap      = 6 // gap between a swatch and its label

	// The Resolution rows are sentences rather than card names, and there are more of them —
	// a busy round merges to a dozen lines where the flow pane draws at most ten.
	paneTextRowHeight = 22
)

// **Resolution is a three-line feed above the hand now** *(2026-08-11)*, in the slot the
// caption box used to hold, and it grows upward while it is long-pressed.
//
// The pane it replaced was 15–78% wide and a third of the screen tall, which bought a whole
// round on screen at once at a cost the screen could not keep paying — it was the largest
// thing on it, and what it mostly held was blank rows. The feed is the opposite trade: the
// last few things that happened, where the eye already is, and the rest one press away.
//
// **Older lines scroll off silently, which suspends a rule this pane wrote.** "Never hide
// part of what you claim to show" is why the deck overlay draws `+N more not shown` and why
// this pane drew `... N earlier`. Three rows cannot afford a line of bookkeeping out of
// three, and the long press is what makes the omission honest — the full account is a press
// away rather than gone. **The expanded view keeps the marker**, because there it is a line
// out of nineteen and it is the only place the count can still be told.
const (
	// feedRowInset is the gap from the box's edge to the first row, and again below the
	// last. It stands in for paneFirstRow, which reserves room for a title the feed has
	// no space for.
	feedRowInset = 8

	// feedRows is how many lines the box shows when it is not being held. Everything about
	// the box's height is derived from it.
	feedRows = 3

	// feedGapAboveCards is how far the box's bottom edge sits above the resting hand row.
	//
	// **A selected card lifts by selectedNudge and does overlap it**, by 21 pixels, and that
	// is accepted rather than overlooked: the box is measured against where the cards live,
	// not against where one of them goes when it is picked. What it costs is that the box's
	// bottom strip cannot take a long press — see updateFeed, which ignores a press inside
	// handZone so a lifted card is still the thing under the cursor there.
	feedGapAboveCards = 5

	// feedExpandTopPct is how far up a held box grows: the top of the band the full-height
	// pane used to occupy. It expands into the space that pane vacated, which is what makes
	// the room free.
	feedExpandTopPct = paneTopPct

	// longPressTicks is how long the button has to be held before it counts as a long press.
	// A third of a second at 60 TPS.
	//
	// **This is the game's first long press.** `CLAUDE.md` has had it in the input vocabulary
	// since the vocabulary was written — left click, drag and drop, long press — and nothing
	// had used it. If a second one arrives this constant and the counter beside it are what
	// should be lifted into a shared widget rather than copied.
	longPressTicks = 20
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

	// firstRow is the gap from the top edge to the first row. A titled pane has to clear its
	// title; the feed has no title and cannot afford to pretend it does — 45 pixels of
	// reserved heading out of an 82-pixel box is most of the box.
	firstRow int
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
		firstRow:  paneFirstRow,
	}

	// **Resolution keeps its colours and loses its column** *(2026-08-11)*. The left and
	// right percentages are unused — the feed takes its width from the hand, like the AP bar
	// and the box it replaced — and the title is dropped, because a heading in a three-line
	// box costs a line and the box is under the cards it is reporting on, which says what it
	// is more directly than a word would.
	//
	// **The off-white ground survives the move deliberately**, even though the box it took
	// over from was a dim pink one. The verbs are coloured, and a light ground is what makes
	// three saturated inks legible — that was the whole reason this pane went off-white on
	// 2026-08-07. `paneEdge` still names it, so the pink identity is in the border.
	resolutionPane = panePlacement{
		title:     "",
		color:     paneEdge,
		fill:      color.RGBA{R: 234, G: 230, B: 224, A: 255},
		ink:       color.RGBA{R: 34, G: 32, B: 38, A: 255},
		nowInk:    color.RGBA{R: 178, G: 22, B: 106, A: 255},
		rowHeight: paneTextRowHeight,
		firstRow:  feedRowInset,
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
// surface ran into `enemySwatch`. The player's green still collides with earth, which went green
// on 2026-08-14, so the element scheme is only half-untangled — and a card border and a row
// swatch are never seen side by side, which is why that half has been allowed to stand.
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
	s.drawPane(gs, screen, actionFlowPane, panePlacementRect(gs, actionFlowPane),
		s.actionFlowRows(s.fighterActions, s.enemyActions, s.concealEnemy(gs)))
}

// drawResolution shows what the round actually did, accumulating as it plays back.
func (s *CombatScene) drawResolution(gs *state.GlobalState, screen *ebiten.Image) {
	r := s.feedRect(gs)
	expanded := s.feedExpanded()

	rows, hidden := s.resolutionLines(gs, feedCapacity(r), expanded)
	s.drawPane(gs, screen, resolutionPane, r, rows)

	if !expanded && hidden > 0 {
		drawMoreAbove(screen, r)
	}
}

// The little upward arrow in the box's top-right corner: **there are lines above this one**.
//
// It is drawn only when something is actually scrolled off, so it is a report and not a
// decoration. That has a cost worth stating: on a short round nothing advertises the long
// press, and the long press is the only way to reach what the arrow points at. The
// alternative — always drawing it — makes the gesture discoverable and makes the arrow lie
// on every round that fits, which is the trade the deck overlay's `+N more not shown` and
// this pane's own `... N earlier` both already decided the same way.
//
// `attentionYellow` is the screen's one "look here" colour, which is exactly what this is.
// **It is drawn twice, black then yellow.** The box's ground is off-white and the arrow is
// a small saturated yellow, which is the one pairing `attentionYellow` is weak at — the deck
// stack wears the same colour against a dark screen and reads instantly. An outline is what
// buys it back without introducing a second "look here" colour.
const (
	moreArrowWidth  = 14
	moreArrowHeight = 8
	moreArrowInset  = 12 // from the box's top and right edges
	moreArrowBorder = 2
)

func drawMoreAbove(screen *ebiten.Image, box image.Rectangle) {
	cx := float32(box.Max.X-moreArrowInset) - moreArrowWidth/2
	top := float32(box.Min.Y + moreArrowInset)

	outline := color.RGBA{A: 255}

	// The outline is the same triangle grown by the border on every side: wider at the base
	// by twice the border, taller by it, and started that much higher so the point keeps its
	// margin. Growing it rather than stroking it means there is only one shape to be wrong.
	fillArrowUp(screen, cx, top-moreArrowBorder,
		moreArrowWidth+2*moreArrowBorder, moreArrowHeight+moreArrowBorder, outline)

	// **The base needs its own bar** *(2026-08-11)*. Growing the triangle cannot supply one:
	// a taller triangle is also a wider one, so its bottom row lands on the same scanline as
	// the yellow base rather than under it, and the arrow drew as two black slopes with an
	// open bottom — a caret, not a triangle. This is the third side, at the grown width so it
	// ends flush with them.
	vector.DrawFilledRect(screen,
		cx-(moreArrowWidth+2*moreArrowBorder)/2, top+moreArrowHeight,
		moreArrowWidth+2*moreArrowBorder, moreArrowBorder, outline, false)

	fillArrowUp(screen, cx, top, moreArrowWidth, moreArrowHeight, attentionYellow)
}

// fillArrowUp draws an upward triangle centred on cx, in scanlines rather than a path — the
// same hand-rolled idiom as everything else on this screen. The point is at the top because
// that is the direction the hidden lines are in.
func fillArrowUp(screen *ebiten.Image, cx, top, w, h float32, c color.RGBA) {
	for i := float32(0); i < h; i++ {
		rowW := w * (i + 1) / h
		vector.DrawFilledRect(screen, cx-rowW/2, top+i, rowW, 1, c, false)
	}
}

// feedRect is the box Resolution is drawn in: hand width, bottom edge a few pixels above the
// cards, and either three rows tall or grown up to the vacated pane band while held.
//
// **The bottom edge is the fixed one.** It is anchored to the hand — the thing the box is
// reporting on — so expanding moves the top and nothing the eye is already resting on
// shifts. A box that grew downward would push into the cards; one that grew both ways would
// move every line already on screen.
func (s *CombatScene) feedRect(gs *state.GlobalState) image.Rectangle {
	band := handBand(gs, s.laidOutCount())
	bottom := gs.PctY(handTopPct) - feedGapAboveCards

	top := bottom - feedHeight()
	if s.feedExpanded() {
		top = gs.PctY(feedExpandTopPct)
	}
	return image.Rect(band.Min.X, top, band.Max.X, bottom)
}

// feedHeight is the collapsed box: the rows it holds, plus a margin above the first and
// below the last. Derived from feedRows so the two cannot disagree.
func feedHeight() int {
	return 2*feedRowInset + feedRows*paneTextRowHeight
}

// feedCapacity is how many lines fit in a box of this size. Derived rather than written down,
// so neither the collapsed height nor the expanded one can leave a constant behind claiming a
// capacity the box does not have.
func feedCapacity(r image.Rectangle) int {
	n := (r.Dy() - 2*feedRowInset) / paneTextRowHeight
	if n < 1 {
		return 1
	}
	return n
}

// feedExpanded reports whether the box is currently grown. It is derived from how long the
// button has been held rather than stored as a mode, for the same reason planning() is
// derived: two things that can disagree about whether the box is open is a bug waiting to be
// written.
func (s *CombatScene) feedExpanded() bool {
	return s.feedPressTicks >= longPressTicks
}

// updateFeed runs the long press: hold the mouse down on the box and it grows, release and it
// snaps back.
//
// **Held rather than latched, deliberately.** A latched panel is a second dialog, and the one
// dialog in the game needs a bright yellow ring around its only exit to stop being a trap. A
// press that ends when the button comes up has nothing to escape from.
//
// It ignores a press inside handZone. A selected card lifts into the box's bottom 21 pixels,
// and the card is what the player means there — see feedGapAboveCards. The action box reads
// that press on the same tick, so without this both would fire.
func (s *CombatScene) updateFeed(gs *state.GlobalState) {
	at := image.Pt(gs.MouseX, gs.MouseY)

	// The overlay is a dialog: nothing behind it responds, and that includes this.
	held := !s.showDeck &&
		ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) &&
		!at.In(handZone(gs))

	// Once it is open the press keeps it open wherever the cursor has wandered to — the box
	// grew out from under it. Starting one still requires the cursor to be on the collapsed
	// box.
	if held && (s.feedExpanded() || at.In(s.feedRect(gs))) {
		s.feedPressTicks++
		return
	}
	s.feedPressTicks = 0
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
//
// **The attack phase is one line, and it is the combo's** *(2026-08-14)*. Prepares and defends
// still write a line each; the attack cards write none. A turn lands one blow, so five sentences
// saying "Duelist attacks with an earth strike" described a round that does not happen — and the
// line that matters, what the five cards came to, was the sixth. The combo line says who, what
// they formed and the arithmetic, and the damage attaches to it. **A blow that formed no hand
// still gets its sentence**, from the card it led with, because a lone Strike is not a combo and
// announcing one over every attack would make the word mean nothing.
// It reports how many lines it dropped off the top as well as the rows themselves, so the
// caller can say so — with a marker line when there is room for one, and with the arrow in
// the corner when there is not.
// planningLines is what the pane holds before a round has been resolved: the prompt, and above
// it the combo the current selection has already formed.
//
// **The pane records and does not propose — with one standing exception, and this joins it.**
// `(press DUEL!)` was already a line telling the player what to do, because the caption box that
// used to say it is gone and this is the only text on the screen with room. The preview goes in
// the same place for the same reason, and it costs nothing the rule was protecting: the pane is
// empty while planning, so nothing is being said twice, and the moment the round resolves these
// lines are replaced by the record.
//
// **It reads the way the fired line will read** — same amber swatch, same COMBO! prefix, same
// name from the same catalogue — so forming a hand and watching it land are recognisably the
// same event. What it deliberately does not carry is the arithmetic: `Base` and `Swing` are the
// resolver's, worked out against a strength and a shock roll that have not happened, and a
// figure printed here that the round then contradicted would be worse than no figure.
func (s *CombatScene) planningLines() []paneRow {
	prompt := paneRow{prefix: "(press DUEL!)"}

	blow, ok := s.previewAttack()
	if !ok {
		return []paneRow{prompt}
	}

	return []paneRow{
		{
			prefix: "COMBO! " + comboNameOf(blow.Hand, blow.Mix) +
				" x" + multiplierText(blow.Multiplier),
			swatch: comboSwatch,
		},
		prompt,
	}
}

func (s *CombatScene) resolutionLines(gs *state.GlobalState, capacity int, markOverflow bool) ([]paneRow, int) {
	end := s.cursor + 1
	if end > len(s.log) {
		end = len(s.log)
	}
	if end <= 0 {
		return s.planningLines(), 0
	}

	var rows []paneRow

	// cur is the line the next outcome attaches to, or -1 when the last thing appended was
	// an announcement rather than an action. curSide is whose line it is.
	//
	// **The side is tracked rather than read back off the row's swatch**, because the combo line
	// wears amber and takes outcomes: a damage event compared against that swatch would read
	// every hit as a Riposte's counter.
	cur := -1
	curSide := combat.SideA
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
	// category's colour. See cardPhrase.
	act := func(side combat.Side, c combat.Card) {
		rows = append(rows, paneRow{
			prefix:  s.sideName(side) + " ",
			verb:    verbFor(c.Action.Category()),
			suffix:  " " + cardPhrase(c),
			verbInk: verbInkFor(c.Action.Category()),
			swatch:  swatchFor(side),
		})
		cur, curSide = len(rows)-1, side
		outcomes = 0
	}

	announce := func(label string, swatch color.RGBA) {
		rows = append(rows, paneRow{prefix: label, swatch: swatch})
		cur = -1
	}

	// blow opens the attack phase's one line: what the hand formed and what it adds up to.
	//
	// **It is an announcement that takes outcomes**, which no other line here is. The blow's
	// damage, a shocked miss and any status it lands all belong to it, because the cards that
	// would otherwise have carried them no longer write lines of their own.
	blow := func(e combat.Event) {
		rows = append(rows, paneRow{
			prefix: fmt.Sprintf("COMBO!  %s lands a %s", s.sideName(e.Side), comboName(e)),
			suffix: "  " + comboMath(e),
			swatch: comboSwatch,
		})
		cur, curSide = len(rows)-1, e.Side
		outcomes = 1 // the sum is already on the line, so the first outcome reads as a list
	}

	for _, e := range s.log[:end] {
		switch e.Kind {
		case combat.KindRoundStart, combat.KindRoundEnd:
			// The pane holds one round, so saying which round it is would be a line spent on
			// something the caption and the character block both already carry.

		case combat.KindAction:
			// **An attack card writes no line.** Its beat still passes — the engine announces
			// every card so the table can light it and playback can count slots — but the
			// sentence for the whole phase is the KindCombo below.
			if e.Action.Category() == combat.CategoryAttack {
				break
			}
			act(e.Side, combat.Card{Action: e.Action, Element: e.Element})

		case combat.KindStaggered:
			announce(fmt.Sprintf("%s is staggered - %v is lost", s.sideName(e.Side), e.Action),
				swatchFor(e.Side))

		case combat.KindMissed:
			// It attaches to the attacker's own line rather than announcing, because the card
			// *was* played — the line above it is real and this is what became of it. Naming
			// the shock is the whole point: a blow that simply missed would look like a bug in
			// a game with no dice in it.
			attach("misses - shocked")

		case combat.KindStatus:
			attach(statusPhrase(e.Element))

		case combat.KindBurned:
			// A tick belongs to nobody's card, so it opens its own line. It carries the
			// victim's swatch because it is a thing happening *to* them, which is also the
			// only side the event names.
			announce(fmt.Sprintf("%s burns for %d", s.sideName(e.Target), e.Amount),
				swatchFor(e.Target))

		case combat.KindCombo:
			// **This is the attack phase's line, whether or not a hand formed.** A blow that
			// formed nothing is not a combo and must not be announced as one — announcing
			// "COMBO!" over every single Strike would make the word mean nothing — so it writes
			// the ordinary sentence for the card it led with, which the engine names on the
			// event for exactly this.
			if e.Hand == combat.HandNone {
				act(e.Side, combat.Card{Action: e.Action, Element: e.Element})
				break
			}
			blow(e)

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
			attach(fmt.Sprintf("stopped by a %v", lower(e.Action.String())))

		case combat.KindDamage:
			// A Riposte's counter belongs to the *defender*, so it lands on the attacker's line
			// as something done back rather than as a hit of its own. It is the only damage in
			// the game that runs the other way, so the side mismatch is the whole test.
			switch {
			case cur >= 0 && curSide != e.Side:
				attach(fmt.Sprintf("hits back for %d", e.Amount))
			default:
				attach(fmt.Sprintf("%d damage", e.Amount))
			}

		case combat.KindDefeated:
			announce(fmt.Sprintf("%s falls", s.sideName(e.Target)), swatchFor(e.Target))
		}
	}

	// **Two ways to overflow, and which one applies is the box's size** *(2026-08-11)*.
	//
	// Expanded, the old rule holds: never silently drop lines, the same reason the deck
	// overlay draws "+N more not shown". A panel that quietly hides part of what it claims to
	// show is a picture that lies.
	//
	// Collapsed, the box takes the newest lines and the rest simply scroll off. Three rows
	// cannot spend one of themselves on a count, and the claim is different at that size —
	// the feed says "here is what just happened", not "here is the round". What keeps it
	// honest is the long press: the full account is one hold away, which is where the count
	// is drawn.
	hidden := 0
	if len(rows) > capacity {
		if markOverflow {
			hidden = len(rows) - capacity + 1
			rows = append([]paneRow{{prefix: fmt.Sprintf("... %d earlier", hidden)}}, rows[hidden:]...)
		} else {
			hidden = len(rows) - capacity
			rows = rows[hidden:]
		}
	}

	// The newest line is the one playback is on, which ties this pane to the lit row in
	// Action Flow — the same moment, told two ways.
	if s.cursor < len(s.log) && len(rows) > 0 {
		rows[len(rows)-1].highlighted = true
	}
	return rows, hidden
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
	combat.Retreat: "behind a retreat",
}

// **What each card does, in words, printed on its face.** A second table beside the one above
// and deliberately not the same strings: `actionPhrases` is prose for a *sentence about a
// round* — "attacks with a heavy strike" — and this is a rules description read while
// deciding whether to play the thing.
//
// **The rule it describes lives in `internal/combat` and the wording lives here**, the same
// split actionPhrases is built on: the rules package names actions and must not grow UI
// strings. `internal/cards` knows how to set the text on a card and not what any card does.
//
// **Verb first, on every card.** They are read in a row while the player is counting action
// points — the first word saying what the card *does to the round* is what makes eight of them
// scannable. "0.5x" and "1x" rather than "half" and "full" because a multiplier is what the
// rule actually is, and **"DMG" rather than "damage"** because the column is about a dozen
// characters wide and the duelist card already labels the figure that way.
//
// A concept with no entry draws nothing rather than a placeholder; TestEveryConceptHasEffectText
// is what stops that being how a card ships.
var cardEffects = map[combat.ActionKind]string{
	combat.Gather:  "Bank 2 AP for next round",
	combat.Sift:    "Throw away 2 more cards, then refill",
	combat.Guard:   "Halve every attack next turn",
	combat.Ritual:  "Bank 6 AP for next round",
	combat.Jab:     "Deal 0.5x DMG",
	combat.Strike:  "Deal 1x DMG",
	combat.Feint:   "Strip a defend card, deal 1x DMG",
	combat.Heavy:   "Deal 2x DMG",
	combat.Brace:   "Halve 1 incoming attack",
	combat.Dodge:   "Negate 1 incoming attack",
	combat.Riposte: "Negate 1 attack, deal 0.5x DMG back",
	combat.Retreat: "Negate 3 incoming attacks",
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

// cardPhrase is actionPhrase with the element worked into it: "with a fire strike".
//
// **The element goes after the article rather than in front of the phrase**, which is what
// makes it a sentence instead of a label. Every phrase that can carry a status has an article —
// the four attacks are all "with a …" — so the insertion lands correctly on exactly the cards
// where it matters most.
//
// A phrase with no article gets the element in brackets: "and gathers their strength (fire)".
// That is deliberately the plainer half of the rule. An elemental prepare is a real card and
// currently does nothing mechanical, so a line that reads slightly like a note is honest about
// what it is — and it is better than a sentence bent around a word that does not fit it.
func cardPhrase(c combat.Card) string {
	phrase := actionPhrase(c.Action)
	if c.Element == combat.Basic {
		return phrase
	}

	name := lower(c.Element.String())
	if i := strings.Index(phrase, "a "); i >= 0 {
		// **The article has to be corrected, not just followed.** Two of the five elements begin
		// with a vowel, so "a earth strike" is a third of the lines this function writes.
		article := "a "
		if strings.ContainsRune("aeiou", rune(name[0])) {
			article = "an "
		}
		return phrase[:i] + article + name + " " + phrase[i+2:]
	}
	return phrase + " (" + name + ")"
}

// statusPhrase is what a landed element says it did, as an outcome attached to the attacker's
// line. Each names the *effect* rather than the element, because "chills them" says what
// happens next and "applies ice" says only that a rule fired.
//
// Basic has no phrase because it applies no status and no KindStatus event is ever raised for
// it; the fallback exists so a fifth element narrates as something rather than as an empty tail.
func statusPhrase(e combat.Element) string {
	switch e {
	case combat.Fire:
		return "sets them burning"
	case combat.Ice:
		return "chills them"
	case combat.Lightning:
		return "shocks them"
	case combat.Earth:
		return "weighs them down"
	default:
		return "leaves " + lower(e.String()) + " on them"
	}
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

// duelistName is the fallback for a duelist record that names nobody. The record is still
// keyed `Fighter1` in duelists.json — a key is not a label, and renaming it would mean
// renaming it in the balance tool and the tests for no gain — but the record now carries a
// Name and that is what is normally shown.
const duelistName = "Duelist"

// sideName is who a Resolution line belongs to, written out beside the swatch that already
// says it in colour. **Saying it twice is deliberate**: the colours carry the pattern at a
// glance, but a line that begins "Strike" reads as an instruction rather than a report, and
// with both sides' actions in one list the reader has to hold which colour is which. The name
// makes each line stand on its own.
//
// **It reads the combatant rather than the roster** *(2026-08-11)*. It used to index the
// fight order and print the record key, which is why the four records were named
// Monster1..Tactician1 — style names standing in for creature names because there was
// nowhere else to put one. Records carry a Name now, so a line says "Ogre Warlord attacks"
// rather than "OgreWarlord attacks".
func (s *CombatScene) sideName(side combat.Side) string {
	c := s.fighter
	if side == combat.SideB {
		c = s.enemy
	}
	if c != nil && c.Name != "" {
		return c.Name
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

// panePlacementRect is the column a full-height pane occupies, from its percentages and the
// shared band. **Only Action Flow is placed this way now** — Resolution takes its rectangle
// from the hand instead, so the rect is a parameter rather than something drawPane works out.
func panePlacementRect(gs *state.GlobalState, p panePlacement) image.Rectangle {
	return image.Rect(
		gs.PctX(p.leftPct), gs.PctY(paneTopPct),
		gs.PctX(p.rightPct), gs.PctY(paneBottomPct),
	)
}

// drawPaneFrame draws a pane's fill, border and title in the rectangle given, and reports it
// back as floats. Split out because the card panes fill themselves rather than drawing text
// rows.
func (s *CombatScene) drawPaneFrame(gs *state.GlobalState, screen *ebiten.Image, p panePlacement, r image.Rectangle) (x, y, w, h float32) {
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

// drawPane draws a read-only pane: the frame, then a row per action.
func (s *CombatScene) drawPane(gs *state.GlobalState, screen *ebiten.Image, p panePlacement, r image.Rectangle, rows []paneRow) {
	x, y, w, _ := s.drawPaneFrame(gs, screen, p, r)

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}

	// **The highlight is centred on the text, not offset from the row's top by a constant.**
	// It used to be drawn at rowY-4 with height rowHeight-2, numbers picked by eye against a
	// single 30px pitch. When the Resolution pane arrived at 22 the bar came out 20 tall
	// against a ~19px line sitting 4px lower, so it clipped the text and the swatch along its
	// bottom edge. Measuring the line and centring on it works at any pitch, which is the
	// point — the pane's pitch is now a property of the placement and free to change again.
	_, lineHeight := text.Measure("Ag", face, 0)

	for i, row := range rows {
		rowY := y + float32(p.firstRow) + float32(i*p.rowHeight)
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
func (s *CombatScene) actionFlowRows(fighter, enemy []combat.Card, concealEnemy bool) []paneRow {
	order := combat.ResolutionOrder(fighter, enemy)
	if len(order) == 0 {
		return []paneRow{{prefix: "(empty)"}}
	}

	playingSlot, playing := s.currentSlot()

	rows := make([]paneRow, 0, len(order))
	for i, slot := range order {
		label, swatch := slot.Card.Action.String(), playerSwatch
		if slot.Side == combat.SideB {
			swatch = enemySwatch
			if concealEnemy {
				label = concealedLabel(slot.Card.Action)
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

// comboName is what the attack phase formed, said in words: the element makeup in front of the
// hand, as "Duo Strike Flurry".
//
// **The mix is dropped when the hand showed no colour**, because "Drab Strike Flurry" is a word
// spent saying nothing. And a blow that formed no hand at all is named only as an attack — the
// pane does not announce those, but the trace does.
//
// The names come from the catalogue rather than being written here, so a hand renamed in
// `data/combos.json` is renamed once.
func comboName(e combat.Event) string {
	hand, ok := combat.HandByID(e.Hand)
	if !ok {
		return "attack"
	}
	mix, _ := combat.MixByID(e.Mix)
	return comboNameOf(hand, mix)
}

// comboNameOf is the same name built from the hand and mix themselves, for the preview the hand
// row draws while the player is still choosing. **One namer, two callers**: a preview that named
// a combo differently from the feed that reports it would be two vocabularies for one thing.
func comboNameOf(hand combat.Hand, mix combat.Mix) string {
	if mix.Colours > 0 {
		return mix.Name + " " + hand.Name
	}
	return hand.Name
}

// comboMath is the blow written out as the sum it is: `20 + 10 x 3.5 = 55`.
//
// **Every figure in it comes off the event.** Base, Swing, Multiplier and the total are all
// worked out by the resolver, so the line cannot claim a sum the round did not use — which is
// the whole reason those fields are on the event rather than being recomputed here.
//
// **The cards' damage is one term, not one term each.** MECHANICS.md writes the formula out
// per card — `10 + 10 + 10x1.5` — and that is the right form for a design document; the feed is
// three rows of a sentence, and an Onslaught spelled out card by card is half a line spent on
// five identical numbers.
//
// It is the swing before the attacker's weight and before anything the defender raised, so the
// damage that follows on the same line is often smaller. That gap is what a defence is worth,
// and it is only legible because both figures are shown.
func comboMath(e combat.Event) string {
	return fmt.Sprintf("(%d + %d x %s = %d)", e.Base, e.Swing, multiplierText(e.Multiplier), e.Amount)
}

// multiplierText writes a percentage multiplier the way the design does: 350 as `3.5`, 200 as
// `2`, 1000 as `10`. Trailing zeros are dropped rather than padded to two places, because
// `x 10.00` reads as a precision the game does not have.
func multiplierText(pct int) string {
	return strconv.FormatFloat(float64(pct)/100, 'f', -1, 64)
}
