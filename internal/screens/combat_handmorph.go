package screens

// Cards changing **in the hand**, mid-fight: the parasite's half of the morph.
//
// The post-battle screen's version is a single card flown to the middle of an empty table and
// changed there, which is what a reward can afford. A parasite is spent between two turns of a live
// round, on cards standing in the row the player is about to play — so the change has to happen
// where the cards already are, on as many of them as the parasite named, and without taking the
// screen.
//
// **One beat for all of them** *(owner's call, 2026-09-08, the same call the shield break was taken
// under)*. Every card a parasite touched changes at once, on one clock. Three cards dissolving in
// sequence would be three beats of pause over a hand the player is in the middle of building, and
// what is being said — "these are the cards it took" — is one statement about the parasite rather
// than one per card.
//
// # It draws at the seat, and the seat is a lookup
//
// A morph carries the **card's identity**, not its index: the row can be sorted or dragged while one
// is running, and a mover holding a slot number would then be painting whatever card had moved into
// it. `at` is the rectangle the card was standing in when the morph was raised, and it is only used
// when the card is no longer in the hand at all — which is exactly the eaten case, where what is
// being drawn is a ghost of a card the run has already lost.
//
// # It may never change an outcome
//
// The deck and the hand are both altered by `spendParasite` before any of this is raised, so a morph
// is a picture of something that has already happened — the rule every flight on this screen is
// under. Nothing waits for it: the card underneath a running morph is the new card and is selectable
// while it changes, exactly as a card is clickable while it flies.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// handMorph is one card in the hand changing into another.
type handMorph struct {
	// id is the card this is about. **Identity rather than a seat**, so a sort or a drag under a
	// running morph moves the picture with the card.
	id int

	// at is where the card stood when this was raised, and it is the fallback for a card that is
	// no longer in the hand — the eaten case, which has no seat to look up.
	at image.Rectangle

	m morph
}

func (h *handMorph) tick()     { h.m.tick() }
func (h handMorph) done() bool { return h.m.done() }

// handFaces is every card in the hand as a finished face, by identity, plus where each one stands.
//
// **Taken before a parasite is applied and again after**, because what changed is the difference
// between the two — which is the only way to raise a morph without this file learning what each
// parasite does. A borer recolours, a grub reforms, a graft overwrites, a rider takes the left
// column: all of them are "this face is not the face that was here", and none of them needs a case.
func (s *CombatScene) handFaces(gs *state.GlobalState) (map[int]cards.Spec, map[int]image.Rectangle) {
	faces := make(map[int]cards.Spec, len(s.hand))
	seats := make(map[int]image.Rectangle, len(s.hand))
	for i, c := range s.hand {
		faces[c.actionCard.ID] = s.handFace(c)
		seats[c.actionCard.ID] = s.cardSlot(gs, i)
	}
	return faces, seats
}

// handFace is one hand card's face, drawn as it would be if nothing else were going on.
//
// **Always enabled and never selected**, deliberately. Those two are the row's state rather than the
// card's, and they move for reasons that have nothing to do with a parasite — a selection reaching
// the cap dims every other card, and a morph that captured that would be comparing the hand's mood
// rather than the cards.
func (s *CombatScene) handFace(c paletteCard) cards.Spec {
	return cardSpec(c.actionCard, heldBy(s.fighter.Duelist, c.actionCard), true, false)
}

// raiseHandMorphs works out what a parasite changed and puts a morph on each of it.
//
// It is handed the faces and seats from *before* the parasite landed and reads the hand as it is
// now, so the three shapes fall out of the comparison rather than out of a switch on the parasite:
// a face that changed is a replacement, a card that has appeared is a copy, and a card that has
// gone is one that was eaten.
func (s *CombatScene) raiseHandMorphs(gs *state.GlobalState, was map[int]cards.Spec, seats map[int]image.Rectangle) {
	for i, c := range s.hand {
		id := c.actionCard.ID
		now := s.handFace(c)
		at := s.cardSlot(gs, i)

		before, had := was[id]
		switch {
		case !had:
			// A card that was not in the hand a moment ago: a copy, arriving out of nothing.
			s.theatre.morphs = append(s.theatre.morphs, handMorph{
				id: id, at: at, m: morphIn(now, cards.Hand),
			})
		case before != now:
			s.theatre.morphs = append(s.theatre.morphs, handMorph{
				id: id, at: at, m: morphInto(before, now, cards.Hand),
			})
		}
	}

	// What is left is what the run no longer owns. **The row has already closed over it**, so this
	// is a ghost at the seat the card had — the same shape a discarded card's flight takes, and the
	// same rule: the model moved first and this is a picture of it.
	live := make(map[int]bool, len(s.hand))
	for _, c := range s.hand {
		live[c.actionCard.ID] = true
	}
	for id, before := range was {
		if live[id] {
			continue
		}
		s.theatre.morphs = append(s.theatre.morphs, handMorph{
			id: id, at: seats[id], m: morphAway(before, cards.Hand),
		})
	}
}

// handMorphFor is the morph running on one hand card, if there is one.
func (s *CombatScene) handMorphFor(id int) (handMorph, bool) {
	for _, h := range s.theatre.morphs {
		if h.id == id {
			return h, true
		}
	}
	return handMorph{}, false
}

// drawHandMorphs draws the ghosts: morphs whose card is no longer in the hand.
//
// **The ones still in the hand are drawn by the row itself**, at their own seats, which is what
// keeps a changing card in the place the player last saw it. Only a card that has left has nowhere
// to be drawn from, so only those are drawn here.
func (s *CombatScene) drawHandMorphs(gs *state.GlobalState, screen *ebiten.Image) {
	if len(s.theatre.morphs) == 0 {
		return
	}
	live := make(map[int]bool, len(s.hand))
	for _, c := range s.hand {
		live[c.actionCard.ID] = true
	}
	for _, h := range s.theatre.morphs {
		if live[h.id] {
			continue
		}
		drawMorph(gs, screen, h.at.Min, h.m)
	}
}
