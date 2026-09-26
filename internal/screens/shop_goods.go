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
// **There is no X, and the way out is SKIP.** Every other modal in the game is a look at something
// and closes without consequence; this stands between a purchase and what it bought, so leaving it
// forfeits the vitae. That is a choice the player may make, and GoodsScene gives it a labelled
// button rather than the red square that means "close" everywhere else. Every card is also an exit
// — the screen ends when one is chosen.

import (
	"fmt"
	"image"
	"math/rand"
	"sort"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/achieve"
	"github.com/curiousjc/ascend-duel/internal/ui"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/journal"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// **There is no goodKind enum any more** *(2026-09-14)*. A shelf seat names a good by its record
// key, the catalog is data/goods.json, and what the dialog does with one is switched on
// session.GoodContents — the closed vocabulary that says which catalog is inside. The three names
// that used to be constants here are the records' own Name fields.

// Where the goods rows sit.
//
// **Measured from the screen, and now standing on it** *(2026-09-19)*. These were always screen
// percentages, because the modal panel covered most of it; what changed is what they have to clear.
// The build band and the two lines of type under it are at the top of the screen, so the cards
// start below them rather than below a panel's title bar — see buildBandBottom and offerHintTop.
const (
	// The bag's and the sack's one row, centered, with nothing under it.
	goodsChoiceRowPct = 42

	// The vial has two rows, and both of them plus the lift a selected card takes have to fit
	// between the hint and the bottom of the screen.
	vialEssenceRowPct = 38
	goodsOfferRowPct  = 68
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

	// selected is which offered cards are picked out, as row slots. **A set, because an essence may
	// take more than one card** — one is the mechanic and a relic scales it, see
	// essence_targets.go. The reward screen's own field, under the same rules. See consumableTarget.
	selected []int

	// tip explains whichever card the cursor is resting on.
	tip models.Tooltip

	// What the spent essence did, held on screen through goodsShowing. **The reward screen's own
	// fields and the same rules** — see PostBattleScene, where each of these is written up: the card
	// that flies is the card the player was looking at, the morph is the change happening in front
	// of them, and the deck is not touched until the hold is over.
	lands       []essenceLanding
	removes     bool
	copied      bool
	held        int
	arrival     ui.Travel
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
	g.good, g.stage, g.selected = good, goodsPick, nil
	g.stones, g.essences, g.runes, g.offer = nil, nil, nil, nil
	g.tip = models.Tooltip{DwellTicks: ui.TipDwell()}

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
	g.good, g.stage, g.selected = session.Good{}, goodsClosed, nil
	g.stones, g.essences, g.runes, g.offer = nil, nil, nil, nil
	g.lands = nil
	g.removes, g.copied, g.held = false, false, 0
	g.arrival, g.arrivedFrom, g.applyNow = ui.Travel{}, image.Rectangle{}, nil
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
	if g.isSelected(i) {
		top -= offerSelectedNudge
	}
	return image.Rect(left, top, left+cardWidth, top+cardHeight)
}

// isSelected reports whether this row slot is one of the picked cards.
func (g *goods) isSelected(i int) bool {
	for _, sel := range g.selected {
		if sel == i {
			return true
		}
	}
	return false
}

// targets is how many cards an essence takes here — one, whatever the relics make of it, and never
// more than the offer is holding. See essence_targets.go.
func (g *goods) targets(gs *state.GlobalState) int {
	return essenceTargetCount(gs, len(g.offer))
}

// reachNow is how many cards a click on an essence would change right now: what is selected, or the
// ceiling when nothing is — the reward screen's rule. See essenceReach.
func (g *goods) reachNow(gs *state.GlobalState) int {
	return essenceReach(len(g.selected), g.targets(gs))
}

// selectedSlots is the picked cards **in row order**, whatever order they were clicked in — the
// reward screen's rule, and the combat screen's.
func (g *goods) selectedSlots() []int {
	out := append([]int(nil), g.selected...)
	sort.Ints(out)
	return out
}

// selectedDeckIndexes is the offer's current picks as indexes into the run deck, in row order. The
// reward screen's function of the same name, and the same job.
func (g *goods) selectedDeckIndexes() []int {
	out := make([]int, 0, len(g.selected))
	for _, slot := range g.selectedSlots() {
		if slot < 0 || slot >= len(g.offer) {
			return nil
		}
		out = append(out, g.offer[slot])
	}
	return out
}

// essenceSpendable is whether clicking this essence now would take it: at least one card and no
// more than the essence reaches is selected, and this essence can change every one of them.
//
// **A player may always take fewer**, the reward screen's rule — see consumableTarget.fewest.
//
// **The same question the click asks and the same one the card's lit state reads**, which is what
// stops an essence looking available and doing nothing. It is the reward screen's predicate over the
// same target rule — see targeting.go.
func (g *goods) essenceSpendable(gs *state.GlobalState, w session.Essence) bool {
	if gs.Run == nil {
		return false
	}
	target := consumableTarget{
		needs:  g.targets(gs),
		fewest: 1,
		legal: func(idx []int) bool {
			for _, i := range idx {
				if !gs.Run.CanApply(w, i) {
					return false
				}
			}
			return true
		},
	}
	return target.satisfiedBy(g.selectedDeckIndexes())
}

// update runs the good and reports whether anything is still open. **It no longer claims the frame**
// *(2026-09-19)*: this is a screen now, so there is nothing behind it to protect and no ModalOpen to
// set. False means the card has been taken and whatever it did has finished playing.
func (g *goods) update(gs *state.GlobalState, pile func(image.Point) bool) bool {
	if !g.openNow() {
		return false
	}

	if g.stage == goodsShowing {
		g.tickShowing(gs)
		return true
	}

	g.hover(gs)
	systems.UpdateTooltip(gs, &g.tip)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && gs.CursorAllowed() {
		// **The pile is asked first and it is the scene's**, because the panel it opens covers the
		// cards this file draws — see GoodsScene.clickedPile.
		if pile == nil || !pile(image.Pt(gs.MouseX, gs.MouseY)) {
			g.click(gs)
		}
	}
	return true
}

func (g *goods) hover(gs *state.GlobalState) {
	if !gs.CursorAllowed() {
		return
	}
	at := image.Pt(gs.MouseX, gs.MouseY)

	// **ui.HoveredSeat** — the vial deals its offer at the hand's pitch, so the row overlaps for the
	// same reason the hand does and the card on top is the last drawn.
	if i := ui.HoveredSeat(at, len(g.offer), func(i int) image.Rectangle {
		return g.offerSlot(gs, i)
	}); i >= 0 {
		seat := g.offerSlot(gs, i)
		if card, ok := gs.Run.Card(g.offer[i]); ok {
			title, lines := ui.CardTip(card, ui.HeldByRun(gs, card))
			g.tip.Point(seat, ui.TipLine(title), ui.TipLines(lines))
		}
		return
	}

	// **ui.HoveredSeat, like every row in the game.** This one is laid out at a pitch wider than a
	// card and cannot overlap today, so the walk's direction decides nothing — it goes through the
	// shared one so that a row nobody has to check stays a row nobody has to check.
	if i := ui.HoveredSeat(at, g.count(), func(i int) image.Rectangle {
		return g.slot(gs, i)
	}); i >= 0 {
		switch g.good.Contains {
		case session.ContentsEssences:
			// **An essence is tooltipped like everything else in this pane** *(owner's call,
			// 2026-09-18)*, now that its face is a picture rather than a sentence on a scrim. What
			// a dim essence means — "not for the card you have selected" — is still left to the row.
			w := g.essences[i]
			title, lines := ui.EssenceTip(w, g.reachNow(gs))
			g.tip.Point(g.slot(gs, i), ui.TipLine(title), ui.TipLines(lines))
		case session.ContentsStones:
			st := g.stones[i]
			g.tip.Point(g.slot(gs, i), ui.TipLine(st.Name), ui.TipLines(stoneTipLines(gs, st)))
		case session.ContentsRunes:
			p := g.runes[i]
			g.tip.Point(g.slot(gs, i), ui.TipLine(p.Name), ui.TipLines(runeTipLines(gs, p)))
		}
		return
	}
}

// click is what this dialog does with a press. **A click on a card is the only input it takes
// itself** — there is no confirm, because the purchase has already happened and what is left is
// which one. The screen around it owns SKIP.
func (g *goods) click(gs *state.GlobalState) {
	at := image.Pt(gs.MouseX, gs.MouseY)

	// **The offer row first**, because it is the row drawn in front: a selected card is lifted and
	// a lifted card overlaps nothing above it, but reading the rows in drawing order is the rule
	// every screen here follows.
	if i := ui.HoveredSeat(at, len(g.offer), func(i int) image.Rectangle {
		return g.offerSlot(gs, i)
	}); i >= 0 {
		g.selectCard(gs, i)
		return
	}

	if i := ui.HoveredSeat(at, g.count(), func(i int) image.Rectangle {
		return g.slot(gs, i)
	}); i >= 0 {
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
		if !g.essenceSpendable(gs, essence) {
			return
		}

		gs.Journal.Write(journal.Record{
			Kind:    journal.KindTake,
			Key:     essence.Record,
			Seat:    i,
			Targets: deckCardIDs(gs, g.selectedDeckIndexes()),
		})
		g.show(gs, essence, g.selectedSlots())

	case session.ContentsStones:
		stone := g.stones[i]
		if gs.Run.UseStone(stone.Record) {
			gs.Journal.Write(journal.Record{Kind: journal.KindTake, Key: stone.Record, Seat: i})
			trace.Logf("shop", "bag of rocks: %s, shape %s now at %d stones",
				stone.Record, stone.Shape, gs.Run.StonesOn(stone.Hands()[0]))
		}
		g.reset()

	case session.ContentsRunes:
		// **A rune is not applied here — it goes into the sack.** That is the whole
		// difference between this good and the other two: a stone and an essence are spent on the
		// spot, and a rune is carried into the next fight and spent between its turns.
		p := g.runes[i]
		if gs.Run.Hold(p.Record) {
			gs.Journal.Write(journal.Record{Kind: journal.KindTake, Key: p.Record, Seat: i})
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
func (g *goods) show(gs *state.GlobalState, essence session.Essence, slots []int) {
	if gs.Run == nil || len(slots) == 0 {
		return
	}

	at := make([]int, 0, len(slots))
	from := make([]image.Rectangle, 0, len(slots))
	ids := make([]int, 0, len(slots))
	for _, slot := range slots {
		if slot < 0 || slot >= len(g.offer) {
			return
		}
		card, ok := gs.Run.Card(g.offer[slot])
		if !ok {
			return
		}
		at = append(at, g.offer[slot])
		from = append(from, g.offerSlot(gs, slot))
		ids = append(ids, card.ID)
	}

	lands, ok := previewEssence(gs, essence, at, from)
	if !ok {
		return
	}

	g.lands = lands
	g.removes = essence.Target == session.TargetRemove
	g.copied = essence.Target == session.TargetDuplicate

	g.stage, g.held = goodsShowing, settledHoldTicks()
	g.arrival, g.arrivedFrom = ui.NewTravel(0, settleFlightTicks()), from[0]

	// **The commitment is by identity, not by position**, the reward screen's rule: an essence may
	// take several cards, and removing one moves every deck position above it.
	g.applyNow = func(run *session.Session) { run.ApplyToAll(essence, ids) }
	g.tip.Forget()

	trace.Logf("shop", "vial of essence: %s aimed at deck positions %v", essence.Record, at)
}

// tickShowing runs the held picture: the flight, then the change, then the hold, and the deck edit
// at the end of it.
//
// **The change does not start until the card has landed**, and the hold does not start until the
// change has finished — the reward screen's ordering, and for its reason: a dissolve running over a
// moving card puts the one thing worth watching on a target the eye is still chasing.
func (g *goods) tickShowing(gs *state.GlobalState) {
	if !g.arrival.Done() {
		g.arrival.Tick()
		return
	}
	if !tickLandings(g.lands) {
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
			for _, l := range g.lands {
				earnMoment(gs, achieve.CardAltered(l.after.Label()))
			}
		}
	}
	g.reset()
}

// selectCard picks a card out of the offer row, or puts it back. **Clicking the selected card
// deselects it**, the hand row's own gesture, so the thing a player already knows how to undo works
// here too.
//
// **A full selection replaces its oldest card rather than refusing the click** — the reward
// screen's rule, written up in PostBattleScene.selectOffered: with one target it reads as the pick
// moving, which is exactly what it always did.
func (g *goods) selectCard(gs *state.GlobalState, i int) {
	for k, sel := range g.selected {
		if sel == i {
			g.selected = append(g.selected[:k], g.selected[k+1:]...)
			g.tip.Forget()
			return
		}
	}

	if n := g.targets(gs); len(g.selected) >= n {
		g.selected = append([]int(nil), g.selected[len(g.selected)-n+1:]...)
	}
	g.selected = append(g.selected, i)
	g.tip.Forget()
}

// drawCards puts the good's own rows up: what was drawn from it, and — for a vial — the cards an
// essence may be aimed at.
//
// **The screen around it is the scene's** *(2026-09-19)*. This was a modal frame with a title bar
// and a scrim; the ground, the build band, the heading and the draw pile are GoodsScene's now, and
// what is left here is the thing the good actually is.
func (g *goods) drawCards(gs *state.GlobalState, screen *ebiten.Image) {
	if !g.openNow() || gs.Run == nil {
		return
	}

	if g.stage == goodsShowing {
		g.drawShowing(gs, screen)
		return
	}

	for i := 0; i < g.count(); i++ {
		at := g.slot(gs, i).Min
		switch g.good.Contains {
		case session.ContentsStones:
			ui.DrawStoneCard(gs, screen, at, g.stones[i], true)
		case session.ContentsRunes:
			ui.DrawRuneCard(gs, screen, at, g.runes[i], true, false)
		default:
			// **An essence is lit only for the cards that are selected.** With nothing selected the
			// whole row is dim, which is what says the gesture starts underneath — the reward
			// screen's rule, and the rune pane's.
			ui.DrawEssenceCard(gs, screen, at, g.essences[i], g.essenceSpendable(gs, g.essences[i]))
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
		ui.DrawCard(gs, screen, g.offerSlot(gs, i).Min, cards.Hand, card, ui.HeldByRun(gs, card),
			true, g.isSelected(i))
	}
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
	drawLandings(gs, screen, g.lands, g.arrival, g.copied)
}

// hint is the line under the title: the record's own, plus how far an essence reaches when a relic
// has widened it.
//
// **The reach is said in words because nothing else on the screen says it** — the row lights the
// essences only once it has enough cards, which says when and never how many. "Up to", because the
// reach is a ceiling rather than a quota; see consumableTarget.fewest.
func (g *goods) hint(gs *state.GlobalState) string {
	line := g.good.Hint
	if g.good.Contains != session.ContentsEssences {
		return line
	}
	if n := g.targets(gs); n > 1 {
		return fmt.Sprintf("%s - up to %d cards each", line, n)
	}
	return line
}

// stoneTipLines is what a stone's tooltip says: one line per rung it raises, with what the stone
// adds to that rung and where the rung stands for this run right now.
//
// **One line per rung because the figures differ.** A stone raises every rung of its shape by a
// tenth of that rung's *own* multiplier, so a Card Three of a Kind and a Form Three of a Kind move
// by different amounts off the same rock, and a single "+N" would be true of neither.
//
// **The run's own figure rather than the catalog's**, because a second stone on a rung is worth
// exactly what the first was and the player has no other way to see what the first one did.
func stoneTipLines(gs *state.GlobalState, st session.Stone) []string {
	var out []string
	for _, hand := range st.Hands() {
		worth := session.StoneWorth(hand)
		line := fmt.Sprintf("%s +%d", stoneHandName(hand), worth)
		if gs.Run != nil {
			if now, ok := gs.Run.HandMultiplier(hand); ok {
				line = fmt.Sprintf("%s: x%d to x%d", stoneHandName(hand), now, now+worth)
			}
		}
		out = append(out, line)
	}
	if gs.Run != nil && len(st.Hands()) > 0 {
		if n := gs.Run.StonesOn(st.Hands()[0]); n > 0 {
			out = append(out, fmt.Sprintf("%d already on these hands", n))
		}
	}
	return out
}

// stoneHandName is the rung a stone raises, by the name the rest of the game calls it.
//
// **Read off the ladder rather than off the stone**: `data/stones.json` names a shape and
// `hands.json` owns what each rung of it is called, so a renamed rung cannot leave a stone's
// tooltip saying the old one. The key itself is the readable failure for a rung that is not there.
func stoneHandName(key string) string {
	for _, h := range combat.Hands() {
		if h.Key == key {
			return h.Name
		}
	}
	return key
}
