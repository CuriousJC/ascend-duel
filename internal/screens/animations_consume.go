package screens

// animations_consume.go — the gallery's `use:` entries: each kind of consumable run through
// `ui.Consume`, the one sequence they all share, with the band seat it leaves at the top left, the
// stage in the middle and what it changes on the right.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// The `use:` layout, as offsets from the stage's corner.
var (
	consumeFromAt   = image.Pt(0, -60)
	consumeStageAt  = image.Pt(400, 90)
	consumeTargetAt = image.Pt(800, 300)
	consumeTargetDX = 220
)

// animConsumeTicks is one run of a `use:` entry, measured off a Consume of the same shape so the
// loop and the sequence cannot disagree.
func animConsumeTicks(exit ui.ConsumeExit, changes int) int {
	return ui.NewConsume(ui.Consume{Exit: exit, Changes: make([]ui.Morph, changes)}).Ticks()
}

func drawAnimConsume(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
	// The stone's destination is not a card, so it is drawn here and marked as the motes arrive.
	if s.use.Exit == ui.ExitCrumble && len(s.use.To) > 0 {
		to := s.use.To[0]
		ink := animStageInk
		if s.use.Arrived() && !s.use.ArrivedTravel().Done() {
			ink = ui.GroundInk
		}
		animText(gs, screen, "HAND LADDER", to.Min.X+20, to.Min.Y+to.Dy()/2, animNameSize, ink)
	}
	animText(gs, screen, "band seat", at.Min.X+consumeFromAt.X, at.Min.Y+consumeFromAt.Y-24, animNoteSize, animStageInk)
	s.use.Draw(gs, screen)
}

// animConsume builds the Consume a `use:` entry runs, or the zero one for any other entry.
func animConsume(gs *state.GlobalState, name string) ui.Consume {
	o := image.Pt(gs.PctX(animStageLeftPct), gs.PctY(animStageTopPct))
	seat := func(off image.Point) image.Rectangle {
		return image.Rect(o.X+off.X, o.Y+off.Y, o.X+off.X+cardWidth, o.Y+off.Y+cardHeight)
	}
	target := func(i int) image.Rectangle {
		return seat(image.Pt(consumeTargetAt.X+i*consumeTargetDX, consumeTargetAt.Y))
	}
	a, b := animFace(gs, animCardA), animFace(gs, animCardB)

	c := ui.Consume{
		Style: cards.EssenceStyle,
		From:  seat(consumeFromAt),
		Stage: seat(consumeStageAt),
		Seed:  cards.MarkSeed(name),
	}

	switch name {
	case "use: essence":
		es := session.Essences()
		if len(es) == 0 {
			return ui.Consume{}
		}
		c.Exit = ui.ExitAbsorb
		c.Face = ui.EssenceSpec(gs, es[0], true)
		c.To = []image.Rectangle{target(0)}
		c.Changes = []ui.Morph{ui.MorphInto(a, b, cards.Hand)}
		c.Motes = ui.Emitter{Element: cards.Arcane, Set: ui.ParticleSparks, Rate: 1.2, Size: 22}

	case "use: rune":
		rs := session.Runes()
		if len(rs) == 0 {
			return ui.Consume{}
		}
		c.Exit = ui.ExitSigil
		c.Face = ui.RuneSpec(gs, rs[0], true, false)
		c.To = []image.Rectangle{target(0), target(1)}
		c.Changes = []ui.Morph{
			ui.MorphInto(a, b, cards.Hand),
			ui.MorphInto(b, animFace(gs, combat.Card{Concept: combat.Bash, Element: combat.Earth}), cards.Hand),
		}
		c.Motes = ui.Emitter{Element: cards.Arcane, Rate: 1.2, Size: 22,
			Sprites: ui.FirstDrawn(ui.RuneMoteSprites, []string{"spark-lightning-1", "spark-lightning-2"})}

	case "use: stone":
		ss := session.Stones()
		if len(ss) == 0 {
			return ui.Consume{}
		}
		c.Exit = ui.ExitCrumble
		c.Face = ui.StoneSpec(gs, ss[0], true)
		c.To = []image.Rectangle{target(0)}
		c.Motes = ui.Emitter{Element: cards.Earth, Rate: 1.2, Size: 22,
			Sprites: ui.FirstDrawn(ui.RubbleSprites, []string{"particle-earth-1", "particle-earth-2"})}

	case "use: cantrip":
		cs := session.Cantrips()
		r, ok := animRelic(gs)
		if len(cs) == 0 || !ok {
			return ui.Consume{}
		}
		c.Exit = ui.ExitBurn
		c.Face = ui.CantripSpec(gs, cs[0], true)
		// **The relic it casts arrives in the band**, standing in for the cantrip-relic here.
		c.To = []image.Rectangle{seat(image.Pt(consumeTargetAt.X, consumeFromAt.Y))}
		c.Changes = []ui.Morph{ui.MorphIn(ui.RelicSpec(gs, r, "", true, false), cards.RelicStyle)}
		c.Motes = ui.Emitter{Element: cards.Fire, Set: ui.ParticleMixed, Rate: 1.2, Size: 22}

	default:
		return ui.Consume{}
	}
	return ui.NewConsume(c)
}
