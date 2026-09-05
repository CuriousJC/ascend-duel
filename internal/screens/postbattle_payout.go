package screens

// **What the reward screen says, and where the worms come in from.**
//
// The sentences are here rather than in prose.go because prose.go is the vocabulary a *log* line
// draws on — a verb for an attack, a name for a status — and this is a fixed script that happens
// once a fight. What the two have in common is the rule: **the words are presentation over figures
// something else already decided**, and nothing in this file computes a payout. It reads
// `session.Spoils`, which `WonFight` froze.

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// payoutLines is the script *(owner's wording, 2026-08-22)*.
//
// **Each line that names a figure claims it**, and the claim is what the flight to the duelist card
// is paying in — see typewriter.tick. The three parts are read out in the order the run decided
// them: interest on what you were carrying, a tenth of the life you kept, then what the room pays.
//
// **The total is computed here and not read off the purse**, because the purse is still climbing
// while the line is typed. It is the figure the three claims are about to add up to, which is what
// the sentence promises.
func payoutLines(gs *state.GlobalState) []proseLine {
	if gs.Run == nil {
		return nil
	}
	spoils := gs.Run.Spoils()
	total := gs.Run.Vitae() + spoils.Total()

	var lines []proseLine

	// **The interest line only appears when there is interest**, which is most fights after the
	// first few. A sentence saying a purse swelled by nothing is a sentence that teaches the player
	// the screen is not reading their run.
	if spoils.Propagated > 0 {
		lines = append(lines, proseLine{
			runs: []proseRun{
				{text: "Vitae", ink: vitaeInk},
				{text: fmt.Sprintf(" proliferates for each %d -- ", session.PropagationPer)},
				{text: fmt.Sprintf("+%d", spoils.Propagated), ink: vitaeInk},
			},
			pays: func(gs *state.GlobalState) int { return gs.Run.ClaimPropagation() },
		})
	}

	lines = append(lines, proseLine{
		runs: []proseRun{
			{text: fmt.Sprintf("Health proliferates for each %d -- ", session.LifeSharePer)},
			{text: fmt.Sprintf("+%d", spoils.FromLife), ink: vitaeInk},
		},
		pays: func(gs *state.GlobalState) int { return gs.Run.ClaimFromLife() },
	})

	lines = append(lines, proseLine{
		runs: []proseRun{
			{text: "Enemy "},
			{text: "vitae", ink: vitaeInk},
			{text: " -- "},
			{text: fmt.Sprintf("+%d", spoils.FromRoom), ink: vitaeInk},
		},
		pays: func(gs *state.GlobalState) int { return gs.Run.ClaimFromRoom() },
	})

	lines = append(lines, proseLine{runs: []proseRun{
		{text: "You have "},
		{text: fmt.Sprintf("%d vitae", total), ink: vitaeInk},
		{text: "."},
	}})

	lines = append(lines, proseLine{runs: []proseRun{
		{text: "Worms flee your enemy, you can only catch one."},
	}})

	return lines
}

// proseLineAt is the middle of one narrated line — where a payment sets off from.
func proseLineAt(gs *state.GlobalState, i int) image.Point {
	return image.Pt(gs.PctX(50), proseTop+i*proseLineGap)
}

// drawProse puts the narration up: every line typed so far, and whatever figure is in the air.
func (s *PostBattleScene) drawProse(gs *state.GlobalState, screen *ebiten.Image,
	face *text.GoTextFace) {

	for i, line := range s.prose.lines {
		runs, on := s.prose.visible(i)
		if !on {
			break
		}
		drawProseLine(screen, face, line.plain(), runs, gs.PctX(50), proseTop+i*proseLineGap)
	}

	s.prose.drawVitaeFlight(gs, screen, face)
}

// beginOffer is what the last sentence leads to: the worms come in from the sides.
//
// **They fly rather than appear**, which is the rule everywhere in this game and is doing real work
// here — the line just read says two creatures are fleeing the enemy, and a card that was already
// on screen would contradict it.
func (s *PostBattleScene) beginOffer(gs *state.GlobalState) {
	s.stage = pickWorm
	for i := range s.entry {
		s.entry[i] = newTravel(i*wormEntryStagger, wormEntryTicks)
	}
	s.place(gs)
}

// The worms' arrival: how long one takes to cross in, and how far apart the two set off.
var (
	wormEntryTicks   = beat(1, 1)
	wormEntryStagger = beat(1, 4)
)

// wormArrivingAt is where one offered worm is *drawn* while it flies in — off the near side of the
// screen at the start of its journey, and in its seat by the end.
//
// **The seat itself never moves**, which is why this is separate from `wormSlot`: the hit test is
// against the seat, so a card can be clicked the moment it is on screen and the flight stays a
// thing to look at. Presentation may never change what a click means.
func (s *PostBattleScene) wormArrivingAt(gs *state.GlobalState, i int) image.Point {
	seat := s.wormSlot(gs, i)
	if i >= len(s.entry) || s.entry[i].done() {
		return seat.Min
	}

	// Left-hand cards come in from the left edge, right-hand ones from the right, so the two arrive
	// from opposite sides rather than sweeping across each other.
	from := image.Rect(-cardWidth-40, seat.Min.Y, -40, seat.Max.Y)
	if seat.Min.X >= gs.PctX(50) {
		from = image.Rect(gs.ScreenWidth+40, seat.Min.Y, gs.ScreenWidth+cardWidth+40, seat.Max.Y)
	}
	return flyingTo(from, seat, s.entry[i])
}
