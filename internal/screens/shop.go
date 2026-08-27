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

// shelfSize is how many rings a visit puts up.
//
// **Three, matching the reward screen's row**, so the two between-fight screens read as one
// language: a short row of cards, and you take what you can afford. A shelf of seventeen would be
// a catalogue rather than an offer, and there would be no reason for the shop to come round again.
const shelfSize = 3

// Where the two rows sit. Percentages anchor the groups; offsets inside a group stay in pixels,
// per CLAUDE.md.
const (
	// **There is one row of rings for sale, and the worn row is the build band's** *(owner's call,
	// 2026-08-22)*. The shop drew its own row of worn rings until the band arrived; with the band
	// up they were the same five rings twice, and the screen had no room for both once the hooded
	// creature started speaking. So the shelf is the only row the shop lays out, and it sits under
	// the narration where the eye lands after reading it.
	shelfRowPct = 48

	// The narration clears the band, whose ring row can now carry a sell price under it. The
	// reward screen's own prose starts at 296 against a band with nothing under the rings.
	shopProseTop = 310

	// shopHintTop is the line between the narration and the shelf. **It is not a title** — the
	// creature's two sentences are the title, exactly as the payout's are on the reward screen —
	// and it is written only when it has something the duelist card does not already say.
	shopHintTop = 392

	// The figure under a card: what it costs on the shelf, what it pays back in the row.
	shopFigureGap  = 10
	shopFigureSize = 22

	// The confirm tab that hangs under an armed ring, in the seat the sell figure was written in.
	// **Narrower than the card it hangs off**, so it reads as attached to that ring rather than as
	// a row of its own.
	sellTabWidth    = 130
	sellTabHeight   = 30
	sellTabTextSize = 18

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

	// leaveButton is the only control that is not a card or a confirm tab. **There is no basket,
	// and buying has no confirm** — a click is a purchase, because the price is on the card and a
	// run cannot go into debt, so there is nothing a confirmation step would be protecting.
	// **Selling does have one** as of 2026-08-22, and the asymmetry is the point: see `armed`.
	leaveButton *models.Button

	// tut is Bob, when a run is being taught. See tutorial.go, and combat.go for the same field.
	tut tutorialOverlay

	// armed is the worn ring a confirm tab is hanging under, by record key, and empty for none.
	//
	// **Selling is the one thing on this screen that asks twice** *(owner's call, 2026-08-22)*.
	// Buying does not and should not: it is refused when it cannot be afforded, the price is on
	// the card, and a run cannot go into debt — so there is nothing a confirmation would protect.
	// A sale is the opposite. The ring is *already yours*, the row it sits in is the row the whole
	// screen invites you to read, and a click meant for a tooltip took a ring off your hand for
	// less than it cost. It is also not symmetric to undo: a growing ring's accumulator goes with
	// it, and buying it back starts that over.
	armed string

	// ringDrag is the press in progress over the worn row. **A press there is now two gestures
	// sharing one button**: a click still arms the sell tab, and a press that travels reorders the
	// row instead. The threshold in carddrag.go is what tells them apart, and it is the same
	// threshold the hand has used since the action box was built.
	ringDrag cardDrag

	// selling is the tab's request, consumed by Update, for the reason `leaving` is: a button's
	// OnClick reaches no global state and a sale needs the run.
	selling string

	// sellButton is the tab itself — **one button moved under whichever ring is armed**, not one
	// per finger. Only one can be armed, so a second button would be a second thing to keep in
	// step with the row's own re-centring.
	sellButton *models.Button

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

	// prose is the shopkeeper. **Nothing it says has a `pays`**, unlike the reward screen's
	// payout — this is flavour rather than arithmetic, and the typewriter is reused for the
	// cadence rather than for the claims.
	prose typewriter

	// deck is the D button in the corner and the panel behind it. A ring is bought against a deck,
	// and until 2026-08-22 the deck could not be looked at from here. See deckpanel.go.
	deck deckToggle

	// hands is the C button beside it: every hand the deck can build, and what each pays. A ring
	// is bought against a deck for the hands that deck can make, which is the question this
	// answers and the shelf does not. See handspanel.go.
	hands handsToggle

	// bagBought, canBought and bucketBought are whether this visit's three sealed goods have been
	// taken.
	//
	// **Once each per visit** *(owner's call, 2026-08-27)*, restocked on the next. It bounds what a
	// rich run can do in one stop and keeps the shop a short offer rather than a vending machine —
	// the same argument the three-ring shelf is under.
	bagBought, canBought, bucketBought bool

	// good is the dialog a purchase opens: the four that were inside, and which one is taken. See
	// shop_goods.go.
	good goods

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

	if s.sellButton == nil {
		s.sellButton = models.NewButton(sellTabWidth, sellTabHeight, "",
			func() { s.selling = s.armed })
		// **The colour a control that commits something wears**, and the same crimson DUEL!
		// takes. A sale is the only thing on this screen that cannot be taken back.
		s.sellButton.BaseColor = color.RGBA{R: 220, G: 20, B: 60, A: 255}
		s.sellButton.TextSize = sellTabTextSize
	}

	s.armed, s.selling = "", ""
	s.leaving = false
	s.from, s.move = nil, travel{}
	s.tip = models.Tooltip{DwellTicks: tipDwell}
	s.bagBought, s.canBought, s.bucketBought = false, false, false
	s.good.reset()
	s.shelf = dealShelf(gs)
	s.prose.setLines(shopkeeperLines())
	s.deck.init()
	s.hands.init(handsCornerPlace)

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

// dealShelf picks which rings are for sale: three weighted draws from everything the run is not
// wearing.
//
// **Rarity is the weight** *(owner's call, 2026-08-22)*. A common ring holds ten tickets to a
// rare one's, so a rare ring is something a run mostly does not see rather than something it sees
// and cannot afford — see data.Rarity for why the price ladder is much flatter than that.
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

	out := make([]shelfItem, 0, shelfSize)
	for len(out) < shelfSize && len(pool) > 0 {
		at := drawWeighted(pool, rng)
		out = append(out, shelfItem{key: pool[at]})
		pool = append(pool[:at], pool[at+1:]...)
	}
	return out
}

// drawWeighted picks one index out of the pool, each key holding as many tickets as its rarity is
// worth, and it is where the rarity mechanic actually bites.
//
// **Without replacement, which is why it is a draw per seat rather than one weighted shuffle.** A
// shelf offering the same ring twice would be a seat spent saying nothing; the caller removes what
// this returns and asks again, so the weights re-normalise over what is left.
//
// **A key the catalogue does not weight holds one ticket rather than none.** The registry refuses a
// bad rarity at load, so reaching here with a zero is a ring the run knows about and the shop does
// not — and dropping it from every shelf forever is a worse failure than offering it as a common.
func drawWeighted(pool []string, rng *rand.Rand) int {
	total := 0
	for _, key := range pool {
		total += weightOf(key)
	}

	ticket := rng.Intn(total)
	for i, key := range pool {
		ticket -= weightOf(key)
		if ticket < 0 {
			return i
		}
	}
	return len(pool) - 1
}

func weightOf(key string) int {
	if w := session.RingWeight(key); w > 0 {
		return w
	}
	return 1
}

func (s *ShopScene) Update(gs *state.GlobalState) error {
	// Before this screen's own input; see combat.go's Update.
	s.tut.update(gs, s)

	s.move.tick()

	if s.leaving {
		s.leaving = false
		advanceRun(gs)
		return nil
	}

	// **The greeting is the whole screen while it types.** A click skips it rather than buying
	// something, which is the reward screen's rule for its payout and for the same reason: a
	// sentence half-read while a ring is already being bought is two things at once.
	if !s.prose.finished() {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && gs.CursorAllowed() {
			s.prose.skip(gs)
		}
		s.prose.tick(gs, func(i int) image.Point { return shopProseLineAt(gs, i) })
		return nil
	}

	// **The goods dialog runs before anything else and swallows the frame.** It stands between a
	// purchase and what it bought, so nothing behind it may be clickable — including the two
	// panels, whose buttons would otherwise sit live under a dialog with no exit but a card.
	if s.good.update(gs) {
		return nil
	}

	// While the deck panel is up the two rows are dead. See deckToggle.update, which counts the
	// frame the panel closes on as a covered one.
	s.deck.block(s.hands.open)
	s.hands.block(s.deck.open)
	if s.deck.update(gs, ownedContents(gs)) {
		return nil
	}
	if s.hands.update(gs) {
		return nil
	}

	if s.selling != "" {
		key := s.selling
		s.selling, s.armed = "", ""
		s.sell(gs, key)
		return nil
	}

	// **The tab runs before the click that might disarm it.** A press on the tab is a press on
	// nothing the rows own, so `click` leaves it armed and the release lands here — the same
	// press-then-release split the action box relies on.
	s.updateSellTab(gs)

	s.click(gs)
	s.updateRingRow(gs)

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

	// A gated step takes the tooltips with the clicks; see the combat screen's hover.
	if !gs.CursorAllowed() {
		return
	}

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

	for _, kind := range goodKinds() {
		seat := s.goodSlot(gs, kind)
		if s.goodTaken(kind) || !at.In(seat) {
			continue
		}
		title, lines := goodTip(kind)
		s.tip.Point(seat, title, lines)
		return
	}

	hoverBuildRings(gs, at, &s.tip)
}

// click is the press on either row. **Both rows are live at once**, unlike the reward screen's two
// stages: buying and selling are not steps of one decision, and needing to be in "sell mode" to
// free a finger for the ring you are looking at would be a mode where a click would do.
//
// **A press on the worn row arms a confirm tab rather than selling** *(owner's call, 2026-08-22)*.
// The worn row is the build band now, which is the row the player hovers all run to read what they
// are wearing — so the seat a tooltip is asked for and the seat a sale is committed in are the
// same pixels, and a click that missed by a frame sold a ring. That is not a mode: nothing else
// on the screen changes while a tab is up, the other rings stay clickable, and clicking the armed
// ring again puts it away.
func (s *ShopScene) click(gs *state.GlobalState) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || !gs.CursorAllowed() {
		return
	}
	at := image.Pt(gs.MouseX, gs.MouseY)

	for i := range s.shelf {
		if !s.shelf[i].bought && at.In(s.shelfSlot(gs, i)) {
			s.armed = ""
			s.buy(gs, i)
			return
		}
	}

	for _, kind := range goodKinds() {
		if at.In(s.goodSlot(gs, kind)) {
			s.armed = ""
			s.openGood(gs, kind)
			return
		}
	}

	// **A press on a worn ring is not this function's** *(2026-08-26)*. It became two gestures when
	// the row became reorderable — a click arms the sale, a drag moves the ring — and only the
	// release knows which it was, so it is answered by the shared drag's rowClick. Leaving the
	// press here as well would arm a ring on the way into a drag.
	worn := gs.Run.Worn()
	for i := range worn {
		if at.In(s.wornSlot(gs, i, len(worn))) {
			return
		}
	}

	// A press anywhere else drops the question — except on the tab itself, which is not a click
	// this screen handles: its own release is what answers.
	if s.armed != "" && !at.In(s.sellTabRect(gs)) {
		s.armed = ""
	}
}

// arm puts the question under one ring, or takes it away again if that ring is already asking it.
//
// **Pulled out of `click` so it can be tested**: the press needs a cursor and a window, and what
// is worth pinning is that arming a ring changes nothing about the run.
func (s *ShopScene) arm(key string) {
	if s.armed == key {
		s.armed = ""
		return
	}
	s.armed = key
}

// updateSellTab positions the tab under whichever ring is armed and runs it.
//
// **It disarms a ring that is no longer worn**, which is what stops a tab surviving the sale it
// asked about — or a scenario arriving with a key the run does not hold.
func (s *ShopScene) updateSellTab(gs *state.GlobalState) {
	if s.armed == "" {
		return
	}

	if _, ok := s.wornSeatOf(gs, s.armed); !ok {
		s.armed = ""
		return
	}

	// The label carries the figure, so the tab is the whole question — the sell figure it replaces
	// said the same number and asked nothing.
	s.sellButton.Text = fmt.Sprintf("Sell for %d?", session.SellValue(s.armed))
	tab := s.sellTabRect(gs)
	s.sellButton.ScreenX = (tab.Min.X + tab.Max.X) / 2
	s.sellButton.ScreenY = (tab.Min.Y + tab.Max.Y) / 2
	systems.UpdateButton(gs, s.sellButton)
}

// wornSeatOf is where one worn ring is sitting, by key.
func (s *ShopScene) wornSeatOf(gs *state.GlobalState, key string) (image.Rectangle, bool) {
	worn := gs.Run.Worn()
	for i, k := range worn {
		if k == key {
			return s.wornSlot(gs, i, len(worn)), true
		}
	}
	return image.Rectangle{}, false
}

// sellTabRect is where the confirm tab hangs: **the seat the sell figure is written in**, centred
// under the armed ring. One rectangle, drawn in and hit-tested against.
func (s *ShopScene) sellTabRect(gs *state.GlobalState) image.Rectangle {
	seat, ok := s.wornSeatOf(gs, s.armed)
	if !ok {
		return image.Rectangle{}
	}
	left := (seat.Min.X+seat.Max.X)/2 - sellTabWidth/2
	top := seat.Max.Y + shopFigureGap
	return image.Rect(left, top, left+sellTabWidth, top+sellTabHeight)
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

// sell takes a ring off and pays its tier's sell-back figure.
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
	return rowSlot(gs, i, s.rowWidth(), gs.PctY(shelfRowPct))
}

// wornSlot is where one worn ring is drawn — **a finger in the build band**, not a row of the
// shop's own. It takes the count rather than reading it, because the row it is being drawn into
// may be the one from before a sale.
//
// **It is `buildRingRect` and `ringSlotAt`, which is what the combat screen and the reward screen
// use.** A ring is in the same place on every screen that shows one, so selling is a click on the
// row the player has been reading all run rather than on a second copy of it.
func (s *ShopScene) wornSlot(gs *state.GlobalState, i, n int) image.Rectangle {
	return ringSlotRect(buildRingRect(gs), i, n)
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

	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 18}
	prose := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 26}

	line := func(y int, face *text.GoTextFace, msg string, ink color.RGBA) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(gs.PctX(50)), float64(y))
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(ink)
		text.Draw(screen, msg, face, op)
	}

	// **The duelist card, then the worn row drawn by this screen** — the band's two halves, split
	// because a ring here carries a price and moves when the row re-centres. See buildband.go.
	drawBuildCard(gs, screen, gs.Run.Vitae())
	s.drawWorn(gs, screen, small)

	s.drawProse(gs, screen, prose)

	// **Nothing else is on screen while the greeting types.** It is the reward screen's rule: the
	// sentences are the whole of the screen until they are finished.
	if !s.prose.finished() {
		return
	}

	line(shopHintTop, small, s.hint(gs), groundInk)
	s.drawShelf(gs, screen, small, line)
	s.drawGoods(gs, screen, small, line)

	systems.DrawButton(gs, screen, s.leaveButton)
	systems.DrawTooltip(gs, screen, &s.tip)

	// Last, and over everything: the panel covers the screen, so nothing of this one may be drawn
	// on top of it.
	s.deck.draw(gs, screen, ownedContents(gs))
	s.hands.draw(gs, screen, ownedHands(gs))

	// The sealed good's dialog, over both panels: it is the one dialog on this screen that a
	// purchase has already been made for, so nothing may be drawn on top of it but the tutorial.
	s.good.draw(gs, screen)

	// **Bob over everything, and the spotlight with him.** See combat.go's Draw, whose last line
	// this is the counterpart of: the scrim dims what is already drawn, so nothing may follow it.
	s.tut.draw(gs, screen, s)
}

// drawShelf draws what is for sale, with its price under it.
//
// **A ring that cannot be bought is dimmed rather than hidden**, so the row still says what was
// offered and the reason one of them is unavailable is visible instead of a click that silently
// does nothing. The price is dimmed with it: the figure and the card say the same thing at once.
func (s *ShopScene) drawShelf(gs *state.GlobalState, screen *ebiten.Image,
	face *text.GoTextFace, line func(int, *text.GoTextFace, string, color.RGBA)) {

	// **The row is never empty now**, because the two sealed goods are always in it — so the old
	// "nothing left to sell you" line went with the day the shelf stopped being rings alone. A run
	// wearing every ring in the catalogue still has a bag and a can to spend on.
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

		// **No badge on the shelf**, whatever the ring is: an accumulator belongs to a worn ring,
		// and a shelf ring is one nobody has ever put on. A ring the run once wore and sold has
		// had its number reset, so there is nothing to show there either.
		drawRingCard(gs, screen, at.Min, record, "", affordable, false)
		s.figure(gs, screen, at, fmt.Sprintf("%d vitae", price), affordable)
	}
}

// drawWorn draws the hand: what the run is wearing, in worn order, with what each would pay back.
//
// **Worn order is a rule and not a presentation detail** — rings fire left to right and compound —
// so the row is the firing order, and selling out of the middle changes it. That is a real cost of
// letting a ring come off, and it is visible here rather than hidden.
func (s *ShopScene) drawWorn(gs *state.GlobalState, screen *ebiten.Image,
	face *text.GoTextFace) {

	worn := gs.Run.Worn()
	counters := runCounters(gs)

	for i, key := range worn {
		record, ok := gs.Rings[key]
		if !ok {
			continue
		}
		// The seat a dragged ring left stays empty; see the combat screen's row.
		if s.ringDrag.dragging() && i == s.ringDrag.origin() {
			continue
		}
		seat := s.wornSlot(gs, i, len(worn))
		at := seat.Min
		if was, moving := s.from[key]; moving && !s.move.done() {
			at = flyingTo(was, seat, s.move)
		}

		drawRingCard(gs, screen, at, record, counters[key], true, false)

		// **The price is only offered once the shopkeeper has finished speaking**, like everything
		// else on this screen — a sell figure under a ring during the greeting would be an offer
		// standing before it was made.
		//
		// **An armed ring shows the tab in that seat instead of the figure**, rather than both:
		// the tab carries the same number and asks the question the figure only stated, so
		// drawing the pair would be the price said twice with one of them clickable.
		if !s.prose.finished() {
			continue
		}
		if s.armed == key {
			systems.DrawButton(gs, screen, s.sellButton)
			continue
		}
		// **The figure follows the card, not the seat**, so a ring still sliding to its new
		// finger keeps its price under it.
		flown := image.Rectangle{Min: at,
			Max: at.Add(image.Pt(cards.RingStyle.Width, cards.RingStyle.Height))}
		s.figure(gs, screen, flown, fmt.Sprintf("sell +%d", session.SellValue(key)), true)
	}

	// Last, so the ring riding the cursor rides over the sell figures too.
	drawDraggedRing(gs, screen, &s.ringDrag, counters)
}

// updateRingRow runs the drag over the worn row.
//
// **A click here arms the sell tab**, which is the one screen where a press on a ring means
// something besides reordering it — see click, which no longer handles that row.
//
// **The row is dead while the shopkeeper is still speaking and under either panel**, exactly as
// buying and selling are: the greeting is the whole screen while it runs.
func (s *ShopScene) updateRingRow(gs *state.GlobalState) {
	worn := gs.Run.Worn()
	row := buildRingRow(gs, func(i int) {
		if i >= 0 && i < len(worn) {
			s.arm(worn[i])
		}
	})

	if !gs.CursorAllowed() {
		s.ringDrag.cancel(row)
		return
	}

	s.ringDrag.update(gs, row)
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

// hint is the line between the narration and the shelf, and **it is usually empty** *(2026-08-22)*.
//
// **The cap surfaces here rather than being displayed as empty slots** — MECHANICS.md's rule is
// that it is never shown until it binds, and a hand of five with rings still on the shelf is the
// moment it binds. That is the whole of what this line is for now.
//
// It used to open with the purse as well. The duelist card in the build band writes the purse in
// crimson two hundred pixels above, so saying it again here would be the screen's only sentence
// spent on a figure already on it.
func (s *ShopScene) hint(gs *state.GlobalState) string {
	if len(gs.Run.Worn()) >= combat.MaxWornRings && s.anyLeft() {
		return fmt.Sprintf("%d vitae - every finger is spoken for, sell one to make room",
			gs.Run.Vitae())
	}
	return ""
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
var _ = func(r data.RingData) (string, data.Rarity) { return r.Name, r.Rarity }

// The sealed goods on the shelf: where they sit, what a click on one does, and what they say.
//
// **They are a row of their own under the rings** — see goodsRowPct, and shop_goods.go for the
// dialog a purchase opens.

// goodSlot is where one good is drawn, and the rectangle it is clicked in. **Two seats, centred**,
// so the row reads as a pair rather than as two things that happen to be near each other.
func (s *ShopScene) goodSlot(gs *state.GlobalState, kind goodKind) image.Rectangle {
	i := len(s.shelf)
	switch kind {
	case goodCan:
		i++
	case goodBucket:
		i += 2
	}
	return rowSlot(gs, i, s.rowWidth(), gs.PctY(shelfRowPct))
}

// rowWidth is how many seats the shelf has: the rings, plus the two sealed goods.
//
// **One row rather than two, and the screen decided it** *(2026-08-27)*. A second row of cards
// under the rings would have read better — a ring is worn and a good is opened, which are not the
// same kind of thing — but the shop is 960 tall with a build band at the top, two sentences of
// narration under it and the Leave button at 88%, and a card is 224. There is room for one row of
// cards between the narration and the button and there is no room for two. So the goods stand at
// the right-hand end of the shelf, told apart by their faces and by the label under them rather
// than by where they are.
// **Six seats since the bucket** *(2026-08-27)*. `rowSlot` is a fixed pitch of `cardWidth + 40`,
// so six is 1172 pixels of 1280 — it fits, with about 54 clear at each end, and a seventh would
// not. The next good has to narrow the pitch or find a second row, which is the constraint the
// paragraph above already describes.
func (s *ShopScene) rowWidth() int { return len(s.shelf) + len(goodKinds()) }

// goodTaken is whether this visit's copy has already been opened.
func (s *ShopScene) goodTaken(kind goodKind) bool {
	switch kind {
	case goodBag:
		return s.bagBought
	case goodBucket:
		return s.bucketBought
	default:
		return s.canBought
	}
}

// goodAffordable is whether the purse covers one. **Asked of the run rather than compared here**,
// which is the line RingPrice already draws: what a thing costs is the shop's arithmetic and this
// file only decides where it is drawn.
func goodAffordable(gs *state.GlobalState, kind goodKind) bool {
	if gs.Run == nil {
		return false
	}
	switch kind {
	case goodBag:
		return gs.Run.CanAffordBag()
	case goodBucket:
		return gs.Run.CanAffordBucket()
	default:
		return gs.Run.CanAffordCan()
	}
}

// openGood pays for a sealed good and opens it.
//
// **The purse moves first and the dialog opens second**, exactly as `Buy` wears the ring after
// spending: a refusal has to leave the run as it was, and `SpendVitae` is the one place that
// refuses. A dialog opened before the payment would be four cards the player could take for free
// if the purse turned out to be short.
func (s *ShopScene) openGood(gs *state.GlobalState, kind goodKind) {
	if gs.Run == nil || s.goodTaken(kind) || !goodAffordable(gs, kind) {
		return
	}

	paid := false
	switch kind {
	case goodBag:
		paid = gs.Run.BuyBag()
	case goodBucket:
		paid = gs.Run.BuyBucket()
	default:
		paid = gs.Run.BuyCan()
	}
	if !paid {
		return
	}

	switch kind {
	case goodBag:
		s.bagBought = true
	case goodBucket:
		s.bucketBought = true
	default:
		s.canBought = true
	}
	s.good.open(gs, kind)
	s.tip.Forget()

	trace.Logf("shop", "opened %s, %d vitae left", goodName(kind), gs.Run.Vitae())
}

// drawGoods draws the two sealed goods with their price under them, on the same terms as the
// shelf: an unaffordable one is dimmed rather than hidden, and one already opened leaves an empty
// seat rather than closing the row up.
func (s *ShopScene) drawGoods(gs *state.GlobalState, screen *ebiten.Image,
	face *text.GoTextFace, line func(int, *text.GoTextFace, string, color.RGBA)) {

	for _, kind := range goodKinds() {
		at := s.goodSlot(gs, kind)
		if s.goodTaken(kind) {
			drawEmptySeat(screen, at)
			continue
		}

		lit := goodAffordable(gs, kind)
		drawGoodCard(gs, screen, at.Min, goodName(kind), goodLine(kind), goodArt(gs, kind), lit)
		s.figure(gs, screen, at, fmt.Sprintf("%d vitae", goodPrice(kind)), lit)
	}
}

// goodName, goodLine, goodPrice and goodArt are what one of the two says on its face.
//
// **The line states the shape of the offer and never its contents** — "4 stones, keep 1" — because
// what is inside is drawn when the bag is opened. A face that could name the four would make these
// shelf items rather than sealed ones, which is the whole mechanic.
func goodName(kind goodKind) string {
	switch kind {
	case goodBag:
		return bagName
	case goodBucket:
		return bucketName
	default:
		return canName
	}
}

func goodLine(kind goodKind) string {
	if kind == goodBag {
		return fmt.Sprintf("%d stones\nkeep 1", session.BagSize())
	}
	return fmt.Sprintf("%d worms\nkeep 1", session.CanSize())
}

func goodPrice(kind goodKind) int {
	switch kind {
	case goodBag:
		return session.BagPrice()
	case goodBucket:
		return session.BucketPrice()
	default:
		return session.CanPrice()
	}
}

// **The bucket borrows the worm's picture**, for the reason a parasite card does: there is no
// parasite art, and a placeholder its sibling already wears is better than a blank face. The two
// goods are told apart by their names and their lines until one of them gets a drawing.
func goodArt(gs *state.GlobalState, kind goodKind) image.Image {
	if kind == goodBag {
		return stoneArt()
	}
	return artwork(gs, wormArtKey)
}

// goodTip is what resting on one says. **It explains what a stone and a worm each are**, since the
// face has room for neither and a player meeting the bag on floor one has never seen a stone.
func goodTip(kind goodKind) (string, []string) {
	if kind == goodBag {
		return bagName, []string{
			fmt.Sprintf("%d stones, and you keep one", session.BagSize()),
			"a stone raises one hand's multiplier",
			"by a tenth of it, for the rest of the run",
			fmt.Sprintf("%d vitae", session.BagPrice()),
		}
	}
	return canName, []string{
		fmt.Sprintf("%d worms, and you keep one", session.CanSize()),
		"a worm changes one card of your deck",
		fmt.Sprintf("%d vitae", session.CanPrice()),
	}
}
