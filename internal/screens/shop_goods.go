package screens

// The two sealed goods, and the dialog that opens them.
//
// **A bag of rocks and a can of worms** *(owner's call, 2026-08-27)*. Both cost five vitae, both
// hold four of something, and both give the player exactly one of the four — the other three are
// gone. What is bought is the *choice*, which is what makes them different from a ring on the
// shelf: a ring is a thing you read and then pay for, and these are paid for and then read.
//
// **The bag is the only way a stone is ever got.** A stone raises one rung of the hand ladder by a
// tenth of its catalogue multiplier, for the rest of the run — see `internal/session/stone.go`.
// Choosing it is using it: there is no inventory, so the click that picks a rock is the click that
// puts it on the ladder.
//
// **The can is a worm, on the same terms as the reward screen's** — and that includes the gesture:
// the four worms and a hand's worth of cards are up together, and you select the card first and
// click the worm second *(owner's call, 2026-09-06)*. It was two stages until then, and what was
// wrong with them is that the worms left the screen at the moment the player had to judge one
// against a card. See targeting.go for the rule, and the reward screen, which merged its own two
// stages the same day. It is worth five vitae over a free offer of two because four is twice the
// choice and because it arrives at the shop rather than at the end of a fight, which is a
// different moment to want one at.
//
// **The can's title is an instruction and its hint is the gesture**, because a dialog whose two
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

// goodKind is which of the two sealed goods a shelf seat is.
type goodKind int

const (
	goodNone goodKind = iota
	goodBag
	goodCan

	// goodBucket is the third, and the one whose contents leave the shop with the player rather
	// than being applied in the dialog. A stone and a worm are both spent the moment they are
	// chosen; a parasite goes into the bucket and is spent mid-fight. See combat_parasite.go.
	goodBucket
)

// goodKinds is the shelf's goods in the order they stand, for anything that walks them.
func goodKinds() []goodKind { return []goodKind{goodBag, goodCan, goodBucket} }

// The two goods as cards. **The name is what it is and the line is the shape of the offer**, never
// what is inside: a bag that named its four rocks on the face would be a shelf item you could read
// before paying for, which is the one thing these are not.
const (
	bagName    = "BAG OF ROCKS"
	canName    = "CAN OF WORMS"
	bucketName = "BUCKET OF PARASITES"
)

// Where the goods row sits, and how the dialog lays its cards out.
const (
	// The dialog's four cards, centred in the panel — the bag's and the bucket's, which have one
	// row and nothing under it.
	goodsChoiceRowPct = 42

	// The can has two rows and they both have to fit inside the panel, so its worms sit high and
	// the cards they may eat sit under them. **Not 42 and 70** *(2026-09-06)*: the offer row ended
	// exactly on the panel's bottom edge, which was survivable while it was the only row on screen
	// and is not now that a selected card is lifted into the row above it.
	canWormRowPct    = 22
	goodsOfferRowPct = 55

	// The line under the title, saying what to do with what is up.
	goodsHintTop = 120
)

// goodsStage is how far through opening a good the player is.
type goodsStage int

const (
	// goodsClosed: nothing is open, and the shop is the shop.
	goodsClosed goodsStage = iota

	// goodsPick: what was drawn is up, and a click takes one. **There is no second stage**
	// *(owner's call, 2026-09-06)*: the can used to move on to a row of cards once a worm was
	// chosen, so the worms disappeared at the moment the player had to judge one against a card.
	// Both rows are up together now and the gesture is the row's own — select the card, then click
	// the worm. See targeting.go, which is the rule the parasite pane and the reward screen's own
	// worm row already follow.
	goodsPick
)

// goods is the dialog: which good was opened, what was drawn from it, and how far through the
// player is.
//
// **It holds no purse and no run.** The vitae is taken by the shop the moment a good is clicked —
// see `ShopScene.openGood` — so what is here is the offer and the choice, exactly as the shelf
// holds a ring's key and not its price.
type goods struct {
	kind  goodKind
	stage goodsStage

	// stones, worms and parasites are what was drawn, and only the one matching kind is filled.
	stones    []session.Stone
	worms     []session.Worm
	parasites []session.Parasite

	// offer is the cards a worm may be aimed at, by index into the run's deck. **Only the can
	// fills it**, and it is dealt when the can is opened rather than when a worm is picked — the
	// two rows are up at once, so the cards cannot be a function of a choice not yet made.
	offer []int

	// selected is which offered card is picked out, or -1. **One card**, because a worm eats
	// exactly one — the reward screen's own field, and the same reason it is an index rather than
	// a set. See consumableTarget.
	selected int

	// tip explains whichever card the cursor is resting on.
	tip models.Tooltip
}

// open puts a good up, drawing what is inside it.
//
// **The contents are drawn here rather than by the run**, from a per-fight stream of their own, so
// a purchase interrupted by a quit leaves nothing to snapshot. Buying the same bag twice in one
// visit is not possible — see the shelf's `bought` flag — so a stream per fight is a stream per
// bag.
func (g *goods) open(gs *state.GlobalState, kind goodKind) {
	g.kind, g.stage, g.selected = kind, goodsPick, -1
	g.stones, g.worms, g.parasites, g.offer = nil, nil, nil, nil
	g.tip = models.Tooltip{DwellTicks: tipDwell}

	switch kind {
	case goodBag:
		g.stones = dealStones(gs)
	case goodCan:
		g.worms = dealCanWorms(gs)
		g.offer = dealCanOffer(gs)
	case goodBucket:
		g.parasites = dealBucketParasites(gs)
	}
}

// openBag reports whether anything is up.
func (g *goods) openNow() bool { return g.stage != goodsClosed }

// close puts it away.
func (g *goods) reset() {
	g.kind, g.stage, g.selected = goodNone, goodsClosed, -1
	g.stones, g.worms, g.parasites, g.offer = nil, nil, nil, nil
	g.tip.Forget()
}

// count is how many cards are in the row that is taken from: the stones, the parasites, or the
// worms. **Not the offer**, which is a second row with its own slot function.
func (g *goods) count() int {
	switch {
	case g.kind == goodBag:
		return len(g.stones)
	case g.kind == goodBucket:
		return len(g.parasites)
	default:
		return len(g.worms)
	}
}

// dealStones is what a bag holds: four of the catalogue, without repeats.
//
// **Its own stream** (`seeds.BagStock`), separate from the shelf's rings and from both worm draws —
// see internal/seeds, where the argument is written down. **Without repeats**, because a bag
// offering the same rock twice is a seat spent saying nothing, exactly as the shelf is.
//
// **Flat, not weighted.** A stone has no rarity: every rung is worth a tenth of itself, so a
// Card Five stone is not a better rock than a Card Pair stone — it is a rock for a rung you may
// never build. Weighting them would be pricing the *hand*, which the ladder already does.
func dealStones(gs *state.GlobalState) []session.Stone {
	all := session.Stones()
	rng := rand.New(rand.NewSource(seeds.ForFight(gs.RunSeed, seeds.BagStock, gs.Run.Fight())))
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	if len(all) > session.BagSize() {
		all = all[:session.BagSize()]
	}
	return all
}

// dealCanWorms is what a can holds: four worms, without repeats.
//
// **A different stream from the reward screen's two** (`seeds.CanStock`), which is the case the
// salts exist for: sharing would make the shop's four a function of which two had just been
// offered free, so buying the can could guarantee — or rule out — the pair the player had turned
// down. See internal/seeds.
func dealCanWorms(gs *state.GlobalState) []session.Worm {
	all := session.Worms()
	rng := rand.New(rand.NewSource(seeds.ForFight(gs.RunSeed, seeds.CanStock, gs.Run.Fight())))
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	if len(all) > session.CanSize() {
		all = all[:session.CanSize()]
	}
	return all
}

// dealBucketParasites is what a bucket holds: four of the catalogue, without repeats.
//
// **Its own stream** (`seeds.BucketStock`), separate from both other goods and from the reward
// screen's worms — see internal/seeds. **Flat, not weighted**, on the bag's argument: a parasite
// has no rarity, and weighting them would be pricing the effect, which nothing has decided yet.
//
// **A catalogue shorter than the bucket is not an error.** Four parasites ship and the bucket holds
// four, so it currently offers the whole file; the cut is what keeps that true as the list grows.
func dealBucketParasites(gs *state.GlobalState) []session.Parasite {
	all := session.Parasites()
	rng := rand.New(rand.NewSource(seeds.ForFight(gs.RunSeed, seeds.BucketStock, gs.Run.Fight())))
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	if len(all) > session.BucketSize() {
		all = all[:session.BucketSize()]
	}
	return all
}

// parasiteTipLines is what resting on a parasite says: what it does, and when it can be spent.
//
// **The "when" is the half the card cannot say.** A parasite's face is a name and a clipped line,
// and the thing that makes it a different object from a worm is not on it — so the tooltip is where
// a player finds out that this one is carried into a fight rather than used now.
func parasiteTipLines(p session.Parasite) []string {
	// **The card's own text, unwrapped.** A `\n` on a face is an authored line break and a tooltip
	// draws its own lines, so the two are the same sentence written for two widths — the same
	// treatment wormTip gives a worm.
	lines := strings.Split(p.Text, "\n")
	return append(lines, "spent between the turns of a fight")
}

// dealCanOffer is the hand the can deals beside its worms: a shuffle of every position in the run's
// deck, cut to a hand's worth.
//
// **It is not filtered by a worm any more** *(owner's call, 2026-09-06)*. It could not be: both rows
// are up at once now, so the cards are dealt before the player has chosen anything, and a filter
// would have to know which worm. What replaces it is the *worm* going dim — a worm that cannot
// change the selected card is unclickable, which is the same information at the other end of the
// gesture, and the reward screen's own rule. See wormSpendable.
func dealCanOffer(gs *state.GlobalState) []int {
	if gs.Run == nil || gs.Run.Size() == 0 {
		return nil
	}

	idx := make([]int, 0, gs.Run.Size())
	for i := 0; i < gs.Run.Size(); i++ {
		idx = append(idx, i)
	}

	rng := rand.New(rand.NewSource(seeds.ForFight(gs.RunSeed, seeds.CanStock, gs.Run.Fight())))
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
	if g.kind == goodCan {
		top = gs.PctY(canWormRowPct)
	}
	pitch := cardWidth + 40

	width := (n-1)*pitch + cardWidth
	left := gs.PctX(50) - width/2
	return image.Rect(left+i*pitch, top, left+i*pitch+cardWidth, top+cardHeight)
}

// offerSlot is where one of the cards a worm may eat is drawn and clicked. **The hand's own
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

// wormSpendable is whether clicking this worm now would take it: a card is selected, and this worm
// can actually change that card.
//
// **The same question the click asks and the same one the card's lit state reads**, which is what
// stops a worm looking available and doing nothing. It is the reward screen's predicate over the
// same target rule — see targeting.go.
func (g *goods) wormSpendable(gs *state.GlobalState, w session.Worm) bool {
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
		g.tip.Point(seat, title, lines)
		return
	}

	for i := 0; i < g.count(); i++ {
		if !at.In(g.slot(gs, i)) {
			continue
		}
		switch {
		case g.kind == goodCan:
			// **The worms are deliberately not tooltipped**, which is the reward screen's own
			// choice on the same row: a worm's whole rule is printed on its face, where a deck
			// card's is not. What a dim worm means — "not for the card you have selected" — is
			// left to the row rather than to a tooltip.
			return
		case g.kind == goodBag:
			st := g.stones[i]
			g.tip.Point(g.slot(gs, i), st.Name, stoneTipLines(gs, st))
		case g.kind == goodBucket:
			p := g.parasites[i]
			g.tip.Point(g.slot(gs, i), p.Name, parasiteTipLines(p))
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
	switch {
	case g.kind == goodCan:
		// **A worm is refused rather than falling back**, on the predicate its lit state already
		// read: a dim worm cannot be taken and a lit one always works. With no card selected every
		// worm is dim, so the dialog waits rather than choosing a card for the player.
		worm := g.worms[i]
		idx, ok := g.selectedDeckIndex()
		if !ok || !g.wormSpendable(gs, worm) {
			return
		}

		if gs.Run.Apply(worm, idx) {
			trace.Logf("shop", "can of worms: %s applied to deck position %d", worm.Record, idx)

			// **The same moment the post-battle screen raises**, because it is the same event: a
			// card in the run's deck is now a different card. Read back out of the deck rather than
			// predicted, and skipped for a removal, which leaves no card to name.
			if worm.Target != session.TargetRemove {
				if card, ok := gs.Run.Card(idx); ok {
					earnMoment(gs, achieve.CardAltered(card.Label()))
				}
			}
		}
		g.reset()

	case g.kind == goodBag:
		stone := g.stones[i]
		if gs.Run.UseStone(stone.Record) {
			trace.Logf("shop", "bag of rocks: %s, %s now at %d stones",
				stone.Record, stone.Hand, gs.Run.StonesOn(stone.Hand))
		}
		g.reset()

	case g.kind == goodBucket:
		// **A parasite is not applied here — it goes into the bucket.** That is the whole
		// difference between this good and the other two: a stone and a worm are spent on the
		// spot, and a parasite is carried into the next fight and spent between its turns.
		p := g.parasites[i]
		if gs.Run.Hold(p.Record) {
			trace.Logf("shop", "bucket of parasites: %s held, %d in the bucket",
				p.Record, gs.Run.HoldCount())
		}
		g.reset()
	}
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

	if line := g.hint(); line != "" {
		hint := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 20}
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(panel.Min.X+panel.Dx()/2), float64(panel.Min.Y+goodsHintTop))
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(color.RGBA{R: 226, G: 228, B: 236, A: 255})
		text.Draw(screen, line, hint, op)
	}

	for i := 0; i < g.count(); i++ {
		at := g.slot(gs, i).Min
		switch {
		case g.kind == goodBag:
			drawStoneCard(gs, screen, at, g.stones[i], true)
		case g.kind == goodBucket:
			drawSpecCard(gs, screen, at, parasiteSpec(gs, g.parasites[i], true, false))
		default:
			// **A worm is lit only for the card that is selected.** With nothing selected the
			// whole row is dim, which is what says the gesture starts underneath — the reward
			// screen's rule, and the parasite pane's.
			drawWormCard(gs, screen, at, g.worms[i], g.wormSpendable(gs, g.worms[i]))
		}
	}

	// **The cards the worms may eat, under them and up at the same time** *(owner's call,
	// 2026-09-06)*. They are drawn after the worms so a lifted card is in front of the row above
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
// **The can is the exception and says one thing** *(owner's call, 2026-09-05)*: the worms on the
// table are the whole of what the dialog is, so it is an instruction rather than a label with a
// caption. hint returns nothing there and draw prints nothing rather than an empty line.
func (g *goods) title() string {
	switch g.kind {
	case goodBag:
		return bagName
	case goodBucket:
		return bucketName
	default:
		return "CHOOSE YOUR WORM"
	}
}

func (g *goods) hint() string {
	switch {
	case g.kind == goodCan:
		return "pick the card, then the worm that eats it"
	case g.kind == goodBag:
		return fmt.Sprintf("take one of the %d, the rest are gone", len(g.stones))
	default:
		return fmt.Sprintf("take one of the %d, the rest are gone", len(g.parasites))
	}
}

// stoneTipLines is what a stone's tooltip says: the rung it raises, what one is worth, and where
// that rung stands for this run right now.
//
// **The run's own figure rather than the catalogue's**, because a second stone on a rung is worth
// exactly what the first was and the player has no other way to see what the first one did.
func stoneTipLines(gs *state.GlobalState, st session.Stone) []string {
	worth := session.StoneWorth(st.Hand)

	out := []string{fmt.Sprintf("raises this hand by %d", worth)}
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
