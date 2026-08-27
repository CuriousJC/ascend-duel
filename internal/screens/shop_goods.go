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
// **The can is a worm, on the same terms as the reward screen's** — pick one of four, then pick
// the card it eats. It is worth five vitae over a free offer of two because four is twice the
// choice and because it arrives at the shop rather than at the end of a fight, which is a
// different moment to want one at.
//
// **This dialog has no X, and that is deliberate.** Every other modal in the game is a look at
// something and closes without consequence; this one stands between a purchase and what it bought,
// so an exit that forfeited five vitae would be a trap wearing the same red square that means
// "close" everywhere else. Every card in it is an exit — the dialog ends when one is chosen — and
// the second stage cannot be reached without the first.

import (
	"fmt"
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
	bagName    = "Bag of Rocks"
	canName    = "Can of Worms"
	bucketName = "Bucket of Parasites"
)

// Where the goods row sits, and how the dialog lays its cards out.
const (
	// The dialog's four cards, and the deck row the can's second stage offers.
	goodsChoiceRowPct = 42
	goodsOfferRowPct  = 70

	// The line under the title, saying what to do with what is up.
	goodsHintTop = 120
)

// goodsStage is how far through opening a good the player is.
type goodsStage int

const (
	// goodsClosed: nothing is open, and the shop is the shop.
	goodsClosed goodsStage = iota

	// goodsPick: the four are up, and one click ends it — or, for the can, moves it on.
	goodsPick

	// goodsAim: a worm is chosen and the cards it may eat are dealt. **Only the can reaches
	// here**; a stone has nothing to aim at, since the rung it raises is written on its face.
	goodsAim
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

	// chosen is which of the four was taken, for the can's second stage. -1 before one is.
	chosen int

	// offer is the deck positions the chosen worm may be applied to, by index into the run's deck.
	// **Re-dealt rather than held across a purchase**, because a worm removes a card and every
	// index after it moves — the same rule the reward screen's offer is under.
	offer []int

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
	g.kind, g.stage, g.chosen = kind, goodsPick, -1
	g.stones, g.worms, g.parasites, g.offer = nil, nil, nil, nil
	g.tip = models.Tooltip{DwellTicks: tipDwell}

	switch kind {
	case goodBag:
		g.stones = dealStones(gs)
	case goodCan:
		g.worms = dealCanWorms(gs)
	case goodBucket:
		g.parasites = dealBucketParasites(gs)
	}
}

// openBag reports whether anything is up.
func (g *goods) openNow() bool { return g.stage != goodsClosed }

// close puts it away.
func (g *goods) reset() {
	g.kind, g.stage, g.chosen = goodNone, goodsClosed, -1
	g.stones, g.worms, g.parasites, g.offer = nil, nil, nil, nil
	g.tip.Forget()
}

// count is how many cards the current stage is showing.
func (g *goods) count() int {
	switch {
	case g.stage == goodsAim:
		return len(g.offer)
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

// dealCanOffer is which cards the chosen worm may be applied to: a shuffle of every position in
// the run's deck, cut to a hand's worth.
//
// **Only the cards the worm would actually change.** `CanApply` is asked before the offer is cut,
// which is the reward screen's rule — a Promote worm pointed at a row of Smashes is a reward that
// does nothing, and dimming four cards the player cannot click is worse than not offering them.
func dealCanOffer(gs *state.GlobalState, w session.Worm) []int {
	if gs.Run == nil || gs.Run.Size() == 0 {
		return nil
	}

	idx := make([]int, 0, gs.Run.Size())
	for i := 0; i < gs.Run.Size(); i++ {
		if gs.Run.CanApply(w, i) {
			idx = append(idx, i)
		}
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
	pitch := cardWidth + 40
	if g.stage == goodsAim {
		top = gs.PctY(goodsOfferRowPct)
		pitch = handPitch(gs, n)
	}

	width := (n-1)*pitch + cardWidth
	left := gs.PctX(50) - width/2
	return image.Rect(left+i*pitch, top, left+i*pitch+cardWidth, top+cardHeight)
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

	for i := 0; i < g.count(); i++ {
		if !at.In(g.slot(gs, i)) {
			continue
		}
		switch {
		case g.stage == goodsAim:
			card, ok := gs.Run.Card(g.offer[i])
			if !ok {
				return
			}
			title, lines := cardTip(card, heldByRun(gs, card))
			g.tip.Point(g.slot(gs, i), title, lines)
		case g.kind == goodBag:
			st := g.stones[i]
			g.tip.Point(g.slot(gs, i), st.Name, stoneTipLines(gs, st))
		case g.kind == goodBucket:
			p := g.parasites[i]
			g.tip.Point(g.slot(gs, i), p.Name, parasiteTipLines(p))
		default:
			title, lines := wormTip(g.worms[i])
			g.tip.Point(g.slot(gs, i), title, lines)
		}
		return
	}
}

// click is the whole of what this dialog does. **A click on a card is the only input it takes** —
// there is no confirm and no close, because the purchase has already happened and what is left is
// which one.
func (g *goods) click(gs *state.GlobalState) {
	at := image.Pt(gs.MouseX, gs.MouseY)

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
	case g.stage == goodsAim:
		worm := g.worms[g.chosen]
		if gs.Run.Apply(worm, g.offer[i]) {
			trace.Logf("shop", "can of worms: %s applied to deck position %d", worm.Record, g.offer[i])
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

	default:
		g.chosen = i
		g.offer = dealCanOffer(gs, g.worms[i])
		if len(g.offer) == 0 {
			// **A worm with nothing to eat ends the dialog rather than stranding it.** The offer
			// is filtered by `CanApply`, so an empty one means every card in the deck is already
			// what this worm would make it — which is a purchase that bought nothing, and a dialog
			// with no clickable card would be a lock-up on top of that.
			trace.Logf("shop", "can of worms: %s has nothing it can change", g.worms[i].Record)
			g.reset()
			return
		}
		g.stage = goodsAim
		g.tip.Forget()
	}
}

// draw puts the dialog up. It takes the shared modal frame, so it reads as the same kind of thing
// as the deck and hands panels — **minus the X**, for the reason at the top of this file.
func (g *goods) draw(gs *state.GlobalState, screen *ebiten.Image) {
	if !g.openNow() {
		return
	}

	panel := drawModalFrame(gs, screen, modalHead{title: g.title()})

	hint := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 20}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(panel.Min.X+panel.Dx()/2), float64(panel.Min.Y+goodsHintTop))
	op.PrimaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(color.RGBA{R: 226, G: 228, B: 236, A: 255})
	text.Draw(screen, g.hint(), hint, op)

	for i := 0; i < g.count(); i++ {
		at := g.slot(gs, i).Min
		switch {
		case g.stage == goodsAim:
			card, ok := gs.Run.Card(g.offer[i])
			if !ok {
				continue
			}
			drawCard(gs, screen, at, cards.Hand, card, heldByRun(gs, card), true, false)
		case g.kind == goodBag:
			drawStoneCard(gs, screen, at, g.stones[i], true)
		case g.kind == goodBucket:
			drawSpecCard(gs, screen, at, parasiteSpec(gs, g.parasites[i], true, false))
		default:
			drawWormCard(gs, screen, at, g.worms[i], true)
		}
	}

	systems.DrawTooltip(gs, screen, &g.tip)
}

// title names what is open, and hint says what to do with it. **Two short lines rather than a
// paragraph**: the cards say what they are, and this says how many of them the player gets.
func (g *goods) title() string {
	switch g.kind {
	case goodBag:
		return bagName
	case goodBucket:
		return bucketName
	default:
		return canName
	}
}

func (g *goods) hint() string {
	switch {
	case g.stage == goodsAim:
		return "and the card it changes"
	case g.kind == goodBag:
		return fmt.Sprintf("take one of the %d, the rest are gone", len(g.stones))
	case g.kind == goodBucket:
		return fmt.Sprintf("take one of the %d, the rest are gone", len(g.parasites))
	default:
		return fmt.Sprintf("take one of the %d, the rest are gone", len(g.worms))
	}
}

// stoneTipLines is what a stone's tooltip says: the rung it raises, what one is worth, and where
// that rung stands for this run right now.
//
// **The run's own figure rather than the catalogue's**, because a second stone on a rung is worth
// exactly what the first was and the player has no other way to see what the first one did.
func stoneTipLines(gs *state.GlobalState, st session.Stone) []string {
	worth := session.StoneWorth(st.Hand)

	out := []string{fmt.Sprintf("raises this hand by %d, for the rest of the run", worth)}
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
