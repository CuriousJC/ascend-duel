package screens

// The shop: **three rings on a shelf, five fingers, and one purse.**
//
// It is the second of the between-fight scenes — the worm, then this, then the room choice — and
// like the first it is an ordinary scene in the registry rather than a mode of anything. Nothing
// here names what comes next: the scene says it is finished and `advanceRun` decides where that
// leads. See flow.go.
//
// **It is what makes thirteen of the seventeen rings reachable.** The grammar has been built since
// 2026-08-17 and a run opened wearing three of them with no way to get a fourth, so most of the
// catalogue existed only in the file. What was missing was never the rules — `Session.Wear`, the
// purse and the `fight-won` accumulator were all already there — it was the screen.
//
// **Two rows, and they are the same object twice.** The shelf is what you can have and the row
// beneath is what you have; both are ring cards, both are clicked, and the difference is which
// direction the vitae moves. A shop built as a list with buttons would have made a ring a line of
// text on the one screen where it is a thing you are choosing to wear.
//
// **The rules of the trade live on the run, not here** — see session/shop.go. This file decides
// where a card is drawn and what a click means; what a ring costs, what it sells back for, and
// what happens to a growing ring's accumulator when it comes off are the run's business, and a
// screen holding a second opinion about any of them is the failure that separation prevents.

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"

	"github.com/curiousjc/ascend-duel/data"
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

// shelfSize is how many rings a visit puts up.
//
// **Three, matching the reward screen's row**, so the two between-fight screens read as one
// language: a short row of cards, and you take what you can afford. A shelf of seventeen would be
// a catalogue rather than an offer, and there would be no reason for the shop to come round again.
const shelfSize = 3

// Where the two rows sit. Percentages anchor the groups; offsets inside a group stay in pixels,
// per CLAUDE.md.
const (
	// The shelf sits where the eye lands and the worn row sits under it, in the band the reward
	// screen puts its deck cards in — so "these are yours" is below "these are for sale" in the
	// same place a card being altered sits below the worm altering it.
	shelfRowPct = 22
	wornRowPct  = 55

	// The figure under a card: what it costs on the shelf, what it pays back in the row.
	shopFigureGap  = 10
	shopFigureSize = 22

	// shopRowLabelGap is how far the row's own label sits above its cards.
	shopRowLabelGap  = 34
	shopRowLabelSize = 18
)

// shopMoveTicks is how long a ring takes to reach its new place — a bought one crossing to the
// finger it lands on, and every ring that shifts along when one is sold.
//
// **A proportion of the game's one speed**, like everything else that moves. See clock.go.
var shopMoveTicks = beat(1, 1)

// shelfItem is one ring on the shelf.
type shelfItem struct {
	key string

	// bought is set once it has been taken. **It stays in the row rather than being removed from
	// it**, the same choice the prize row makes: a card leaving would move the two beside it, and
	// the shelf you are reading must not rearrange itself under your hand. The seat is drawn empty.
	bought bool
}

// ShopScene sells rings and buys them back.
type ShopScene struct {
	// shelf is what this visit offers, drawn from the rings the run is not already wearing.
	shelf []shelfItem

	// leaveButton is the only control that is not a card. **There is no confirm and no basket** —
	// a click is a purchase, because the price is on the card and a run cannot go into debt, so
	// there is nothing a confirmation step would be protecting.
	leaveButton *models.Button

	// leaving is the button's request, consumed by Update. A button's OnClick reaches no global
	// state, and advancing the run needs it.
	leaving bool

	// from is where each worn ring was sitting before the last change, keyed by record, and move
	// is the one clock they all travel on.
	//
	// **Every ring in the row moves when one is bought or sold**, because the row is centred: the
	// seats themselves shift. So this is a map rather than a single mover, and it is *seats* being
	// remembered rather than journeys — the destination is recomputed from the layout every frame,
	// which is what lets a flight survive the window being resized. See travel.go.
	from map[string]image.Rectangle
	move travel

	// tip explains a ring: what it does, what it costs, and where it would sit in the firing order.
	// **The case the tooltip was built for** — a shelf offering Keen Ring says a name and a price
	// and nothing at all about slashes.
	tip models.Tooltip
}

// Init deals the shelf. **Re-entered on every visit**, because each fight earns its own.
func (s *ShopScene) Init(gs *state.GlobalState) {
	if s.leaveButton == nil {
		s.leaveButton = models.NewButton(offerButtonWidth, offerButtonHeight, "Leave",
			func() { s.leaving = true })
		s.leaveButton.BaseColor = color.RGBA{R: 120, G: 132, B: 150, A: 255}
	}

	s.leaving = false
	s.from, s.move = nil, travel{}
	s.tip = models.Tooltip{DwellTicks: tipDwell}
	s.shelf = dealShelf(gs)

	trace.Logf("shop", "after fight %d: %v for sale, %d vitae in hand, wearing %d",
		gs.Run.Fight(), shelfKeys(s.shelf), gs.Run.Vitae(), len(gs.Run.Worn()))
}

func shelfKeys(items []shelfItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.key)
	}
	return out
}

// dealShelf picks which rings are for sale: a shuffle of everything the run is not wearing, cut to
// three.
//
// **Its own stream** (`seeds.ShopStock`), and per fight — so a defeat and a retry walk into the
// same shop, exactly as they meet the same opponent. Sharing the worm offer's stream would have
// made authoring a worm change which rings every run was ever sold; see internal/seeds.
//
// **What is already worn is off the shelf**, rather than shown and refused. A ring on your hand
// offered back to you is a seat spent saying nothing, and `Buy` would turn the click down anyway.
func dealShelf(gs *state.GlobalState) []shelfItem {
	if gs.Run == nil {
		return nil
	}

	worn := make(map[string]bool, len(gs.Run.Worn()))
	for _, key := range gs.Run.Worn() {
		worn[key] = true
	}

	var pool []string
	for _, key := range session.Rings() {
		if !worn[key] {
			pool = append(pool, key)
		}
	}

	rng := rand.New(rand.NewSource(seeds.ForFight(gs.RunSeed, seeds.ShopStock, gs.Run.Fight())))
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	if len(pool) > shelfSize {
		pool = pool[:shelfSize]
	}

	out := make([]shelfItem, 0, len(pool))
	for _, key := range pool {
		out = append(out, shelfItem{key: key})
	}
	return out
}

func (s *ShopScene) Update(gs *state.GlobalState) error {
	s.move.tick()

	if s.leaving {
		s.leaving = false
		advanceRun(gs)
		return nil
	}

	s.click(gs)

	s.leaveButton.ScreenX, s.leaveButton.ScreenY = gs.PctX(50), gs.PctY(offerButtonsPct)
	systems.UpdateButton(gs, s.leaveButton)

	s.hover(gs)
	systems.UpdateTooltip(gs, &s.tip)
	return nil
}

// hover points the tooltip at whichever ring the cursor is resting on. **The shelf first, then the
// hand**, which is the order they are drawn and the order they are read.
func (s *ShopScene) hover(gs *state.GlobalState) {
	at := image.Pt(gs.MouseX, gs.MouseY)

	for i, item := range s.shelf {
		seat := s.shelfSlot(gs, i)
		if item.bought || !at.In(seat) {
			continue
		}
		if record, ok := gs.Rings[item.key]; ok {
			title, lines := shopRingTip(record)
			s.tip.Point(seat, title, lines)
		}
		return
	}

	worn := gs.Run.Worn()
	for i, key := range worn {
		seat := s.wornSlot(gs, i, len(worn))
		if !at.In(seat) {
			continue
		}
		if record, ok := gs.Rings[key]; ok {
			title, lines := ringTip(record, i, len(worn))
			s.tip.Point(seat, title, lines)
		}
		return
	}
}

// click is the press on either row. **Both rows are live at once**, unlike the reward screen's two
// stages: buying and selling are not steps of one decision, and needing to be in "sell mode" to
// free a finger for the ring you are looking at would be a mode where a click would do.
func (s *ShopScene) click(gs *state.GlobalState) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	at := image.Pt(gs.MouseX, gs.MouseY)

	for i := range s.shelf {
		if !s.shelf[i].bought && at.In(s.shelfSlot(gs, i)) {
			s.buy(gs, i)
			return
		}
	}

	worn := gs.Run.Worn()
	for i, key := range worn {
		if at.In(s.wornSlot(gs, i, len(worn))) {
			s.sell(gs, key)
			return
		}
	}
}

// buy takes a ring off the shelf and puts it on the hand.
//
// **The flight is raised after the run has already changed**, so it is a ghost of something that
// has happened rather than an animation the model is waiting on — the same rule every other mover
// in the game follows. A refused purchase raises nothing, which is what makes an unaffordable card
// read as unavailable rather than as broken.
func (s *ShopScene) buy(gs *state.GlobalState, i int) {
	key := s.shelf[i].key
	if !gs.Run.CanBuy(key) {
		return
	}

	seats := s.seats(gs)
	// The bought ring sets off from the shelf seat the player clicked, so the card that travels is
	// the card they were looking at.
	seats[key] = s.shelfSlot(gs, i)

	if !gs.Run.Buy(key) {
		return
	}
	s.shelf[i].bought = true
	s.start(seats)
	s.tip.Forget()

	price, _ := session.RingPrice(key)
	trace.Logf("shop", "bought %s for %d, %d vitae left, wearing %d",
		key, price, gs.Run.Vitae(), len(gs.Run.Worn()))
}

// sell takes a ring off and pays a quarter of its price back.
//
// **The sold ring has nothing to fly**, which is the documented exception to cards always
// travelling: what happened is an absence. What does travel is every ring to its right, sliding
// into the seats the row's re-centring gives them.
func (s *ShopScene) sell(gs *state.GlobalState, key string) {
	seats := s.seats(gs)

	if !gs.Run.Sell(key) {
		return
	}
	s.start(seats)
	s.tip.Forget()

	trace.Logf("shop", "sold %s for %d, %d vitae in hand, wearing %d",
		key, session.SellValue(key), gs.Run.Vitae(), len(gs.Run.Worn()))
}

// seats is where every worn ring is sitting right now, keyed by record — the picture taken before a
// change, so the row can be seen moving from it.
func (s *ShopScene) seats(gs *state.GlobalState) map[string]image.Rectangle {
	worn := gs.Run.Worn()
	out := make(map[string]image.Rectangle, len(worn)+1)
	for i, key := range worn {
		out[key] = s.wornSlot(gs, i, len(worn))
	}
	return out
}

// start runs the row from where it was to wherever the change has put it.
func (s *ShopScene) start(from map[string]image.Rectangle) {
	s.from, s.move = from, newTravel(0, shopMoveTicks)
}

// shelfSlot is where one offered ring is drawn, and the rectangle it is clicked in. **One function
// for both**, the same rule every other row in the game follows: a card hit-tested against a
// rectangle it is not drawn in is exactly the bug this shape prevents.
func (s *ShopScene) shelfSlot(gs *state.GlobalState, i int) image.Rectangle {
	return rowSlot(gs, i, len(s.shelf), gs.PctY(shelfRowPct))
}

// wornSlot is where one worn ring is drawn. It takes the count rather than reading it, because the
// row it is being drawn into may be the one from before a sale.
func (s *ShopScene) wornSlot(gs *state.GlobalState, i, n int) image.Rectangle {
	return rowSlot(gs, i, n, gs.PctY(wornRowPct))
}

// rowSlot is a centred row of cards at a fixed pitch. **Full-size cards at a fixed gap**, not the
// hand's compressing pitch: neither row here can exceed five, so nothing has to be squeezed.
func rowSlot(gs *state.GlobalState, i, n, top int) image.Rectangle {
	pitch := cardWidth + 40
	width := (n-1)*pitch + cardWidth
	left := gs.PctX(50) - width/2 + i*pitch
	return image.Rect(left, top, left+cardWidth, top+cardHeight)
}

func (s *ShopScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	screen.Fill(screenGround)

	heading := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 34}
	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 18}

	line := func(y int, face *text.GoTextFace, msg string, ink color.RGBA) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(gs.PctX(50)), float64(y))
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(ink)
		text.Draw(screen, msg, face, op)
	}

	line(offerTitleTop, heading, "The shop", groundInk)
	line(offerHintTop, small, s.hint(gs), groundInk)

	s.drawShelf(gs, screen, small, line)
	s.drawWorn(gs, screen, small, line)

	systems.DrawButton(gs, screen, s.leaveButton)
	systems.DrawTooltip(gs, screen, &s.tip)
}

// drawShelf draws what is for sale, with its price under it.
//
// **A ring that cannot be bought is dimmed rather than hidden**, so the row still says what was
// offered and the reason one of them is unavailable is visible instead of a click that silently
// does nothing. The price is dimmed with it: the figure and the card say the same thing at once.
func (s *ShopScene) drawShelf(gs *state.GlobalState, screen *ebiten.Image,
	face *text.GoTextFace, line func(int, *text.GoTextFace, string, color.RGBA)) {

	if len(s.shelf) == 0 {
		line(gs.PctY(shelfRowPct)+cardHeight/2, face, "nothing left to sell you", groundInk)
		return
	}

	line(gs.PctY(shelfRowPct)-shopRowLabelGap, face, "for sale", groundInk)

	for i, item := range s.shelf {
		at := s.shelfSlot(gs, i)
		if item.bought {
			drawEmptySeat(screen, at)
			continue
		}

		record, ok := gs.Rings[item.key]
		if !ok {
			continue
		}
		affordable := gs.Run.CanBuy(item.key)
		price, _ := session.RingPrice(item.key)

		drawRingCard(gs, screen, at.Min, record, affordable)
		s.figure(gs, screen, at, fmt.Sprintf("%d vitae", price), affordable)
	}
}

// drawWorn draws the hand: what the run is wearing, in worn order, with what each would pay back.
//
// **Worn order is a rule and not a presentation detail** — rings fire left to right and compound —
// so the row is the firing order, and selling out of the middle changes it. That is a real cost of
// letting a ring come off, and it is visible here rather than hidden.
func (s *ShopScene) drawWorn(gs *state.GlobalState, screen *ebiten.Image,
	face *text.GoTextFace, line func(int, *text.GoTextFace, string, color.RGBA)) {

	worn := gs.Run.Worn()

	label := fmt.Sprintf("worn - %d/%d, click one to sell it", len(worn), combat.MaxWornRings)
	if len(worn) == 0 {
		label = "worn - nothing"
	}
	line(gs.PctY(wornRowPct)-shopRowLabelGap, face, label, groundInk)

	for i, key := range worn {
		record, ok := gs.Rings[key]
		if !ok {
			continue
		}
		seat := s.wornSlot(gs, i, len(worn))
		at := seat.Min
		if was, moving := s.from[key]; moving && !s.move.done() {
			at = flyingTo(was, seat, s.move)
		}

		drawRingCard(gs, screen, at, record, true)
		s.figure(gs, screen, image.Rectangle{Min: at, Max: at.Add(image.Pt(cardWidth, cardHeight))},
			fmt.Sprintf("sell +%d", session.SellValue(key)), true)
	}
}

// figure writes the number under a card, centred on it. Dimmed toward the ground rather than
// scaled toward black, because it is written straight onto the table — see the colour rules in
// CLAUDE.md, and `systems.ColorToward`, which exists for exactly this.
func (s *ShopScene) figure(gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle,
	msg string, lit bool) {

	ink := groundInk
	if !lit {
		ink = systems.ColorToward(groundInk, screenGround, 55)
	}

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(at.Min.X+cardWidth/2), float64(at.Max.Y+shopFigureGap))
	op.PrimaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(ink)
	text.Draw(screen, msg, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: shopFigureSize}, op)
}

// hint is the line under the title: the purse, and whichever of the two things standing between
// the player and a ring is actually true.
//
// **The cap surfaces here rather than being displayed as empty slots** — MECHANICS.md's rule is
// that it is never shown until it binds, and a hand of five with rings still on the shelf is the
// moment it binds.
func (s *ShopScene) hint(gs *state.GlobalState) string {
	vitae := gs.Run.Vitae()

	if len(gs.Run.Worn()) >= combat.MaxWornRings && s.anyLeft() {
		return fmt.Sprintf("%d vitae - every finger is spoken for, sell one to make room", vitae)
	}
	return fmt.Sprintf("%d vitae in hand", vitae)
}

// anyLeft reports whether the shelf still holds something to buy — so the cap is only mentioned
// when it is what is stopping the player, rather than on an empty shelf.
func (s *ShopScene) anyLeft() bool {
	for _, item := range s.shelf {
		if !item.bought {
			return true
		}
	}
	return false
}

// Compile-time assurance that the record the shelf draws still carries what this screen reads off
// it. A ring losing its price would otherwise be a shelf of free rings rather than a build failure.
var _ = func(r data.RingData) (string, int) { return r.Name, r.Price }
