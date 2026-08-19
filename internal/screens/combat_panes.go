package screens

// The pane machinery — the row model, the placements and colours, and the prose that turns an
// event into a sentence.
//
// **Nothing in this file is drawn on the combat screen as of 2026-08-18.** Action Flow is built
// and unwired, and the Resolution feed is gone; what draws these rows now is the fight log — see
// combat_log.go. Split out of combat.go on 2026-08-07. **The prose lives here and not in
// internal/combat** on purpose: the rules package names actions, it does not describe them.
// Everything here is presentation over a log the engine has already finished deciding, which is
// what makes it structurally impossible for a panel to disagree with the round it reports.

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

// **The band above the hand, which the Resolution feed used to occupy** *(vacated 2026-08-18)*.
// Nothing is drawn there at rest now; what claims it is the hand dialog, which writes the
// blow's arithmetic across it, and `drawPlannedHand`, which writes the name of the hand the
// selection has already formed in the same place. See combat_mathbox.go.
//
// The two constants stay because the band does: `tableRowTop` keeps the played cards clear of
// it, and `handMathRect` is measured from it. Their **values are the feed's** — the sum was
// laid out and looked at against a box of exactly this height, so keeping the number is what
// stops removing the feed from quietly re-laying out the arithmetic.
const (
	// mathBandHeight is how deep it is: what the feed's collapsed three rows came to.
	mathBandHeight = 82

	// mathBandGapAboveCards is how far its bottom edge sits above the resting hand row.
	//
	// **A selected card lifts by selectedNudge and does overlap it**, by 21 pixels, and that is
	// accepted rather than overlooked: the band is measured against where the cards live, not
	// against where one of them goes when it is picked.
	mathBandGapAboveCards = 5
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
// open question of how one pane could be both: the flow pane never learned to mark a hand
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
)

// paneEdge is the pink a pane is bordered and named in. Still a placeholder palette.
var paneEdge = color.RGBA{R: 235, G: 105, B: 170, A: 255}

// handSwatch marks a line that is not one side acting but something the round did — a hand
// forming. **It is the yellow the enemy used to be**, freed when the opponent went grey on
// 2026-08-07: a hand is the loudest thing that can happen in a round and had been sharing a
// hue with every enemy action on screen.
//
// Darker than a screen yellow because it sits on a light pane now — the same figure that read
// as amber on plum reads as washed-out cream on off-white.
var handSwatch = color.RGBA{R: 198, G: 142, B: 16, A: 255}

// The two sides' colours: **green is you, grey is them.**
//
// The opponent was yellow until 2026-08-07 and went grey to give the yellow to `handSwatch` —
// a hand is the loudest thing that can happen in a round and was sharing a hue with every
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

// logRows writes the sentences for a run of events: one line per thing that happened, in the
// order the resolver produced them.
//
// **One line per slot, not one per event.** A busy round is 25-30 events, so writing the log
// verbatim would be a panel nobody could read. Merging an action with its outcome is
// presentation of events the engine already decided; it computes nothing, so what is written
// here still cannot disagree with what the round did. **Hands and chills get lines of their
// own**, because they are not something a card did — folding a hand into the line of the card
// that happened to start it would bury the one thing worth reading.
//
// **The attack phase is one line, and it is the hand's** *(2026-08-14)*. Prepares and defends
// still write a line each; the attack cards write none. A turn lands one blow, so five sentences
// saying "Duelist attacks with an earth strike" described a round that does not happen, and the
// line that mattered — what the five cards came to — was the sixth. **Every hand takes that line,
// the High Card included** *(2026-08-19)* — a lone attack is the catalogue's one-card hand and is
// announced like any other.
//
// **It takes the events rather than reading the round off the scene** *(2026-08-18)*. It was the
// Resolution feed's walk, over `s.log[:cursor+1]`; the feed is gone and the fight log is what
// draws these rows now, over every round of the fight. Passing the slice is what let one walk
// serve both while both existed, and it is what keeps this function free of any opinion about
// where the rows are going.
//
// It knows nothing about capacity, overflow or which row is live. Those are properties of the
// panel the rows are poured into, not of the round, and they stay with the caller.
func (s *CombatScene) logRows(events []combat.Event) []paneRow {
	var rows []paneRow

	// cur is the line the next outcome attaches to, or -1 when the last thing appended was
	// an announcement rather than an action. curSide is whose line it is.
	//
	// **The side is tracked rather than read back off the row's swatch**, because the hand line
	// wears amber and takes outcomes: a damage event compared against that swatch would read
	// every hit as belonging to the wrong duelist.
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
			verb:    verbFor(c.Category()),
			suffix:  " " + cardPhrase(c),
			verbInk: verbInkFor(c.Category()),
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
			prefix: fmt.Sprintf("HAND!  %s lands a %s", s.sideName(e.Side), handName(e)),
			suffix: "  " + handMath(e),
			swatch: handSwatch,
		})
		cur, curSide = len(rows)-1, e.Side
		outcomes = 1 // the sum is already on the line, so the first outcome reads as a list
	}

	for _, e := range events {
		switch e.Kind {
		case combat.KindRoundStart, combat.KindRoundEnd:
			// The feed holds one round, so saying which round it is would be a line spent on
			// something the caption and the character block both already carry. **The fight log
			// does need it and writes its own heading**, from the position of the round in the
			// fight rather than from these events — a heading belongs to the caller, which is
			// the only one of the two that knows whether a round has anything before it.

		case combat.KindAction:
			// **A hand-forming side's attack card writes no line.** Its beat still passes — the engine
			// announces every card so the table can light it and playback can count slots — but the
			// sentence for the whole phase is the KindHand below.
			//
			// **A solo attacker has no phase line, so the card's own sentence is the line**
			// *(2026-08-17)*. There is no KindHand coming for it, and an attack that reported
			// nothing but a damage figure with no verb in front of it would be the one kind of
			// action in the round that never says what it was.
			if combat.Plain(e.Action).Category() == combat.CategoryAttack && !s.soloAttacker(e.Side) {
				break
			}
			act(e.Side, combat.Card{Concept: e.Action, Element: e.Element})

		case combat.KindChilled:
			announce(fmt.Sprintf("%s is chilled - %v is lost", s.sideName(e.Side), combat.ConceptOf(e.Action).Label),
				swatchFor(e.Side))

		case combat.KindMissed:
			// It attaches to the attacker's own line rather than announcing, because the card
			// *was* played — the line above it is real and this is what became of it. Naming
			// the shock is the whole point: a blow that simply missed would look like a bug in
			// a game with no dice in it.
			attach("misses - shocked")

		case combat.KindStatus:
			attach(statusPhrase(e.Status))

		case combat.KindBurned:
			// A tick belongs to nobody's card, so it opens its own line. It carries the
			// victim's swatch because it is a thing happening *to* them, which is also the
			// only side the event names.
			//
			// **The status names itself** *(2026-08-17)*: with statuses decoupled from the colours,
			// a second damage-over-time status would otherwise narrate identically to the first.
			announce(fmt.Sprintf("%s %s %d",
				s.sideName(e.Target), tickVerb(e.Status), e.Amount), swatchFor(e.Target))

		case combat.KindHand:
			// **This is the attack phase's line, and every hand takes it — the High Card
			// included** *(2026-08-19)*. There used to be a branch here writing an ordinary attack
			// sentence when `e.Hand` was `HandNone`, on the argument that announcing "HAND!" over
			// a single Strike empties the word. **It had been unreachable for some time**:
			// `blowFor` falls back to the catalogue's `high-card` entry, so a turn with an attack
			// in it always names a hand and the branch could not fire. What the log actually
			// printed was the hand line, correctly, while the code beside it said otherwise.
			//
			// The High Card is an equal citizen throughout now, on the owner's call, so this is
			// deliberate rather than merely true.
			blow(e)

		case combat.KindGathered:
			attach(fmt.Sprintf("+%d AP", e.Amount))

		case combat.KindDrew:
			// **The cards say "next round" and this line does not**, because a feed line is read
			// while a round is being replayed and the sentence around it is already in the past.
			// The card face is the place that has to state the timing.
			attach(fmt.Sprintf("+%d cards", e.Amount))

		case combat.KindNegated:
			// The card that answered the blow is named rather than assumed. Defend is the only one
			// that can reach here today, and the sentence is written off the event anyway — a
			// second card that reduced damage would read correctly without touching this.
			attach(fmt.Sprintf("halved by a %v", lower(combat.ConceptOf(e.Action).Label)))

		case combat.KindDamage:
			// **Damage whose side does not match the line it is attaching to is damage running the
			// other way**, which reads as something done back rather than as a hit of its own.
			// Nothing in the game produces it as of 2026-08-15 — the counter-attacking card that
			// did was cut — and it is kept because the test is a side comparison rather than a
			// card name, so it costs one branch and catches the case rather than mis-narrating it.
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
// **Every card's prose is generated from its verb, not written down** *(2026-08-16)*. There were
// two hand-maintained tables here, one string per concept, which worked while there were fourteen
// concepts. There are hundreds now — every enemy carries its own cards — so a table would be a
// list nobody could keep complete, and a card with no entry read as though nothing had happened.
//
// **The wording still lives here and not in `internal/combat`.** The rules package names cards; it
// does not describe them. What changed is that the description is now a function of the rule
// rather than a lookup beside it, which is the only version that can cover a deck written in JSON.

// attackVerb is the word a form hits with. **The player's three forms are told apart by it** —
// nine attack cards on one ladder, and a card whose text began "Deal" on all nine would leave the
// corner letter carrying the distinction alone. An enemy card belongs to no form and simply hits.
func attackVerb(f combat.Form) string {
	switch f {
	case combat.FormStab:
		return "Stabs"
	case combat.FormSlash:
		return "Slashes"
	case combat.FormCrush:
		return "Crushes"
	default:
		return "Hits"
	}
}

// multiplierText writes a damage multiplier the way a card says it: 0.5x, 1x, 1.5x, 2x.
//
// **A multiplier rather than a word** — "0.5x" instead of "half" — because a multiplier is what
// the rule actually is, and because the column is about a dozen characters wide.
func multiplierText(amount int) string {
	whole, frac := amount/100, amount%100
	if frac == 0 {
		return strconv.Itoa(whole) + "x"
	}
	if frac%10 == 0 {
		return fmt.Sprintf("%d.%dx", whole, frac/10)
	}
	return fmt.Sprintf("%d.%02dx", whole, frac)
}

// cardEffect is what a card does, in words, printed on its face.
//
// **Verb first, on every card.** They are read in a row while the player is counting action
// points — the first word saying what the card *does to the round* is what makes eight of them
// scannable. **"DMG" rather than "damage"** because the column is about a dozen characters wide
// and the duelist card already labels the figure that way.
//
// It is deliberately not the same wording as cardPhrase: that is prose for a *sentence about a
// round* — "attacks with a heavy strike" — and this is a rules description read while deciding
// whether to play the thing.
// **It reads the card, not the concept** *(2026-08-17)*. Every figure printed here comes from
// `Card.Amount()`, which is where a worm's scaling is applied — so an altered Defend says the
// percentage it actually cuts and an altered Prepare says what it actually banks. The wording was
// already a template over the value; what changed is which value it reads. A card whose face
// disagreed with its behaviour would be the worst thing an alteration mechanic could produce.
func cardEffect(card combat.Card) string {
	c := card.Spec()
	amount := card.Amount()
	switch c.Verb {
	case combat.VerbDefend:
		return "Cuts damage by " + strconv.Itoa(amount) + "%"
	case combat.VerbBank:
		return "Bank " + strconv.Itoa(amount) + " AP for next round"
	case combat.VerbDraw:
		return "Draw " + strconv.Itoa(amount) + " cards next round"
	default:
		if c.Target == combat.TargetSelf {
			return "Costs you " + multiplierText(amount) + " DMG"
		}
		return attackVerb(c.Form) + " for " + multiplierText(amount) + " DMG"
	}
}

// actionPhrase is what follows the verb in a Resolution line, and every phrase carries an article
// so cardPhrase can slot an element into it.
//
// **The card's own label is the noun.** That is what lets one function narrate four hundred
// concepts: "with a fire strike" for the player, "behind a congeal" for a slime.
func actionPhrase(id combat.ConceptID) string {
	c := combat.ConceptOf(id)
	name := lower(c.Label)
	switch c.Verb {
	case combat.VerbDefend:
		return "behind a " + name
	case combat.VerbBank:
		return "and gathers with a " + name
	case combat.VerbDraw:
		return "and looks ahead with a " + name
	default:
		if c.Target == combat.TargetSelf {
			return "with a " + name + ", at their own cost"
		}
		return "with a " + name
	}
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
	phrase := actionPhrase(c.Concept)
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

// statusPhrase is what a landed status says it did, as an outcome attached to the attacker's line.
// Each names the *effect* rather than the status, because "chills them" says what happens next and
// "applies chilled" says only that a rule fired.
//
// **Keyed by record rather than by element** *(2026-08-17)*, since a status is no longer a colour:
// two rings can put two different statuses on the same fire card, and one phrase per colour could
// not tell them apart. The fallback is what a status with no sentence of its own narrates as — its
// own name, which is at least true — so authoring a status in the file does not need a Go change to
// read properly.
func statusPhrase(id combat.StatusID) string {
	spec := combat.StatusOf(id)
	switch spec.Key {
	case "burning":
		return "sets them burning"
	case "chilled":
		return "chills them"
	case "shocked":
		return "shocks them"
	case "weighted":
		return "weighs them down"
	default:
		return "leaves them " + lower(spec.Name)
	}
}

// tickVerb is how a damage-over-time status reads when it bites at the end of a round: "Goblin burns
// for 2". A status with no verb of its own falls back to its name, which is true rather than
// graceful — and is what stops a second such status narrating as a burn.
func tickVerb(id combat.StatusID) string {
	if combat.StatusOf(id).Key == "burning" {
		return "burns for"
	}
	return "takes " + lower(combat.StatusOf(id).Name) + " damage:"
}

// verbFor is the verb a category is spoken with.
func verbFor(c combat.Category) string {
	if c == combat.CategoryPlan {
		return "plans"
	}
	return "attacks"
}

// The colour the verb is *written* in. **Red for attack, blue for plan** — the category made loud
// enough to scan a round by, without reading it.
//
// **The verb was a filled chip until 2026-08-08 and is now the word itself**, coloured, bolded
// and underlined. The chip was a saturated block in a pane that already carries a swatch and a
// sentence, and it drew the eye to a rectangle rather than to the word inside it. Marking the
// word spends the same signal on the thing being read, which is the reasoning that already
// retired the full-width highlight bar a day earlier — this is the same mistake one scale
// smaller.
//
// **Plan takes the blue defend used to have** *(2026-08-15)*, rather than the no-hue prepare had.
// With two categories the second colour is the whole distinction, and a category rendered in the
// row's own ink would leave "attacks" as the only marked verb — which is a highlight, not a
// scheme.
func verbInkFor(c combat.Category) color.RGBA {
	if c == combat.CategoryPlan {
		return color.RGBA{R: 52, G: 104, B: 196, A: 255}
	}
	return color.RGBA{R: 186, G: 52, B: 52, A: 255}
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
// soloAttacker reports whether a side's attack cards resolve one at a time rather than as a hand.
//
// **It reads the duelist the round was resolved for, not the side** — see
// `combat.Duelist.SoloAttacks`. Two things on this screen change with it and both would otherwise
// have to guess: the feed writes a sentence per attack card because no hand line is coming, and
// the table lights one card at a time because no single blow is being assembled.
//
// A missing combatant answers false, which is the hand-forming case: this is asked while drawing, and
// a half-built scene should read as the ordinary round rather than as an enemy's.
func (s *CombatScene) soloAttacker(side combat.Side) bool {
	c := s.fighter
	if side == combat.SideB {
		c = s.enemy
	}
	return c != nil && c.SoloAttacks
}

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
// **It no longer has to draw a hand spanning non-adjacent slots**, which was an open problem
// for as long as this was the only pane: one row per slot with a single walking highlight has
// no way to say "these together did a thing". The Resolution pane says it in words instead.
// The same goes for a slot a chill deleted — this pane still draws it as a row, and the
// other one is where it is reported as lost.
func (s *CombatScene) actionFlowRows(fighter, enemy []combat.Card, concealEnemy bool) []paneRow {
	order := combat.ResolutionOrder(fighter, enemy)
	if len(order) == 0 {
		return []paneRow{{prefix: "(empty)"}}
	}

	playingSlot, playing := s.currentSlot()

	rows := make([]paneRow, 0, len(order))
	for i, slot := range order {
		label, swatch := slot.Card.Label(), playerSwatch
		if slot.Side == combat.SideB {
			swatch = enemySwatch
			if concealEnemy {
				label = concealedLabel(slot.Card)
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
func concealedLabel(c combat.Card) string {
	return fmt.Sprintf("??? (%s)", c.Category())
}

// handName is what the attack phase formed, said in words: "Two Pair", "Four of a Kind".
//
// **A hand carries its whole name** *(2026-08-17)*. The name used to be assembled here from two
// parts — the element makeup in front of the hand, "Duo Strike Flurry" — and both of those axes
// are gone: colour buys statuses rather than a multiplier, and a hand is named for its shape
// rather than for the card that formed it. A blow that formed no hand at all is named only as an
// attack; the pane does not announce those, but the trace does.
//
// The name comes from the catalogue rather than being written here, so a hand renamed in
// `data/hands.json` is renamed once.
func handName(e combat.Event) string {
	hand, ok := combat.HandByID(e.Hand)
	if !ok {
		return "attack"
	}
	return hand.Name
}

// handMath is the blow written out as the sum it is: `20 x 1.5 = 30`.
//
// **Every figure in it comes off the event.** Base, Multiplier and the total are all worked out
// by the resolver, so the line cannot claim a sum the round did not use — which is the whole
// reason those fields are on the event rather than being recomputed here.
//
// **The cards' damage is one term, not one term each.** The hand dialog is what spells the hand
// out card by card; the feed is three rows of a sentence, and four identical numbers would be
// half a line saying what the dialog just showed at four times the size.
//
// **The multiplier is always written, the identity included** *(2026-08-19, owner's call)*. A High
// Card used to print `(20)` rather than `(20 x 1 = 20)`, on the argument that a sum times one says
// nothing. **Hands are going to be upgradable**, which makes that 1 a number that will change — and
// a term that appears only once it stops being 1 would make an upgrade read as a new rule rather
// than as a bigger figure. The dialog does the same, so the line and the sum say one thing.
//
// It is the blow before the attacker's weight and before anything the defender raised, so the
// damage that follows on the same line is often smaller. That gap is what a defence is worth,
// and it is only legible because both figures are shown.
func handMath(e combat.Event) string {
	return fmt.Sprintf("(%d x %s = %d)", e.Base, handMultiplierText(e.Multiplier), e.Amount)
}

// handMultiplierText writes a *hand's* percentage multiplier the way the design does: 350 as
// `3.5`, 200 as `2`, 1000 as `10`. Trailing zeros are dropped rather than padded to two places,
// because `x 10.00` reads as a precision the game does not have.
//
// It is not `multiplierText`, which writes a *card's* multiplier and keeps its `x`. The two read
// almost the same and are printed in different sentences: this one lands inside the arithmetic
// line, where the `x` is already there as an operator.
func handMultiplierText(pct int) string {
	return strconv.FormatFloat(float64(pct)/100, 'f', -1, 64)
}
