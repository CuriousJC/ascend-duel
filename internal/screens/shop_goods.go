package screens

// The two sealed goods, and the dialog that opens them.
//
// **A bag of rocks and a vial of essence** *(owner's call, 2026-08-27)*. Both cost five vitae, both
// hold four of something, and both give the player exactly one of the four — the other three are
// gone. What is bought is the *choice*, which is what makes them different from a relic on the
// shelf: a relic is a thing you read and then pay for, and these are paid for and then read.
//
// **The bag is the only way a stone is ever got.** A stone raises one rung of the hand ladder by a
// tenth of its catalog multiplier, for the rest of the run — see `internal/session/stone.go`.
// Choosing it is using it: there is no inventory, so the click that picks a rock is the click that
// puts it on the ladder.
//
// **The vial is an essence, on the same terms as the reward screen's** — and that includes the gesture:
// the four essences and a hand's worth of cards are up together, and you select the card first and
// click the essence second *(owner's call, 2026-09-06)*. It was two stages until then, and what was
// wrong with them is that the essences left the screen at the moment the player had to judge one
// against a card. See targeting.go for the rule, and the reward screen, which merged its own two
// stages the same day. It is worth five vitae over a free offer of two because four is twice the
// choice and because it arrives at the shop rather than at the end of a fight, which is a
// different moment to want one at.
//
// **The vial's title is an instruction and its hint is the gesture**, because a dialog whose two
// rows are both live has to say which one is clicked first.
//
// **This dialog has no X, and that is deliberate.** Every other modal in the game is a look at
// something and closes without consequence; this one stands between a purchase and what it bought,
// so an exit that forfeited five vitae would be a trap wearing the same red square that means
// "close" everywhere else. Every card in it is an exit — the dialog ends when one is chosen — and
// the second stage cannot be reached without the first.

import (
	"fmt"
	"github.com/curiousjc/ascend-duel/internal/achieve"
	"image"
	"image/color"
	"math/rand"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// **There is no goodKind enum any more** *(2026-09-14)*. A shelf seat names a good by its record
// key, the catalog is data/goods.json, and what the dialog does with one is switched on
// session.GoodContents — the closed vocabulary that says which catalog is inside. The three names
// that used to be constants here are the records' own Name fields.

// Where the goods row sits, and how the dialog lays its cards out.
const (
	// The dialog's four cards, centered in the panel — the bag's and the sack's, which have one
	// row and nothing under it.
	goodsChoiceRowPct = 42

	// The vial has two rows and they both have to fit inside the panel, so its essences sit high and
	// the cards they may eat sit under them. **Not 42 and 70** *(2026-09-06)*: the offer row ended
	// exactly on the panel's bottom edge, which was survivable while it was the only row on screen
	// and is not now that a selected card is lifted into the row above it.
	vialEssenceRowPct = 22
	goodsOfferRowPct  = 55

	// The line under the title, saying what to do with what is up.
	goodsHintTop = 120
)

// goodsStage is how far through opening a good the player is.
type goodsStage int

const (
	// goodsClosed: nothing is open, and the shop is the shop.
	goodsClosed goodsStage = iota

	// goodsPick: what was drawn is up, and a click takes one. **There is no second stage**
	// *(owner's call, 2026-09-06)*: the vial used to move on to a row of cards once an essence was
	// chosen, so the essences disappeared at the moment the player had to judge one against a card.
	// Both rows are up together now and the gesture is the row's own — select the card, then click
	// the essence. See targeting.go, which is the rule the rune pane and the reward screen's own
	// essence row already follow.
	goodsPick

	// goodsShowing: an essence has been spent and what it did is on screen — the card flies to the
	// middle and changes there, exactly as it does on the reward screen *(owner's call,
	// 2026-09-16)*. It was missing here: the vial applied the essence on the click and closed, so
	// the one screen where a player pays five vitae for an alteration was the one that never showed
	// them the alteration. **Only the vial reaches this stage** — a stone and a rune change no card,
	// so there is nothing to watch.
	goodsShowing
)

// goods is the dialog: which good was opened, what was drawn from it, and how far through the
// player is.
//
// **It holds no purse and no run.** The vitae is taken by the shop the moment a good is clicked —
// see `ShopScene.openGood` — so what is here is the offer and the choice, exactly as the shelf
// holds a relic's key and not its price.
type goods struct {
	// good is the record that was opened: its name, its size, and what is inside it. **The whole
	// record rather than a key**, because every question this dialog asks of it — how many were
	// drawn, what to head the panel with, which catalog to deal — is a field on it.
	good  session.Good
	stage goodsStage

	// stones, essences and runes are what was drawn, and only the one matching kind is filled.
	stones   []session.Stone
	essences []session.Essence
	runes    []session.Rune

	// offer is the cards an essence may be aimed at, by index into the run's deck. **Only the vial
	// fills it**, and it is dealt when the vial is opened rather than when an essence is picked — the
	// two rows are up at once, so the cards cannot be a function of a choice not yet made.
	offer []int

	// selected is which offered card is picked out, or -1. **One card**, because an essence eats
	// exactly one — the reward screen's own field, and the same reason it is an index rather than
	// a set. See consumableTarget.
	selected int

	// tip explains whichever card the cursor is resting on.
	tip models.Tooltip

	// What the spent essence did, held on screen through goodsShowing. **The reward screen's own
	// fields and the same rules** — see PostBattleScene, where each of these is written up: the card
	// that flies is the card the player was looking at, the morph is the change happening in front
	// of them, and the deck is not touched until the hold is over.
	before      combat.Card
	after       combat.Card
	change      morph
	removes     bool
	copied      bool
	held        int
	arrival     travel
	arrivedFrom image.Rectangle
	applyNow    func(*session.Session)
}

// open puts a good up, drawing what is inside it.
//
// **The contents are drawn here rather than by the run**, from a per-fight stream of their own, so
// a purchase interrupted by a quit leaves nothing to snapshot. Buying the same bag twice in one
// visit is not possible — see the shelf's `bought` flag — so a stream per fight is a stream per
// bag.
func (g *goods) open(gs *state.GlobalState, good session.Good) {
	g.good, g.stage, g.selected = good, goodsPick, -1
	g.stones, g.essences, g.runes, g.offer = nil, nil, nil, nil
	g.tip = models.Tooltip{DwellTicks: tipDwell()}

	switch good.Contains {
	case session.ContentsStones:
		g.stones = dealStones(gs, good.Record, good.Size)
	case session.ContentsEssences:
		g.essences = dealVialEssences(gs, good.Record, good.Size)
		g.offer = dealVialOffer(gs)
	case session.ContentsRunes:
		g.runes = dealSackRunes(gs, good.Record, good.Size)
	}
}

// openBag reports whether anything is up.
func (g *goods) openNow() bool { return g.stage != goodsClosed }

// close puts it away.
func (g *goods) reset() {
	g.good, g.stage, g.selected = session.Good{}, goodsClosed, -1
	g.stones, g.essences, g.runes, g.offer = nil, nil, nil, nil
	g.before, g.after, g.change = combat.Card{}, combat.Card{}, morph{}
	g.removes, g.copied, g.held = false, false, 0
	g.arrival, g.arrivedFrom, g.applyNow = travel{}, image.Rectangle{}, nil
	g.tip.Forget()
}

// count is how many cards are in the row that is taken from: the stones, the runes, or the
// essences. **Not the offer**, which is a second row with its own slot function.
func (g *goods) count() int {
	switch g.good.Contains {
	case session.ContentsStones:
		return len(g.stones)
	case session.ContentsRunes:
		return len(g.runes)
	default:
		return len(g.essences)
	}
}

// dealStones is what a bag holds: four of the catalog, without repeats.
//
// **Its own stream** (`seeds.BagStock`), separate from the shelf's relics and from both essence draws —
// see internal/seeds, where the argument is written down. **Without repeats**, because a bag
// offering the same rock twice is a seat spent saying nothing, exactly as the shelf is.
//
// **Flat, not weighted.** A stone has no rarity: every rung is worth a tenth of itself, so a
// Card Five stone is not a better rock than a Card Pair stone — it is a rock for a rung you may
// never build. Weighting them would be pricing the *hand*, which the ladder already does.
func dealStones(gs *state.GlobalState, seat string, size int) []session.Stone {
	all := session.Stones()
	rng := rand.New(rand.NewSource(
		seeds.ForFightSeat(gs.RunSeed, seeds.BagStock, gs.Run.Fight(), seat)))
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	if len(all) > size {
		all = all[:size]
	}
	return all
}

// dealVialEssences is what a vial holds: four essences, without repeats.
//
// **A different stream from the reward screen's two** (`seeds.VialStock`), which is the case the
// salts exist for: sharing would make the shop's four a function of which two had just been
// offered free, so buying the vial could guarantee — or rule out — the pair the player had turned
// down. See internal/seeds.
func dealVialEssences(gs *state.GlobalState, seat string, size int) []session.Essence {
	all := session.Essences()
	rng := rand.New(rand.NewSource(
		seeds.ForFightSeat(gs.RunSeed, seeds.VialStock, gs.Run.Fight(), seat)))
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	if len(all) > size {
		all = all[:size]
	}
	return all
}

// dealSackRunes is what a sack holds: four of the catalog, without repeats.
//
// **Its own stream** (`seeds.SackStock`), separate from both other goods and from the reward
// screen's essences — see internal/seeds. **Flat, not weighted**, on the bag's argument: a rune
// has no rarity, and weighting them would be pricing the effect, which nothing has decided yet.
//
// **A catalog shorter than the sack is not an error.** Four runes ship and the sack holds
// four, so it currently offers the whole file; the cut is what keeps that true as the list grows.
func dealSackRunes(gs *state.GlobalState, seat string, size int) []session.Rune {
	all := session.Runes()
	rng := rand.New(rand.NewSource(
		seeds.ForFightSeat(gs.RunSeed, seeds.SackStock, gs.Run.Fight(), seat)))
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	if len(all) > size {
		all = all[:size]
	}
	return all
}

// runeTipLines is what resting on a rune says: what it does, what it would fire, and when it can be
// spent.
//
// **It is the whole of what a rune says now** *(owner's call, 2026-09-16)*. The card is a picture
// and a name; the authored line moved here rather than sitting on a scrim over the art. See
// runeSpec.
//
// **The "when" is the half the card never could say.** The thing that makes a rune a different
// object from an essence is not on its face — so the tooltip is where a player finds out that this
// one is carried into a fight rather than used now.
//
// **A chimera says what it would fire**, because its authored line cannot: its whole subject is a
// rune named somewhere else, and "copies the last" is a line the player has to remember the answer
// to. On a run that has spent nothing there is no answer and it keeps its own line.
func runeTipLines(gs *state.GlobalState, p session.Rune) []string {
	// **The card's own text, unwrapped.** An authored line break on a face is a tooltip's own line:
	// the two are the same sentence written for two widths — the treatment essenceTip already gives
	// an essence.
	lines := strings.Split(p.Text, "\n")
	if gs.Run != nil {
		if echoed := gs.Run.EchoedName(p); echoed != "" {
			lines = append(lines, "it would copy "+echoed)
		}
	}
	return append(lines, "spent between the turns of a fight")
}

// dealVialOffer is the hand the vial deals beside its essences: a shuffle of every position in the run's
// deck, cut to a hand's worth.
//
// **It is not filtered by an essence any more** *(owner's call, 2026-09-06)*. It could not be: both rows
// are up at once now, so the cards are dealt before the player has chosen anything, and a filter
// would have to know which essence. What replaces it is the *essence* going dim — an essence that cannot
// change the selected card is unclickable, which is the same information at the other end of the
// gesture, and the reward screen's own rule. See essenceSpendable.
func dealVialOffer(gs *state.GlobalState) []int {
	if gs.Run == nil || gs.Run.Size() == 0 {
		return nil
	}

	idx := make([]int, 0, gs.Run.Size())
	for i := 0; i < gs.Run.Size(); i++ {
		idx = append(idx, i)
	}

	rng := rand.New(rand.NewSource(seeds.ForFight(gs.RunSeed, seeds.VialStock, gs.Run.Fight())))
	rng.Shuffle(len(idx), func(i, j int) { idx[i], idx[j] = idx[j], idx[i] })

	if len(idx) > handSize {
		idx = idx[:handSize]
	}
	sortInts(idx)
	return idx
}

// slot is where one card of the current stage is drawn, and the rectangle it is clicked in. One
// function for both, like every other row in the game.
func (g *goods) slot(gs *state.GlobalState, i int) image.Rectangle {
	n := g.count()
	if n == 0 {
		return image.Rectangle{}
	}

	top := gs.PctY(goodsChoiceRowPct)
	if g.good.Contains == session.ContentsEssences {
		top = gs.PctY(vialEssenceRowPct)
	}
	pitch := cardWidth + 40

	width := (n-1)*pitch + cardWidth
	left := gs.PctX(50) - width/2
	return image.Rect(left+i*pitch, top, left+i*pitch+cardWidth, top+cardHeight)
}

// offerSlot is where one of the cards an essence may eat is drawn and clicked. **The hand's own
// compressing pitch**, because this row is a hand's worth of cards and the fixed pitch above it is
// for four.
//
// **A selected card is lifted**, exactly as it is in the hand and on the reward screen: the row a
// consumable is aimed at says which card is picked by standing it up, not by a highlight nobody
// has to learn.
func (g *goods) offerSlot(gs *state.GlobalState, i int) image.Rectangle {
	n := len(g.offer)
	if n == 0 || i < 0 || i >= n {
		return image.Rectangle{}
	}

	pitch := handPitch(gs, n)
	width := (n-1)*pitch + cardWidth
	left := gs.PctX(50) - width/2 + i*pitch
	top := gs.PctY(goodsOfferRowPct)
	if i == g.selected {
		top -= offerSelectedNudge
	}
	return image.Rect(left, top, left+cardWidth, top+cardHeight)
}

// selectedDeckIndex is the offer's current pick as an index into the run deck, and whether there is
// one. The reward screen's function of the same name, and the same job.
func (g *goods) selectedDeckIndex() (int, bool) {
	if g.selected < 0 || g.selected >= len(g.offer) {
		return 0, false
	}
	return g.offer[g.selected], true
}

// essenceSpendable is whether clicking this essence now would take it: a card is selected, and this essence
// can actually change that card.
//
// **The same question the click asks and the same one the card's lit state reads**, which is what
// stops an essence looking available and doing nothing. It is the reward screen's predicate over the
// same target rule — see targeting.go.
func (g *goods) essenceSpendable(gs *state.GlobalState, w session.Essence) bool {
	if gs.Run == nil {
		return false
	}
	target := consumableTarget{
		needs: 1,
		legal: func(ids []int) bool { return gs.Run.CanApply(w, ids[0]) },
	}
	if idx, ok := g.selectedDeckIndex(); ok {
		return target.satisfiedBy([]int{idx})
	}
	return target.satisfiedBy(nil)
}

// update runs the dialog and reports whether it swallowed the frame. **The shop is dead while it
// is up**, which is what `gs.ModalOpen` says to the game's own chrome as well.
func (g *goods) update(gs *state.GlobalState) bool {
	if !g.openNow() {
		return false
	}
	gs.ModalOpen = true

	if g.stage == goodsShowing {
		g.tickShowing(gs)
		return true
	}

	g.hover(gs)
	systems.UpdateTooltip(gs, &g.tip)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && gs.CursorAllowed() {
		g.click(gs)
	}
	return true
}

func (g *goods) hover(gs *state.GlobalState) {
	if !gs.CursorAllowed() {
		return
	}
	at := image.Pt(gs.MouseX, gs.MouseY)

	for i := range g.offer {
		seat := g.offerSlot(gs, i)
		if !at.In(seat) {
			continue
		}
		card, ok := gs.Run.Card(g.offer[i])
		if !ok {
			return
		}
		title, lines := cardTip(card, heldByRun(gs, card))
		g.tip.Point(seat, tipLine(title), tipLines(lines))
		return
	}

	for i := 0; i < g.count(); i++ {
		if !at.In(g.slot(gs, i)) {
			continue
		}
		switch g.good.Contains {
		case session.ContentsEssences:
			// **The essences are deliberately not tooltipped**, which is the reward screen's own
			// choice on the same row: an essence's whole rule is printed on its face, where a deck
			// card's is not. What a dim essence means — "not for the card you have selected" — is
			// left to the row rather than to a tooltip.
			return
		case session.ContentsStones:
			st := g.stones[i]
			g.tip.Point(g.slot(gs, i), tipLine(st.Name), tipLines(stoneTipLines(gs, st)))
		case session.ContentsRunes:
			p := g.runes[i]
			g.tip.Point(g.slot(gs, i), tipLine(p.Name), tipLines(runeTipLines(gs, p)))
		}
		return
	}
}

// click is the whole of what this dialog does. **A click on a card is the only input it takes** —
// there is no confirm and no close, because the purchase has already happened and what is left is
// which one.
func (g *goods) click(gs *state.GlobalState) {
	at := image.Pt(gs.MouseX, gs.MouseY)

	// **The offer row first**, because it is the row drawn in front: a selected card is lifted and
	// a lifted card overlaps nothing above it, but reading the rows in drawing order is the rule
	// every screen here follows.
	for i := range g.offer {
		if at.In(g.offerSlot(gs, i)) {
			g.selectCard(i)
			return
		}
	}

	for i := 0; i < g.count(); i++ {
		if !at.In(g.slot(gs, i)) {
			continue
		}
		g.take(gs, i)
		return
	}
}

// take commits whichever card was clicked.
func (g *goods) take(gs *state.GlobalState, i int) {
	switch g.good.Contains {
	case session.ContentsEssences:
		// **An essence is refused rather than falling back**, on the predicate its lit state already
		// read: a dim essence cannot be taken and a lit one always works. With no card selected every
		// essence is dim, so the dialog waits rather than choosing a card for the player.
		essence := g.essences[i]
		idx, ok := g.selectedDeckIndex()
		if !ok || !g.essenceSpendable(gs, essence) {
			return
		}

		g.show(gs, essence, idx, g.offerSlot(gs, g.selected))

	case session.ContentsStones:
		stone := g.stones[i]
		if gs.Run.UseStone(stone.Record) {
			trace.Logf("shop", "bag of rocks: %s, %s now at %d stones",
				stone.Record, stone.Hand, gs.Run.StonesOn(stone.Hand))
		}
		g.reset()

	case session.ContentsRunes:
		// **A rune is not applied here — it goes into the sack.** That is the whole
		// difference between this good and the other two: a stone and an essence are spent on the
		// spot, and a rune is carried into the next fight and spent between its turns.
		p := g.runes[i]
		if gs.Run.Hold(p.Record) {
			trace.Logf("shop", "sack of runes: %s held, %d in the sack",
				p.Record, gs.Run.HoldCount())
		}
		g.reset()
	}
}

// show is the vial's last stage: the picked card flies to the middle, the essence changes it there,
// and the dialog closes once the player has had time to read what it became.
//
// **The preview runs the real essence against a throwaway copy of the run**, and the deck is not
// touched until the hold is over — both are the reward screen's rules, written up in
// PostBattleScene.aimAt. A preview computed by its own arithmetic is a preview that can disagree
// with the thing it is previewing, and a deck altered while the result is on screen would make the
// after-card impossible to draw from anything but a deck that has already moved on.
func (g *goods) show(gs *state.GlobalState, essence session.Essence, deckIndex int, from image.Rectangle) {
	before, ok := gs.Run.Card(deckIndex)
	if !ok {
		return
	}

	trial := session.New(gs.Run.Deck())
	if !trial.Apply(essence, deckIndex) {
		return
	}

	g.before = before
	g.removes = essence.Target == session.TargetRemove
	g.copied = essence.Target == session.TargetDuplicate

	switch {
	case g.copied:
		// The copy is appended, so the card that arrived is the last one.
		g.after, _ = trial.Card(trial.Size() - 1)
	case !g.removes:
		g.after, _ = trial.Card(deckIndex)
	}

	// **What the essence did decides which shape the change takes** — recolored, eaten, or copied.
	// See cardmorph.go; the morph is handed two finished faces and works out the rest.
	beforeSpec := cardSpec(before, heldByRun(gs, before), true, false)
	switch {
	case g.removes:
		g.change = morphAway(beforeSpec, cards.Hand)
	case g.copied:
		g.change = morphIn(cardSpec(g.after, heldByRun(gs, g.after), true, false), cards.Hand)
	default:
		g.change = morphInto(beforeSpec,
			cardSpec(g.after, heldByRun(gs, g.after), true, false), cards.Hand)
	}

	g.stage, g.held = goodsShowing, settledHoldTicks()
	g.arrival, g.arrivedFrom = newTravel(0, settleFlightTicks()), from
	g.applyNow = func(run *session.Session) { run.Apply(essence, deckIndex) }
	g.tip.Forget()

	trace.Logf("shop", "vial of essence: %s aimed at deck position %d", essence.Record, deckIndex)
}

// tickShowing runs the held picture: the flight, then the change, then the hold, and the deck edit
// at the end of it.
//
// **The change does not start until the card has landed**, and the hold does not start until the
// change has finished — the reward screen's ordering, and for its reason: a dissolve running over a
// moving card puts the one thing worth watching on a target the eye is still chasing.
func (g *goods) tickShowing(gs *state.GlobalState) {
	if !g.arrival.done() {
		g.arrival.tick()
		return
	}
	if !g.change.done() {
		g.change.tick()
		return
	}

	g.held--
	if g.held > 0 {
		return
	}

	if g.applyNow != nil {
		g.applyNow(gs.Run)
		trace.Logf("shop", "vial of essence applied, deck now %d", gs.Run.Size())

		// **The same moment the post-battle screen raises**, because it is the same event: a card in
		// the run's deck is now a different card. Skipped for a removal, which leaves nothing to name.
		if !g.removes {
			earnMoment(gs, achieve.CardAltered(g.after.Label()))
		}
	}
	g.reset()
}

// selectCard picks a card out of the offer row, or puts it back. **Clicking the selected card
// deselects it**, the hand row's own gesture, so the thing a player already knows how to undo works
// here too.
func (g *goods) selectCard(i int) {
	if g.selected == i {
		g.selected = -1
		return
	}
	g.selected = i
	g.tip.Forget()
}

// draw puts the dialog up. It takes the shared modal frame, so it reads as the same kind of thing
// as the deck and hands panels — **minus the X**, for the reason at the top of this file.
func (g *goods) draw(gs *state.GlobalState, screen *ebiten.Image) {
	if !g.openNow() {
		return
	}

	panel := drawModalFrame(gs, screen, modalHead{title: g.title()})

	if line := g.hint(); line != "" && g.stage != goodsShowing {
		hint := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 20}
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(panel.Min.X+panel.Dx()/2), float64(panel.Min.Y+goodsHintTop))
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(color.RGBA{R: 226, G: 228, B: 236, A: 255})
		text.Draw(screen, line, hint, op)
	}

	if g.stage == goodsShowing {
		g.drawShowing(gs, screen)
		return
	}

	for i := 0; i < g.count(); i++ {
		at := g.slot(gs, i).Min
		switch g.good.Contains {
		case session.ContentsStones:
			drawStoneCard(gs, screen, at, g.stones[i], true)
		case session.ContentsRunes:
			drawRuneCard(gs, screen, at, g.runes[i], true, false)
		default:
			// **An essence is lit only for the card that is selected.** With nothing selected the
			// whole row is dim, which is what says the gesture starts underneath — the reward
			// screen's rule, and the rune pane's.
			drawEssenceCard(gs, screen, at, g.essences[i], g.essenceSpendable(gs, g.essences[i]))
		}
	}

	// **The cards the essences may eat, under them and up at the same time** *(owner's call,
	// 2026-09-06)*. They are drawn after the essences so a lifted card is in front of the row above
	// it, which is the only place the two rows can meet.
	for i, deckIndex := range g.offer {
		card, ok := gs.Run.Card(deckIndex)
		if !ok {
			continue
		}
		drawCard(gs, screen, g.offerSlot(gs, i).Min, cards.Hand, card, heldByRun(gs, card),
			true, i == g.selected)
	}

	systems.DrawTooltip(gs, screen, &g.tip)
}

// title names what is open, and hint says what to do with it. **Two short lines rather than a
// paragraph**: the cards say what they are, and this says how many of them the player gets.
//
// **Both come off the record now** *(2026-09-14)*, resolved once at load — see session.Good, where a
// blank Title becomes the good's Name and a blank Hint becomes "take one of the four, the rest are
// gone", with the figure the record's own Size rather than a number authored twice. The vial is
// still the exception and still says one thing *(owner's call, 2026-09-05)*: the essences on the
// table are the whole of what the dialog is, so it heads itself with an instruction rather than a
// label with a caption. It authors both fields to say so.
func (g *goods) title() string {
	if g.stage == goodsShowing {
		if g.removes {
			return "EATEN"
		}
		return "CHANGED"
	}
	return g.good.Title
}

// drawShowing is the picked card flying to the middle, changing there, and held while it is read.
//
// **The flight carries the old face and the morph carries the new one**, which is why nothing here
// asks what the essence did — `change` was handed the two faces in show and is the only thing that
// knows which of the three shapes this is. A removal ends on an empty seat, a duplicate ends on two
// cards, everything else ends on one.
func (g *goods) drawShowing(gs *state.GlobalState, screen *ebiten.Image) {
	seats := settledSeats(gs, 1)
	if g.copied {
		seats = settledSeats(gs, 2)
	}

	at := flyingTo(g.arrivedFrom, seats[0], g.arrival)

	// While a copy is being made the card that flew is the original, untouched: the morph in the
	// second seat is the whole of what is happening.
	if g.copied {
		drawCard(gs, screen, at, cards.Hand, g.before, heldByRun(gs, g.before), true, false)
		drawMorph(gs, screen, seats[1].Min, g.change)
		return
	}

	drawMorph(gs, screen, at, g.change)
}

func (g *goods) hint() string { return g.good.Hint }

// stoneTipLines is what a stone's tooltip says: the rung it raises, what one is worth, and where
// that rung stands for this run right now.
//
// **The run's own figure rather than the catalog's**, because a second stone on a rung is worth
// exactly what the first was and the player has no other way to see what the first one did.
func stoneTipLines(gs *state.GlobalState, st session.Stone) []string {
	worth := session.StoneWorth(st.Hand)

	out := []string{fmt.Sprintf("raises %s by %d", stoneHandName(st.Hand), worth)}
	if gs.Run == nil {
		return out
	}
	if now, ok := gs.Run.HandMultiplier(st.Hand); ok {
		out = append(out, fmt.Sprintf("it pays x%d today, and x%d after this",
			now, now+worth))
	}
	if n := gs.Run.StonesOn(st.Hand); n > 0 {
		out = append(out, fmt.Sprintf("%d already on this rung", n))
	}
	return out
}

// stoneHandName is the rung a stone raises, by the name the rest of the game calls it.
//
// **Read off the ladder rather than off the stone**, which is the same split stoneLine made when the
// figure was computed: `data/stones.json` names a rung by key and `hands.json` owns what that rung
// is called, so a renamed rung cannot leave a stone's tooltip saying the old one. The key itself is
// the readable failure for a stone naming a rung that is not there.
func stoneHandName(key string) string {
	for _, h := range combat.Hands() {
		if h.Key == key {
			return h.Name
		}
	}
	return key
}
