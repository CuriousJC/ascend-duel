package screens

// **The combat screen's half of the tutorial**: what it can say is true, and where it draws the
// things Bob points at.
//
// It is a file of its own rather than three methods scattered through combat.go, because the
// whole of the feature's footprint on this screen is here — the overlay's own drawing is in
// tutorial.go and the state machine is in `internal/tutorial`. A tutorial that grew tendrils
// through the screen would be one nobody could take back out.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// tutorialFacts is what this screen can honestly report about the run this frame.
//
// **Everything here is read, never derived for the tutorial's benefit.** `Resolving` is the
// screen's own playback cursor and `RoundsPlayed` is the counter it already keeps, so a condition
// cannot end up watching a number that exists only because a lesson wanted one — which would be
// a fact about the tutorial rather than about the game.
func (s *CombatScene) tutorialFacts(gs *state.GlobalState) tutorial.Facts {
	f := tutorial.Facts{
		Queued: len(s.fighterActions),

		// **Unqueued, not the hand's length.** A queued card stays in the row with `selected` set —
		// see `toggle` and `syncQueue` — so `len(s.hand)` is the same number before and after the
		// player has picked everything, and a step waiting on that waited forever.
		Unqueued:  len(s.hand) - s.selectedCount(),
		Resolving: !s.planning() && !s.duelSettled(),

		// **Rounds *finished*, which is not what `s.round` counts** *(2026-09-06)*. That counter is
		// bumped when DUEL! is pressed — it is the round now being played, so it reads 1 for the
		// whole of the first playback — and `tutorial.Facts.RoundsPlayed` is documented as how many
		// rounds have resolved. Publishing it raw made `round-done` unsatisfiable: `duel-pressed`
		// gives way on the same frame the counter moves, so the following step's baseline was
		// already the round it was waiting for and the lesson hung on the math band forever.
		//
		// **Corrected here rather than in the condition**, because the condition is right and the
		// number was wrong. A `>=` in `satisfied` would have made every other reading of the field
		// off by one to compensate for this one.
		RoundsPlayed: s.roundsFinished(),

		// **Read off the frame, not off this screen.** The ledger button belongs to `internal/game`
		// and its panel eats the whole frame while it is up, so what this screen can honestly say
		// is the tally the frame keeps — which is why it is on GlobalState rather than here. The
		// value arrives on the first frame after the panel closes, since this method is not called
		// while it is open at all.
		LedgerOpens: gs.LedgerOpens,

		// **Counted off the marks, which is what is actually on screen.** A break that has settled
		// is a card the player can see is broken; one still crossing the table is not yet
		// anything. Reading the resolved log instead would let a step fire on a break that had not
		// been drawn.
		ShieldBreaks: len(s.theatre.shatteredSeats),
	}
	match := s.matchingCards(gs)
	f.Matching = len(match)
	for _, i := range match {
		if s.hand[i].selected {
			f.MatchingQueued++
		}
	}
	if gs.Run != nil {
		f.Phase = gs.Run.Phase().String()
	}
	return f
}

// roundsFinished is how many rounds of this duel have been played out to the end.
//
// **`s.round` is the round in progress, not the count of completed ones**, so a round being
// replayed right now is subtracted back off. See the field it feeds, and tutorial.CondRoundDone.
func (s *CombatScene) roundsFinished() int {
	if !s.planning() && !s.duelSettled() {
		return s.round - 1
	}
	return s.round
}

// tutorialRects is where each of this screen's anchors is drawn.
//
// **The rectangles are asked of the same functions that draw the things**, never re-derived from
// the constants they were laid out with. A spotlight around where a control used to be is the one
// failure this whole feature cannot survive, and it is exactly what a second copy of a layout
// produces the first time the first copy moves.
func (s *CombatScene) tutorialRects(gs *state.GlobalState, a tutorial.Anchor) ([]image.Rectangle, bool) {
	switch a {
	case tutorial.AnchorEnemyCard:
		return one(s.enemyCardRect(gs)), true
	case tutorial.AnchorDuelistCard:
		return one(s.duelistCardRect(gs)), true
	case tutorial.AnchorTowerPlace:
		return one(s.towerPlaceRect(gs)), true
	case tutorial.AnchorRoundTimer:
		// **The whole bar rather than the cell about to light.** What the step is teaching is the
		// count, and a square around one segment would say the opposite — that this round is the
		// thing to look at — while also moving every time the fight advanced.
		//
		// **It reports false for a fight on no clock**, which is the same honesty `shop-worn` keeps
		// about an empty row: there is no bar drawn, so there is nothing to point at.
		if s.roundTimerLimit() < 1 {
			return nil, false
		}
		return one(s.roundTimerRect(gs)), true
	case tutorial.AnchorHand:
		return one(handZone(gs)), true
	case tutorial.AnchorFirstCard:
		// **The card, not the band.** `cardSlot` is the same rectangle the click is hit-tested
		// against, so the lit square and the one legal click cannot describe different pixels.
		// An empty hand has no first card and reports false, which drops the gate rather than
		// shielding the screen around a seat with nothing in it.
		if len(s.hand) == 0 {
			return nil, false
		}
		return one(s.cardSlot(gs, 0)), true
	case tutorial.AnchorMatchingCards:
		// **One rectangle per card, not the box round them** *(2026-09-08)*. This was a union and
		// the union was a bug: the lit square is also the click gate, so anything sitting between
		// two matching cards was lit and clickable while not being part of the set.
		//
		// **They are not contiguous, and the note that said they were reasoned about the wrong
		// axis.** It argued that cards sharing a *concept* differ only by element and so land side
		// by side whichever sort key leads — true, and not what this anchor matches on. The lesson
		// matches on **element**, which is four different concepts at four different costs, and
		// under the default cost-led sort the taught set lands at seats 0, 1, 2 and 4 with an
		// arcane card at seat 3. The player could queue that card; the taught four cost 1+1+2+2,
		// which is the whole budget, so the fourth could then never be paid for and the round
		// committed a Three of a Kind having just been promised a Four.
		//
		// `cardSlot` is the same rectangle the click is hit-tested against, seat by seat, so the
		// lit squares and the legal clicks are the same pixels — which is what "true by
		// construction" was always supposed to mean. See TestTheMatchingCardsGateLightsOnlyTheTaughtCards.
		match := s.matchingCards(gs)
		if len(match) == 0 {
			return nil, false
		}
		lit := make([]image.Rectangle, 0, len(match))
		for _, i := range match {
			lit = append(lit, s.cardSlot(gs, i))
		}
		return lit, true

	case tutorial.AnchorMatchingCardsLeft:
		// **The same set minus what is already queued** — the cards the player still has to take.
		// A card that has been taken is not something to click, and while it was in this list the
		// only thing the step invited was clicking it back off. See the anchor.
		var left []image.Rectangle
		for _, i := range s.matchingCards(gs) {
			if s.hand[i].selected {
				continue
			}
			left = append(left, s.cardSlot(gs, i))
		}
		if len(left) == 0 {
			return nil, false
		}
		return left, true

	case tutorial.AnchorShatteredCards:
		// **One rectangle per broken seat.** They need not be adjacent — a shield picks the
		// heaviest blows and a creature does not queue them in order — so the box round them would
		// light the cards that *did* land, which is the opposite of what the step is saying.
		//
		// **Walked in seat order rather than over the map**, because Go randomises map iteration
		// and the spotlight sorts what it is given: a stable order costs nothing and keeps the
		// picture the same frame to frame.
		//
		// **It reads the same layout the row draws with**, through breakSeatRect, so a card still
		// flying to its seat is lit where it actually is.
		var broken []image.Rectangle
		for seat := range s.theatre.enemyDealt {
			if !s.theatre.shatteredSeats[seat] {
				continue
			}
			at, ok := s.breakSeatRect(gs, seat)
			if !ok {
				continue
			}
			broken = append(broken, at)
		}
		if len(broken) == 0 {
			return nil, false
		}
		return broken, true

	case tutorial.AnchorDeckStack:
		return one(deckStackBounds(gs)), true
	case tutorial.AnchorMathBand:
		// The band the blow is added up in. **The whole band rather than the figures in it**: the
		// sum is laid out centred and its width is a function of how many terms the round produced,
		// so a rectangle round the figures would be a different size every round and the square
		// would appear to twitch.
		return one(s.handMathRect(gs)), true

	case tutorial.AnchorAPBar:
		// The bar is drawn from the hand band's bottom edge — see drawAPBar's caller — and it is
		// eight pixels tall, which is too thin to spotlight on its own. The rectangle returned is
		// the bar plus the figure written under it, since "3/6 AP" is the half of the pair a
		// player can actually read.
		band := handZone(gs)
		return one(image.Rect(band.Min.X, band.Max.Y+apBarBelow-4,
			band.Max.X, band.Max.Y+apBarBelow+apBarHeight+apFigureBelowBar+20)), true

	case tutorial.AnchorLedgerButton:
		// **The column's own answer, not a copy of it.** `internal/game` places the LEDGER button
		// by asking this same function for the same slot — see chrome.go's ledgerButtonRect — so
		// the lit square is derived from the thing that positions the button rather than from a
		// second reading of the layout.
		return one(ControlColumnSlot(gs, SlotLedger)), true

	case tutorial.AnchorDuelButton:
		return one(buttonRect(s.duelButton)), true
	case tutorial.AnchorHandsButton:
		return one(buttonRect(s.hands.button)), true
	}
	return nil, false
}

// tutorialCovered is whether one of this screen's three dialogs is up: the deck overlay, the fight
// log or the hands ladder. **It is `modalUp` and nothing else**, rather than a second list of the
// same three — a spotlight that kept pointing after a fourth dialog was added would be exactly the
// bug this method exists to fix.
func (s *CombatScene) tutorialCovered(*state.GlobalState) bool { return s.modalUp() }

// matchingCards is the largest set of cards in the hand matching on the script's axis, as hand
// indices in order. It is what [tutorial.AnchorMatchingCards] points at and what
// [tutorial.CondMatchQueued] counts, and both read this one function so the square and the
// condition cannot describe different cards.
//
// **The axis comes from the script** *(2026-08-25)*. It counted concepts and nothing else while the
// lesson was five Jabs; the taught hand is now four cards of one colour, and a set is only a set
// relative to the axis it is counted on — see tutorial.MatchAxis, which is refused rather than
// defaulted for exactly this reason.
//
// **A tie goes to whichever value appears first in the hand**, which is a rule rather than a
// preference: ranging the tally would be map order, and the tutorial would point at a different
// pair of cards on different launches of the same seed. See the determinism rules in CLAUDE.md.
//
// **Fewer than two is no set at all.** One card matches nothing, and a step asking the player to
// find a hand in a hand that has not got one would gate the screen down to a single card and then
// wait for a condition that is already satisfied.
func (s *CombatScene) matchingCards(gs *state.GlobalState) []int {
	axis := runOf(gs).Match()
	if axis == tutorial.MatchNone {
		return nil
	}

	key := func(c combat.Card) int {
		switch axis {
		case tutorial.MatchElement:
			return int(c.Element)
		case tutorial.MatchForm:
			return int(combat.ConceptOf(c.Concept).Form)
		default:
			return int(c.Concept)
		}
	}

	counts := make(map[int]int, len(s.hand))
	for _, c := range s.hand {
		counts[key(c.actionCard)]++
	}

	best, bestN := 0, 0
	for _, c := range s.hand {
		if n := counts[key(c.actionCard)]; n > bestN {
			best, bestN = key(c.actionCard), n
		}
	}
	if bestN < 2 {
		return nil
	}

	var out []int
	for i, c := range s.hand {
		if key(c.actionCard) == best {
			out = append(out, i)
		}
	}
	return out
}
