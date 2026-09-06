package screens

// Which sealed packs a visit puts up, and the two reroll buttons.
//
// **A visit offers two of the three** *(owner's call, 2026-09-06)*. The bag, the can and the bucket
// were all on the shelf every time, which meant the shop asked no question about them: three seats,
// three goods, and the only decision was what the purse could cover. Two of three makes a visit
// something to read — and it is what buys the room the potions pane needed on a row that was
// already too long. All three catalogues stay reachable; which two you meet is the roll.
//
// **Its own stream, `seeds.PackOffer`.** Sharing the ring shelf's would make authoring a ring change
// which packs every run was ever offered, and would reroll the packs every time the rings were
// rerolled — see internal/seeds, where the argument is written down beside the other four.
//
// **Rerolls are per pane and they escalate** *(owner's call, 2026-09-06)*: 2 vitae, then 4, then 8,
// doubling within a visit and starting again at the next shop. The rings and the packs each have
// their own button and their own count, so pressing one does not make the other dearer. The potions
// and the brand have no button at all — the potion pane is the whole catalogue every visit, so a
// reroll would offer what is already offered, and the brand is one seat.
//
// **A reroll advances the visit's cursor rather than seeding a second stream.** The scene holds the
// two `*rand.Rand`s from Init and every deal draws from them, which is what keeps a replayed run
// exact however many times the button is pressed.

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
)

// packsOffered is how many sealed packs stand on the shelf at once.
const packsOffered = 2

// rerollBase is what the first reroll of a pane costs, and every one after it doubles.
//
// **A placeholder in the same sense the brand's price is** — it was 2 on the design canvas and
// nothing has been tuned against it. The doubling is the shape of the decision; the 2 is a number
// to move.
const rerollBase = 2

// rerollPrice is what the next reroll of a pane costs: the base doubled once per reroll already
// taken this visit.
//
// **Computed rather than stored**, so the figure on the button and the figure charged cannot
// disagree — the split `goodPrice` already makes between a face and a purse.
func (s *ShopScene) rerollPrice(p shopPane) int {
	price := rerollBase
	for i := 0; i < s.rerolls[p]; i++ {
		price *= 2
	}
	return price
}

// rerollable is whether a pane has a button under it at all.
func rerollable(p shopPane) bool { return p == shopPanePacks || p == shopPaneRings }

// shopRNG is one of the visit's two streams, seeded per fight.
func shopRNG(gs *state.GlobalState, stream seeds.Stream) *rand.Rand {
	if gs.Run == nil {
		return nil
	}
	return rand.New(rand.NewSource(seeds.ForFight(gs.RunSeed, stream, gs.Run.Fight())))
}

// dealPacks picks which two of the three stand on the shelf.
//
// **A shuffle of the catalogue, cut to two** rather than two draws without replacement, because the
// three are equally weighted: there is no rarity here, so a weighted draw would be the ring shelf's
// machinery doing nothing. The order is the roll's, so the bag is not always on the left.
func dealPacks(rng *rand.Rand) []goodKind {
	kinds := goodKinds()
	if rng == nil {
		return kinds[:packsOffered]
	}

	rng.Shuffle(len(kinds), func(i, j int) { kinds[i], kinds[j] = kinds[j], kinds[i] })
	return kinds[:packsOffered]
}

// rerollPacks redraws whichever pack seats have not been opened.
//
// **An opened seat is left as it is**, which is the shelf's rule everywhere: what has been bought is
// gone and its seat stays empty, so nothing moves under the hand of a player who is mid-decision.
// A pane with nothing left to redraw refuses the press rather than charging for it.
func (s *ShopScene) rerollPacks(gs *state.GlobalState) {
	fresh := dealPacks(s.packRNG)

	// Keep the kinds already opened where they stand, and fill the rest from the fresh pair,
	// skipping anything already on the shelf so the two seats are never the same pack twice.
	out := make([]goodKind, len(s.offered))
	taken := map[goodKind]bool{}
	for i, kind := range s.offered {
		if s.goodTaken(kind) {
			out[i] = kind
			taken[kind] = true
		}
	}

	next := 0
	for i := range out {
		if out[i] != goodNone {
			continue
		}
		for next < len(fresh) && taken[fresh[next]] {
			next++
		}
		if next >= len(fresh) {
			// Nothing left that is not already standing: keep what was there.
			out[i] = s.offered[i]
			taken[out[i]] = true
			continue
		}
		out[i] = fresh[next]
		taken[fresh[next]] = true
		next++
	}

	s.offered = out
}

// rerollRings redraws the whole shelf, leaving bought seats empty.
//
// **A bought ring does not come back.** The seat it left is spent for the visit, exactly as it is
// after a purchase — a reroll that refilled it would let one reroll be worth three rings.
func (s *ShopScene) rerollRings(gs *state.GlobalState) {
	fresh := dealShelf(gs, s.stockRNG)

	for i := range s.shelf {
		if s.shelf[i].bought || i >= len(fresh) {
			continue
		}
		s.shelf[i] = fresh[i]
	}
}

// paneHasSomethingToReroll is whether a press would change anything. A pane whose every seat is
// spent has nothing to redraw, and taking vitae for that would be the shop selling nothing.
func (s *ShopScene) paneHasSomethingToReroll(p shopPane) bool {
	switch p {
	case shopPaneRings:
		return s.anyLeft()
	case shopPanePacks:
		for _, kind := range s.offered {
			if !s.goodTaken(kind) {
				return true
			}
		}
	}
	return false
}

// canReroll is whether a pane's button is live: it has something to redraw and the purse covers the
// next price.
func (s *ShopScene) canReroll(gs *state.GlobalState, p shopPane) bool {
	if gs.Run == nil || !rerollable(p) || !s.paneHasSomethingToReroll(p) {
		return false
	}
	return gs.Run.Vitae() >= s.rerollPrice(p)
}

// reroll pays for a redraw and takes it.
//
// **The purse moves first**, the rule every purchase on this screen is under: `SpendVitae` is the
// one place a refusal happens, so a reroll that could not be paid for changes nothing.
func (s *ShopScene) reroll(gs *state.GlobalState, p shopPane) {
	if !s.canReroll(gs, p) {
		return
	}
	price := s.rerollPrice(p)
	if !gs.Run.SpendVitae(price) {
		return
	}

	s.rerolls[p]++
	switch p {
	case shopPaneRings:
		s.rerollRings(gs)
	case shopPanePacks:
		s.rerollPacks(gs)
	}
	s.armed = ""
	s.tip.Forget()

	trace.Logf("shop", "rerolled %s for %d, next costs %d, %d vitae left",
		paneName(p), price, s.rerollPrice(p), gs.Run.Vitae())
}

func paneName(p shopPane) string {
	if p == shopPaneRings {
		return "rings"
	}
	return "packs"
}

// initRerollButtons builds the two, once. **The crimson is not theirs** — a reroll is not a thing
// that cannot be taken back, so it wears the Leave button's slate rather than the sell tab's red.
func (s *ShopScene) initRerollButtons() {
	if s.ringReroll != nil {
		return
	}
	// `build` rather than `make`, which is a builtin worth not shadowing.
	build := func(p shopPane) *models.Button {
		b := models.NewButton(shopRerollWidth, shopRerollHeight, "", func() {
			s.rerolling, s.rerollNow = p, true
		})
		b.BaseColor = color.RGBA{R: 120, G: 132, B: 150, A: 255}
		b.TextSize = shopRerollText
		return b
	}
	s.ringReroll = build(shopPaneRings)
	s.packReroll = build(shopPanePacks)
}

// updateRerollButtons positions and runs them, and takes the press the previous frame recorded.
func (s *ShopScene) updateRerollButtons(gs *state.GlobalState) {
	if s.rerollNow {
		s.rerollNow = false
		s.reroll(gs, s.rerolling)
		return
	}

	for _, p := range []shopPane{shopPaneRings, shopPanePacks} {
		b := s.rerollButton(p)
		at := shopRerollRect(gs, p)
		b.ScreenX, b.ScreenY = (at.Min.X+at.Max.X)/2, (at.Min.Y+at.Max.Y)/2
		b.Text = fmt.Sprintf("REROLL %d", s.rerollPrice(p))
		setEnabled(b, s.canReroll(gs, p))
		systems.UpdateButton(gs, b)
	}
}

func (s *ShopScene) drawRerollButtons(gs *state.GlobalState, screen *ebiten.Image) {
	systems.DrawButton(gs, screen, s.ringReroll)
	systems.DrawButton(gs, screen, s.packReroll)
}

func (s *ShopScene) rerollButton(p shopPane) *models.Button {
	if p == shopPaneRings {
		return s.ringReroll
	}
	return s.packReroll
}
