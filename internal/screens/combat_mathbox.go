package screens

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// **The band above the hand, which the Resolution feed used to occupy** *(vacated 2026-08-18)*.
// Nothing is drawn there at rest now; what claims it is the hand dialog, which writes the
// blow's arithmetic across it, and `drawPlannedHand`, which writes the name of the hand the
// selection has already formed in the same place. See combat_mathbox.go.
//
// **It is an overlay and reserves nothing** *(2026-09-04, owner's call)*. `tableRowTop` used to
// subtract this height, so a band nothing is drawn in at rest cost the played cards a row of the
// screen; the cards now come down over it and the sum is drawn on top of them, last. The
// constants stay because the sum's own layout does: `handMathRect` is measured from them, and
// their **values are the feed's** — the arithmetic was laid out and looked at against a box of
// exactly this height, so changing the number is re-laying out the sum.
const (
	// mathBandHeight is how deep it is: what the feed's collapsed three rows came to.
	mathBandHeight = 82

	// mathBandGapAboveCards is how far its bottom edge sits above the resting hand row.
	//
	// **A selected card lifts by selectedNudge and does overlap it**, by 21 pixels, and that is
	// accepted rather than overlooked: the band is measured against where the cards live, not
	// against where one of them goes when it is picked.
	mathBandGapAboveCards = 5
)

// The hand dialog: the blow's arithmetic acted out at the size of the screen, on the beat the
// hand fires.
//
// **It exists because the sum was the one number on this screen nobody could source**
// *(2026-08-18)*. The Resolution feed printed it — `(40 x 1.5 = 60)` — in sixteen-point text on
// the third line of a three-line box, beside the sentence it belonged to. That is a record, and a
// record is the right thing for a feed to be; what it is not is an *explanation*. A player
// watching a blow land could see the total and could not see which card paid for which part of
// it, so the multiplier read as a number the game had decided rather than one they had built. The
// fix is not a better sentence: it is showing the figures leaving the cards.
//
// **It says nothing the event does not.** Every figure in it comes off the `KindHand` event —
// `HandAmounts`, `Multiplier`, `Amount` — and nothing here multiplies, adds or rounds. That is
// the rule to hold: this is a second *drawing* of one event, never a second arithmetic. If it
// ever needs a figure the event does not carry, the field goes on the event.
//
// **It may not change an outcome**, the same constraint as playback speed, the debug flags,
// `internal/trace` and every card in flight. What it does change is *pacing* — playback stops
// while it runs, and `advancePlayback` holds the cursor rather than the box racing a dwell it
// cannot fit inside. A round therefore takes longer with a hand in it, which is the point: the
// blow is the moment of the round and it used to go past in one and a quarter seconds.
//
// **It draws across the band above the hand**, which the Resolution feed used to hold and left on
// 2026-08-18, and clears itself before the damage lands. That band was chosen while the feed was
// still in it, because it is where the player was already looking for what happened and because
// it is the only place on the screen with room for forty-point numerals. **The feed leaving costs
// this nothing and is why the band is still measured the way it is** — see handMathRect.

const (
	// The script's beats, in ticks at 60 a second.
	//
	// **They are separate constants rather than one scaled number**, because they answer
	// different questions: how long a shout needs to be read, how long a figure takes to cross a
	// quarter of the screen, and how long an operator needs to register. Tuning one should not
	// move the others.
	// **They are proportions of `beatTicks` rather than durations** *(2026-08-19)*, so the
	// dialog speeds up and slows down with the round it is part of. The fractions reproduce the
	// numbers they were tuned to — 35, 22, 10, 25, 40 — at a speed of 25. See beat.

	// Type sizes. The figures are the point of the box, so they are the biggest thing on the
	// screen that is not a card; the operators are smaller because they are punctuation.
	//
	// **All three doubled on 2026-08-19, with the hand's name** *(owner's call)*: at 38 points a
	// term was being overwhelmed by the cards and the name around it, and the arithmetic is the
	// thing the box exists to make readable. `mathTotalSize` carries the landing damage figure
	// with it — see `hitFigureSize`, which is this constant rather than a size of its own.
	//
	// **Width is checked and depth is not.** The widest sum the rules could ever produce is about
	// 830 pixels against a 1232-wide band, which is what `TestTheWidestSumFitsItsBand` holds — but
	// a 100-point total measures 85 pixels tall against `mathBandHeight`'s 82, so the line now
	// stands a pixel and a half past the band top and bottom. It clears its neighbors either way
	// (`firingGap` above, `mathBandGapAboveCards` below), so this is a note rather than a bug: the
	// next increase is the one that has to move the band rather than only the type.
	mathTermSize = 76

	// mathGrowthSize is a relic's multiplier inside a term, and it is **mathTermSize** — the same
	// size as every other figure on the line *(owner's call, 2026-09-19)*. It was well under it
	// while the relic's figure was an annotation *beside* a term the relic had already been folded
	// into; inside a bracket it is one of the factors being multiplied, and a factor set smaller
	// than the ones either side of it reads as a footnote on the product rather than part of it.
	//
	// **It is still its own constant**, because how loud a relic's figure is against the cards'
	// remains an open question: the pink already says whose it is.
	mathGrowthSize = mathTermSize
	mathSymbolSize = 60

	// mathInnerSymbolSize is the `x` inside a term, against mathSymbolSize for the one that
	// multiplies the whole sum. **The smaller operator is the tighter-binding one**, which is the
	// typographic half of the same argument the gaps make: a product inside a term is a detail of
	// that term, and the sum's own multiplier is an instruction about everything to its left.
	mathInnerSymbolSize = 40
	mathTotalSize       = 100

	// mathNameSize is the hand's name, and **it is one size wherever the name is written**
	// *(2026-08-19, owner's call)*: proposed in the middle of the table, traveling to the hand
	// row at DUEL!, resting there for the round, and shouted by the box for a hand the banner
	// never carried. It is deliberately the biggest type on the screen — the hand is what the
	// round is about, and everything else the screen says about it is a figure.
	// `TestTheWidestHandNameFitsTheScreen` is what stops the longest name in the catalog
	// running off the edges at it.
	//
	// **It used to grow from 80 to 124 on the flight down** — a preview proposing and an
	// announcement recording, the split the caption and the Resolution feed kept. That was worth
	// having while the two were separate drawings and stopped being worth it once the word
	// traveled: a name that swells while it moves is a second thing happening to it, and what
	// says the hand is now committed is the journey itself, plus the alpha coming up from
	// `mathPreviewAlpha` to solid. **The word is the same word, so it is the same size.**
	mathNameSize = 80

	// mathItemGap is the air between one item of the sum and the next.
	mathItemGap = 16

	// mathHugGap is what a bracket leaves between itself and the term inside it. **Not zero**: the
	// type is set with its own side bearings and a bracket hard against a numeral at 76 points
	// touches it.
	mathHugGap = 4

	// mathTightGap is the air inside a bracket, and mathWideGap the air round the multiplier that
	// applies to the whole sum.
	//
	// **Three gaps rather than one, because the line has three levels** *(owner's call,
	// 2026-09-19)*. At a single gap everything on it is equally far from everything else, so the
	// `x` inside a term and the `x` that multiplies the finished sum read as the same operation —
	// and the hand's multiplier appeared to belong to the bracket on its left rather than to all of
	// them. What says which is which is distance: a product is set close, a sum is set apart.
	mathTightGap = 7
	mathWideGap  = 30

	// mathBoldMinStep is the smallest faux-bold offset, in pixels. **Bold is the same run drawn
	// again a step to the right** — the pane's own idiom, and for the pane's own reason: `text/v2`
	// has no synthetic bold and kubasta ships one weight. The step is scaled off the type size
	// above this floor, because a one-pixel thickening on an 80-point word is not visible at all.
	mathBoldMinStep  = 1
	mathBoldSizeStep = 32 // one pixel of thickening per this many points

	// mathPreviewAlpha is how solid it is. Far enough back to read as a proposal rather than an
	// event, and it is the second mark saying so: the size alone was ambiguous against a long
	// hand name, which fills more of the band than a short one at any size.
	mathPreviewAlpha = 0.62

	// The breath: how far the hand's name swells past its own size and how long a full
	// expand-and-contract takes, in ticks at 60 a second.
	//
	// **It is small and slow on purpose.** The name is the only thing on this screen that moves
	// while nothing is happening — the whole planning phase is otherwise still — so it has to read
	// as a thing alive rather than as a thing flashing. Six per cent is about four pixels on the
	// preview and it is enough to catch an eye that is looking elsewhere.
	//
	// **It is a multiplier on the scale, never on the size**, so it costs no re-measuring and
	// cannot reflow anything: the text is laid out once at its own size and the breath is applied
	// to the drawing.
	//
	// **It is the one clock on this screen that is not a fraction of the playback speed**, and
	// deliberately: everything `beat` scales is a *duration between two things* — how long a beat
	// is held, how long a journey takes — and this is neither. It is an idle oscillation on a word
	// that is mostly on screen while nothing at all is playing back, so tying it to playback would
	// make the label breathe faster the faster a round is watched, which is the opposite of what a
	// resting pulse means.
	mathBreathAmount = 0.06
	mathBreathTicks  = 84

	// mathMultLineSize is the `1.15x DMG` line under the hand's name, and mathMultLineGap the air
	// between the two, in points of the name's own size.
	//
	// **It is `mathTermSize` — the size a figure is written at in the sum — and it does not grow
	// with the name** *(2026-08-19, owner's call)*. That is what makes the handoff work: this line
	// is not a caption that gets replaced by the multiplier, it *is* the multiplier, sitting in
	// the hand row until the sum calls for it and then flying into the line at the size it was
	// already being read at. A second size here would make the figure jump on the frame it set
	// off, which is exactly the "two numbers swapping" failure the damage figure's handoff is
	// written to avoid.
	//
	// The gap stays proportional to the name, so the pair opens up as the name swells to the
	// shout and the big word never lands on the figure under it.
	mathMultLineSize = mathTermSize
	mathMultLineGap  = 0.10

	// The scales an item is drawn at as it arrives. A flown figure grows into place, which reads
	// as coming toward the reader; an operator and the total drop *onto* the line from bigger,
	// which reads as being stamped there.
	mathFlyFromScale  = 0.45
	mathPopFromScale  = 1.7
	mathTotalPopScale = 2.4
	mathShoutPopScale = 2.1
)

// The script's beats, as fractions of the one playback speed. See the const block above for what
// each one is, and `beat` for why they are written this way.
func mathShoutTicks() int  { return ui.Beat(7, 5) }  // the hand's name popping in
func mathTermTicks() int   { return ui.Beat(9, 10) } // one card's figure flying down into the row
func mathSymbolTicks() int { return ui.Beat(2, 5) }  // a +, an x or an = appearing in place
func mathRelicTicks() int  { return ui.Beat(7, 10) } // one relic's multiplier flying out of its own card
func mathTotalTicks() int  { return ui.Beat(1, 1) }  // the answer landing
func mathHoldTicks() int   { return ui.Beat(8, 5) }  // the finished sum held before the box clears

// bannerFlyTicks() is the hand's name traveling from the planning seat to the hand row when
// DUEL! is pressed — a shade longer than a card's own flight, because it crosses more screen
// and grows by half again while it does it.
func bannerFlyTicks() int { return ui.Beat(1, 1) }

// handNameInk is the color the hand's name is written in, planned and shouted alike, and the
// color the multiplier that comes out of it is written in with it.
//
// **Pink since 2026-08-19** *(owner's call)*, where it was the screen's attention yellow. The yellow had just
// become the lightning element's color as well — see cards.BorderOf — so the loudest word on the
// screen was wearing a hue that also means "this card is lightning", and a lightning figure flying
// up out of a card into the sum was arriving in the name's own color.
//
// **The multiplier follows it rather than staying yellow**, because the multiplier flies out of
// the word: `PAIR!` and `1.5` are one fact said twice, and a figure that leaves a pink word in
// yellow reads as a second thing appearing rather than as that word's own number setting off.
//
// **It stopped being a hue at all on 2026-09-02** *(owner's call)*, and this is the rule the
// screen's palette now runs on: **hue belongs to the elements**, and a hand is not one.
//
// It was the screen's pink, which is also `boostInk` — a *relic's* figure — so a hand's multiplier
// and a relic's multiplier arrived in one color in the same sum, in the one place the player is
// trying to tell them apart. Deep purple fixed that and immediately collided with arcane, which
// was the moment the real problem was visible: the wheel is full. Fire, ice, lightning, earth and
// arcane take five hues, relic takes pink, the two verbs take red and blue, the two sides take
// green and gray. There is no unclaimed hue, and every candidate is a near-collision waiting to be
// re-litigated.
//
// **So a hand is marked by weight and by the amber swatch on its row, not by color.** Nothing
// else in a pane is heavy, the shout is eighty points, and two of the three axes a hand counts on
// — concept and form — have nothing to do with elements in the first place. It takes the ground's
// own ink: the color text on this screen is written in when nothing is claiming it.
var handNameInk = ui.GroundInk

// mathItem is one thing written on the line: a card's figure, an operator, the multiplier, or
// the answer.
//
// **Every item's resting place is computed once, before any of them is shown.** The alternative —
// laying the line out again as each item appears — would recenter the whole sum on every beat, so
// the figures already on screen would crawl sideways while the player was reading them. The line
// is measured in full at the start and revealed left to right into space it has already claimed.
type mathItem struct {
	text string
	size float64
	tint color.RGBA

	// from is where a flying item sets off: the center of the card that paid the figure. Items
	// that are not flown leave this zero and pop in place.
	from image.Point
	fly  bool

	// fromScale is how big a flying item is when it sets off, as a fraction of its own size. Zero
	// means `mathFlyFromScale`, the growing-toward-the-reader gesture every card's figure uses.
	//
	// **The multiplier is the one item that sets off at 1**, because it is not appearing — it has
	// been on the screen under the hand's name since DUEL!, and it flies out of that line at the
	// size it was already being read at. See mathMultLineSize.
	fromScale float64

	// fromDuelist says this item's figure flies out of the acting duelist's own card rather than
	// out of a played card: the DMG the hand was swung at, which is the one figure in the sum that
	// belongs to the fighter instead of to a card. **It is a flag rather than another seat
	// convention** because there is one duelist and any number of cards.
	fromDuelist bool

	// hugPrev and hugNext take the air out of one side of an item, so a bracket sits against the
	// figure it encloses instead of floating a word's width off it. Only the parens use them.
	hugPrev bool
	hugNext bool

	// burst says this item throws the signal's own firework as it sets off, and flies the way a
	// signal flies: out of the thing that produced it at the size a signal's figure is drawn, and
	// receding into its place on the line. **A relic's figure and a rider's are the same event** —
	// something worn paying into this blow — so they are one gesture rather than two.
	burst bool

	// tightPrev sets an item close to the one before it — the inside of a bracket, where the
	// figures are one product — and widePrev sets it far from it, which is what puts the sum's own
	// multiplier apart from the terms it multiplies. See mathTightGap.
	tightPrev bool
	widePrev  bool

	// relicSeat is the worn seat this item's figure flies out of, plus one, and 0 for an item that
	// is not a relic's multiplier. **Plus one so the zero value means "not a relic"**, which is what
	// lets every other item in the script leave the field alone.
	relicSeat int

	// cardSeat is the played card this item's figure flies out of, plus one, on the same
	// convention. Filled by `startHandMath`, which is the half of the box that knows the table.
	cardSeat int

	// signalsSent says this item's card has already thrown whatever its riders parked, so an item
	// held on screen for its whole beat cannot throw them once a frame. **The signals themselves are not stored here** — the
	// scene parked them against a seat when the events went past, and this only says when the seat
	// is up. See combat_signal.go, and takeSignalSeat below.
	signalsSent bool

	// shakeRelics are worn seats that shake as this item runs without their figure being the one
	// flying: the echo relic behind an extra landing, which buys a *term* rather than a multiplier
	// and so has no number of its own in the line.
	shakeRelics []bool

	// at is the item's resting center, filled by layOutMath.
	at image.Point

	// t is this item's own clock, started when the script reaches it.
	t ui.Travel
}

// handBanner is the name of the hand the player has built, and it is **one object with two homes
// rather than two drawings of one word** *(2026-08-19, owner's call)*.
//
// While the round is being planned the name sits in the middle of the player's half of the table,
// where their cards are about to land. When DUEL! is pressed those cards fly *up* into that half
// and the name flies *down* into the hand row they left, at one size the whole way and coming up
// to full alpha as it goes — and there it stays for the rest of the round, which is where the
// multiplier later flies out of it.
//
// **The point is that it never leaves the screen.** It used to be a preview that vanished at DUEL!
// and an unrelated shout that popped into existence several beats later, which asked the player to
// recognize the same word twice rather than to watch it move — the card-flight argument, applied
// to the one thing on this screen that is not a card.
//
// **It ends when its own figure sets off** *(2026-08-19, owner's call)*: the multiplier flies out
// of the second line and into the sum, and the whole banner is taken down on that frame rather
// than left lit over the hand while the sum finishes and the opponent swings back. It has been
// carried down, read, and spent by then — see `advancePlayback`, which is where the clear happens
// because that is where the box's own clock runs.
//
// It holds no rules and cannot change an outcome: the name is the one the resolver already
// decided, and the flight is a drawing on its own clock beside playback.
type handBanner struct {
	// name is the exclamation as it will be read — `PAIR!` — captured at DUEL! from the same
	// `handShout` the fired event goes through, so the word cannot change as it travels.
	name string

	// mult is the second line — `1.15x DMG` — and it travels with the name as one object.
	//
	// **It is what the name is worth** *(2026-08-19, owner's call)*. The hand's name alone says
	// which rung of the ladder was built and says nothing about what building it bought, so the
	// multiplier was a number the player first met when it flew out of the word several beats
	// after the round was committed. Naming it while the hand is still being chosen is what makes
	// the ladder something to play toward rather than something to be told about afterwards.
	//
	// It is a string captured at DUEL! for the reason `name` is: the figure cannot change as it
	// travels, and nothing here formats a second opinion of it — `handMultiplierText` is the same
	// formatting the sum's own term goes through.
	//
	// **It is not replaced by the sum's copy, it becomes it.** The line rests under the name
	// through every card's figure flying down, and the banner is cleared on the frame the sum's
	// multiplier sets off from exactly this spot at exactly this size — see `handMultiplierOrigin`
	// and `mathMultLineSize`. A figure that vanished and popped up elsewhere would be two numbers;
	// one that leaves is one number moving.
	mult string

	// flight is the journey to the hand row. Started when the round starts; once it is done the
	// banner rests in the hand row until the round is over.
	flight ui.Travel

	// flying is set for the whole of the committed half of the banner's life, `flight` still
	// running or not. Without it a banner that had arrived would be indistinguishable from one
	// that had never set off.
	flying bool

	// flash is the swell on the beat the hand actually scores *(owner's call, 2026-09-15)*.
	//
	// **The word travels at DUEL! and then has to wait**, because the defenses now resolve first —
	// so between the journey and the sum there is a stretch of shields going up during which the
	// name is sitting in the hand row saying something that has not happened yet. The flash is what
	// gives it a second moment: it arrives as a promise and snaps once when it becomes the
	// multiplier.
	//
	// **It is a scale and not an alpha or a color.** The banner is already at full alpha by the
	// time it lands, and the hue wheel is full — see CLAUDE.md. Size is the axis the word has left,
	// and it is the one the shout's own pop already uses, so the two read as the same gesture.
	flash ui.Travel
}

// flashTicks() is how long that swell lasts, and bannerFlashScale how far it goes. **Under the
// shout's own pop** (mathShoutPopScale is 2.1): that one is a word arriving out of nothing, where
// this is a word already on screen asking to be looked at again.
func bannerFlashTicks() int { return ui.Beat(3, 5) }

const bannerFlashScale = 1.5

// flashNow starts it. **Safe to call on a banner that is not flying** — a hand the banner never
// carried has nothing to swell, and the box pops its own shout for that case.
func (b *handBanner) flashNow() {
	if !b.flying {
		return
	}
	b.flash = ui.NewTravel(0, bannerFlashTicks())
}

// showing reports whether the banner is already saying this word, which is what stops the hand
// dialog popping a second copy of it on the beat the hand fires.
func (b handBanner) showing(name string) bool {
	return b.flying && name != "" && b.name == name
}

// tick advances the flight. Called every frame from `Update` rather than from playback, because
// the journey starts at DUEL! — before the first event is reached — and must not be held up by a
// dialog that stops the cursor.
func (b *handBanner) Tick() {
	if !b.flying {
		return
	}
	b.flight.Tick()
	b.flash.Tick()
}

// clear takes the banner down: the round is over and the hand it named has been spent.
func (b *handBanner) Clear() { *b = handBanner{} }

// handMathBox is the whole dialog: a shout, a line of items, and a hold at the end.
//
// **It holds no arithmetic and no rules.** Everything in `items` is a string formatted once from
// the event; nothing here can disagree with the resolver, because nothing here computes.
type handMathBox struct {
	active bool

	// shout is the hand's name — `PAIR!` — and shoutAt its center. Empty when no hand was built,
	// which is the No Hand: a lone attack shouting its own name is the same emptying of the
	// word that keeps `HAND!` off a single Bash in the feed.
	shout   string
	shoutAt image.Point
	shoutT  ui.Travel

	items []mathItem

	// at is the item the script is currently running, and equals len(items) once they are all up.
	at int

	// multAt is which item is the multiplier, so the banner knows when its own second line has
	// been taken over by the sum. Filled by `startHandMath`, which is already the half of the box
	// that knows which flying item is which.
	multAt int

	// Hold is the pause on the finished sum, before the box clears and playback resumes.
	Hold ui.Travel

	// side is whose blow this is, and grown[t] is what that duelist's worn relics had accumulated
	// after term t was counted — both copied straight off the event.
	//
	// **This is what makes the relic badges move while the sum is read** *(2026-08-26)*. A growing
	// relic now steps between the terms of one blow, so the row has a different number to show at
	// each beat, and the alternative to carrying the figures here is the relic row re-deriving which
	// relic grew — a resolver in a screen, which is the thing the whole event exists to prevent.
	side  combat.Side
	grown [][]int

	// termOf[i] is how many card terms have been counted by the time item i is up. Punctuation and
	// the annotations share the count of the term they follow, so the row holds still through them
	// and steps on the beat a figure lands.
	termOf []int
}

// termsShown is how many of the blow's terms have been counted so far, which is what the relic row
// reads to know which accumulators to draw.
func (b handMathBox) termsShown() int {
	if !b.active || len(b.termOf) == 0 {
		return 0
	}
	at := b.at
	if at >= len(b.termOf) {
		at = len(b.termOf) - 1
	}
	return b.termOf[at]
}

// shaking is what the item the box is running **right now** sets moving: the worn seats whose cards
// shake, and the played card that does, if any.
//
// **Every figure in the sum is accompanied by its own card shaking** *(owner's call, 2026-08-26)*.
// A card's damage shakes the card, a relic's multiplier shakes that relic, and an echo's extra term
// shakes the relic that bought the landing even though it has no figure of its own on the line. They
// read off the box's item cursor rather than the term cursor, so they happen one at a time in the
// order the engine applied them — which is the whole point of the sequence.
func (b handMathBox) shaking(side combat.Side) (relics []bool, card int, ok bool) {
	if !b.active || b.side != side || b.at >= len(b.items) {
		return relics, 0, false
	}

	it := b.items[b.at]
	// **A copy, because the caller may write the lead seat into it** and the item is the box's own
	// record of what happened. A shared slice would make reading the box change it.
	relics = append([]bool(nil), it.shakeRelics...)
	if seat := it.relicSeat; seat > 0 && seat-1 < len(relics) {
		relics[seat-1] = true
	}
	return relics, it.cardSeat, true
}

// growthNow is the accumulators one side's relics have reached at this point in the sum, and false
// when the box is not running that side's blow or has not reached a term yet.
func (b handMathBox) growthNow(side combat.Side) ([]int, bool) {
	if !b.active || b.side != side {
		return nil, false
	}

	t := b.termsShown()
	if t < 1 || t > len(b.grown) {
		return nil, false
	}
	return b.grown[t-1], true
}

// startHandMath builds the dialog for one KindHand event and starts it running.
//
// **The layout needs the screen and `applyEvent` does not have it** — playback runs from
// `Update`, which does — so this takes `gs` and is called from there.
func (s *CombatScene) startHandMath(gs *state.GlobalState, e combat.Event) {
	if e.Kind != combat.KindHand || e.HandCardCount < 1 {
		return
	}

	box := handMathBox{
		active: true,
		shout:  shoutFor(e),
		shoutT: ui.NewTravel(0, mathShoutTicks()),
		Hold:   ui.NewTravel(0, mathHoldTicks()),
		items:  mathScript(e),
		side:   e.Side,
		grown:  append([][]int{}, e.HandGrown[:e.HandCardCount]...),
	}

	// **The name snaps on this beat** *(owner's call, 2026-09-15)*. It flew to the hand row at
	// DUEL! and has been resting there through the defend phase, which now runs first — so without
	// this the word is on screen for the whole of the shields going up and then simply is there
	// when the sum starts. The flash is what says *now it matters*: this is the beat the hand
	// scores and the multiplier it has been advertising becomes the thing the sum uses.
	s.Theater.banner.flashNow()

	// **The banner is already saying it, so the box does not say it again** *(2026-08-19)*. The
	// player's hand was named at DUEL! and the word has been sitting in the hand row ever since;
	// popping a second copy of it over the top would be the same announcement twice, and the
	// multiplier would fly out of whichever of the two happened to be drawn last. The box keeps
	// its own shout for a hand it did not carry down — an opponent's, which nothing produces
	// today but which the engine can still emit.
	if s.Theater.banner.showing(box.shout) {
		box.shout = ""
	} else if s.Theater.banner.flying {
		// **The announcement wins over the banner it disagrees with.** The two can only differ if
		// the hand that fired is not the hand that was planned — which nothing produces today,
		// since only a chill can take a queued card away and no enemy can put a status on the
		// player — but if it ever does, the truth is the event, and two words at one point would
		// be worse than either alone.
		s.Theater.banner.Clear()
	}

	// **Where each figure sets off from is the screen's business, not the script's.** The script
	// says what the sum reads; this says which card on the table paid which term, which is the one
	// part of the box that has to know how a row is laid out.
	term, raised := 0, false
	for i := range box.items {
		if !box.items[i].fly {
			continue
		}
		// **A relic's figure sets off from the relic**, and is skipped by the term counter: these are
		// interleaved with the cards' own figures, and counting them would pair every figure after
		// the first relic with the wrong card.
		if seat := box.items[i].relicSeat; seat > 0 {
			box.items[i].from = s.relicCardCenter(gs, seat-1)
			continue
		}
		// **The DMG figure comes off the duelist's own card**, which is where the player has just
		// watched a rung relic raise it. It is skipped by the term counter for the relics' reason:
		// one of these leads every term, and counting them would pair every card's multiplier with
		// the card after it.
		if box.items[i].fromDuelist {
			box.items[i].from = s.fighterCardMid(gs, e.Side)

			// **The rung relic shakes on the figure it raised** *(owner's call, 2026-09-19)*. It
			// used to shake on the first card's term, which was the nearest thing available while
			// the raise was invisible; the DMG is now on the line, so the relic can move on the
			// beat its own contribution is read. **The first term only** — the same figure leads
			// every term, and a relic shaking five times would read as five raises.
			if !raised {
				box.items[i].shakeRelics = append([]bool(nil), e.HandBonusSeats...)
				raised = true
			}
			continue
		}
		if term < e.HandCardCount {
			box.items[i].cardSeat = e.HandCards[term] + 1
			box.items[i].shakeRelics = e.HandLanding[term]

			// **A figure is drawn in the color of whatever produced it** *(2026-08-19, owner's
			// call)*, and a card's figure is produced by the card — so it wears that card's
			// element, which is the color of its border. It leaves the card in the card's own
			// color and keeps it all the way into the line, which is what says the sum is made
			// of the cards rather than handed down by the game.
			seat := e.HandCards[term]
			box.items[i].from = s.handCardCenter(gs, e.Side, seat)
			box.items[i].tint = s.handCardInk(e.Side, seat)

			term++
			continue
		}
		// The only flying item past the blow's own cards is the multiplier.
		box.items[i].from = s.handMultiplierOrigin(gs, e)
		box.multAt = i
	}

	// **After the walk, because it needs multAt.** Every flying item but the multiplier is one card's
	// figure, in order, so counting them is what turns an item index into a term index.
	box.termOf = make([]int, len(box.items))
	counted := 0
	for i := range box.items {
		// **The card's own multiplier is what counts a term**, which is the one flying item a term
		// has exactly one of: the DMG leading it belongs to the duelist and a relic's factor
		// belongs to the relic, so counting either would step the row mid-bracket.
		it := box.items[i]
		if it.fly && i != box.multAt && !it.fromDuelist && it.relicSeat == 0 {
			counted++
		}
		box.termOf[i] = counted
	}

	s.layOutMath(gs, &box)
	box.shoutAt = s.handShoutAt(gs)

	s.Theater.mathBox = box

	// **Everything the hand kept back fires now, before the box has run a frame** *(owner's call,
	// 2026-09-10)*. The sum does not begin until those figures have finished their journey — the
	// held cards pay, the figures land on the duelist card, and only then does the hand start
	// counting. `advancePlayback` freezes the box while any signal is up, so releasing them here is
	// what puts them in front of the shout rather than beside it.
	//
	// **Together, because what the turn kept back is one fact about the turn.** A hand holding four
	// paying cards would otherwise be four pauses in front of a sum that has not started.
	s.releaseHeldSignals()

	// **And the rung relic pays into the duelist before the hand is counted** *(owner's call,
	// 2026-09-19)*, on the same beat and for the same reason: the DMG every term of the sum is
	// about to be worked out at is the figure this raises, so the player watches it climb and then
	// watches the sum use it. It goes onto the stage directly rather than being parked — there is
	// no card to park it against, and `advancePlayback` freezes the box while any signal is up.
	s.raiseDMGSignal(e)
}

// raiseDMGSignal throws the figure a rung relic put into this blow's DMG, out of the relic and onto
// the duelist card's DMG row.
//
// **The climb is what it is worth to this blow, not the relic's own number** — see
// combat.Event.DMGRaise, which is the distance the riders left between the two figures.
func (s *CombatScene) raiseDMGSignal(e combat.Event) {
	raise := e.DMGRaise()
	if raise <= 0 {
		return
	}
	seat := firstSeat(e.HandBonusSeats)
	if seat == 0 {
		return
	}
	s.Theater.signals = append(s.Theater.signals, cardSignal{
		dest:   signalRaise,
		amount: raise,
		side:   e.Side,
		relic:  seat,
		t:      ui.NewTravel(0, signalFlyTicks()+signalHoldTicks()),
	})
}

// mathScript is the sum as a list of things to write, in the order they appear: a figure per card
// of the hand, a plus between each pair, then the multiplier, then the answer.
//
// **It takes no screen and computes no arithmetic**, which is what makes it the testable half of
// the box. Every string in it is formatted from a field the resolver already filled, so a test can
// pin what the player is shown without needing a window to draw it in — the same property that
// keeps `internal/combat` testable, applied to the one part of this file that is not geometry.
func mathScript(e combat.Event) []mathItem {
	var items []mathItem

	for i := 0; i < e.HandCardCount; i++ {
		if i > 0 {
			items = append(items, mathOperator("+"))
		}
		items = append(items, termItems(e, i)...)
	}

	// **The rung's own figure is not in this sum and has not been since 2026-09-14** *(owner's
	// call)*. It was a term here from 2026-09-05 — the last of the additions, in the ground's ink
	// — and it is now a raise on the DMG the whole hand is swung at, so it is already inside every
	// card's figure above. Writing it again would print a sum over its own total. What says the
	// relic fired is the first term shaking it on its way past; see startHandMath.

	// **And the cards kept back pay after the ones spent**, on the same terms as the rung's own
	// term above: its own figure, the ground's ink, flying out of the relic that paid it. Two terms
	// rather than one merged figure, because they answer different questions — what the hand
	// formed, and what it is still holding.
	if e.HeldBonus != 0 {
		items = append(items, mathOperator("+"), mathItem{
			text:      strconv.Itoa(e.HeldBonus),
			size:      mathTermSize,
			tint:      ui.GroundInk,
			fly:       true,
			relicSeat: firstSeat(e.HeldBonusSeats),
			t:         ui.NewTravel(0, mathTermTicks()),
		})
	}

	// **And the purse, third and last of the flat terms.** Same ink and same flight as the other
	// two: it is a figure a relic put into the sum, not a number the cards carried.
	if e.VitaeBonus != 0 {
		items = append(items, mathOperator("+"), mathItem{
			text:      strconv.Itoa(e.VitaeBonus),
			size:      mathTermSize,
			tint:      ui.GroundInk,
			fly:       true,
			relicSeat: firstSeat(e.VitaeBonusSeats),
			t:         ui.NewTravel(0, mathTermTicks()),
		})
	}

	// **The multiplier is always shown, the identity included** *(2026-08-19, owner's call)*. It
	// was dropped at `x 1` on the argument that `20 x 1 = 20` is a sum with nothing in it — true
	// of the arithmetic and wrong about the game: **hands are going to be upgradable**, so the
	// No Hand's 1 is a number that will change, and a term that appears only once it stops being
	// 1 would make an upgrade look like a new rule rather than a bigger figure. Every hand's sum
	// reads the same shape, and the one the player sees most is the one teaching it.
	items = append(items, wide(mathOperator("x")), mathItem{
		text: ui.HandMultiplierText(e.Multiplier),
		size: mathTermSize,
		// The hand's own color, because the hand is what produced it — and the word it flies out
		// of is written in it. See handNameInk.
		tint: handNameInk,
		fly:  true,
		// **It sets off at its own size**, unlike a card's figure: it is already on the screen as
		// the banner's second line, so growing into place would make it a new number appearing
		// rather than the one the player has been reading since DUEL!.
		fromScale: 1,
		t:         ui.NewTravel(0, mathTermTicks()),
	})

	// **A rung relic is its own multiplier in the sum, after the hand's** *(owner's call,
	// 2026-09-05)*. It is deliberately not folded into the figure beside it: the hand's number is
	// the rung the player built and is what the banner and the ladder say, so a relic that quietly
	// grew it would make the ladder look wrong. In the pane's pink and flying out of the relic that
	// paid, like every other figure a relic put in this box.
	if e.HandScale != 0 && e.HandScale != 100 {
		items = append(items, wide(mathOperator("x")), mathItem{
			text:      ui.HandMultiplierText(e.HandScale),
			size:      mathTermSize,
			tint:      ui.PaneEdge,
			fly:       true,
			relicSeat: firstSeat(e.HandScaleSeats),
			t:         ui.NewTravel(0, mathTermTicks()),
		})
	}

	return append(items, wide(mathOperator("=")), mathItem{
		text: strconv.Itoa(e.Amount),
		size: mathTotalSize,
		tint: ui.VerbInkFor(combat.CategoryAttack),
		t:    ui.NewTravel(0, mathTotalTicks()),
	})
}

// termItems is one card's term, written as the product the game actually worked out:
// `(12 x 3)` — the DMG the whole hand was swung at, times the card's own multiplier — with every
// relic that priced it as a further factor inside the same bracket.
//
// **The bracket is the point** *(owner's call, 2026-09-19)*. A term used to be the landed figure
// and nothing else, so the DMG under it was arithmetic the player was shown the answer to and
// never the working: a relic that raised DMG by two made every figure on the line bigger with
// nothing on the line saying so. Written this way the raise is visible where it happened — the
// duelist's own figure — and one card's contribution reads as the two things it is made of.
//
// **The flat form survives for the term the split cannot describe.** See combat.Event.TermSplit:
// an echo's fraction and CardDamage's floor can each put the product a point off the figure that
// landed, and a bracket coming to the wrong number is worse than a bare one.
func termItems(e combat.Event, i int) []mathItem {
	dmg, pct, ok := e.TermSplit(i)
	if !ok {
		out := []mathItem{{
			text: strconv.Itoa(e.HandAmounts[i]),
			size: mathTermSize,
			// **The ground ink is a fallback, not the color a term is drawn in.** Every figure
			// wears the color of what produced it, and a card's figure is its card's element —
			// which is a question about a row on a screen, so `startHandMath` fills it in. This
			// half of the box has no screen and must not grow one.
			tint: ui.GroundInk,
			fly:  true,
			t:    ui.NewTravel(0, mathTermTicks()),
		}}
		return append(out, relicFactorItems(e, i)...)
	}

	mult := mathItem{
		// The card's own multiplier, in the card's color and flying out of the card — the seat and
		// the ink are `startHandMath`'s, exactly as they were when this item was the landed figure.
		text:      ui.HandMultiplierText(pct),
		size:      mathTermSize,
		tint:      ui.GroundInk,
		fly:       true,
		tightPrev: true,
		t:         ui.NewTravel(0, mathTermTicks()),
	}
	out := []mathItem{
		openBracket(),
		{
			// **The duelist's figure, flying off the duelist's card.** It is the same number in
			// every term of the blow, which is the fact the line is being rewritten to show: one
			// DMG, several cards taking their own multiple of it.
			text:        strconv.Itoa(dmg),
			size:        mathTermSize,
			tint:        ui.GroundInk,
			fly:         true,
			fromDuelist: true,
			t:           ui.NewTravel(0, mathTermTicks()),
		},
		innerOperator("x"),
		mult,
	}
	out = append(out, relicFactorItems(e, i)...)
	return append(out, closeBracket())
}

// relicFactorItems is every relic that priced one term, in worn order — which is firing order.
//
// **A factor rather than a note** *(owner's call, 2026-09-19)*. It was a label beside the term —
// `1.1x` — because the figure it annotated had already been multiplied by it; the bracket now
// shows the multiplication happening, so the relic is an `x` in the product like the card's own
// multiplier is. A product would still be wrong: one figure per relic is what says which of five
// fingers did what.
func relicFactorItems(e combat.Event, term int) []mathItem {
	var out []mathItem
	for seat, pct := range e.HandRelicScale[term] {
		if g := relicNote(pct, seat); g != nil {
			g.tightPrev = true
			out = append(out, innerOperator("x"), *g)
		}
	}
	return out
}

// openBracket and closeBracket are the punctuation round a term. **They hug what they enclose**:
// at the sum's own item gap a bracket stands a figure's width off its own term and reads as a
// symbol in the sum rather than as something holding it together.
func openBracket() mathItem {
	it := mathOperator("(")
	it.tint = mathBracketInk()
	it.hugNext = true
	return it
}

func closeBracket() mathItem {
	it := mathOperator(")")
	it.tint = mathBracketInk()
	it.hugPrev = true
	return it
}

// mathBracketInk is what the punctuation *inside* a term is drawn in: a step quieter again than
// the sum's own operators, which are already a step quieter than its figures.
//
// **Punctuation that is as loud as a figure is a figure.** A bracket is there to group, so it has
// to be legible and must not be read — three levels of quiet is what keeps the numerals the thing
// the eye lands on while the shape of the line still holds together.
func mathBracketInk() color.RGBA { return systems.ColorToward(ui.GroundInk, ui.ScreenGround, 64) }

// firstSeat is the leftmost worn seat in a set of contributors, as a 1-based relicSeat, or 0.
func firstSeat(paid []bool) int {
	for seat, did := range paid {
		if did {
			return seat + 1
		}
	}
	return 0
}

// relicNote is the little multiplier one relic put on one term, or nil when that relic did not fire.
//
// **It is here rather than on the card** *(owner's call, 2026-08-26)*. Nothing a relic does reaches a
// card's printed damage any more: a growing relic's figure moves between the cards of a single blow —
// the first fire card steps it and the second is counted at the bigger number — so a figure printed
// on the card would be right in one queue position and wrong in every other. The sum is where every
// relic is accounted for, on the beat the term lands, and the relic's own card bounces with it.
//
// **In the relic pink and smaller than the figure it annotates.** The pink already means "a relic did
// this" everywhere else on screen — see boostInk — and the size is what keeps it a label on the term
// rather than a second term in the sum.
//
// **It flies out of its own relic's card** *(owner's call, 2026-08-26)*, like a card's figure flies
// out of the card. Every figure in this box comes from the thing that produced it, and a relic's
// multiplier appearing beside a term it had no visible part in was the one number on the line with
// no source. `startHandMath` fills in where from — it is the half of the box that knows where a row
// is laid out — and `relicSeat` is how it tells these apart from the cards' own figures, which it
// pairs with `HandCards` in order.
//
// **One at a time, in worn order.** The box already runs its items strictly in sequence, so putting
// these in the script *is* the sequencing: the card's figure lands, then the first relic's, then the
// second's, which is the order the engine applied them in.
func relicNote(pct, seat int) *mathItem {
	// **Zero is "did not fire", and it is the only thing that draws nothing.** A relic firing at the
	// identity still fired — a fresh Enflamed is 1x — and its card bounces on this beat, so leaving
	// the figure out would be a card jumping with nothing to show for it. It is also how the player
	// watches a growing relic climb off 1x.
	if pct <= 0 {
		return nil
	}
	return &mathItem{
		fly:       true,
		burst:     true,
		relicSeat: seat + 1,
		// **The same arc a rider's figure takes onto the duelist card** *(owner's call,
		// 2026-09-19)*: it leaves the ring at a signal figure's size and recedes into the line.
		// The relic's own raise already arrives that way, and the two are the same fact — a worn
		// thing paying into this blow — so the eye should not have to learn two gestures for it.
		fromScale: signalFigureSize / mathGrowthSize,
		t:         ui.NewTravel(0, signalFlyTicks()),
		// **The `x` is a separate operator, like every other factor in the bracket** *(2026-09-19)*.
		// It was written `1.1x` while the figure beside it had already been multiplied and the
		// label was all there was to say so; inside a bracket that shows the product being formed,
		// a second way of writing a multiplication is two notations for one thing.
		text: ui.HandMultiplierText(pct),
		size: mathGrowthSize,
		tint: ui.BoostInk,
	}
}

// innerOperator is the `x` inside a term — smaller and set closer than the one that multiplies the
// whole sum, which is how the line says which of its multiplications binds tighter.
func innerOperator(str string) mathItem {
	it := mathOperator(str)
	it.size = mathInnerSymbolSize
	it.tint = mathBracketInk()
	it.tightPrev = true
	return it
}

// wide sets an operator apart from what is on its left: the multiplier that applies to the whole
// sum, and the equals in front of the answer. **It is the pair of the tight join inside a term** —
// what the two together say is which level of the line an operator belongs to.
func wide(it mathItem) mathItem {
	it.widePrev = true
	return it
}

// mathOperator is a `+`, an `x` or an `=`: punctuation, so it pops in place rather than flying.
func mathOperator(str string) mathItem {
	return mathItem{
		text: str,
		size: mathSymbolSize,
		tint: mathOperatorInk(),
		t:    ui.NewTravel(0, mathSymbolTicks()),
	}
}

// shoutFor is the hand's name made into an exclamation: `PAIR!`, `FOUR OF A KIND!`, and
// `NO HAND!` for the turn that built nothing bigger.
//
// **The No Hand is shouted like any other hand.** It is a real entry in the catalog at the identity
// multiplier, it is the commonest turn in the game, and it is the name the banner has been carrying
// since DUEL! — so falling silent here would take the word off the screen at the exact moment the
// blow lands. What it does not get is the lift; see builtARung.
//
// **The name comes from the catalog**, like every other place the screen names a hand, so a hand
// renamed in `data/hands.json` is renamed once. An event naming no hand at all still shouts
// nothing; nothing in the game emits one, since a turn with an attack in it always produces a
// blow. Caps are safe at this size — the kubasta note in CLAUDE.md is about small text, where
// `VITAE` renders as `VITRE`.
func shoutFor(e combat.Event) string {
	hand, ok := combat.HandByID(e.Hand)
	if !ok {
		return ""
	}
	return handShout(hand.Name)
}

// builtARung reports whether this blow built a rung of two cards or more.
//
// **It gates the lift and nothing else** *(owner's call, 2026-09-17)*. Every rung announces —
// `NO HAND!` as much as `PAIR!`, since the bottom of the ladder is a rung the player can price,
// upgrade with a stone and name in a relic like any other. What it may not do is **raise a card**:
// the announcement lifts the cards that *made* the rung, and the No Hand is the turn that made none
// — its `Blow.Rung` is a single card picked by damage rather than by counting, so raising it stood
// one card up as though it had done something while every other attack landed beside it unraised.
// **A card jumping up for having done nothing is worse than no lift.**
//
// See noteHand, which is the one reader.
func builtARung(e combat.Event) bool {
	hand, ok := combat.HandByID(e.Hand)
	return ok && hand.Cards() >= 2
}

// handShout is a hand's name made into the exclamation, and it is one function because the planned
// hand and the fired one have to be the same words. `drawPlannedHand` writes the name during
// selection and `shoutFor` writes it on the beat it fires; two spellings of PAIR would read as two
// different things happening.
func handShout(name string) string { return upper(name) + "!" }

// mathOperatorInk is the ground ink faded toward the screen: an operator is punctuation between
// two figures, and drawing it at the figures' weight would make `+` compete with `20`.
//
// **Faded toward the ground, not scaled toward black.** The combat screen's ground is cream, so
// `ColorAtStrength` would make this *louder* than the figures rather than quieter — the trap
// CLAUDE.md's color section describes, and one this screen has fallen into before.
func mathOperatorInk() color.RGBA { return systems.ColorToward(ui.GroundInk, ui.ScreenGround, 45) }

// handMathRect is the band the sum is written in: the strip above the hand, at the table's
// width.
//
// **It takes its height from the band and its width from the table, and neither is an accident.**
//
// The vertical band is `mathBandHeight`, which is the Resolution feed's collapsed box — the size
// it was when the sum was laid out and looked at against it. The feed itself is gone
// *(2026-08-18)*, and the constant deliberately survived it: the arithmetic was tuned against
// that depth, so changing the number is re-laying out the sum rather than tidying up after a
// deleted pane. While the feed was still there this was computed rather than read off `feedRect`
// for a related reason — a player holding the box open grew it upward, and a dialog that moved
// with it would have re-laid a line of figures out from under a reader mid-flight.
//
// The width was deliberately **not** the feed's. `feedRect` spanned `handBand`, which is a
// function of how many cards are in the hand — fine for left-aligned sentences that simply have
// less room, wrong for a centered line of large figures that does not wrap and cannot shrink. A
// two-card hand gives a band around 330px against a widest sum of roughly 640, so the arithmetic
// would have run off both ends of it in exactly the rounds a duel is decided in.
// `TestTheWidestSumFitsItsBand` is what found that and what holds it. **The trap is still live
// anywhere else that borrows `handBand` for something that is not the hand.**
//
// The table's insets are the right width because the box is drawn on the bare ground rather than
// inside a pane — it is an overlay at the same width as the two rows of cards it is about.
func (s *CombatScene) handMathRect(gs *state.GlobalState) image.Rectangle {
	bottom := handTop(gs) - mathBandGapAboveCards
	return image.Rect(tableInset, bottom-mathBandHeight, gs.ScreenWidth-tableInset, bottom)
}

// layOutMath measures every item and centers the finished line in the band.
func (s *CombatScene) layOutMath(gs *state.GlobalState, box *handMathBox) {
	r := s.handMathRect(gs)

	widths := make([]float64, len(box.items))
	measure := func() float64 {
		total := 0.0
		for i := range box.items {
			widths[i], _ = text.Measure(box.items[i].text, mathFace(gs, box.items[i].size), 0)
			total += widths[i]
			if i > 0 {
				total += gapBefore(box.items, i)
			}
		}
		return total
	}

	total := measure()

	// **A sum too wide for its band is shrunk to fit rather than allowed off the screen**
	// *(2026-08-22)*. Echo made this reachable: a five-card turn whose lead card lands three times
	// is seven terms where the band was laid out for five, and a figure hanging off the edge is
	// worse than a small one — the whole box exists to be read.
	//
	// **Every item shrinks by the same factor**, so the multiplier stays smaller than the terms and
	// the line keeps its shape. Nothing else changes: the script, the colors and the flights are
	// what they were, and a sum that already fits is untouched.
	if band := float64(r.Dx()) - mathBandInset; total > band && total > 0 {
		shrink := band / total
		if shrink < minMathShrink {
			shrink = minMathShrink
		}
		for i := range box.items {
			box.items[i].size *= shrink
		}
		total = measure()
	}

	x := float64(r.Min.X+r.Max.X)/2 - total/2
	cy := (r.Min.Y + r.Max.Y) / 2
	for i := range box.items {
		if i > 0 {
			x += gapBefore(box.items, i)
		}
		box.items[i].at = image.Pt(int(x+widths[i]/2), cy)
		x += widths[i]
	}
}

// gapBefore is the air between one item and the one before it: the sum's own gap, unless a bracket
// on either side of the join asked to hug what it encloses.
//
// **Measured and laid out by the same function**, which is the whole of why it is one: the line is
// centered on a total, so a gap the measurer did not know about puts every figure half a bracket
// off where it was measured to be.
func gapBefore(items []mathItem, i int) float64 {
	switch {
	case items[i].hugPrev || items[i-1].hugNext:
		return mathHugGap
	case items[i].widePrev:
		return mathWideGap
	case items[i].tightPrev:
		return mathTightGap
	}
	return mathItemGap
}

// mathBandInset is the breathing room a sum keeps inside its band, so a shrunk line does not rest
// against the edge it was shrunk away from.
const mathBandInset = 16

// minMathShrink is as small as the sum will ever be drawn, as a fraction of its authored sizes.
// **A floor rather than a guarantee**: past this the figures stop being readable, and a line that
// overflows is a better bug report than one nobody can read.
const minMathShrink = 0.6

// mathFace is the box's type. One font at four sizes; the screen has exactly one face.
func mathFace(gs *state.GlobalState, size float64) *text.GoTextFace {
	return &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: size}
}

// handCardCenter is where a figure sets off from: the middle of the card that paid it, **at rest**.
//
// **Nothing is lifted while a figure is in the air.** The cards go back down when the tally starts
// and the shield pips fly in a phase where nothing is raised at all, so a point measured from the
// lifted position would sit a card's height above the card the figure comes out of.
//
// **It recomputes the seat rather than storing a point**, exactly as every card in flight on this
// screen does — the row re-lays out under a moving thing, and a cached coordinate goes stale.
func (s *CombatScene) handCardCenter(gs *state.GlobalState, side combat.Side, seat int) image.Point {
	var at image.Point
	if side == combat.SideB {
		at = enemySeatAt(gs, seat, len(s.Theater.enemyDealt), s.enemySplit())
	} else {
		at = playedSeatAt(gs, seat, len(s.Theater.resolved), s.playedSplit())
	}
	at = lift(at, false)
	return image.Pt(at.X+cardWidth/2, at.Y+cardHeight/2)
}

// handCardInk is the color of the card in a seat: its element's border, the same color the
// card on the table is wearing round its edge.
//
// **It reads the card rather than the event**, because the event carries one `Element` for the
// blow's lead card and the sum has a figure per card. Nothing here is arithmetic — the amount is
// still the event's — and a seat the table does not hold falls back to the ground ink rather than
// inventing a color.
func (s *CombatScene) handCardInk(side combat.Side, seat int) color.RGBA {
	if side == combat.SideB {
		if seat < 0 || seat >= len(s.Theater.enemyDealt) {
			return ui.GroundInk
		}
		return cards.BorderOf(ui.ArtFor(s.Theater.enemyDealt[seat].card.Element))
	}
	if seat < 0 || seat >= len(s.Theater.resolved) {
		return ui.GroundInk
	}
	return cards.BorderOf(ui.ArtFor(s.Theater.resolved[seat].card.Element))
}

// handCardElement is the element of the card in one seat — handCardInk's question, asked one step
// earlier.
//
// **It exists because a drawing is picked by element and a figure is tinted by color**
// *(2026-09-16)*. The shield pips were a near-white mark multiplied by handCardInk's answer; they
// are five authored marks now, so what a flight has to carry is which one, and deriving that back
// out of a color would be a reverse lookup over a palette.
func (s *CombatScene) handCardElement(side combat.Side, seat int) cards.Element {
	if side == combat.SideB {
		if seat < 0 || seat >= len(s.Theater.enemyDealt) {
			return cards.Basic
		}
		return ui.ArtFor(s.Theater.enemyDealt[seat].card.Element)
	}
	if seat < 0 || seat >= len(s.Theater.resolved) {
		return cards.Basic
	}
	return ui.ArtFor(s.Theater.resolved[seat].card.Element)
}

// handShoutAt is where the hand's name is written when it fires: **across the hand row**, dead
// center of it, whichever side formed the hand.
//
// **It used to stand beside the cards it named** *(until 2026-08-19)*, on whichever side of the
// row was free — which only worked because the ring round those cards said which ones it was
// about. With the ring gone the shout is not a label on a group any more, it is the round's one
// announcement, so it goes where the eye already is: over the hand, in the space the player has
// been reading for the whole planning phase.
//
// **The row is inert while this is up**, so nothing is hidden that could still be acted on — and
// `handRowCenter` rather than `handBand` is what it is centered on, or it would drift sideways as
// the row narrowed under it.
func (s *CombatScene) handShoutAt(gs *state.GlobalState) image.Point {
	return handRowCenter(gs)
}

// handMultiplierOrigin is where the multiplier flies from: **the second line under the hand's
// name**, which is the figure itself sitting in the hand row, not a label about it.
//
// **The figure sets off from where it was already being read** *(2026-08-19, owner's call)*. It
// left the *word* until then — `PAIR!` and `1.5` being one fact said twice — which was right while
// the multiplier was first met on that beat, and became wrong the moment the banner started
// carrying `1.15x DMG` down from DUEL!: a figure leaving the name while the same figure sat
// untouched a line below it is two numbers, not one moving.
//
// **It is the `1.15` inside `1.15x DMG` that flies, so the origin is that run's own center** and
// not the line's. The banner line is centered on the whole string, so setting off from the middle
// of it would put the figure under the `x` and shift it sideways on the first frame — the same
// "two numbers swapping" tell the damage figure's handoff is written to avoid.
//
// **The box's own shout is the fallback**, for the hand the banner did not carry down: an
// opponent's, which nothing produces today but which the engine can still emit. There is no second
// line under that one, so the word is the only thing there to leave.
func (s *CombatScene) handMultiplierOrigin(gs *state.GlobalState, e combat.Event) image.Point {
	if s.Theater.banner.mult == "" || !s.Theater.banner.flying {
		return s.handShoutAt(gs)
	}

	at := s.handShoutAt(gs)
	at.Y += int(multLineDrop(gs, s.Theater.banner.name, mathNameSize))

	line, _ := text.Measure(s.Theater.banner.mult, mathFace(gs, mathMultLineSize), 0)
	figure, _ := text.Measure(ui.HandMultiplierText(e.Multiplier), mathFace(gs, mathMultLineSize), 0)
	at.X += int(figure/2 - line/2)
	return at
}

// --- the clock ---------------------------------------------------------------------------

// running reports whether the box is holding the round. **Playback does not advance while this is
// true**, which is what makes the dialog a beat of the round rather than something drawn over
// one — and it is the only thing on this screen that can stop the cursor.
func (b *handMathBox) Running() bool {
	if !b.active {
		return false
	}
	return b.at < len(b.items) || !b.Hold.Done()
}

// tick runs one frame of the script: the announcement, then each item in turn, then the hold.
//
// **One item at a time and never two at once.** The whole point of the box is that a figure
// arrives, is read, and is then joined by an operator; overlapping the beats would put the sum on
// screen at the speed the feed already manages.
//
// **The announcement beat runs whether or not the box owns the word** *(owner's call, 2026-09-17)*.
// It was gated on `shout` — so an opponent's hand got a beat to be named in and the player's, whose
// name the banner has been carrying since DUEL!, went straight into counting. That left the
// announcement and the tally on one frame, and the hand's cards raised for the whole sum rather
// than for the moment the hand was named. The beat is the same length either way: the box shouts
// into it when it has a word, and the banner flashes into it when it has not. **Every rung gets
// it, the No Hand included** — that rung is announced like any other and only the *lift* is withheld
// from it, see builtARung. See counting below.
func (b *handMathBox) Tick() {
	if !b.active {
		return
	}
	if !b.shoutT.Done() {
		b.shoutT.Tick()
		return
	}
	if b.at < len(b.items) {
		b.items[b.at].t.Tick()
		if b.items[b.at].t.Done() {
			b.at++
		}
		return
	}
	b.Hold.Tick()
}

// **The box carried a defend card's pips until 2026-09-15** *(owner's call)*, flying them on the
// beat that card's figure was written into the sum. That existed because the defend phase came
// *last*: a defense paid a visible 0 into the sum and then did the thing it was for several beats
// later, on a card the player had stopped watching. The phases flipped — see combat.Categories —
// so the raise now happens before the sum exists, with a beat of its own, and the box carrying a
// second copy of it would fly every pip twice. What is left is noteShieldRaise.

// counting reports whether the box has moved past the announcement into the sum itself. **It is
// what puts the hand's cards back down**: they are raised to say which cards the announcement is
// about, and the tally that follows needs no card raised at all — the figures fly out of the cards
// where they stand. See advancePlayback, which is the one reader.
func (b *handMathBox) counting() bool { return b.active && b.shoutT.Done() }

// takeSignalSeat hands back the played seat of the item now running, once, so whatever its riders
// parked can be thrown on the beat that card's own figure sets off.
//
// **Asked on the beat the item starts rather than when it ends**, so a card's signals and its
// figure leave together: they are two things one card did, and staggering them would make the
// grant read as a consequence of the sum rather than of the card. The pips used to ride the same
// mechanism and no longer do — see above.
func (b *handMathBox) takeSignalSeat() (int, bool) {
	if !b.active || b.at >= len(b.items) {
		return 0, false
	}
	it := &b.items[b.at]
	if it.signalsSent {
		return 0, false
	}
	it.signalsSent = true
	if it.cardSeat <= 0 {
		return 0, false
	}
	return it.cardSeat - 1, true
}

// clear takes the box down. Called when the script finishes and whenever a round or a fight
// starts, so a box left up by a screen change cannot outlive the round it describes.
func (b *handMathBox) Clear() { *b = handMathBox{} }

// --- drawing -----------------------------------------------------------------------------

// drawPlannedHand writes the name of the hand the current selection has already formed, in the
// middle of the player's half of the table.
//
// **The name goes where the cards it names are about to land** *(2026-08-19, owner's call)*. The
// half is empty for the whole of the planning phase and fills at DUEL!, so the words are standing
// in the space their own cards fly into — and being on the table rather than in the band above the
// hand leaves the sum's line clear to arrive into.
//
// **It is the only thing saying a hand has formed.** The yellow ring round the cards in the row
// went with the move, so a player choosing a Two Pair out of five sees the *name* and not which
// two pairs earned it. That is the trade, and it is the owner's; it is worth knowing before
// something else is hung off the assumption that the row marks its own hand.
//
// **It breathes** — see mathBreath. Nothing else on the screen moves while the player is choosing,
// which is what makes a slow swell enough to be noticed without being a flash.
//
// **It carries no number, and that is a rule rather than an omission.** `Blow.Base` is the
// resolver working against a strength and a shock roll that have not happened, so a figure shown
// here could be contradicted by the round a second later — worse than no figure. The name is the
// part that is already true.
//
// **Every hand the engine can name is named here, the No Hand included** *(corrected
// 2026-08-19)*. The comment that stood here said the opposite — that `Blow.Formed` kept NO HAND
// out of both the preview and the shout — and both halves of that had stopped being true: the
// predicate is gone, a lone attack card falls back to the catalog's No Hand, and
// `previewAttack` names whatever `BlowFor` returns. A single attack is a hand at the identity
// multiplier, and the word is not emptied by it because the label names the hand rather than
// shouting HAND! — the log still writes a lone attack as an ordinary attack sentence.
//
// It draws nothing once playback starts, because `previewBlow` is gated on `planning()` — so this
// and the real shout can never be on screen together.
func (s *CombatScene) drawPlannedHand(gs *state.GlobalState, screen *ebiten.Image) {
	// The committed half: the same word on its way to, or resting in, the hand row.
	if s.Theater.banner.flying {
		// **The word travels and does not grow.** The journey is what says it has been committed,
		// and the alpha coming up to solid says it with it; swelling as well made the size a
		// second announcement, and the size is the one thing about the name that is the same in
		// both of its homes. See mathNameSize.
		t := ui.EaseOut(s.Theater.banner.flight.Progress())
		at := ui.LerpPoint(tableCenter(gs), handRowCenter(gs), t)
		alpha := mathPreviewAlpha + (1-mathPreviewAlpha)*t

		// **The flash multiplies the breath rather than replacing it**, the same way the shout's
		// pop does, so there is no step in the middle of the only thing moving. A banner that has
		// not been flashed is at popScale(_, a finished travel) = 1 and nothing changes.
		scale := mathBreath(gs) * popScale(bannerFlashScale, s.Theater.banner.flash)

		drawHandName(gs, screen, s.Theater.banner.name, s.Theater.banner.mult, mathNameSize, at,
			scale, float32(alpha))
		return
	}

	blow, ok := s.previewAttack()
	if !ok {
		return
	}

	drawHandName(gs, screen, handShout(blow.Hand.Name), handMultiplierLine(blow.Multiplier),
		mathNameSize, tableCenter(gs), mathBreath(gs), mathPreviewAlpha)
}

// handMultiplierLine is the multiplier written as what it does: `1.15x DMG`.
//
// **The figure goes through `handMultiplierText`**, the same formatting the sum's own multiplier
// term uses, so the number the player reads while planning is character-for-character the one that
// flies into the line when the hand fires. `DMG` rather than `damage` for the reason the cards use
// it — it is the word this game already writes for the stat.
func handMultiplierLine(pct int) string { return ui.HandMultiplierText(pct) + "x DMG" }

// drawHandName draws the hand's name and the multiplier under it as **one object**: two lines that
// breathe, fade and travel together.
//
// **Only the name grows.** The multiplier is written at `mathMultLineSize` wherever it is drawn,
// because it is the figure that later flies into the sum and it has to arrive there the size it
// left, and the name is one size in both of its homes as well. The gap between the two is a
// fraction of the name's size and the whole offset is scaled by the breath — a fixed gap under
// breathing words would read as the lines drifting rather than as the pair breathing.
//
// **The name is bold and the multiplier is not.** Both are `handNameInk`, being one fact said two
// ways, and the weight is what keeps the name first in the reading order.
func drawHandName(gs *state.GlobalState, screen *ebiten.Image, name, mult string, size float64,
	at image.Point, scale float64, alpha float32) {

	drawMathText(gs, screen, name, size, handNameInk, at, scale, alpha, true)
	if mult == "" {
		return
	}
	drawMathText(gs, screen, mult, mathMultLineSize, handNameInk,
		image.Pt(at.X, at.Y+int(multLineDrop(gs, name, size)*scale)), scale, alpha, false)
}

// multLineDrop is how far under the hand's name its multiplier sits, for a name at the given size.
//
// **It is measured rather than written down** because the two lines are different sizes and the
// name's own size moves: the drop is half of each line's measured height plus the gap. One
// function, because the drawing and `handMultiplierOrigin` have to agree on where that line is to
// within a pixel — the figure flies out of exactly the spot it was resting in.
func multLineDrop(gs *state.GlobalState, name string, size float64) float64 {
	_, nameH := text.Measure(name, mathFace(gs, size), 0)
	_, multH := text.Measure("0", mathFace(gs, mathMultLineSize), 0)
	return nameH/2 + size*mathMultLineGap + multH/2
}

// mathBreath is the swell the hand's name is drawn at: a slow expand and contract, forever, for
// as long as the name is up.
//
// **It is read off `gs.Count` rather than kept on the box**, because it belongs to no beat of any
// script — the preview has no clock at all and the shout's own clock finishes while the word is
// still on screen. A free-running tick is what makes both of them breathe at the same rate
// without either owning the other's timing.
//
// It is presentation, and the usual constraint applies: nothing here can change an outcome, and
// pacing is untouched — the box holds the cursor for exactly as long either way.
func mathBreath(gs *state.GlobalState) float64 {
	return 1 + mathBreathAmount*math.Sin(2*math.Pi*float64(gs.Count)/mathBreathTicks)
}

// drawHandMath draws the shout and however much of the sum has been revealed.
//
// **Items past `at` are not drawn at all**, rather than drawn transparent. A figure fading up
// from nothing over the eleven ticks before its turn would make the line look pre-written, which
// is exactly the impression the box exists to break.
func (s *CombatScene) drawHandMath(gs *state.GlobalState, screen *ebiten.Image) {
	b := &s.Theater.mathBox
	if !b.active {
		return
	}

	if b.shout != "" {
		// **The pop and the breath multiply rather than take turns.** The pop is the arrival and
		// the breath is what the word does for the rest of its time on screen; handing over from
		// one to the other at the moment the pop finished would put a step in the middle of the
		// only thing moving.
		drawMathText(gs, screen, b.shout, mathNameSize, handNameInk, b.shoutAt,
			popScale(mathShoutPopScale, b.shoutT)*mathBreath(gs), alphaOf(b.shoutT), true)
	}

	for i := range b.items {
		it := &b.items[i]
		switch {
		case i < b.at:
			drawMathText(gs, screen, it.text, it.size, it.tint, it.at, 1, 1, false)
		case i == b.at:
			drawArrivingMathItem(gs, screen, it)
		}
	}
}

// drawArrivingMathItem is the one item currently in motion.
//
// **A flown figure travels and grows; a stamped one only shrinks onto the line.** The difference
// is the whole grammar of the box: something that flies came from a card, and something that pops
// is punctuation the game supplied.
func drawArrivingMathItem(gs *state.GlobalState, screen *ebiten.Image, it *mathItem) {
	if !it.fly {
		pop := mathPopFromScale
		if it.size == mathTotalSize {
			pop = mathTotalPopScale
		}
		drawMathText(gs, screen, it.text, it.size, it.tint, it.at, popScale(pop, it.t), alphaOf(it.t),
			false)
		return
	}

	t := ui.EaseOut(it.t.Progress())
	from := it.fromScale
	if from == 0 {
		from = mathFlyFromScale
	}
	// **The firework is the signal's, thrown where the figure set off.** See drawBurst, which this
	// shares with the riders rather than reproducing. The seat is what makes one ring's scatter
	// different from the next one's.
	if it.burst {
		drawBurst(screen, it.from, it.t.Age, uint32(it.relicSeat*40503+7), it.tint)
	}
	drawMathText(gs, screen, it.text, it.size, it.tint,
		ui.LerpPoint(it.from, it.at, t), from+(1-from)*t, 1, false)
}

// popScale eases a scale down to 1 from `from`. Used by everything that appears in place.
func popScale(from float64, t ui.Travel) float64 {
	return from + (1-from)*ui.EaseOut(t.Progress())
}

// alphaOf fades an appearing item up over the first half of its beat, so a stamped item is not
// simply absent on one frame and present on the next.
func alphaOf(t ui.Travel) float32 {
	p := t.Progress() * 2
	if p > 1 {
		p = 1
	}
	return float32(p)
}

// drawMathText writes one string centered on a point, at a scale and an alpha.
//
// **Centered by its measured box rather than by an alignment flag**, because the item also scales:
// an aligned draw scales about the text's own origin, so the figure would slide sideways as it
// grew. Measuring and translating by half puts the middle of the glyphs on the point at every
// scale, which is what lets a figure grow without drifting.
func drawMathText(gs *state.GlobalState, screen *ebiten.Image, str string, size float64,
	tint color.RGBA, at image.Point, scale float64, alpha float32, bold bool) {

	if str == "" || alpha <= 0 {
		return
	}
	face := mathFace(gs, size)
	w, h := text.Measure(str, face, 0)

	draw := func(dx float64) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(-w/2, -h/2)
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(float64(at.X)+dx, float64(at.Y))
		op.ColorScale.ScaleWithColor(tint)
		op.ColorScale.ScaleAlpha(alpha)
		text.Draw(screen, str, face, op)
	}

	draw(0)
	if bold {
		// **Faux bold: the same word drawn again a step right**, the pane's own idiom — `text/v2`
		// has no synthetic bold and kubasta ships one weight. The step is drawn *after* the scale,
		// so it is a screen-space thickening and a breathing word does not pulse between bold and
		// not.
		draw(mathBoldStep(size))
	}
}

// mathBoldStep is how far the second pass is offset, in pixels: proportional to the type size,
// never less than one. A single pixel is a bold face at a pane's 22 points and invisible at the
// name's 80.
func mathBoldStep(size float64) float64 {
	if step := size / mathBoldSizeStep; step > mathBoldMinStep {
		return step
	}
	return mathBoldMinStep
}

// upper is the counterpart of `lower` in prose.go, and it is written out rather than
// taken from `strings` for no reason beyond that `lower` is: the two sit next to each other in
// the same sentence-building code and reading one should not send you to a different idiom.
func upper(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		out = append(out, r)
	}
	return string(out)
}
