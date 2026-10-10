package screens

// The potions pane, and the tonic seat beside it.
//
// **A potion is the one thing in the shop that changes the duelist** *(owner's call, 2026-09-06)*.
// A relic is worn, a stone raises a rung, an essence eats a card, a rune waits in a sack; a potion
// is drunk on the spot and moves one of the three figures the fighter is made of. See
// `internal/session/potion.go`, which owns what each one does and refuses a record the rules cannot
// apply.
//
// **The whole catalog is on the shelf every visit, and it does not reroll** *(owner's call)*.
// There is no draw here and so no stream: three vessels, always the same three, in the order
// `data/potions.json` writes them. A reroll would be asking to be offered the thing that is already
// being offered.
//
// **A potion is bought with one click and no confirm**, the shelf's rule rather than the worn row's:
// the price is on the card, the purse cannot go into debt, and there is nothing a confirmation
// would protect. What it does is immediate and visible on the duelist card two hundred pixels
// above — a Salve moves the life fraction, a Draught the DMG line, an Elixir both.
//
// **The tonic seat holds the realm's one tonic** — see internal/session/tonic.go. It is bought on
// the potions' terms, one click and no confirm, and once it is drunk the seat stands empty until the
// run walks through a portal into the next realm.

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/journal"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// shopPotions is the catalog as the shelf stands it. **Asked of the run's package rather than
// held on the scene**: every visit offers all three, so there is nothing to deal and nothing to
// remember between visits.
func shopPotions() []session.Potion { return session.Potions() }

// shopPrice is what the shop charges for something listed at base, through every discount the run
// has drunk. A scene with no run charges the list price.
func shopPrice(gs *state.GlobalState, base int) int {
	if gs.Run == nil {
		return base
	}
	return gs.Run.Price(base)
}

// shopTonic is the tonic on offer in the seat, if there is one.
func shopTonic(gs *state.GlobalState) (session.Tonic, bool) {
	if gs.Run == nil {
		return session.Tonic{}, false
	}
	return gs.Run.TonicOnOffer()
}

// potionSeat and tonicSeat are where a card in either pane is drawn and clicked.
func potionSeat(gs *state.GlobalState, i int) image.Rectangle {
	return shopSeatRect(gs, shopPanePotions, i)
}

func tonicSeat(gs *state.GlobalState) image.Rectangle {
	return shopSeatRect(gs, shopPaneTonic, 0)
}

// drawPotions puts the three vessels up with their prices under them, on the shelf's terms: one
// that cannot be afforded is dimmed rather than hidden, so the pane still says what was offered.
func (s *ShopScene) drawPotions(gs *state.GlobalState, screen *ebiten.Image) {
	drawShopPaneBack(gs, screen, shopPanePotions)

	for i, p := range shopPotions() {
		at := potionSeat(gs, i)
		if s.drunk[p.Record] {
			continue
		}

		lit := gs.Run != nil && gs.Run.CanDrink(p.Record)
		ui.BlitCard(gs, screen, at.Min, potionSpec(gs, p, lit), cards.EssenceStyle)
		s.figure(gs, screen, at, shopPrice(gs, p.Price), lit)
	}
}

// drawTonic puts the realm's tonic up, or leaves the pane bare when there is none — nothing left to
// offer, or this realm's already drunk. **The pane is drawn either way**, so the row keeps its shape
// and an empty seat reads as an empty seat.
func (s *ShopScene) drawTonic(gs *state.GlobalState, screen *ebiten.Image) {
	drawShopPaneBack(gs, screen, shopPaneTonic)

	t, ok := shopTonic(gs)
	if !ok {
		return
	}
	at := tonicSeat(gs)
	lit := gs.Run.CanDrinkTonic(t.Record)
	ui.BlitCard(gs, screen, at.Min, tonicSpec(gs, t, lit), cards.EssenceStyle)
	s.figure(gs, screen, at, shopPrice(gs, t.Price), lit)
}

// potionSpec is a potion drawn as a card.
//
// **Basic, not an element.** A potion changes the duelist rather than a card, and the duelist is not
// a color — so its border is the mid gray `cards.BorderOf` gives `basic`, exactly as a stone's and
// a sealed good's are. The hue wheel is full and a fourth kind of shelf card cannot have one.
//
// **The picture comes off the record** *(2026-09-14)*, through `data.PotionData.ArtKey` — the shape
// the relics, essences and runes are already in.
//
// **The picture is now the whole card** *(owner's call, 2026-09-15)*. It carried "HEAL 15 / LIFE"
// on a scrim across the lower half until then, which is the one thing potionTip already says at
// full length — so the band was covering a painted bottle to repeat, in two clipped words, what
// resting on it explains. A shelf card the player can reach is a card whose tooltip can carry the
// prose; what the face is for is being recognized.
func potionSpec(gs *state.GlobalState, p session.Potion, enabled bool) cards.Spec {
	return cards.Spec{
		Name:    p.Name,
		Form:    cards.FormNone,
		Cost:    0,
		Element: ui.ArtFor(combat.Basic),
		Art:     ui.Artwork(gs, p.Art),
		Enabled: enabled,
	}
}

// tonicSpec is a tonic drawn as a card, on a potion's terms: basic, the picture as the whole face,
// and what it does in the tooltip.
func tonicSpec(gs *state.GlobalState, t session.Tonic, enabled bool) cards.Spec {
	return cards.Spec{
		Name:    t.Name,
		Form:    cards.FormNone,
		Cost:    0,
		Element: ui.ArtFor(combat.Basic),
		Art:     ui.Artwork(gs, t.Art),
		Enabled: enabled,
	}
}

// drinkTonic pays for the realm's tonic and drinks it. The seat empties until the next realm,
// because the run no longer has it on offer.
func (s *ShopScene) drinkTonic(gs *state.GlobalState, key string) {
	if gs.Run == nil || !gs.Run.DrinkTonic(key) {
		return
	}
	s.tip.Forget()
	gs.Journal.Write(journal.Record{Kind: journal.KindTonic, Key: key})
}

// drinkPotion pays for one and applies it.
//
// **The purse moves inside `Drink`**, which is the one place a refusal happens — so a potion the
// run could not afford changes nothing at all, and the seat it was clicked in stays full.
func (s *ShopScene) drinkPotion(gs *state.GlobalState, key string) {
	if gs.Run == nil || s.drunk[key] || !gs.Run.CanDrink(key) {
		return
	}
	if !gs.Run.Drink(key) {
		return
	}

	// **The seat empties for the rest of the visit**, the sealed goods' rule rather than the relic
	// shelf's: a potion is not taken off a shelf of one-offs, but three of one bottle bought in a
	// row would be a shop that sells the same thing until the purse is empty.
	s.drunk[key] = true
	s.tip.Forget()

	gs.Journal.Write(journal.Record{Kind: journal.KindPotion, Key: key})
}

// potionTip is what resting on one says. **It is the whole of what the card says** *(owner's call,
// 2026-09-15)* — the face is a painted bottle and a price, so the figure, what it moves and when it
// is drunk are all read here, by a player who on realm one has never seen a potion.
//
// price is what the shop charges for it, discounts included.
func potionTip(p session.Potion, price int) (string, []string) {
	var what string
	switch p.Effect {
	case session.PotionHeal:
		what = fmt.Sprintf("heals %d of the wound you are carrying", p.Amount)
	case session.PotionDMG:
		what = fmt.Sprintf("adds %d to your DMG for the rest of the run", p.Amount)
	default:
		what = fmt.Sprintf("raises your max life by %d for the rest of the run", p.Amount)
	}
	return p.Name, []string{what, "drunk on the spot", fmt.Sprintf("%d vitae", price)}
}

// tonicTip is what resting on the tonic says: what it changes, that it lasts the whole run, and its
// price. **The whole of what the card says**, as a potion's tip is.
func tonicTip(t session.Tonic, price int) (string, []string) {
	var what []string
	switch t.Effect {
	case session.TonicDiscount:
		what = []string{fmt.Sprintf("everything in the shop costs %d%% less", t.Amount), "and at least 1 vitae less"}
	case session.TonicCostCut:
		what = []string{fmt.Sprintf("your cards cost %d AP less", t.Amount), "but never less than 1 AP"}
	case session.TonicPressure:
		what = []string{"every fight is 1 round shorter", fmt.Sprintf("and every vitae you earn is x%d", t.Amount)}
	case session.TonicHandSize:
		what = []string{fmt.Sprintf("%d more card in every hand", t.Amount)}
	case session.TonicDiscards:
		what = []string{fmt.Sprintf("%d more discard every round", t.Amount)}
	case session.TonicRelicSlots:
		what = []string{fmt.Sprintf("wear %d more relic", t.Amount)}
	}
	what = append(what, "for the rest of the run")
	return t.Name, append(what, fmt.Sprintf("%d vitae", price))
}
