package screens

// The two boosters, and the dialog that opens them.
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
// forfeits the vitae. That is a choice the player may make, and BoostersScene gives it a labelled
// button rather than the red square that means "close" everywhere else. Every card is also an exit
// — the screen ends when one is chosen.

import (
	"fmt"
	"image"
	"math/rand"
	"sort"
	"strings"

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

// **There is no boosterKind enum any more** *(2026-09-14)*. A shelf seat names a booster by its record
// key, the catalog is data/boosters.json, and what the dialog does with one is switched on
// session.BoosterContents — the closed vocabulary that says which catalog is inside. The three names
// that used to be constants here are the records' own Name fields.

// Where the boosters rows sit.
//
// **Measured from the screen, and now standing on it** *(2026-09-19)*. These were always screen
// percentages, because the modal panel covered most of it; what changed is what they have to clear.
// The build band and the two lines of type under it are at the top of the screen, so the cards
// start below them rather than below a panel's title bar — see buildBandBottom and offerHintTop.
const (
	// The bag's and the sack's one row, centered, with nothing under it.
	boostersChoiceRowPct = 42

	// The vial has two rows, and both of them plus the lift a selected card takes have to fit
	// between the hint and the bottom of the screen.
	vialEssenceRowPct   = 38
	boostersOfferRowPct = 68
)

// boostersStage is how far through opening a booster the player is.
type boostersStage int

const (
	// boostersClosed: nothing is open, and the shop is the shop.
	boostersClosed boostersStage = iota

	// boostersPick: what was drawn is up, and a click takes one. **There is no second stage**
	// *(owner's call, 2026-09-06)*: the vial used to move on to a row of cards once an essence was
	// chosen, so the essences disappeared at the moment the player had to judge one against a card.
	// Both rows are up together now and the gesture is the row's own — select the card, then click
	// the essence. See targeting.go, which is the rule the rune pane and the reward screen's own
	// essence row already follow.
	boostersPick

	// boostersShowing: an essence has been spent and what it did is on screen — the essence is used
	// where it stands and the picked cards change where they stand in the row, exactly as on the
	// reward screen. See essenceInPlace. **Only the vial reaches this stage** — a stone and a rune
	// change no card, so there is nothing to watch.
	boostersShowing

	// boostersCarrying: a rune or a cantrip has been taken and is flying into its seat in the
	// consumables pane *(owner's call, 2026-10-04)* — cards fly, they never appear. The run already
	// holds it; this is the picture catching up, and the screen leaves when it lands.
	boostersCarrying

	// boostersUsing: a stone has been taken out of a bag of rocks and is being used up — lifted to the
	// middle, crumbled, its dust carried to the duelist card whose ladder it raised. The run already
	// holds the raise; this is the picture, and the screen leaves when it ends. See stoneUseFrom.
	boostersUsing
)

// boosters is the dialog: which booster was opened, what was drawn from it, and how far through the
// player is.
//
// **It holds no purse and no run.** The vitae is taken by the shop the moment a booster is clicked —
// see `ShopScene.openBooster` — so what is here is the offer and the choice, exactly as the shelf
// holds a relic's key and not its price.
type boosters struct {
	// booster is the record that was opened: its name, its size, and what is inside it. **The whole
	// record rather than a key**, because every question this dialog asks of it — how many were
	// drawn, what to head the panel with, which catalog to deal — is a field on it.
	booster session.Booster
	stage   boostersStage

	// stones, essences, runes, cantrips and actions are what was drawn, and only the one matching
	// kind is filled. actions are an action bundle's playing cards, traits already rolled onto them.
	stones   []session.Stone
	essences []session.Essence
	runes    []session.Rune
	cantrips []session.Cantrip
	actions  []combat.Card

	// row is the cards an essence may be aimed at — `offer` by index into the run's deck, `selected`
	// as row slots — and the sort and the drag over them: the row every screen dealing the player's
	// cards shares, see dealtrow.go. **Only the vial fills it**, and it is dealt when the vial is
	// opened rather than when an essence is picked — the two rows are up at once, so the cards cannot
	// be a function of a choice not yet made.
	row dealtRow

	// tip explains whichever card the cursor is resting on.
	tip models.Tooltip

	// What the spent essence did, held on screen through boostersShowing. **The reward screen's own
	// fields and the same rules** — see PostBattleScene, where each of these is written up: the
	// cards change in the seats the player picked them in, and the deck is not touched until the
	// hold is over. The change itself runs on `use`.
	lands    []essenceLanding
	removes  bool
	copied   bool
	held     int
	applyNow func(*session.Session)

	// What a taken rune or cantrip looks like on its way to the pane, through boostersCarrying: its
	// face, the seat it left, the pane seat it is landing in, and the clock. **A taken playing card
	// flies to the deck pile instead** — carryTo is the pile and carrySeat is -1 — in the hand's style.
	carryFace  cards.Spec
	carryStyle cards.Style
	carryFrom  image.Rectangle
	carryTo    image.Rectangle
	carrySeat  int
	carryTrip  ui.Travel

	// use is a card being used up: a stone taken from a bag, through boostersUsing, or a vial's
	// essence absorbed in its own seat, through boostersShowing.
	use ui.Consume
}

// open puts a booster up, drawing what is inside it.
//
// **The contents are drawn here rather than by the run**, from a per-fight stream of their own, so
// a purchase interrupted by a quit leaves nothing to snapshot. Buying the same bag twice in one
// visit is not possible — see the shelf's `bought` flag — so a stream per fight is a stream per
// bag.
func (g *boosters) open(gs *state.GlobalState, booster session.Booster) {
	g.booster, g.stage = booster, boostersPick
	g.stones, g.essences, g.runes, g.cantrips, g.actions = nil, nil, nil, nil, nil
	g.row.initRow(gs, func(gs *state.GlobalState) int { return gs.PctY(boostersOfferRowPct) })
	g.tip = models.Tooltip{DwellTicks: ui.TipDwell()}

	switch booster.Contains {
	case session.ContentsStones:
		g.stones = dealStones(gs, booster.Record, booster.Size)
	case session.ContentsEssences:
		g.essences = dealVialEssences(gs, booster.Record, booster.Size)
		g.row.deal(gs, dealVialOffer(gs))
	case session.ContentsRunes:
		g.runes = dealSackRunes(gs, booster.Record, booster.Size)
	case session.ContentsCantrips:
		g.cantrips = dealScrolls(gs, booster.Record, booster.Size)
	case session.ContentsCards:
		g.actions = dealActionCards(gs, booster.Record, booster.Size)
	}
}

// roomFull is whether what this booster holds has nowhere to go: it is carried into the consumables
// pane rather than spent here, and the pane is full. **The cards are held back rather than the booster
// refused** *(owner's call, 2026-10-04)* — the player opens it, sees what is inside, and either sells
// a carried card to make room or skips. See BoostersScene.
func (g *boosters) roomFull(gs *state.GlobalState) bool {
	return gs.Run != nil && carried(g.booster.Contains) && gs.Run.ConsumablesFull()
}

// openBag reports whether anything is up.
func (g *boosters) openNow() bool { return g.stage != boostersClosed }

// close puts it away.
func (g *boosters) reset() {
	g.booster, g.stage = session.Booster{}, boostersClosed
	g.stones, g.essences, g.runes, g.cantrips = nil, nil, nil, nil
	g.row.offer, g.row.selected, g.row.hidden = nil, nil, nil
	g.lands = nil
	g.removes, g.copied, g.held, g.applyNow = false, false, 0, nil
	g.carryFace, g.carryFrom, g.carrySeat, g.carryTrip = cards.Spec{}, image.Rectangle{}, -1, ui.Travel{}
	g.carryStyle, g.carryTo = cards.Style{}, image.Rectangle{}
	g.use = ui.Consume{}
	g.tip.Forget()
}

// carry starts a taken card's flight from the seat it was picked in to its seat in the pane.
//
// **The run has already moved**: the card is held before this is called, so the seat it lands in is
// read off the pane as it now stands — the last carried thing of that kind and that record.
func (g *boosters) carry(gs *state.GlobalState, i int, face cards.Spec, kind session.ConsumableKind, key string) {
	g.carrySeat = -1
	for k, c := range heldConsumables(gs) {
		if c.Kind == kind && consumableKey(c) == key {
			g.carrySeat = k
		}
	}
	if g.carrySeat < 0 {
		g.reset()
		return
	}
	g.stage, g.carryFace, g.carryFrom = boostersCarrying, face, g.slot(gs, i)
	g.carryStyle, g.carryTo = cards.EssenceStyle, heldSeat(gs, g.carrySeat)
	g.carryTrip = ui.NewTravel(0, flightTicks())
	g.tip.Forget()
}

// carryToDeck sends the playing card taken from seat i into the deck pile — **it travels**, because a
// card never simply appears in a place, and the pile is where the deck is on this screen.
func (g *boosters) carryToDeck(gs *state.GlobalState, i int, face cards.Spec) {
	g.carrySeat = -1
	g.stage, g.carryFace, g.carryStyle = boostersCarrying, face, cards.Hand
	g.carryFrom, g.carryTo = g.slot(gs, i), deckStackRect(gs)
	g.carryTrip = ui.NewTravel(0, flightTicks())
	g.tip.Forget()
}

// landingSeat is whether a pane seat is the one a card is flying into, and so is drawn empty until
// it lands — the seat a mover is headed for is an absence, never the card arriving twice.
func (g *boosters) landingSeat(i int) bool {
	return g.stage == boostersCarrying && i == g.carrySeat
}

// drawCarry is the taken card mid-flight, shrinking from the size it was offered at to the pane's.
func (g *boosters) drawCarry(gs *state.GlobalState, screen *ebiten.Image) {
	to, st := g.carryTo, g.carryStyle
	at := ui.FlyingTo(g.carryFrom, to, g.carryTrip)
	p := ui.EaseOut(g.carryTrip.Progress())
	w := float64(g.carryFrom.Dx()) + float64(to.Dx()-g.carryFrom.Dx())*p
	h := float64(g.carryFrom.Dy()) + float64(to.Dy()-g.carryFrom.Dy())*p

	var geo ebiten.GeoM
	geo.Scale(w/float64(st.Width), h/float64(st.Height))
	geo.Translate(float64(at.X), float64(at.Y))
	ui.DrawFlyingCard(gs, screen, g.carryFace, st, geo)
}

// count is how many cards are in the row that is taken from: the stones, the runes, or the
// essences. **Not the offer**, which is a second row with its own slot function.
func (g *boosters) count() int {
	switch g.booster.Contains {
	case session.ContentsStones:
		return len(g.stones)
	case session.ContentsRunes:
		return len(g.runes)
	case session.ContentsCantrips:
		return len(g.cantrips)
	case session.ContentsCards:
		return len(g.actions)
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
// **Its own stream** (`seeds.SackStock`), separate from both other boosters and from the reward
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

// dealScrolls is what a bundle of scrolls holds: its size in cantrips, **without repeats**
// *(owner's call)*, flat and unweighted on the sack's argument.
//
// **The catalog has to be at least as long as the biggest bundle**, or a seat that says "5 cantrips"
// opens on fewer — `TestEveryScrollBundleFillsWithoutRepeats` holds it.
//
// **Its own stream** (`seeds.ScrollStock`), separate from the sack's — see internal/seeds.
func dealScrolls(gs *state.GlobalState, seat string, size int) []session.Cantrip {
	all := session.Cantrips()
	rng := rand.New(rand.NewSource(
		seeds.ForFightSeat(gs.RunSeed, seeds.ScrollStock, gs.Run.Fight(), seat)))
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	if len(all) > size {
		all = all[:size]
	}
	return all
}

// dealActionCards is what an action bundle holds: `size` of the duelist's card pool, without repeats,
// each rolled for every card trait the file prices.
//
// **Two streams of its own.** `seeds.ActionStock` deals the cards and `seeds.ActionTraits` rolls the
// traits, each split by the bundle's own record — and the traits split again by trait — so a trait's
// odds moving rerolls no card, and one bundle's contents are not the first cards of a bigger one's.
// **Flat, not weighted**, like the bag of rocks: a playing card has no rarity.
//
// **A bundle-upgrades tonic adds a third**, `seeds.ActionUpgrades`, split by the bundle: each card
// rolls once for one upgrade, picked evenly from every upgrade rune. A run that has drunk none draws
// nothing from it.
func dealActionCards(gs *state.GlobalState, seat string, size int) []combat.Card {
	pool := session.ActionPool()
	if len(pool) == 0 || gs.Run == nil {
		return nil
	}
	rng := rand.New(rand.NewSource(seeds.ForFightSeat(gs.RunSeed, seeds.ActionStock, gs.Run.Fight(), seat)))
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	out := pool[:min(size, len(pool))]

	for _, t := range session.CardTraitCatalog() {
		if t.Percent <= 0 {
			continue
		}
		roll := rand.New(rand.NewSource(seeds.ForFightSeat(gs.RunSeed, seeds.ActionTraits,
			gs.Run.Fight(), seat+"/"+t.Trait.String())))
		for i := range out {
			if roll.Intn(100) < t.Percent {
				out[i] = out[i].WithTrait(t.Trait, true)
			}
		}
	}

	if pct := gs.Run.BundleUpgradePercent(); pct > 0 {
		roll := rand.New(rand.NewSource(seeds.ForFightSeat(gs.RunSeed, seeds.ActionUpgrades, gs.Run.Fight(), seat)))
		for i := range out {
			out[i] = session.RollBundleUpgrade(out[i], pct, roll)
		}
	}
	return out
}

// runeTitle is what a rune's tooltip is headed with: its name in capitals *(owner's call)*, the
// case its own line is written in. The record keeps its name as authored, for the log and the
// ledger to say in a sentence.
func runeTitle(p session.Rune) string { return strings.ToUpper(p.Name) }

// runeTipLines is what resting on a rune says: what it does, what it would fire, and when it can be
// spent.
//
// **It is the whole of what a rune says now** *(owner's call, 2026-09-16)*. The card is a picture
// and a name; the authored line moved here rather than sitting on a scrim over the art. See
// runeSpec.
//
// **A chimera says what it would become**, in brackets under its own line, because its authored
// line cannot: its whole subject is a rune named somewhere else. On a run that has spent nothing
// there is no answer and it says only its own line.
func runeTipLines(gs *state.GlobalState, p session.Rune) []string {
	// **The card's own text, unwrapped.** An authored line break on a face is a tooltip's own line:
	// the two are the same sentence written for two widths — the treatment essenceTip already gives
	// an essence.
	lines := strings.Split(p.Text, "\n")
	if gs.Run != nil {
		if echoed := gs.Run.EchoedName(p); echoed != "" {
			lines = append(lines, "("+strings.ToUpper(echoed)+")")
		}
	}
	return lines
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
func (g *boosters) slot(gs *state.GlobalState, i int) image.Rectangle {
	n := g.count()
	if n == 0 {
		return image.Rectangle{}
	}

	top := gs.PctY(boostersChoiceRowPct)
	if g.booster.Contains == session.ContentsEssences {
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
func (g *boosters) offerSlot(gs *state.GlobalState, i int) image.Rectangle {
	return g.row.slot(gs, i)
}

// targets is how many cards an essence takes here — one, whatever the relics make of it, and never
// more than the offer is holding. See essence_targets.go.
func (g *boosters) targets(gs *state.GlobalState) int {
	return essenceTargetCount(gs, len(g.row.offer))
}

// reachNow is how many cards a click on an essence would change right now: what is selected, or the
// ceiling when nothing is — the reward screen's rule. See essenceReach.
func (g *boosters) reachNow(gs *state.GlobalState) int {
	return essenceReach(len(g.row.selected), g.targets(gs))
}

// selectedSlots is the picked cards **in row order**, whatever order they were clicked in — the
// reward screen's rule, and the combat screen's.
func (g *boosters) selectedSlots() []int {
	out := append([]int(nil), g.row.selected...)
	sort.Ints(out)
	return out
}

// selectedDeckIndexes is the offer's current picks as indexes into the run deck, in row order. The
// reward screen's function of the same name, and the same job.
func (g *boosters) selectedDeckIndexes() []int {
	out := make([]int, 0, len(g.row.selected))
	for _, slot := range g.selectedSlots() {
		if slot < 0 || slot >= len(g.row.offer) {
			return nil
		}
		out = append(out, g.row.offer[slot])
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
func (g *boosters) essenceSpendable(gs *state.GlobalState, w session.Essence) bool {
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

// update runs the booster and reports whether anything is still open. **It no longer claims the frame**
// *(2026-09-19)*: this is a screen now, so there is nothing behind it to protect and no ModalOpen to
// set. False means the card has been taken and whatever it did has finished playing.
func (g *boosters) update(gs *state.GlobalState, pile func(image.Point) bool) bool {
	if !g.openNow() {
		return false
	}

	if g.stage == boostersShowing {
		g.tickShowing(gs)
		return true
	}
	if g.stage == boostersCarrying {
		g.carryTrip.Tick()
		if g.carryTrip.Done() {
			g.reset()
			return false
		}
		return true
	}
	if g.stage == boostersUsing {
		g.use.Tick()
		if g.use.Done() {
			g.reset()
			return false
		}
		return true
	}

	// The vial's row of cards: the shared sort and drag, with a press that never travels selecting.
	if len(g.row.offer) > 0 {
		g.row.update(gs, true, func(i int) { g.selectCard(gs, i) })
	}

	g.hover(gs)
	systems.UpdateTooltip(gs, &g.tip)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && gs.CursorAllowed() {
		// **The pile is asked first and it is the scene's**, because the panel it opens covers the
		// cards this file draws — see BoostersScene.clickedPile.
		if pile == nil || !pile(image.Pt(gs.MouseX, gs.MouseY)) {
			g.click(gs)
		}
	}
	return true
}

func (g *boosters) hover(gs *state.GlobalState) {
	if !gs.CursorAllowed() {
		return
	}
	at := image.Pt(gs.MouseX, gs.MouseY)

	// **The band explains itself here as on every screen that shows it** — the relics, the carried
	// cards and the guide.
	if hoverBuildRelics(gs, at, &g.tip) {
		return
	}
	if g.row.tabs != nil && len(g.row.offer) > 0 && g.row.tabs.Hover(gs, at, &g.tip) {
		return
	}

	// **ui.HoveredSeat** — the vial deals its offer at the hand's pitch, so the row overlaps for the
	// same reason the hand does and the card on top is the last drawn.
	if i := ui.HoveredSeat(at, len(g.row.offer), func(i int) image.Rectangle {
		return g.offerSlot(gs, i)
	}); i >= 0 {
		seat := g.offerSlot(gs, i)
		if card, ok := gs.Run.Card(g.row.offer[i]); ok {
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
		switch g.booster.Contains {
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
			g.tip.Point(g.slot(gs, i), ui.TipLine(runeTitle(p)), ui.TipLines(runeTipLines(gs, p)))
		case session.ContentsCantrips:
			c := g.cantrips[i]
			g.tip.Point(g.slot(gs, i), ui.TipLine(c.Name), ui.TipLines(cantripTipLines(c)))
		case session.ContentsCards:
			c := g.actions[i]
			title, lines := ui.CardTip(c, ui.HeldByRun(gs, c))
			g.tip.Point(g.slot(gs, i), ui.TipLine(title), ui.TipLines(lines))
		}
		return
	}
}

// click is what this dialog does with a press. **A click on a card is the only input it takes
// itself** — there is no confirm, because the purchase has already happened and what is left is
// which one. The screen around it owns SKIP.
func (g *boosters) click(gs *state.GlobalState) {
	at := image.Pt(gs.MouseX, gs.MouseY)

	// **The offer row is the drag's**: a press that never travels selects, one that does reorders.
	// See dealtrow.go, and update, which runs it.
	if ui.HoveredSeat(at, len(g.row.offer), func(i int) image.Rectangle {
		return g.offerSlot(gs, i)
	}) >= 0 {
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
func (g *boosters) take(gs *state.GlobalState, i int) {
	switch g.booster.Contains {
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
		g.show(gs, essence, i, g.selectedSlots())

	case session.ContentsStones:
		stone := g.stones[i]
		if !gs.Run.UseStone(stone.Record) {
			g.reset()
			return
		}
		gs.Journal.Write(journal.Record{Kind: journal.KindTake, Key: stone.Record, Seat: i})
		trace.Logf("shop", "bag of rocks: %s, shape %s now at %d stones",
			stone.Record, stone.Shape, gs.Run.StonesOn(stone.Hands()[0]))

		// **It is used up in front of the player**: lifted out of the bag, crumbled in the middle,
		// its dust carried to the hands button — the ladder it raised — as one used from the
		// consumables pane.
		g.use = stoneUseFrom(gs, ui.StoneSpec(gs, stone, true), g.slot(gs, i))
		g.stage = boostersUsing
		g.tip.Forget()

	case session.ContentsRunes:
		// **A rune is not applied here — it goes into the sack.** That is the whole
		// difference between this booster and the other two: a stone and an essence are spent on the
		// spot, and a rune is carried into the next fight and spent between its turns.
		// **A full pane takes nothing and closes nothing**: the card is dim, and the sack stays open
		// until a carried card is sold or the booster is skipped.
		p := g.runes[i]
		if !gs.Run.Hold(p.Record) {
			return
		}
		gs.Journal.Write(journal.Record{Kind: journal.KindTake, Key: p.Record, Seat: i})
		trace.Logf("shop", "sack of runes: %s held, %d in the sack", p.Record, gs.Run.HoldCount())
		g.carry(gs, i, ui.RuneSpec(gs, p, true, false), session.ConsumableRune, p.Record)

	case session.ContentsCantrips:
		// **A cantrip goes into the scroll case**, the rune's rule: it is cast between the turns of a
		// fight, so nothing happens to the run here but the carrying.
		c := g.cantrips[i]
		if !gs.Run.HoldCantrip(c.Record) {
			return
		}
		gs.Journal.Write(journal.Record{Kind: journal.KindTake, Key: c.Record, Seat: i})
		trace.Logf("shop", "bundle of scrolls: %s held, %d in the case", c.Record, gs.Run.ScrollCount())
		g.carry(gs, i, ui.CantripSpec(gs, c, true), session.ConsumableCantrip, c.Record)

	case session.ContentsCards:
		// **A playing card goes straight into the deck** and flies to the pile — nothing is carried
		// and nothing waits for a fight, so the card the player picked is in the run before it lands.
		c := g.actions[i]
		face := ui.CardSpec(c, ui.HeldByRun(gs, c), true, false)
		gs.Run.TakeActionCard(c)
		gs.Journal.Write(journal.Record{Kind: journal.KindTake, Key: combat.ConceptOf(c.Concept).Key, Seat: i})
		trace.Logf("shop", "action bundle: %s %s taken, traits %v", c.Element, c.Label(), c.Traits.Names())
		g.carryToDeck(gs, i, face)
	}
}

// show is the vial's last stage: the essence in `seat` is used where it stands, the picked cards
// change where they stand, and the dialog closes once the player has had time to read what they
// became.
//
// **The preview runs the real essence against a throwaway copy of the run**, and the deck is not
// touched until the hold is over — both are the reward screen's rules, written up in
// PostBattleScene.aimAt. A preview computed by its own arithmetic is a preview that can disagree
// with the thing it is previewing, and a deck altered while the result is on screen would make the
// after-card impossible to draw from anything but a deck that has already moved on.
func (g *boosters) show(gs *state.GlobalState, essence session.Essence, seat int, slots []int) {
	if gs.Run == nil || len(slots) == 0 {
		return
	}

	at := make([]int, 0, len(slots))
	from := make([]image.Rectangle, 0, len(slots))
	ids := make([]int, 0, len(slots))
	for _, slot := range slots {
		if slot < 0 || slot >= len(g.row.offer) {
			return
		}
		card, ok := gs.Run.Card(g.row.offer[slot])
		if !ok {
			return
		}
		at = append(at, g.row.offer[slot])
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

	g.stage, g.held = boostersShowing, settledHoldTicks()

	// **The picked seats are the morphs' while it runs**, so the row leaves them empty — except for
	// a copy, whose original is untouched and stays drawn. PostBattleScene.settle's rule.
	g.use = essenceInPlace(gs, essence, g.slot(gs, seat), lands, g.copied)
	if !g.copied {
		g.row.hidden = make(map[int]bool, len(slots))
		for _, slot := range slots {
			g.row.hidden[slot] = true
		}
	}

	// **The commitment is by identity, not by position**, the reward screen's rule: an essence may
	// take several cards, and removing one moves every deck position above it.
	g.applyNow = func(run *session.Session) { run.ApplyToAll(essence, ids) }
	g.tip.Forget()

	trace.Logf("shop", "vial of essence: %s aimed at deck positions %v", essence.Record, at)
}

// tickShowing runs the held picture: the essence used in its seat and the cards changing in theirs,
// then the hold, and the deck edit at the end of it. **The hold does not start until every change
// has landed** — the reward screen's ordering.
func (g *boosters) tickShowing(gs *state.GlobalState) {
	if !g.use.Done() {
		g.use.Tick()
		return
	}

	g.held--
	if g.held > 0 {
		return
	}

	if g.applyNow != nil {
		g.applyNow(gs.Run)
		trace.Logf("shop", "vial of essence applied, deck now %d", gs.Run.Size())

		// **The same moment the post-battle screen raises**, because it is the same event. Skipped
		// for a removal, which leaves nothing to have wrapped.
		if !g.removes {
			earnLadderWraps(gs, g.lands)
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
func (g *boosters) selectCard(gs *state.GlobalState, i int) {
	for k, sel := range g.row.selected {
		if sel == i {
			g.row.selected = append(g.row.selected[:k], g.row.selected[k+1:]...)
			g.tip.Forget()
			return
		}
	}

	if n := g.targets(gs); len(g.row.selected) >= n {
		g.row.selected = append([]int(nil), g.row.selected[len(g.row.selected)-n+1:]...)
	}
	g.row.selected = append(g.row.selected, i)
	g.tip.Forget()
}

// drawCards puts the booster's own rows up: what was drawn from it, and — for a vial — the cards an
// essence may be aimed at.
//
// **The screen around it is the scene's** *(2026-09-19)*. This was a modal frame with a title bar
// and a scrim; the ground, the build band, the heading and the draw pile are BoostersScene's now, and
// what is left here is the thing the booster actually is.
func (g *boosters) drawCards(gs *state.GlobalState, screen *ebiten.Image) {
	if !g.openNow() || gs.Run == nil {
		return
	}

	if g.stage == boostersShowing {
		g.row.draw(gs, screen)
		g.use.Draw(gs, screen)
		return
	}
	if g.stage == boostersCarrying {
		g.drawCarry(gs, screen)
		return
	}
	if g.stage == boostersUsing {
		g.use.Draw(gs, screen)
		return
	}

	for i := 0; i < g.count(); i++ {
		at := g.slot(gs, i).Min
		switch g.booster.Contains {
		case session.ContentsStones:
			ui.DrawStoneCard(gs, screen, at, g.stones[i], true)
		case session.ContentsRunes:
			ui.DrawRuneCard(gs, screen, at, g.runes[i], !g.roomFull(gs), false)
		case session.ContentsCantrips:
			ui.DrawCantripCard(gs, screen, at, g.cantrips[i], !g.roomFull(gs))
		case session.ContentsCards:
			// **An effortless card is drawn aloft here too**, so the bundle says what the card is
			// before it is taken; a mirrored one carries its echo in its face.
			c := g.actions[i]
			ui.DrawSeatedCard(gs, screen, at, i, ui.CardSpec(c, ui.HeldByRun(gs, c), true, false),
				cards.Hand, !c.Counted())
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
	if len(g.row.offer) > 0 {
		g.row.draw(gs, screen)
	}
}

// title names what is open, and hint says what to do with it. **Two short lines rather than a
// paragraph**: the cards say what they are, and this says how many of them the player gets.
//
// **Both come off the record now** *(2026-09-14)*, resolved once at load — see session.Booster, where a
// blank Title becomes the booster's Name and a blank Hint becomes session.DefaultBoosterHint, with the figure the record's own Size rather than a number authored twice. The vial is
// still the exception and still says one thing *(owner's call, 2026-09-05)*: the essences on the
// table are the whole of what the dialog is, so it heads itself with an instruction rather than a
// label with a caption. It authors both fields to say so.
func (g *boosters) title() string {
	if g.stage == boostersShowing {
		if g.removes {
			return "EATEN"
		}
		return "CHANGED"
	}
	return g.booster.Title
}

// hint is the line under the title: the record's own, plus how far an essence reaches when a relic
// has widened it.
//
// **The reach is said in words because nothing else on the screen says it** — the row lights the
// essences only once it has enough cards, which says when and never how many. "Up to", because the
// reach is a ceiling rather than a quota; see consumableTarget.fewest.
func (g *boosters) hint(gs *state.GlobalState) string {
	line := g.booster.Hint
	if g.booster.Contains != session.ContentsEssences {
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
// share of that rung's *own* multiplier, so a Card Three of a Kind and a Form Three of a Kind move
// by different amounts off the same rock, and a single "+N" would be true of neither.
//
// **The run's own figure rather than the catalog's**, because what the next stone adds depends on
// how many the run already holds, and the player has no other way to see what the last one did.
func stoneTipLines(gs *state.GlobalState, st session.Stone) []string {
	var out []string
	for _, hand := range st.Hands() {
		have := 0
		if gs.Run != nil {
			have = gs.Run.StonesOn(hand)
		}
		worth := session.StoneWorth(hand, have)
		line := fmt.Sprintf("%s +%s", stoneHandName(hand), stoneMultiplier(worth))
		if gs.Run != nil {
			// **The rung this stone makes, and what it adds to get there**: the level counted as the
			// hands panel counts it — level 1 is the rung as shipped, so a stone never shows it — and
			// the multiplier at that level, with the stone's share beside it.
			if now, ok := gs.Run.HandMultiplier(hand); ok {
				line = fmt.Sprintf("%s Level %d: %s (+%s)", stoneHandName(hand), have+2,
					stoneMultiplier(now+worth), stoneMultiplier(worth))
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

// stoneMultiplier is a rung's multiplier as the player reads it, to the tenth: 110 is `1.1`, 100 is
// `1.0`. **Always a tenth, never more** — every multiplier is a multiple of 10, which the hand loader
// holds — and the `.0` is kept so a rung that has not moved still reads as a multiplier.
func stoneMultiplier(pct int) string {
	return fmt.Sprintf("%d.%d", pct/100, pct%100/10)
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
