package screens

// Spending a parasite: the run's half, with no dialog in front of it.
//
// **A worm is spent between rooms and a parasite is spent between turns**, which is most of what
// makes them different things. The catalogue and the rules are `internal/session/parasite.go`; this
// file is what the screen does with them, and it decides nothing — it hands the run a parasite and
// the cards it names, and brings the hand back into line with what the run says afterwards.
//
// **The `P` button and its dialog are gone** *(owner's call, 2026-09-06)*. A parasite is now a card
// standing in the consumables pane on the top row, clicked directly: select the cards in the hand,
// then click the parasite. See targeting.go for the rule that joins those two halves, and
// consumables.go for the pane. What that removed is a modal, a button, a two-stage prompt and a
// second drawing of the hand — the panel used to redraw the row of cards the player was already
// looking at, one row lower, so that it could be clicked.
//
// **It is still only live while `planning()`.** A parasite alters the deck, and the deck a round was
// resolved against is the deck that round has to be replayed with — `ResolveRound` decides
// everything before a frame of playback runs, so a card changed mid-playback would put a face on
// screen that disagrees with the blow already computed. Losing the dialog did not loosen that; it
// moved the predicate onto the pane.
//
// **Targets come out of the hand, not out of the whole deck** *(a call taken while building it, and
// the one most worth revisiting)*. Two arguments for it: a mid-fight consumable aimed at the card
// you are about to play is a decision about *this* turn, where one aimed at a card somewhere in a
// pile of forty is the reward screen's decision taken in a worse place; and the hand is already on
// screen. The second argument is stronger now than it was — the hand is not merely on screen, it is
// the row the targets are selected in.

import (
	"image"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// heldParasites is what the run is carrying, as records, in acquisition order.
//
// **A parasite the catalogue no longer holds is skipped rather than drawn blank.** `Session.Hold`
// refuses one on the way in and `Resume` refuses a save carrying one, so this cannot fire today —
// it is the belt to those braces, because a nil record reaching the card renderer is a crash where
// a missing card is a gap.
func heldParasites(gs *state.GlobalState) []session.Parasite {
	if gs.Run == nil {
		return nil
	}
	keys := gs.Run.Held()
	out := make([]session.Parasite, 0, len(keys))
	for _, key := range keys {
		if p, ok := session.ParasiteByKey(key); ok {
			out = append(out, p)
		}
	}
	return out
}

// parasiteSpec is a parasite drawn as a card.
//
// **It borrows the worm's picture**, `wormArtKey`, because there is no parasite art and a card with
// no face at all would be worse than one wearing a placeholder its sibling already wears. It is one
// constant rather than a field on the record for the reason the worm's is: when parasites get art
// it becomes a key per parasite, and that should be a `data/parasites.json` field appearing rather
// than a fallback being unpicked.
// chimeraBreak is the authored line break on a chimera's face — see cards.WrapText, which honours
// one. It is a constant rather than a literal so the escape does not have to survive being read
// back out of this file.
const chimeraBreak = "\n"

// **A chimera says what it would fire**, because its authored line cannot: the card's whole subject
// is a parasite named somewhere else, and "COPIES THE LAST" is a card the player has to remember
// the answer to. On a run that has spent nothing there is no answer, and it keeps its own line — a
// card that is about to be drawn dim anyway.
func parasiteSpec(gs *state.GlobalState, p session.Parasite, enabled, selected bool) cards.Spec {
	text := p.Text
	if gs.Run != nil {
		if echoed := gs.Run.EchoedName(p); echoed != "" {
			text = "COPIES" + chimeraBreak + strings.ToUpper(echoed)
		}
	}

	return cards.Spec{
		Name:     p.Name,
		Form:     cards.FormNone,
		Element:  cards.Basic,
		Art:      artwork(gs, wormArtKey),
		Text:     text,
		Enabled:  enabled,
		Selected: selected,
	}
}

// parasiteRowGap is the air between two cards in a row of them. The same gap the shop's shelf
// takes, so a row of cards reads the same wherever it stands.
const parasiteRowGap = 18

// parasiteRowSlots is the left edges of n cards laid out in a centred row.
//
// **It tightens rather than overflowing**: the pitch closes up exactly as the hand's does rather
// than the row running off both edges of the panel.
//
// **Its one caller is the shop's pouch now** *(2026-09-06)*, the parasite dialog this was written
// for having gone. It stays here rather than moving because the pouch's row is the same row of
// full-size cards in a modal, and a second copy is what would drift.
func parasiteRowSlots(r image.Rectangle, n int) []int {
	if n <= 0 {
		return nil
	}

	pitch := cards.Hand.Width + parasiteRowGap
	if width := r.Dx() - 2*parasiteRowGap; n*pitch > width {
		pitch = width / n
	}

	left := r.Min.X + r.Dx()/2 - (n*pitch-parasiteRowGap)/2
	out := make([]int, n)
	for i := range out {
		out[i] = left + i*pitch
	}
	return out
}

// parasiteCardRects is where each card of a row stands.
func parasiteCardRects(r image.Rectangle, n, centreY int) []image.Rectangle {
	slots := parasiteRowSlots(r, n)
	out := make([]image.Rectangle, len(slots))
	for i, x := range slots {
		top := centreY - cards.Hand.Height/2
		out[i] = image.Rect(x, top, x+cards.Hand.Width, top+cards.Hand.Height)
	}
	return out
}

// canSpendParasites is the one predicate for "a parasite may be spent at all".
//
// **`planning()` is the whole of it now.** It used to also ask whether the bucket had anything in
// it, because an empty dialog was something to open; the pane draws its two seats empty or full and
// there is nothing to open, so what is left is the rule that keeps an alteration out of a round that
// has already been resolved.
func (s *CombatScene) canSpendParasites(gs *state.GlobalState) bool {
	return gs.Run != nil && s.planning() && !s.modalUp()
}

// selectedCardIDs is what the player has selected in the hand, by identity, in row order.
//
// **Row order, because that is the order the queue is read in** — see syncQueue, which walks the
// same list. A parasite naming a first and a second target reads them left to right, and the player
// reorders by dragging, exactly as they reorder the round.
func (s *CombatScene) selectedCardIDs() []int {
	out := make([]int, 0, len(s.hand))
	for _, c := range s.hand {
		if c.selected {
			out = append(out, c.actionCard.ID)
		}
	}
	return out
}

// parasiteTarget is what one held parasite needs from the selection.
//
// **The legality question goes to the run**, which is the only thing that knows whether these
// particular cards can take it — see `Session.CanApplyParasite`, which is also what the apply itself
// checks, so a parasite that lit up cannot then be refused.
// **A chimera is asked through the run as well.** Its own record names no cards; how many it wants
// comes from whatever it is copying, so the count is read off `Session.Echoes` rather than off the
// card in the pane. A chimera with nothing to copy resolves to a parasite that cannot be satisfied,
// which is what draws it dim.
func (s *CombatScene) parasiteTarget(gs *state.GlobalState, p session.Parasite) consumableTarget {
	echoed, ok := gs.Run.Echoes(p)
	if !ok {
		return consumableTarget{needs: -1, legal: func([]int) bool { return false }}
	}
	return consumableTarget{
		needs: echoed.Count,
		legal: func(ids []int) bool { return gs.Run.CanApplyParasite(p, ids) },
	}
}

// spendParasite hands one to the run against the cards the player has selected.
//
// **Apply, then drop, and only drop if the apply succeeded.** A parasite dropped from the bucket by
// an application that then refused would be a consumable the player paid for and did not get;
// `ApplyParasite` is all-or-nothing, so asking it first is what makes the pair safe.
func (s *CombatScene) spendParasite(gs *state.GlobalState, i int) {
	held := heldParasites(gs)
	if i < 0 || i >= len(held) {
		return
	}
	p := held[i]

	ids := s.selectedCardIDs()
	if !s.parasiteTarget(gs, p).satisfiedBy(ids) {
		return
	}
	if !gs.Run.ApplyParasiteRolling(p, ids, s.parasiteRNG(gs, p)) {
		return
	}
	gs.Run.Drop(i)

	// **The hand is rebuilt from the run, because the cards in it are copies.** `s.hand` holds
	// values dealt off the deck at the start of the round; a rider attached to the run's card would
	// otherwise not appear on the card the player is looking at until the next deal, and a removed
	// card would still be sitting in the row.
	s.resyncHandFromRun(gs)

	// **A copy joins the hand it was copied from** *(owner's call, 2026-09-02)*. The worm version
	// of this only has to put a card in the deck, because it is spent between fights; a parasite is
	// spent in the middle of one, and the fight's piles were dealt before it existed — so a copy
	// that went only into the run would not be playable until the next fight and would read as a
	// dud. `resyncHandFromRun` cannot do it, because it walks the hand and the copy is not in it.
	//
	// **It arrives unselected, whatever the card it came from was doing.** A copy that queued
	// itself would spend action points the player had not committed.
	for _, copied := range gs.Run.Duplicated() {
		s.hand = append(s.hand, paletteCard{actionCard: copied})
	}
	s.syncQueue()

	// **A rock shower's stones fly to the duelist card rather than stopping the screen**
	// *(owner's call, 2026-09-06)*. They are in the run's pouch by the time this runs — there is
	// nothing to choose and nothing to confirm — so what is owed the player is a picture of where
	// they went, not a panel to dismiss. See stoneflight.go.
	if shown := gs.Run.Granted(); len(shown) > 0 {
		s.flyStonesToPouch(gs, i, shown)
	}

	saveRun(gs)
}

// parasiteRNG is the source the parasite about to be spent draws from, and nil for the ones that
// draw nothing.
//
// **It is picked off the *resolved* parasite**, so a chimera copying a rock shower gets the
// shower's stream rather than none — which is the whole reason this is a switch rather than the
// single `showerRNG` it replaced.
//
// **Two streams and never one.** Sharing would make what a gamble grants a function of how many
// rock showers the run had spent, and vice versa; the `randomness` skill's test is what a shared
// stream would silently reroll, and the answer here is "both of them".
func (s *CombatScene) parasiteRNG(gs *state.GlobalState, p session.Parasite) *rand.Rand {
	if gs.Run == nil {
		return nil
	}
	echoed, ok := gs.Run.Echoes(p)
	if !ok {
		return nil
	}

	switch echoed.Target {
	case session.ParasiteStones:
		return s.showerRNG(gs)
	case session.ParasiteLuck:
		return s.luckRNG(gs)
	default:
		return nil
	}
}

// showerRNG is the source a rock shower draws its stones from.
//
// **Its own salted stream, plus the number of stones the run has already placed** — see
// `seeds.StoneShower`. The fight index alone is not enough here, because a run may carry three
// showers and spend all three in one fight; the placed count is a number the snapshot already
// carries, so a resumed run rolls what it would have rolled.
func (s *CombatScene) showerRNG(gs *state.GlobalState) *rand.Rand {
	placed := 0
	for _, n := range gs.Run.StoneCounts() {
		placed += n
	}
	seed := seeds.ForFight(gs.RunSeed, seeds.StoneShower, gs.Run.Fight()) + int64(placed)*stoneShowerStride
	return rand.New(rand.NewSource(seed))
}

// luckRNG is the source a luck parasite rolls against.
//
// **The shower's shape, with the run's roll count as the cursor** — see `seeds.LuckRoll`. The count
// steps on a dud as well as on a win, which is what stops two consecutive empty rolls being seeded
// identically and coming up empty for ever.
func (s *CombatScene) luckRNG(gs *state.GlobalState) *rand.Rand {
	seed := seeds.ForFight(gs.RunSeed, seeds.LuckRoll, gs.Run.Fight()) +
		int64(gs.Run.LuckRolls())*luckRollStride
	return rand.New(rand.NewSource(seed))
}

// luckRollStride separates one gamble from the next inside a fight, on the argument
// `stoneShowerStride` is under. A different number from the shower's, so two consumables spent at
// the same count do not land on neighbouring seeds.
const luckRollStride int64 = 0x7F4A_7C15

// stoneShowerStride separates one shower from the next inside a fight. A large odd number, on the
// argument `seeds.fightStride` is under: consecutive draws should not be consecutive seeds.
const stoneShowerStride int64 = 0x3B9A_CA07

// resyncHandFromRun brings the hand back in line with the run's deck after a parasite has altered
// it: an altered card is redrawn as it now is, and a card the run no longer owns leaves the row.
//
// **It walks by identity**, which is the whole reason a card has one. A card in the hand is a copy —
// and, with a flip ring worn, a copy in a colour the run's card never had — so the match cannot be
// made by looking at the two cards.
//
// **A card the run has lost is dropped from the hand, the queue and the selection together.**
// Leaving it in the queue would put a card into a round the player does not own.
func (s *CombatScene) resyncHandFromRun(gs *state.GlobalState) {
	kept := make([]paletteCard, 0, len(s.hand))
	for _, c := range s.hand {
		owned, ok := gs.Run.CardByID(c.actionCard.ID)
		if !ok {
			continue
		}
		// **The element is the hand's, not the run's.** A flip ring recoloured this card as it was
		// drawn and that colour is a fact about the card in play; taking the run's colour back
		// would undo a ring mid-round.
		owned.Element = c.actionCard.Element
		kept = append(kept, paletteCard{actionCard: owned, selected: c.selected})
	}

	s.hand = kept
	s.syncQueue()
}

// parasiteRiderLine is what a rider is called in the fight log and anywhere else a sentence has to
// name one. It is here rather than in prose.go because the vocabulary is the parasite's.
func parasiteRiderLine(k combat.RiderKind) string {
	switch k {
	case combat.RiderHealOnPlay:
		return "heals its owner"
	default:
		return "does nothing"
	}
}

// parasiteSpendable is the pane's "would clicking this do anything" predicate on this screen.
//
// **The card's lit state and the click read the same function**, which is what stops a control
// looking available and doing nothing.
func (s *CombatScene) parasiteSpendable(gs *state.GlobalState) func(session.Parasite) bool {
	if !s.canSpendParasites(gs) {
		return nil
	}
	ids := s.selectedCardIDs()
	return func(p session.Parasite) bool {
		return s.parasiteTarget(gs, p).satisfiedBy(ids)
	}
}

// updateConsumables runs the click on the consumables pane. Called every tick from Update.
//
// **The gate is the same predicate the card's own lit state reads**, so a card that is drawn dim
// cannot be clicked and a card that is lit always works. Two predicates here is how a control comes
// to look available and do nothing.
func (s *CombatScene) updateConsumables(gs *state.GlobalState) {
	if !s.canSpendParasites(gs) || !gs.CursorAllowed() {
		return
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	at := image.Pt(gs.MouseX, gs.MouseY)
	i := consumableClicked(gs, s.consumablePaneRect(gs), at)
	if i < 0 {
		return
	}
	s.spendParasite(gs, i)
}
