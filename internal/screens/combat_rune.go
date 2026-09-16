package screens

// Spending a rune: the run's half, with no dialog in front of it.
//
// **An essence is spent between rooms and a rune is spent between turns**, which is most of what
// makes them different things. The catalog and the rules are `internal/session/rune.go`; this
// file is what the screen does with them, and it decides nothing — it hands the run a rune and
// the cards it names, and brings the hand back into line with what the run says afterwards.
//
// **The `P` button and its dialog are gone** *(owner's call, 2026-09-06)*. A rune is now a card
// standing in the consumables pane on the top row, clicked directly: select the cards in the hand,
// then click the rune. See targeting.go for the rule that joins those two halves, and
// consumables.go for the pane. What that removed is a modal, a button, a two-stage prompt and a
// second drawing of the hand — the panel used to redraw the row of cards the player was already
// looking at, one row lower, so that it could be clicked.
//
// **It is still only live while `planning()`.** A rune alters the deck, and the deck a round was
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

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// heldRunes is what the run is carrying, as records, in acquisition order.
//
// **A rune the catalog no longer holds is skipped rather than drawn blank.** `Session.Hold`
// refuses one on the way in and `Resume` refuses a save carrying one, so this cannot fire today —
// it is the belt to those braces, because a nil record reaching the card renderer is a crash where
// a missing card is a gap.
func heldRunes(gs *state.GlobalState) []session.Rune {
	if gs.Run == nil {
		return nil
	}
	keys := gs.Run.Held()
	out := make([]session.Rune, 0, len(keys))
	for _, key := range keys {
		if p, ok := session.RuneByKey(key); ok {
			out = append(out, p)
		}
	}
	return out
}

// runeSpec is a rune drawn as a card.
//
// **The picture comes off the record** *(2026-09-12)*, through `data.RuneData.ArtKey`, already
// resolved by the time a `session.Rune` exists. It borrowed the essence's placeholder through one
// constant until then, and the note on that constant said the day runes got art it should be a
// `data/runes.json` field appearing rather than a fallback being unpicked — so the fallback is
// `assets/rune/default-rune.png` now, a seat of the catalog's own.
//
// **The card says nothing at all** *(owner's call, 2026-09-16)*. It carried its authored line
// across the lower half of its picture, and what that cost is the picture: a rune is a full-bleed
// card, so the sentence is a scrim over the one thing on the card worth looking at. The line has
// not gone anywhere — it is in the tooltip, which is where a player who does not recognize a
// picture yet already goes, and where there is room to say the half a face cannot fit. Same call
// the sealed goods took on 2026-09-15 and the stones took with this one.
//
// **The chimera's answer went with it.** Its face printed `COPIES <name>` because its authored line
// cannot say what it would fire; that is now the tooltip's first line — see runeTipLines.
func runeSpec(gs *state.GlobalState, p session.Rune, enabled, selected bool) cards.Spec {
	return cards.Spec{
		Name:     p.Name,
		Form:     cards.FormNone,
		Element:  cards.Basic,
		Art:      artwork(gs, p.Art),
		Enabled:  enabled,
		Selected: selected,
	}
}

// runeRowGap is the air between two cards in a row of them. The same gap the shop's shelf
// takes, so a row of cards reads the same wherever it stands.
const runeRowGap = 18

// runeRowSlots is the left edges of n cards laid out in a centered row.
//
// **It tightens rather than overflowing**: the pitch closes up exactly as the hand's does rather
// than the row running off both edges of the panel.
//
// **Its one caller is the shop's pouch now** *(2026-09-06)*, the rune dialog this was written
// for having gone. It stays here rather than moving because the pouch's row is the same row of
// full-size cards in a modal, and a second copy is what would drift.
func runeRowSlots(r image.Rectangle, n int) []int {
	if n <= 0 {
		return nil
	}

	pitch := cards.Hand.Width + runeRowGap
	if width := r.Dx() - 2*runeRowGap; n*pitch > width {
		pitch = width / n
	}

	left := r.Min.X + r.Dx()/2 - (n*pitch-runeRowGap)/2
	out := make([]int, n)
	for i := range out {
		out[i] = left + i*pitch
	}
	return out
}

// runeCardRects is where each card of a row stands.
func runeCardRects(r image.Rectangle, n, centerY int) []image.Rectangle {
	slots := runeRowSlots(r, n)
	out := make([]image.Rectangle, len(slots))
	for i, x := range slots {
		top := centerY - cards.Hand.Height/2
		out[i] = image.Rect(x, top, x+cards.Hand.Width, top+cards.Hand.Height)
	}
	return out
}

// canSpendRunes is the one predicate for "a rune may be spent at all".
//
// **`planning()` is the whole of it now.** It used to also ask whether the sack had anything in
// it, because an empty dialog was something to open; the pane draws its two seats empty or full and
// there is nothing to open, so what is left is the rule that keeps an alteration out of a round that
// has already been resolved.
func (s *CombatScene) canSpendRunes(gs *state.GlobalState) bool {
	return gs.Run != nil && s.planning() && !s.modalUp()
}

// selectedCardIDs is what the player has selected in the hand, by identity, in row order.
//
// **Row order, because that is the order the queue is read in** — see syncQueue, which walks the
// same list. A rune naming a first and a second target reads them left to right, and the player
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

// runeTarget is what one held rune needs from the selection.
//
// **The legality question goes to the run**, which is the only thing that knows whether these
// particular cards can take it — see `Session.CanApplyRune`, which is also what the apply itself
// checks, so a rune that lit up cannot then be refused.
// **A chimera is asked through the run as well.** Its own record names no cards; how many it wants
// comes from whatever it is copying, so the count is read off `Session.Echoes` rather than off the
// card in the pane. A chimera with nothing to copy resolves to a rune that cannot be satisfied,
// which is what draws it dim.
func (s *CombatScene) runeTarget(gs *state.GlobalState, p session.Rune) consumableTarget {
	echoed, ok := gs.Run.Echoes(p)
	if !ok {
		return consumableTarget{needs: -1, legal: func([]int) bool { return false }}
	}
	return consumableTarget{
		needs: echoed.Count,
		legal: func(ids []int) bool { return gs.Run.CanApplyRune(p, ids) },
	}
}

// spendRune hands one to the run against the cards the player has selected.
//
// **Apply, then drop, and only drop if the apply succeeded.** A rune dropped from the sack by
// an application that then refused would be a consumable the player paid for and did not get;
// `ApplyRune` is all-or-nothing, so asking it first is what makes the pair safe.
func (s *CombatScene) spendRune(gs *state.GlobalState, i int) {
	held := heldRunes(gs)
	if i < 0 || i >= len(held) {
		return
	}
	p := held[i]

	ids := s.selectedCardIDs()
	if !s.runeTarget(gs, p).satisfiedBy(ids) {
		return
	}

	// **The hand as it stands, before any of this lands.** What the rune changed is the
	// difference between this and the hand a few lines below, which is what lets the morphs be
	// raised without this file knowing what any particular rune does. See raiseHandMorphs.
	was, seats := s.handFaces(gs)

	if !gs.Run.ApplyRuneRolling(p, ids, s.runeRNG(gs, p)) {
		return
	}
	gs.Run.Drop(i)

	// **The hand is rebuilt from the run, because the cards in it are copies.** `s.hand` holds
	// values dealt off the deck at the start of the round; a rider attached to the run's card would
	// otherwise not appear on the card the player is looking at until the next deal, and a removed
	// card would still be sitting in the row.
	s.resyncHandFromRun(gs)

	// **A copy joins the hand it was copied from** *(owner's call, 2026-09-02)*. The essence version
	// of this only has to put a card in the deck, because it is spent between fights; a rune is
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

	// **The change is shown where it happened**, on the cards in the row rather than in a panel
	// about them. Raised after the hand is final — the resync, the copies and the queue are all
	// done — so every morph is a ghost of something that has already happened, which is the rule
	// every mover on this screen is under. See combat_handmorph.go.
	s.raiseHandMorphs(gs, was, seats)

	// **A rock shower's stones fly to the duelist card rather than stopping the screen**
	// *(owner's call, 2026-09-06)*. They are in the run's pouch by the time this runs — there is
	// nothing to choose and nothing to confirm — so what is owed the player is a picture of where
	// they went, not a panel to dismiss. See stoneflight.go.
	if shown := gs.Run.Granted(); len(shown) > 0 {
		s.flyStonesToPouch(gs, i, shown)
	}

	saveRun(gs)
}

// runeRNG is the source the rune about to be spent draws from, and nil for the ones that
// draw nothing.
//
// **It is picked off the *resolved* rune**, so a chimera copying a rock shower gets the
// shower's stream rather than none — which is the whole reason this is a switch rather than the
// single `showerRNG` it replaced.
//
// **One stream today and it is still asked for by target rather than assumed.** The gamble used to
// be the second caller and moved into the resolver on 2026-09-09, when it stopped being a
// consumable and became something a card permanently carries — see combat.RiderGolden. What is left
// is the shower, and the shape stays because the question "which stream does this rune draw
// from" is the one a second rolling target has to answer again.
func (s *CombatScene) runeRNG(gs *state.GlobalState, p session.Rune) *rand.Rand {
	if gs.Run == nil {
		return nil
	}
	echoed, ok := gs.Run.Echoes(p)
	if !ok {
		return nil
	}

	switch echoed.Target {
	case session.RuneStones:
		return s.showerRNG(gs)
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

// stoneShowerStride separates one shower from the next inside a fight. A large odd number, on the
// argument `seeds.fightStride` is under: consecutive draws should not be consecutive seeds.
const stoneShowerStride int64 = 0x3B9A_CA07

// resyncHandFromRun brings the hand back in line with the run's deck after a rune has altered
// it: an altered card is redrawn as it now is, and a card the run no longer owns leaves the row.
//
// **It walks by identity**, which is the whole reason a card has one. A card in the hand is a copy —
// and, with a flip relic worn, a copy in a color the run's card never had — so the match cannot be
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
		// **The card is re-dealt rather than re-colored** *(2026-09-08)*. It used to take the
		// element straight off the card in the hand, on the argument that a flip relic had recolored
		// it as it was drawn and that color is a fact about the card in play. That argument is
		// right about the flip and wrong about everything else, and it made every element rune
		// do nothing at all: Hexmark turned the run's card arcane, this line wrote the hand's fire
		// back over it, and what the player saw was a consumable vanishing.
		//
		// **`drawnAs` is the honest answer to both.** It is the same function the deal itself uses,
		// so the card in the hand is the card the run would deal now — the rune's new color
		// with the worn flips applied on top of it, exactly as the next fight will deal it.
		kept = append(kept, paletteCard{actionCard: s.drawnAs(owned), selected: c.selected})
	}

	s.hand = kept
	s.syncQueue()
}

// runeRiderLine is what a rider is called in the fight log and anywhere else a sentence has to
// name one. It is here rather than in prose.go because the vocabulary is the rune's.
//
// **Total over combat.RiderKinds(), and TestEveryRiderKindHasALine holds it that way** *(2026-09-09)*.
// It had one arm and a `default` of "does nothing", which was a lie about seven of the eight kinds
// that existed and would have been a lie about ten of ten — the same failure the choreography
// table's missing default exists to prevent.
func runeRiderLine(k combat.RiderKind) string {
	switch k {
	case combat.RiderHealOnPlay:
		return "heals its owner"
	case combat.RiderShieldOnPlay:
		return "raises a shield"
	case combat.RiderDamageOnPlay:
		return "adds DMG to the blow"
	case combat.RiderDamageInHand:
		return "adds DMG while held"
	case combat.RiderScaleInHand:
		return "multiplies the blow while held"
	case combat.RiderVitaeInHand:
		return "pays vitae while held"
	case combat.RiderScaleInCombo:
		return "multiplies the blow it makes"
	case combat.RiderWildElement:
		return "counts as every element"
	case combat.RiderGolden:
		return "gambles for DMG or life"
	case combat.RiderSilver:
		return "gambles for vitae"
	default:
		return "does nothing"
	}
}

// runeSpendable is the pane's "would clicking this do anything" predicate on this screen.
//
// **The card's lit state and the click read the same function**, which is what stops a control
// looking available and doing nothing.
func (s *CombatScene) runeSpendable(gs *state.GlobalState) func(session.Rune) bool {
	if !s.canSpendRunes(gs) {
		return nil
	}
	ids := s.selectedCardIDs()
	return func(p session.Rune) bool {
		return s.runeTarget(gs, p).satisfiedBy(ids)
	}
}

// updateConsumables runs the click on the consumables pane. Called every tick from Update.
//
// **The gate is the same predicate the card's own lit state reads**, so a card that is drawn dim
// cannot be clicked and a card that is lit always works. Two predicates here is how a control comes
// to look available and do nothing.
func (s *CombatScene) updateConsumables(gs *state.GlobalState) {
	if !s.canSpendRunes(gs) || !gs.CursorAllowed() {
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
	s.spendRune(gs, i)
}
