package screens

// The shop: **three relics on a shelf, five fingers, and one purse.**
//
// It is the second of the between-fight scenes — the essence, then this, then the room choice — and
// like the first it is an ordinary scene in the registry rather than a mode of anything. Nothing
// here names what comes next: the scene says it is finished and `advanceRun` decides where that
// leads. See flow.go.
//
// **It is what makes thirteen of the seventeen relics reachable.** The grammar has been built since
// 2026-08-17 and a run opened wearing three of them with no way to get a fourth, so most of the
// catalog existed only in the file. What was missing was never the rules — `Session.Wear`, the
// purse and the `fight-won` accumulator were all already there — it was the screen.
//
// **Two rows, and they are the same object twice.** The shelf is what you can have and the row
// beneath is what you have; both are relic cards, both are clicked, and the difference is which
// direction the vitae moves. A shop built as a list with buttons would have made a relic a line of
// text on the one screen where it is a thing you are choosing to wear.
//
// **The rules of the trade live on the run, not here** — see session/shop.go. This file decides
// where a card is drawn and what a click means; what a relic costs, what it sells back for, and
// what happens to a growing relic's accumulator when it comes off are the run's business, and a
// screen holding a second opinion about any of them is the failure that separation prevents.

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// shelfSize is how many relics a visit puts up.
//
// **Three, matching the reward screen's row**, so the two between-fight screens read as one
// language: a short row of cards, and you take what you can afford. A shelf of seventeen would be
// a catalog rather than an offer, and there would be no reason for the shop to come round again.
const shelfSize = 3

// Where the two rows sit. Percentages anchor the groups; offsets inside a group stay in pixels,
// per CLAUDE.md.
const (
	// The narration clears the band, whose relic row can now carry a sell price under it. The
	// reward screen's own prose starts at 296 against a band with nothing under the relics.
	// **360 since 2026-09-04**, from 310. The band above it is a card tall and the card grew by a
	// sixth, so the sell figure under a worn relic — and the confirm tab that replaces it — now
	// reach further down than the narration used to start. TestTheSellFiguresClearTheNarration and
	// TestTheSellTabClearsTheNarration are what hold the gap.
	shopProseTop = 360

	// shopHintTop is the line between the narration and the shelf. **It is not a title** — the
	// creature's two sentences are the title, exactly as the payout's are on the reward screen —
	// and it is written only when it has something the duelist card does not already say.
	shopHintTop = 444

	// The figure under a card: what it costs on the shelf, what it pays back in the row.
	shopFigureGap  = 10
	shopFigureSize = 22

	// The confirm tab that hangs under an armed relic, in the seat the sell figure was written in.
	// **Narrower than the card it hangs off**, so it reads as attached to that relic rather than as
	// a row of its own.
	sellTabWidth    = 200
	sellTabHeight   = 30
	sellTabTextSize = 26
)

// shopMoveTicks() is how long a relic takes to reach its new place — a bought one crossing to the
// finger it lands on, and every relic that shifts along when one is sold.
//
// **A proportion of the game's one speed**, like everything else that moves. See clock.go.
func shopMoveTicks() int { return ui.Beat(1, 1) }

// shelfItem is one relic on the shelf.
type shelfItem struct {
	key string

	// bought is set once it has been taken. **It stays in the row rather than being removed from
	// it**, the same choice the prize row makes: a card leaving would move the two beside it, and
	// the shelf you are reading must not rearrange itself under your hand. The seat is drawn empty.
	bought bool
}

// ShopScene sells relics and buys them back.
type ShopScene struct {
	// shelf is what this visit offers, drawn from the relics the run is not already wearing.
	shelf []shelfItem

	// leaveButton is the only control that is not a card or a confirm tab. **There is no basket,
	// and buying has no confirm** — a click is a purchase, because the price is on the card and a
	// run cannot go into debt, so there is nothing a confirmation step would be protecting.
	// **Selling does have one** as of 2026-08-22, and the asymmetry is the point: see `armed`.
	leaveButton *models.Button

	// tut is Bob, when a run is being taught. See tutorial.go, and combat.go for the same field.
	tut tutorialOverlay

	// armed is the worn relic a confirm tab is hanging under, by record key, and empty for none.
	//
	// **Selling is the one thing on this screen that asks twice** *(owner's call, 2026-08-22)*.
	// Buying does not and should not: it is refused when it cannot be afforded, the price is on
	// the card, and a run cannot go into debt — so there is nothing a confirmation would protect.
	// A sale is the opposite. The relic is *already yours*, the row it sits in is the row the whole
	// screen invites you to read, and a click meant for a tooltip took a relic off your hand for
	// less than it cost. It is also not symmetric to undo: a growing relic's accumulator goes with
	// it, and buying it back starts that over.
	armed string

	// relicDrag is the press in progress over the worn row. **A press there is now two gestures
	// sharing one button**: a click still arms the sell tab, and a press that travels reorders the
	// row instead. The threshold in carddrag.go is what tells them apart, and it is the same
	// threshold the hand has used since the action box was built.
	relicDrag ui.CardDrag

	// selling is the tab's request, consumed by Update, for the reason `leaving` is: a button's
	// OnClick reaches no global state and a sale needs the run.
	selling string

	// sellButton is the tab itself — **one button moved under whichever relic is armed**, not one
	// per finger. Only one can be armed, so a second button would be a second thing to keep in
	// step with the row's own re-centering.
	sellButton *models.Button

	// leaving is the button's request, consumed by Update. A button's OnClick reaches no global
	// state, and advancing the run needs it.
	leaving bool

	// from is where each worn relic was sitting before the last change, keyed by record, and move
	// is the one clock they all travel on.
	//
	// **Every relic in the row moves when one is bought or sold**, because the row is centered: the
	// seats themselves shift. So this is a map rather than a single mover, and it is *seats* being
	// remembered rather than journeys — the destination is recomputed from the layout every frame,
	// which is what lets a flight survive the window being resized. See travel.go.
	from map[string]image.Rectangle
	move ui.Travel

	// prose is the shopkeeper. **Nothing it says has a `pays`**, unlike the reward screen's
	// payout — this is flavor rather than arithmetic, and the typewriter is reused for the
	// cadence rather than for the claims.
	prose typewriter

	// deck is the D button in the corner and the panel behind it. A relic is bought against a deck,
	// and until 2026-08-22 the deck could not be looked at from here. See deckpanel.go.
	deck ui.DeckToggle

	// hands is the C button beside it: every hand the deck can build, and what each pays. A relic
	// is bought against a deck for the hands that deck can make, which is the question this
	// answers and the shelf does not. See handspanel.go.
	hands ui.HandsToggle

	// offered is which packs this visit put up: two of the three, dealt in Init and fixed for the
	// visit unless the reroll button under them is pressed. **A slice rather than three flags**,
	// because the pane's two seats are positions and which kind stands in each is the decision.
	offered []string

	// stockRNG and packRNG are the visit's two streams, kept so a reroll advances a cursor rather
	// than starting a second sequence. See Init.
	stockRNG *rand.Rand
	packRNG  *rand.Rand

	// relicReroll and packReroll are the two buttons under those panes. **Two buttons rather than
	// one moved between two places**, unlike the worn row's sell tab, because both are up at once.
	relicReroll, packReroll *models.Button

	// rerolling is the pane a button asked to redraw, consumed on the next frame for the reason
	// `selling` is: a button's OnClick reaches no global state and a reroll needs the purse.
	rerolling shopPane
	rerollNow bool

	// rerolls is how many times each pane has been rerolled this visit, by pane. It is what the
	// price is doubled against — see rerollPrice — and it is per visit rather than per run, so a
	// fresh shop opens at the base price whatever the last one cost.
	rerolls map[shopPane]int

	// drunk is which potions have been bought this visit, by record key. **Per visit, like a
	// sealed good's flag**: the catalog is the same three every shop, and a run that could buy
	// three Salves in a row would be buying a life bar rather than a potion.
	drunk map[string]bool

	// visit is which shop visit the state below belongs to — the run and the fight it follows.
	//
	// **It exists because this screen can now be left and come back to.** A sealed good is its own
	// screen, so returning from one runs Init again, and a shelf re-dealt at that moment would be a
	// second offer the player did not earn. See Init.
	visit shopVisit

	// opened is which of this visit's sealed goods have been taken, by record key.
	//
	// **Once each per visit** *(owner's call, 2026-08-27)*, restocked on the next. It bounds what a
	// rich run can do in one stop and keeps the shop a short offer rather than a vending machine —
	// the same argument the three-ring shelf is under. **A map rather than a flag each** since the
	// catalog became data/goods.json: a fourth good is a record, not a field on this scene.
	opened map[string]bool

	// pouch is the S button beside them: the stones the run is carrying, and the two things
	// that can be done to one. **A panel rather than a row**, because the screen has no vertical
	// room left for a fourth row of cards - see shop_pouch.go.
	pouch pouchToggle

	// tip explains a relic: what it does, what it costs, and where it would sit in the firing order.
	// **The case the tooltip was built for** — a shelf offering The Sickle says a name and a price
	// and nothing at all about slashes.
	tip models.Tooltip
}

// Init deals the shelf. **Re-entered on every visit**, because each fight earns its own.
//
// **A visit is one deal, however many times the screen is entered** *(2026-09-19)*. Opening a
// sealed good is a screen now, so coming back from one re-enters the shop — and a second deal would
// restock the shelf, forget which goods had been opened, un-drink the potions and replay the
// shopkeeper. `visit` is which visit the state on this scene belongs to, and a matching one is
// picked up rather than dealt again.
func (s *ShopScene) Init(gs *state.GlobalState) {
	if gs.Run != nil {
		if now := (shopVisit{seed: gs.RunSeed, fight: gs.Run.Fight()}); now == s.visit {
			// **Only the widgets are rebuilt.** Everything a visit accumulates — the shelf, what
			// has been opened, what has been drunk, both streams — is the visit's and stays.
			s.armed, s.selling = "", ""
			s.leaving = false
			s.from, s.move = nil, ui.Travel{}
			s.tip.Forget()
			return
		}
		s.visit = shopVisit{seed: gs.RunSeed, fight: gs.Run.Fight()}
	}

	if s.leaveButton == nil {
		s.leaveButton = models.NewButton(offerButtonWidth, offerButtonHeight, "LEAVE",
			func() { s.leaving = true })
		s.leaveButton.BaseColor = color.RGBA{R: 120, G: 132, B: 150, A: 255}
	}

	if s.sellButton == nil {
		s.sellButton = models.NewButton(sellTabWidth, sellTabHeight, "",
			func() { s.selling = s.armed })
		// **The color a control that commits something wears**, and the same crimson DUEL!
		// takes. A sale is the only thing on this screen that cannot be taken back.
		s.sellButton.BaseColor = color.RGBA{R: 220, G: 20, B: 60, A: 255}
		s.sellButton.TextSize = sellTabTextSize
	}

	s.initRerollButtons()
	s.pouch.init()

	s.armed, s.selling = "", ""
	s.leaving = false
	s.from, s.move = nil, ui.Travel{}
	s.tip = models.Tooltip{DwellTicks: ui.TipDwell()}
	s.opened = map[string]bool{}

	// **Both stocks are dealt from an rng the visit keeps**, rather than from one built per call.
	// That is what makes a reroll *advance* the stream instead of drawing a second sequence
	// beside it — the property TODO.md asked for, and the one that keeps a replayed run exact.
	s.stockRNG = shopRNG(gs, seeds.ShopStock)
	s.packRNG = shopRNG(gs, seeds.PackOffer)
	s.rerolls = map[shopPane]int{}
	s.drunk = map[string]bool{}
	s.offered = dealPacks(s.packRNG)
	s.shelf = dealShelf(gs, s.stockRNG)
	s.prose.setLines(shopkeeperLines())
	// **The same places the combat screen uses** *(owner's call, 2026-09-06)*. HANDS is a rung of
	// the control column and the two square panels stand on the bottom line beside the frame's cog,
	// so the corner reads the same on every screen that has one — see controlcolumn.go, which is
	// the one place either is measured from.
	s.deck.InitAsPile()
	s.hands.InitInColumn(func(gs *state.GlobalState) image.Point {
		return ControlColumnSlotCenter(gs, SlotHands)
	})

	trace.Logf("shop", "after fight %d: %v for sale, %d vitae in hand, wearing %d",
		gs.Run.Fight(), shelfKeys(s.shelf), gs.Run.Vitae(), len(gs.Run.Worn()))
}

// shopVisit names one stop at the shop: a run and the fight it comes after. **Comparable, so Init
// can ask whether it is looking at the same visit it dealt** — and it carries the seed as well as
// the fight because a fresh run starts at fight one too.
type shopVisit struct {
	seed  int64
	fight int
}

func shelfKeys(items []shelfItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.key)
	}
	return out
}

// dealShelf picks which relics are for sale: three weighted draws from everything the run is not
// wearing.
//
// **Rarity is the weight** *(owner's call, 2026-08-22)*. A common relic holds ten tickets to a
// rare one's, so a rare relic is something a run mostly does not see rather than something it sees
// and cannot afford — see data.Rarity for why the price ladder is much flatter than that.
//
// **The stream is handed in rather than built here** *(2026-09-06)*, so the reroll button can deal
// a second shelf off the same cursor. A function that rebuilt the rng from the seed would hand back
// the same three relics however many times it was pressed.
//
// **Its own stream** (`seeds.ShopStock`), and per fight — so a defeat and a retry walk into the
// same shop, exactly as they meet the same opponent. Sharing the essence offer's stream would have
// made authoring an essence change which relics every run was ever sold; see internal/seeds.
//
// **What is already worn is off the shelf**, rather than shown and refused. A relic on your hand
// offered back to you is a seat spent saying nothing, and `Buy` would turn the click down anyway.
func dealShelf(gs *state.GlobalState, rng *rand.Rand) []shelfItem {
	if gs.Run == nil || rng == nil {
		return nil
	}

	worn := make(map[string]bool, len(gs.Run.Worn()))
	for _, key := range gs.Run.Worn() {
		worn[key] = true
	}

	var pool []string
	for _, key := range session.Relics() {
		if !worn[key] {
			pool = append(pool, key)
		}
	}

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
// shelf offering the same relic twice would be a seat spent saying nothing; the caller removes what
// this returns and asks again, so the weights re-normalize over what is left.
//
// **A key the catalog does not weight holds one ticket rather than none.** The registry refuses a
// bad rarity at load, so reaching here with a zero is a relic the run knows about and the shop does
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
	if w := session.RelicWeight(key); w > 0 {
		return w
	}
	return 1
}

func (s *ShopScene) Update(gs *state.GlobalState) error {
	// Before this screen's own input; see combat.go's Update.
	s.tut.update(gs, s)

	s.move.Tick()

	if s.leaving {
		s.leaving = false
		advanceRun(gs)
		return nil
	}

	// **The greeting is the whole screen while it types.** A click skips it rather than buying
	// something, which is the reward screen's rule for its payout and for the same reason: a
	// sentence half-read while a relic is already being bought is two things at once.
	//
	// **It releases itself the moment it is complete**, which is where the two screens part
	// *(2026-09-08)*. The payout is held for a second click because its figures are the thing the
	// player came to read; a greeting is flavor in front of a shelf, so making it a gesture would
	// be charging a click for a sentence nobody is studying.
	if !s.prose.finished() {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && gs.CursorAllowed() {
			s.prose.skip(gs)
		}
		s.prose.tick(gs, func(i int) image.Point { return shopProseLineAt(gs, i) })
		if s.prose.filled() {
			s.prose.release()
		}
		return nil
	}

	// While the deck panel is up the two rows are dead. See deckToggle.update, which counts the
	// frame the panel closes on as a covered one.
	s.deck.Block(s.hands.IsOpen() || s.pouch.IsOpen())
	s.hands.Block(s.deck.IsOpen() || s.pouch.IsOpen())
	if s.deck.Update(gs, ui.OwnedContents(gs)) {
		return nil
	}
	if s.hands.Update(gs) {
		return nil
	}
	if s.updatePouch(gs) {
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
	s.updateRerollButtons(gs)

	s.click(gs)
	s.updateRelicRow(gs)

	s.leaveButton.ScreenX, s.leaveButton.ScreenY = gs.PctX(50), gs.PctY(offerButtonsPct)
	systems.UpdateButton(gs, s.leaveButton)

	s.hover(gs)
	systems.UpdateTooltip(gs, &s.tip)
	return nil
}

// hover points the tooltip at whichever relic the cursor is resting on. **The shelf first, then the
// hand**, which is the order they are drawn and the order they are read.
func (s *ShopScene) hover(gs *state.GlobalState) {
	at := image.Pt(gs.MouseX, gs.MouseY)

	// A gated step takes the tooltips with the clicks; see the combat screen's hover.
	if !gs.CursorAllowed() {
		return
	}

	// **The three shelves hit-test through ui.HoveredSeat**, like every row in the game: the panes
	// share one pitch across every seat the shop offers, so the cards overlap by a dozen pixels and
	// the one on top is the last drawn.
	//
	// **A seat nothing is drawn in answers an empty rectangle**, which HoveredSeat then walks past.
	// That is how a sold relic, an opened good and a drunk potion stay out of the way rather than
	// swallowing the card behind them — they are skipped in the drawing too, and a seat that
	// blocked a tooltip while showing nothing would be the worst version of this bug.
	if i := ui.HoveredSeat(at, len(s.shelf), func(i int) image.Rectangle {
		if s.shelf[i].bought {
			return image.Rectangle{}
		}
		return s.shelfSlot(gs, i)
	}); i >= 0 {
		if record, ok := gs.Relics[s.shelf[i].key]; ok {
			title, lines := ui.ShopRelicTip(record)
			s.tip.Point(s.shelfSlot(gs, i), ui.TipLine(title), ui.TipLines(lines))
		}
		return
	}

	if i := ui.HoveredSeat(at, len(s.offered), func(i int) image.Rectangle {
		if _, ok := session.GoodByKey(s.offered[i]); !ok || s.goodTaken(s.offered[i]) {
			return image.Rectangle{}
		}
		return s.goodSlot(gs, s.offered[i])
	}); i >= 0 {
		good, _ := session.GoodByKey(s.offered[i])
		title, lines := goodTip(good)
		s.tip.Point(s.goodSlot(gs, s.offered[i]), ui.TipLine(title), ui.TipLines(lines))
		return
	}

	potions := shopPotions()
	if i := ui.HoveredSeat(at, len(potions), func(i int) image.Rectangle {
		if s.drunk[potions[i].Record] {
			return image.Rectangle{}
		}
		return potionSeat(gs, i)
	}); i >= 0 {
		title, lines := potionTip(potions[i])
		s.tip.Point(potionSeat(gs, i), ui.TipLine(title), ui.TipLines(lines))
		return
	}

	// **The brand explains itself even though it cannot be bought**, which is the whole reason it
	// has a tooltip: a dim card with no explanation is one the player keeps clicking.
	if seat := brandSeat(gs); at.In(seat) {
		title, lines := brandTip()
		s.tip.Point(seat, ui.TipLine(title), ui.TipLines(lines))
		return
	}

	hoverBuildRelics(gs, at, &s.tip)
}

// click is the press on either row. **Both rows are live at once**, unlike the reward screen's two
// stages: buying and selling are not steps of one decision, and needing to be in "sell mode" to
// free a finger for the relic you are looking at would be a mode where a click would do.
//
// **A press on the worn row arms a confirm tab rather than selling** *(owner's call, 2026-08-22)*.
// The worn row is the build band now, which is the row the player hovers all run to read what they
// are wearing — so the seat a tooltip is asked for and the seat a sale is committed in are the
// same pixels, and a click that missed by a frame sold a relic. That is not a mode: nothing else
// on the screen changes while a tab is up, the other relics stay clickable, and clicking the armed
// relic again puts it away.
func (s *ShopScene) click(gs *state.GlobalState) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || !gs.CursorAllowed() {
		return
	}
	at := image.Pt(gs.MouseX, gs.MouseY)

	// **The pile is tested before the rows**, because it is the opener for a panel that covers
	// them: a press that lands on both is the one that opens the deck.
	if s.clickedPile(gs, at) {
		return
	}

	// **The same walk the tooltips take**, so a click lands on the card the panel just described.
	if i := ui.HoveredSeat(at, len(s.shelf), func(i int) image.Rectangle {
		if s.shelf[i].bought {
			return image.Rectangle{}
		}
		return s.shelfSlot(gs, i)
	}); i >= 0 {
		s.armed = ""
		s.buy(gs, i)
		return
	}

	if i := ui.HoveredSeat(at, len(s.offered), func(i int) image.Rectangle {
		return s.goodSlot(gs, s.offered[i])
	}); i >= 0 {
		s.armed = ""
		s.openGood(gs, s.offered[i])
		return
	}

	// **The potions are the shelf's rule, not the worn row's**: one click buys and drinks, because
	// the price is on the card and a purse cannot go into debt. The brand is not in this list at
	// all — it is a placeholder and a click on it does nothing. See shop_potions.go.
	clickable := shopPotions()
	if i := ui.HoveredSeat(at, len(clickable), func(i int) image.Rectangle {
		if s.drunk[clickable[i].Record] {
			return image.Rectangle{}
		}
		return potionSeat(gs, i)
	}); i >= 0 {
		s.armed = ""
		s.drinkPotion(gs, clickable[i].Record)
		return
	}

	// **A press on a worn relic is not this function's** *(2026-08-26)*. It became two gestures when
	// the row became reorderable — a click arms the sale, a drag moves the relic — and only the
	// release knows which it was, so it is answered by the shared drag's rowClick. Leaving the
	// press here as well would arm a relic on the way into a drag.
	//
	// **It asks ui.HoveredSeat although it wants no seat**, only whether the press landed on the
	// row at all. Which relic answers cannot change what happens here — every seat returns — so
	// this is the one row walk in the game that a forward loop would have got right. It goes
	// through the shared one anyway: a hand-rolled walk beside eleven that are not is a walk the
	// next reader has to check, and checking it is how the four that *were* wrong went unnoticed.
	worn := gs.Run.Worn()
	if ui.HoveredSeat(at, len(worn), func(i int) image.Rectangle {
		return s.wornSlot(gs, i, len(worn))
	}) >= 0 {
		return
	}

	// A press anywhere else drops the question — except on the tab itself, which is not a click
	// this screen handles: its own release is what answers.
	if s.armed != "" && !at.In(s.sellTabRect(gs)) {
		s.armed = ""
	}
}

// arm puts the question under one relic, or takes it away again if that relic is already asking it.
//
// **Pulled out of `click` so it can be tested**: the press needs a cursor and a window, and what
// is worth pinning is that arming a relic changes nothing about the run.
func (s *ShopScene) arm(key string) {
	if s.armed == key {
		s.armed = ""
		return
	}
	s.armed = key
}

// updateSellTab positions the tab under whichever relic is armed and runs it.
//
// **It disarms a relic that is no longer worn**, which is what stops a tab surviving the sale it
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
	s.sellButton.Text = fmt.Sprintf("SELL FOR %d?", session.SellValue(s.armed))
	tab := s.sellTabRect(gs)
	s.sellButton.ScreenX = (tab.Min.X + tab.Max.X) / 2
	s.sellButton.ScreenY = (tab.Min.Y + tab.Max.Y) / 2
	systems.UpdateButton(gs, s.sellButton)
}

// wornSeatOf is where one worn relic is sitting, by key.
func (s *ShopScene) wornSeatOf(gs *state.GlobalState, key string) (image.Rectangle, bool) {
	worn := gs.Run.Worn()
	for i, k := range worn {
		if k == key {
			return s.wornSlot(gs, i, len(worn)), true
		}
	}
	return image.Rectangle{}, false
}

// sellTabRect is where the confirm tab hangs: **the seat the sell figure is written in**, centered
// under the armed relic. One rectangle, drawn in and hit-tested against.
func (s *ShopScene) sellTabRect(gs *state.GlobalState) image.Rectangle {
	seat, ok := s.wornSeatOf(gs, s.armed)
	if !ok {
		return image.Rectangle{}
	}
	left := (seat.Min.X+seat.Max.X)/2 - sellTabWidth/2
	top := seat.Max.Y + shopFigureGap
	return image.Rect(left, top, left+sellTabWidth, top+sellTabHeight)
}

// buy takes a relic off the shelf and puts it on the hand.
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
	// The bought relic sets off from the shelf seat the player clicked, so the card that travels is
	// the card they were looking at.
	seats[key] = s.shelfSlot(gs, i)

	if !gs.Run.Buy(key) {
		return
	}
	s.shelf[i].bought = true
	s.start(seats)
	s.tip.Forget()

	price, _ := session.RelicPrice(key)
	trace.Logf("shop", "bought %s for %d, %d vitae left, wearing %d",
		key, price, gs.Run.Vitae(), len(gs.Run.Worn()))
}

// sell takes a relic off and pays its tier's sell-back figure.
//
// **The sold relic has nothing to fly**, which is the documented exception to cards always
// traveling: what happened is an absence. What does travel is every relic to its right, sliding
// into the seats the row's re-centering gives them.
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

// seats is where every worn relic is sitting right now, keyed by record — the picture taken before a
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
	s.from, s.move = from, ui.NewTravel(0, shopMoveTicks())
}

// shelfSlot is where one offered relic is drawn, and the rectangle it is clicked in. **One function
// for both**, the same rule every other row in the game follows: a card hit-tested against a
// rectangle it is not drawn in is exactly the bug this shape prevents.
func (s *ShopScene) shelfSlot(gs *state.GlobalState, i int) image.Rectangle {
	return shopSeatRect(gs, shopPaneRelics, i)
}

// wornSlot is where one worn relic is drawn — **a finger in the build band**, not a row of the
// shop's own. It takes the count rather than reading it, because the row it is being drawn into
// may be the one from before a sale.
//
// **It is `buildRelicRect` and `relicSlotAt`, which is what the combat screen and the reward screen
// use.** A relic is in the same place on every screen that shows one, so selling is a click on the
// row the player has been reading all run rather than on a second copy of it.
func (s *ShopScene) wornSlot(gs *state.GlobalState, i, n int) image.Rectangle {
	return relicSlotRect(buildRelicRect(gs), i, n)
}

func (s *ShopScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	ui.FillGround(screen)

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
	// because a relic here carries a price and moves when the row re-centers. See buildband.go.
	drawBuildCard(gs, screen, gs.Run.Vitae())
	// **The pane, without its fraction** *(2026-09-06)*. Every relic in this row carries a sell
	// figure under it and the count hangs off the same corner on the same line, so with five worn
	// the `5/5 relics` and the last `sell +3` are two numbers in one place. The pane is what was
	// missing here; the fraction is already said by a row you can count.
	drawRelicPaneBack(screen, buildRelicRect(gs))
	s.drawWorn(gs, screen, small)

	// **The runes the run is carrying, in the pane the fight draws them in** *(2026-09-06)*.
	// The shop draws the band's two halves itself, because a relic here carries a price — and the
	// consumables pane went missing in the split, so a run walked into a shop and its sack
	// vanished. nil: a rune is carried on this screen, not spent. See buildband.go.
	drawConsumablePane(gs, screen, buildConsumableRect(gs), nil, nil, s.tip.Showing())

	s.drawProse(gs, screen, prose)

	// **Nothing else is on screen while the greeting types.** It is the reward screen's rule: the
	// sentences are the whole of the screen until they are finished.
	if !s.prose.finished() {
		return
	}

	line(shopHintTop, small, s.hint(gs), ui.GroundInk)

	// **The four panes, left to right.** Each paints its own surface first and its cards on top;
	// see shop_panes.go, which owns where they stand.
	s.drawGoods(gs, screen)
	s.drawShelf(gs, screen)
	s.drawPotions(gs, screen)
	s.drawBrand(gs, screen)
	s.drawRerollButtons(gs, screen)
	drawDeckPile(gs, screen)

	systems.DrawButton(gs, screen, s.leaveButton)
	systems.DrawTooltip(gs, screen, &s.tip)

	// Last, and over everything: the panel covers the screen, so nothing of this one may be drawn
	// on top of it.
	s.deck.Draw(gs, screen, ui.OwnedContents(gs))
	s.hands.Draw(gs, screen, ui.OwnedHands(gs))
	s.drawPouch(gs, screen)

	// **Bob over everything, and the spotlight with him.** See combat.go's Draw, whose last line
	// this is the counterpart of: the scrim dims what is already drawn, so nothing may follow it.
	s.tut.draw(gs, screen, s)
}

// drawShelf draws what is for sale, with its price under it.
//
// **A relic that cannot be bought is dimmed rather than hidden**, so the row still says what was
// offered and the reason one of them is unavailable is visible instead of a click that silently
// does nothing. The price is dimmed with it: the figure and the card say the same thing at once.
func (s *ShopScene) drawShelf(gs *state.GlobalState, screen *ebiten.Image) {
	drawShopPaneBack(gs, screen, shopPaneRelics)

	for i, item := range s.shelf {
		at := s.shelfSlot(gs, i)
		if item.bought {
			// **A spent seat draws nothing; the pane is the hole** *(owner's call, 2026-09-06)*.
			// It used to be outlined, and on a pane an outline is a dark rectangle overlapping the
			// card beside it, because the row overlaps. See the consumables pane, which took the
			// same decision a day later.
			continue
		}

		record, ok := gs.Relics[item.key]
		if !ok {
			continue
		}
		affordable := gs.Run.CanBuy(item.key)
		price, _ := session.RelicPrice(item.key)

		// **No badge on the shelf**, whatever the relic is: an accumulator belongs to a worn relic,
		// and a shelf relic is one nobody has ever put on. A relic the run once wore and sold has
		// had its number reset, so there is nothing to show there either.
		ui.DrawRelicCard(gs, screen, at.Min, record, "", affordable, false)
		s.figure(gs, screen, at, fmt.Sprintf("%d vitae", price), affordable)
	}
}

// drawWorn draws the hand: what the run is wearing, in worn order, with what each would pay back.
//
// **Worn order is a rule and not a presentation detail** — relics fire left to right and compound —
// so the row is the firing order, and selling out of the middle changes it. That is a real cost of
// letting a relic come off, and it is visible here rather than hidden.
func (s *ShopScene) drawWorn(gs *state.GlobalState, screen *ebiten.Image,
	face *text.GoTextFace) {

	worn := gs.Run.Worn()
	counters := runCounters(gs)

	for i, key := range worn {
		record, ok := gs.Relics[key]
		if !ok {
			continue
		}
		// The seat a dragged relic left stays empty; see the combat screen's row.
		if s.relicDrag.Dragging() && i == s.relicDrag.Origin() {
			continue
		}
		seat := s.wornSlot(gs, i, len(worn))
		at := seat.Min
		if was, moving := s.from[key]; moving && !s.move.Done() {
			at = ui.FlyingTo(was, seat, s.move)
		}

		ui.DrawRelicCard(gs, screen, at, record, counters[key], true, false)

		// **The price is only offered once the shopkeeper has finished speaking**, like everything
		// else on this screen — a sell figure under a relic during the greeting would be an offer
		// standing before it was made.
		//
		// **An armed relic shows the tab in that seat instead of the figure**, rather than both:
		// the tab carries the same number and asks the question the figure only stated, so
		// drawing the pair would be the price said twice with one of them clickable.
		if !s.prose.finished() {
			continue
		}
		if s.armed == key {
			systems.DrawButton(gs, screen, s.sellButton)
			continue
		}
		// **The figure follows the card, not the seat**, so a relic still sliding to its new
		// finger keeps its price under it.
		flown := image.Rectangle{Min: at,
			Max: at.Add(image.Pt(cards.RelicStyle.Width, cards.RelicStyle.Height))}
		s.figure(gs, screen, flown, fmt.Sprintf("sell +%d", session.SellValue(key)), true)
	}

	// Last, so the relic riding the cursor rides over the sell figures too.
	drawDraggedRelic(gs, screen, &s.relicDrag, counters)
}

// updateRelicRow runs the drag over the worn row.
//
// **A click here arms the sell tab**, which is the one screen where a press on a relic means
// something besides reordering it — see click, which no longer handles that row.
//
// **The row is dead while the shopkeeper is still speaking and under either panel**, exactly as
// buying and selling are: the greeting is the whole screen while it runs.
func (s *ShopScene) updateRelicRow(gs *state.GlobalState) {
	worn := gs.Run.Worn()
	row := buildRelicRow(gs, func(i int) {
		if i >= 0 && i < len(worn) {
			s.arm(worn[i])
		}
	})

	if !gs.CursorAllowed() {
		s.relicDrag.Cancel(row)
		return
	}

	s.relicDrag.Update(gs, row)
}

// figure writes the number under a card, centered on it. Dimmed toward the ground rather than
// scaled toward black, because it is written straight onto the table — see the color rules in
// CLAUDE.md, and `systems.ColorToward`, which exists for exactly this.
func (s *ShopScene) figure(gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle,
	msg string, lit bool) {

	ink := ui.GroundInk
	if !lit {
		ink = systems.ColorToward(ui.GroundInk, ui.ScreenGround, 55)
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
// that it is never shown until it binds, and a hand of five with relics still on the shelf is the
// moment it binds. That is the whole of what this line is for now.
//
// It used to open with the purse as well. The duelist card in the build band writes the purse in
// crimson two hundred pixels above, so saying it again here would be the screen's only sentence
// spent on a figure already on it.
func (s *ShopScene) hint(gs *state.GlobalState) string {
	// **Against the run's own cap, not a width** *(2026-09-17)*. It read combat.MaxWornRelics — the
	// duelist array's width, eight — so the hint about every finger being spoken for waited for a
	// sixth, seventh and eighth relic the shop had already refused to sell. The same mistake the
	// pane's `0/8` fraction made, and the constant is gone now.
	if len(gs.Run.Worn()) >= gs.Run.RelicSlots() && s.anyLeft() {
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
// it. A relic losing its price would otherwise be a shelf of free relics rather than a build failure.
var _ = func(r data.RelicData) (string, data.Rarity) { return r.Name, r.Rarity }

// The sealed goods on the shelf: where they sit, what a click on one does, and what they say.
//
// **They are a row of their own under the relics** — see goodsRowPct, and shop_goods.go for the
// dialog a purchase opens.

// goodSlot is where one good is drawn, and the rectangle it is clicked in. **Two seats, centered**,
// so the row reads as a pair rather than as two things that happen to be near each other.
func (s *ShopScene) goodSlot(gs *state.GlobalState, key string) image.Rectangle {
	for i, offered := range s.offered {
		if offered == key {
			return shopSeatRect(gs, shopPanePacks, i)
		}
	}
	return image.Rectangle{}
}

// **The row's geometry moved to shop_panes.go on 2026-09-06** *(owner's call)*. It was one flat
// row of six seats at a fixed pitch, told apart by the labels written under them; it is four panes
// at one solved pitch now, and there is no label anywhere on the shelf. `rowWidth` went with it —
// nothing counts the shelf's seats any more, because each pane knows its own.

// goodTaken is whether this visit's copy has already been opened.
func (s *ShopScene) goodTaken(key string) bool { return s.opened[key] }

// goodAffordable is whether the purse covers one. **Asked of the run rather than compared here**,
// which is the line RelicPrice already draws: what a thing costs is the shop's arithmetic and this
// file only decides where it is drawn.
func goodAffordable(gs *state.GlobalState, key string) bool {
	if gs.Run == nil {
		return false
	}
	return gs.Run.CanAffordGood(key)
}

// goodAvailable is whether a seat can be clicked at all: the purse covers it, and there is somewhere
// to put what comes out.
//
// **Only the sack has the second question** *(2026-09-06)*. A stone is spent in the dialog that
// opened the bag and an essence is spent in the dialog that opened the vial, so neither can hand the run
// something it has no room for; a rune goes into a sack that now holds two — see
// session.MaxHeld — and a full one would take five vitae for a card that `Hold` refuses. The seat
// goes dim rather than the purchase failing afterwards, which is the same courtesy an unaffordable
// good already gets.
func goodAvailable(gs *state.GlobalState, key string) bool {
	if !goodAffordable(gs, key) {
		return false
	}
	good, ok := session.GoodByKey(key)
	if !ok {
		return false
	}
	if good.Contains == session.ContentsRunes && gs.Run.HoldFull() {
		return false
	}
	return true
}

// openGood pays for a sealed good and opens it.
//
// **The purse moves first and the dialog opens second**, exactly as `Buy` wears the relic after
// spending: a refusal has to leave the run as it was, and `SpendVitae` is the one place that
// refuses. A dialog opened before the payment would be four cards the player could take for free
// if the purse turned out to be short.
func (s *ShopScene) openGood(gs *state.GlobalState, key string) {
	if gs.Run == nil || s.goodTaken(key) || !goodAvailable(gs, key) {
		return
	}

	good, ok := session.GoodByKey(key)
	if !ok || !gs.Run.BuyGood(key) {
		return
	}

	s.opened[key] = true
	s.tip.Forget()
	openGoods(gs, key)

	trace.Logf("shop", "opened %s, %d vitae left", good.Name, gs.Run.Vitae())
}

// drawGoods draws the two sealed goods with their price under them, on the same terms as the
// shelf: an unaffordable one is dimmed rather than hidden, and one already opened leaves an empty
// seat rather than closing the row up.
func (s *ShopScene) drawGoods(gs *state.GlobalState, screen *ebiten.Image) {
	drawShopPaneBack(gs, screen, shopPanePacks)

	for _, key := range s.offered {
		good, ok := session.GoodByKey(key)
		if !ok || s.goodTaken(key) {
			continue
		}
		at := s.goodSlot(gs, key)

		lit := goodAvailable(gs, key)
		ui.DrawGoodCard(gs, screen, at.Min, good.Name, goodArt(gs, good), lit)
		s.figure(gs, screen, at, fmt.Sprintf("%d vitae", good.Price), lit)
	}
}

// goodArt is the picture a sealed good draws.
//
// **A record naming its own Art wins, and since 2026-09-15 all three do** — the bag, the vial and
// the sack are painted as the vessels they are named after.
//
// **The borrow is what is left underneath**: a good with no Art draws the default face of whatever
// is inside it — the stone catalog's, the essence catalog's, the rune catalog's. It is kept rather than deleted because it is what a *new* good draws on the day it is
// authored and before it is drawn, and a picture of its contents says more than a fourth
// placeholder would.
func goodArt(gs *state.GlobalState, good session.Good) image.Image {
	if good.Art != "" {
		return ui.Artwork(gs, good.Art)
	}
	switch good.Contains {
	case session.ContentsStones:
		return ui.Artwork(gs, data.DefaultStoneArt)
	case session.ContentsRunes:
		return ui.Artwork(gs, data.DefaultRuneArt)
	default:
		return ui.Artwork(gs, data.DefaultEssenceArt)
	}
}

// goodTip is what resting on one says. **It is the whole of what the card says** *(owner's call,
// 2026-09-15)* — the face is a picture and a price now, so the count, the noun and what that noun
// even is are all read here, by a player who on floor one has never seen a stone.
//
// **The prose is the record's and the figures are not** *(2026-09-14)*. The Tip lines say what a
// stone or an essence or a rune *is*; how many are in the good and what it costs are computed from
// the same fields the face and the purse read, so a tooltip cannot quote a price the shop does not
// charge. It also gave the sack a tooltip of its own — it fell through to the vial's until the
// catalog landed.
func goodTip(good session.Good) (string, []string) {
	lines := make([]string, 0, len(good.Tip)+2)
	lines = append(lines, fmt.Sprintf("%d %s, and you keep one", good.Size, good.Contains.Noun()))
	lines = append(lines, good.Tip...)
	return good.Name, append(lines, fmt.Sprintf("%d vitae", good.Price))
}
