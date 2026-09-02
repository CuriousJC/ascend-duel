package screens

// The bucket: the board piece a parasite is spent from, and the one place a run's deck is altered
// while a duel is going on.
//
// **A worm is spent between rooms and a parasite is spent between turns**, which is most of what
// makes them different things. The catalogue and the rules are `internal/session/parasite.go`; this
// file is the dialog, and it decides nothing — it picks a parasite, picks the cards, shows what
// would happen, and asks the run to do it.
//
// **It is the fourth dialog on this screen and it takes the other three's shape**: the same
// footprint, the same scrim, the same red X, one live exit. `modalUp` gained a fourth term rather
// than this growing a rule of its own.
//
// **It is only live while `planning()`.** A parasite alters the deck, and the deck a round was
// resolved against is the deck that round has to be replayed with — `ResolveRound` decides
// everything before a frame of playback runs, so a card changed mid-playback would put a face on
// screen that disagrees with the blow already computed. Spending between turns is the whole design
// and this predicate is what enforces it.
//
// **Targets are picked out of the hand, not out of the whole deck** *(a call taken while building
// it, and the one most worth revisiting)*. Two arguments for it: a mid-fight consumable aimed at
// the card you are about to play is a decision about *this* turn, where one aimed at a card
// somewhere in a pile of forty is the reward screen's decision taken in a worse place; and the hand
// is already on screen, so the picker is the row the player is looking at rather than a second grid
// of the deck. What it costs is reach — a card sitting in the draw pile cannot be touched until it
// is drawn.

import (
	"fmt"
	"image"
	"math/rand"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The button that opens the bucket.
//
// **`P` on a 44px square**, the shape every dialog opener on this screen takes, and a letter that
// does not collide with the `L` beside the pile or the `$`/`T`/`E` of the sort column.
const (
	parasiteToggleLabel = "P"
	parasiteToggleText  = 30

	// The gap between this button and the Log button under it. The same 10 two toggles standing
	// together take everywhere else.
	parasiteButtonUp = modalToggleGap
)

// parasiteButtonPlace is where it stands: **directly above the Log button, sharing its column.**
//
// **Stacked rather than added to the bottom strip**, which is deliberate and is the cheap half of
// a decision the owner has said he wants to move later. `buttonStripSlots` measures the strip's
// span from the AP figure to the Log button and divides what is left in three; putting a fifth
// control on that line would re-space Discard and DUEL! as a side effect of adding a bucket. The
// air above the Log button is empty — the hand row ends well clear of it — so a square there costs
// nothing and moves nothing.
func parasiteButtonPlace(gs *state.GlobalState) image.Point {
	log := logButtonRect(gs)
	return image.Pt(
		log.Min.X+logButtonSize/2,
		log.Min.Y-parasiteButtonUp-logButtonSize/2,
	)
}

// parasiteToggle is the bucket behind its button: which parasite is armed, and which cards it has
// been aimed at so far.
//
// **Nothing is committed until the take**, which is the rule the worm morph is under and the reason
// these two fields exist rather than the dialog spending as it goes. Back steps out of the targets
// to the bucket and out of the bucket to the screen, so a player who armed the wrong parasite or
// aimed it at the wrong card is never stuck with either.
type parasiteToggle struct {
	modalToggle

	// armed is the position in the run's bucket of the parasite being spent, or -1 for none.
	//
	// **A position rather than a record key**, because the bucket may hold two of the same
	// parasite and spending one must not be ambiguous about which. It is only ever held across the
	// frames of one dialog, and nothing else may alter the bucket while a dialog is up.
	armed int

	// picked is the identities of the cards chosen so far, in the order they were clicked.
	//
	// **Identities, not hand positions.** The hand is re-laid-out by a sort and re-dealt by a
	// refill; an identity survives both, and it is what `session.ApplyParasite` takes.
	picked []int

	// shown is a rock shower's receipt: the stones it just handed over, held on screen until the
	// player dismisses them. Empty for every other parasite, which close on the take.
	shown []session.Stone
}

// initParasites wires the button. **The button survives a re-entry and the state does not** —
// `Init` runs again on every fight, and arriving in a duel with the bucket open would be a dialog
// nobody asked for.
func (s *CombatScene) initParasites() {
	s.parasites.modalToggle.init(parasiteToggleLabel, logButtonSize, logButtonSize,
		parasiteToggleText, parasiteButtonPlace)
	s.parasites.disarm()
}

// disarm puts the dialog back to its first stage.
func (t *parasiteToggle) disarm() {
	t.armed = -1
	t.picked = nil
}

// dismiss clears a rock shower's receipt and closes the dialog behind it.
func (t *parasiteToggle) dismiss() {
	t.shown = nil
	t.open = false
}

// updateParasites runs the button and, while the bucket is open, whatever is under the cursor.
//
// **It is taken out of the frame entirely when it cannot be used** — no parasites to spend, or the
// round is playing back. `blocked` is the same field another dialog sets, and it neither runs nor
// draws the button, which is what stops a control being lit for something the player cannot do.
func (s *CombatScene) updateParasites(gs *state.GlobalState) bool {
	s.parasites.block(s.showDeck || s.showLog || s.hands.open || !s.canSpendParasites(gs))
	if !s.parasites.open {
		s.parasites.disarm()
		s.parasites.shown = nil
	}
	return s.parasites.modalToggle.update(gs, func(at image.Point, tip *models.Tooltip) {
		s.parasiteHover(gs, at, tip)
	})
}

// canSpendParasites is the one predicate for "the bucket may be opened".
//
// **`planning()` is half of it and the bucket having something in it is the other.** Neither is a
// presentation rule: the first is what keeps an alteration out of a round that has already been
// resolved, and the second is what stops an empty dialog being something to open.
func (s *CombatScene) canSpendParasites(gs *state.GlobalState) bool {
	if gs.Run == nil || !s.planning() {
		return false
	}
	// **A receipt keeps the dialog alive after the bucket has emptied.** Spending the last rock
	// shower leaves nothing to spend, and without this the panel showing what it just handed over
	// would be taken off screen in the same frame it went up.
	return gs.Run.HoldCount() > 0 || len(s.parasites.shown) > 0
}

// drawParasites puts the button and, if it is open, the panel on screen.
func (s *CombatScene) drawParasites(gs *state.GlobalState, screen *ebiten.Image) {
	s.parasites.modalToggle.draw(gs, screen, func() { s.drawParasitePanel(gs, screen) })
}

// heldParasites is the bucket, resolved.
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
func parasiteSpec(gs *state.GlobalState, p session.Parasite, enabled, selected bool) cards.Spec {
	return cards.Spec{
		Name:     p.Name,
		Form:     cards.FormNone,
		Element:  cards.Basic,
		Art:      artwork(gs, wormArtKey),
		Text:     p.Text,
		Enabled:  enabled,
		Selected: selected,
	}
}

// The panel's geometry. A row of parasite cards across the middle, and under it — once one is
// armed — the hand it is being aimed at.
const (
	// parasiteRowGap is the air between two cards in either row. The same gap the shop's shelf
	// takes, so a row of cards reads the same wherever it stands.
	parasiteRowGap = 18

	// parasitePromptDrop is how far under the panel's top the one line of instruction sits.
	parasitePromptDrop = modalBareBodyTop + 6
	parasitePromptSize = 22

	// The two rows' centres, as fractions of the panel's own height. The bucket sits high when a
	// hand is under it and the hand sits low, so the two read as "this, applied to that".
	parasiteBucketRowPct = 34
	parasiteTargetRowPct = 70
)

// parasiteRowSlots is the left edges of n cards laid out in a centred row.
//
// **It tightens rather than overflowing.** A bucket is not capped, so a rich run can be holding
// more parasites than a row of full-size cards fits; the pitch closes up exactly as the hand's does
// rather than the row running off both edges of the panel.
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

// bucketRects is where the parasite cards stand.
func (s *CombatScene) bucketRects(gs *state.GlobalState) []image.Rectangle {
	r := modalPanelRect(gs)
	return parasiteCardRects(r, len(heldParasites(gs)), r.Min.Y+r.Dy()*parasiteBucketRowPct/100)
}

// targetRects is where the hand stands while a parasite is being aimed.
func (s *CombatScene) targetRects(gs *state.GlobalState) []image.Rectangle {
	r := modalPanelRect(gs)
	return parasiteCardRects(r, len(s.hand), r.Min.Y+r.Dy()*parasiteTargetRowPct/100)
}

// armedParasite is the one being spent, and whether there is one.
func (s *CombatScene) armedParasite(gs *state.GlobalState) (session.Parasite, bool) {
	held := heldParasites(gs)
	if s.parasites.armed < 0 || s.parasites.armed >= len(held) {
		return session.Parasite{}, false
	}
	return held[s.parasites.armed], true
}

// parasiteHover is what the dialog does with the cursor: arm a parasite, aim it, and take it.
//
// **One click does one thing and the stage decides which**, rather than a mode the player has to
// be told they are in. With nothing armed every parasite card is live; with one armed the hand is
// live and the bucket is not.
func (s *CombatScene) parasiteHover(gs *state.GlobalState, at image.Point, tip *models.Tooltip) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || !gs.CursorAllowed() {
		return
	}

	// **A receipt takes the whole panel and the only gesture is dismissing it.** Nothing is being
	// chosen — the stones are already on their rungs — so any click anywhere puts it away rather
	// than the player having to find a control.
	if len(s.parasites.shown) > 0 {
		s.parasites.dismiss()
		return
	}

	p, armed := s.armedParasite(gs)
	if !armed {
		for i, r := range s.bucketRects(gs) {
			if at.In(r) {
				s.armParasite(gs, i)
				return
			}
		}
		return
	}

	for i, r := range s.targetRects(gs) {
		if !at.In(r) {
			continue
		}
		s.aimParasite(gs, p, s.hand[i].actionCard.ID)
		return
	}
}

// armParasite picks one out of the bucket.
//
// **A parasite that needs no target is taken on the spot**, because there is nothing to aim it at
// and a confirm step for a card that says "gain 20 vitae" would be asking the player to agree with
// themselves.
func (s *CombatScene) armParasite(gs *state.GlobalState, i int) {
	s.parasites.armed = i
	s.parasites.picked = nil

	if p, ok := s.armedParasite(gs); ok && p.Count == 0 {
		s.takeParasite(gs, p)
	}
}

// aimParasite adds or removes one target, and takes the parasite once it has all of them.
//
// **Clicking a chosen card takes it back off the list**, which is the hand row's own gesture — a
// card clicked into the queue is clicked out of it — so the one thing a player already knows how
// to undo works here too.
func (s *CombatScene) aimParasite(gs *state.GlobalState, p session.Parasite, id int) {
	for i, chosen := range s.parasites.picked {
		if chosen == id {
			s.parasites.picked = append(s.parasites.picked[:i], s.parasites.picked[i+1:]...)
			return
		}
	}

	// **A card this parasite would do nothing to is not a legal target**, which is the same check
	// the worm offer makes before it puts a card up: a consumable that lands and changes nothing is
	// something bought and taken away. It is asked one card at a time, so the answer is about the
	// card under the cursor rather than about the set.
	if !gs.Run.CanApplyParasite(p, []int{id}) && p.Count == 1 {
		return
	}
	if len(s.parasites.picked) >= p.Count {
		return
	}
	s.parasites.picked = append(s.parasites.picked, id)

	if len(s.parasites.picked) == p.Count {
		s.takeParasite(gs, p)
	}
}

// takeParasite spends it: the run applies it, the bucket loses it, and the dialog closes.
//
// **Drop and apply in that order, and only if the apply succeeded.** A parasite dropped from the
// bucket by an application that then refused would be a consumable the player paid for and did not
// get; `ApplyParasite` is all-or-nothing, so asking it first is what makes the pair safe.
func (s *CombatScene) takeParasite(gs *state.GlobalState, p session.Parasite) {
	if !gs.Run.ApplyParasiteRolling(p, s.parasites.picked, s.showerRNG(gs)) {
		s.parasites.disarm()
		return
	}
	gs.Run.Drop(s.parasites.armed)

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

	s.parasites.disarm()

	// **A rock shower stays on screen instead of closing the dialog** *(owner's call, 2026-09-02)*:
	// "show them all and put them all in". They are already in the run's pouch by the time this
	// runs — the panel is a receipt, not an offer, and the only click it takes is the one that
	// dismisses it. What can be done with them is the shop's S button; see shop_pouch.go.
	if shown := gs.Run.Granted(); len(shown) > 0 {
		s.parasites.shown = shown
		saveRun(gs)
		return
	}

	s.parasites.open = false
	saveRun(gs)
}

// showerRNG is the source a rock shower draws its stones from, and nil for every other parasite.
//
// **Its own salted stream, plus the number of stones the run has already placed** — see
// `seeds.StoneShower`. The fight index alone is not enough here, because a run may carry three
// showers and spend all three in one fight; the placed count is a number the snapshot already
// carries, so a resumed run rolls what it would have rolled.
func (s *CombatScene) showerRNG(gs *state.GlobalState) *rand.Rand {
	if gs.Run == nil {
		return nil
	}
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

// drawParasitePanel is the dialog: the bucket, and the hand under it once one is armed.
func (s *CombatScene) drawParasitePanel(gs *state.GlobalState, screen *ebiten.Image) {
	r := drawModalFrame(gs, screen, modalHead{})

	if len(s.parasites.shown) > 0 {
		s.drawStoneReceipt(gs, screen, r)
		return
	}

	p, armed := s.armedParasite(gs)
	s.drawParasitePrompt(gs, screen, r, p, armed)

	for i, rect := range s.bucketRects(gs) {
		held := heldParasites(gs)
		if i >= len(held) {
			break
		}
		// **The armed one stays lit and the rest go quiet**, so the panel says which parasite the
		// hand under it is being aimed with. It is the card format's selected state, which is
		// already what "this is the one" means everywhere else on this screen.
		drawSpecCard(gs, screen, rect.Min,
			parasiteSpec(gs, held[i], !armed || i == s.parasites.armed, armed && i == s.parasites.armed))
	}

	if !armed || p.Count == 0 {
		return
	}
	s.drawParasiteTargets(gs, screen, p)
}

// drawStoneReceipt is what a rock shower leaves on screen: the stones it handed over.
//
// **They are drawn where the bucket's cards stand**, not where the hand does, because they are the
// thing that was just spent rather than the thing being aimed at — and they are drawn *enabled*,
// since every one of them was kept. There is nothing to choose here.
func (s *CombatScene) drawStoneReceipt(gs *state.GlobalState, screen *ebiten.Image,
	r image.Rectangle) {

	shown := s.parasites.shown

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: parasitePromptSize}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(r.Min.X+r.Dx()/2), float64(r.Min.Y+parasitePromptDrop))
	op.PrimaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, fmt.Sprintf("%d %s into your pouch - spend them at the shop",
		len(shown), stoneWord(len(shown))), face, op)

	rects := parasiteCardRects(r, len(shown), r.Min.Y+r.Dy()*parasiteBucketRowPct/100)
	for i, rect := range rects {
		if i >= len(shown) {
			break
		}
		drawStoneCard(gs, screen, rect.Min, shown[i], true)
	}
}

// stoneWord is "stone" or "stones", so a shower of one does not read "1 stones".
func stoneWord(n int) string {
	if n == 1 {
		return "stone"
	}
	return "stones"
}

// drawParasiteTargets is the hand, drawn as the thing being aimed at.
//
// **A card is enabled if this parasite could legally take it**, so an illegal target is dim before
// it is clicked rather than a click that silently does nothing. Chosen cards are lit.
func (s *CombatScene) drawParasiteTargets(gs *state.GlobalState, screen *ebiten.Image,
	p session.Parasite) {

	for i, rect := range s.targetRects(gs) {
		if i >= len(s.hand) {
			break
		}
		c := s.hand[i].actionCard
		drawCard(gs, screen, rect.Min, cards.Hand, c, heldBy(s.fighter.Duelist, c),
			gs.Run.CanApplyParasite(p, []int{c.ID}), s.parasiteChosen(c.ID))
	}
}

// parasiteChosen reports whether this card is already one of the targets.
func (s *CombatScene) parasiteChosen(id int) bool {
	for _, chosen := range s.parasites.picked {
		if chosen == id {
			return true
		}
	}
	return false
}

// drawParasitePrompt is the one line saying what to do next.
//
// **A sentence rather than a title**, because the panel is a two-stage thing and which stage it is
// in is the only thing the player needs told. The deck panel's argument against words at the top of
// a dialog was that the picture already said what it was; here the picture is the same at both
// stages and the instruction is what differs.
func (s *CombatScene) drawParasitePrompt(gs *state.GlobalState, screen *ebiten.Image,
	r image.Rectangle, p session.Parasite, armed bool) {

	line := "Spend a parasite"
	if armed {
		left := p.Count - len(s.parasites.picked)
		line = fmt.Sprintf("%s - pick %d more %s", p.Name, left, cardWord(left))
	}

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: parasitePromptSize}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(r.Min.X+r.Dx()/2), float64(r.Min.Y+parasitePromptDrop))
	op.PrimaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, line, face, op)
}

// cardWord is "card" or "cards", so the prompt does not read "pick 1 cards".
func cardWord(n int) string {
	if n == 1 {
		return "card"
	}
	return "cards"
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
