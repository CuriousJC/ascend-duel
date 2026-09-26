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

// **The band above the hand**: the strip `handMathRect` measures, which is where the tutorial
// points when it talks about the arithmetic. **It is an overlay and reserves nothing** *(owner's
// call)* — the played cards come down over it, and the hits' lines are placed by `mathLineAt`
// under each card rather than in the band.
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

// The hand dialog: every hit's arithmetic acted out at the size of the screen, on the beat the hand
// fires, **each hit worked out under the card that threw it and all of them at once**.
//
// A turn is a hit per landing, so a five-card turn is five lines of arithmetic, and an echoed card
// is three lines stacked under itself. Each line runs its own script — the DMG flying off the
// duelist, the card's multiplier off the card, each relic's factor off the relic, the hand's
// multiplier off the banner, then the hit's figure — and the lines run in parallel, so a
// simple hit finishes first and its figure flies into the target first. **They do not wait for each
// other**: a line that is done is thrown.
//
// **It says nothing the event does not.** Every figure in it comes off the `KindHand` event —
// `HandAmounts`, `HitAmounts`, `Multiplier` — and nothing here multiplies, adds or rounds. That is
// the rule to hold: this is a second *drawing* of one event, never a second arithmetic. If it ever
// needs a figure the event does not carry, the field goes on the event.
//
// **What became of each hit is read ahead in the log.** The hand event is followed by one outcome
// per thrown hit, and a finished line looks its own up — see hitOutcomes — so a hit that landed
// flies, a miss says MISS, a block says BLOCKED, and a hit after a death fades where it stands. The
// damage that line throws is applied to the target's bar as it lands, and the log's own event for
// it is walked past without a second flight: see CombatScene.throwColumn.
//
// **It may not change an outcome**, the same constraint as playback speed, the debug flags,
// `internal/trace` and every card in flight. What it does change is *pacing* — playback stops
// while it runs, and `advancePlayback` holds the cursor rather than the box racing a dwell it
// cannot fit inside.
//
// **The type is not shrunk to fit** *(owner's call)*. A line is set at the sizes below whatever its
// width, centered on its card, so neighboring lines overlap and a long one runs off the screen.

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
	// **Neither width nor depth is held.** A hit's line is centered on its card and set at these
	// sizes whatever that costs, so lines overlap their neighbors and a card that lands several
	// times stacks lines down the screen and off it.
	mathTermSize = 76

	// mathGrowthSize is a relic's multiplier inside a term, and it is **mathTermSize** — the same
	// size as every other figure on the line *(owner's call, 2026-09-19)*. It was well under it
	// while the relic's figure was an annotation *beside* a term the relic had already been folded
	// into; in the term's product it is one of the factors being multiplied, and a factor set smaller
	// than the ones either side of it reads as a footnote on the product rather than part of it.
	//
	// **It is still its own constant**, because how loud a relic's figure is against the cards'
	// remains an open question: the pink already says whose it is.
	mathGrowthSize = mathTermSize
	mathSymbolSize = 60

	// mathInnerSymbolSize is the `x` inside a term, against mathSymbolSize for the one that
	// multiplies the whole sum. **The smaller operator is the tighter-binding one**, which is the
	// typographic half of the same argument the gaps make: a product inside a term is a detail of
	// that term, and the line's own multiplier is an instruction about everything to its left.
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

	// mathItemGap is the air between one item of a line and the next.
	mathItemGap = 16

	// mathLineGap is the air between the card's bottom edge and the first row of its first hit,
	// mathRowGap the air between two rows of one hit, and mathHitGap the extra air between one hit
	// and the next under the same card, so an echoed card's hits read as separate sums.
	// mathRowShare is a row's height against its digits', leaving room for the drop shadow.
	mathLineGap  = 4
	mathRowGap   = 6
	mathHitGap   = 24
	mathRowShare = 1.15

	// mathUnthrownAlpha is how solid a line stays once it is known its hit was never thrown — the
	// target fell to an earlier hit — and mathVerdictDim how solid the arithmetic under a MISS or a
	// BLOCKED is.
	mathUnthrownAlpha = 0.35
	mathVerdictDim    = 0.45

	// mathTightGap is the air inside a term's product, and mathWideGap the air round the multiplier that
	// applies to the whole line.
	//
	// **Three gaps rather than one, because the line has three levels** *(owner's call)*. At a
	// single gap everything on it is equally far from everything else, so the `x` inside a term and
	// the `x` that multiplies the finished line read as the same operation. What says which is which
	// is distance: a product is set close, a sum is set apart.

	mathTightGap = 7
	mathWideGap  = 30

	// mathBoldMinStep is the smallest faux-bold offset, in pixels. **Bold is the same run drawn
	// again a step to the right** — the pane's own idiom, and for the pane's own reason: `text/v2`
	// has no synthetic bold and kubasta ships one weight. The step is scaled off the type size
	// above this floor, because a one-pixel thickening on an 80-point word is not visible at all.
	mathBoldMinStep  = 1
	mathBoldSizeStep = 32 // one pixel of thickening per this many points

	// mathPreviewAlpha is how solid it is: fully. **A see-through word over a painted backdrop
	// takes on the scene under it**, and the figure glyphs carry their own black contour
	// precisely so they read on anything — a fade throws that away. The breath is what says the
	// word is a proposal rather than an event.
	mathPreviewAlpha = 1.0

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

	// mathMultLineSize is the `1.15x` line under the hand's name, and mathMultLineGap the air
	// between the two, in points of the name's own size.
	//
	// **It is `mathTermSize` — the size a figure is written at in a line — and it does not grow
	// with the name** *(2026-08-19, owner's call)*. That is what makes the handoff work: this line
	// is not a caption that gets replaced by the multiplier, it *is* the multiplier, sitting in
	// the hand row until the line calls for it and then flying into the line at the size it was
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
// up out of a card into its line was arriving in the name's own color.
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

// mathItem is one thing written on a hit's line: a figure, an operator, the multiplier, or the
// hit's total.
//
// **Every item's resting place is computed once, before any of them is shown.** The alternative —
// laying a line out again as each item appears — would recenter it on every beat, so the figures
// already on screen would crawl sideways while the player was reading them. Each line is measured
// in full at the start and revealed left to right into space it has already claimed.
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
	// out of a played card: the DMG the hand was swung at, which is the one figure in a line that
	// belongs to the fighter instead of to a card. **It is a flag rather than another seat
	// convention** because there is one duelist and any number of cards.
	fromDuelist bool

	// burst says this item throws the signal's own firework as it sets off, and flies the way a
	// signal flies: out of the thing that produced it at the size a signal's figure is drawn, and
	// receding into its place on the line. **A relic's figure and a rider's are the same event** —
	// something worn paying into this turn — so they are one gesture rather than two.
	burst bool

	// tightPrev sets an item close to the one before it — inside a term, where the figures are
	// one product — and widePrev sets it far from it, which is what puts the line's own
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

	// cardTerm marks the card's own figure on its line — the one item a hit has exactly one of that
	// comes off the card — and handMult marks the hand's multiplier. The script sets both and
	// `startHandMath` reads them, so nothing has to count items to find out which is which.
	cardTerm bool
	handMult bool

	// cardRider marks a figure one of the card's own riders put on its hit — a Goad's +10, a
	// Hunger's x2. It flies out of the card like the card's term does, but keeps the rider's tint
	// the script gave it: the rider is what produced it, not the card's element.
	cardRider bool

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
// **It ends when its own figure sets off** *(owner's call)*: the multiplier flies out of the second
// line and into every hit's line, and the whole banner is taken down on the frame the first one
// leaves rather than left lit over the hand while the hits finish and the opponent swings back. It has been
// carried down, read, and spent by then — see `advancePlayback`, which is where the clear happens
// because that is where the box's own clock runs.
//
// It holds no rules and cannot change an outcome: the name is the one the resolver already
// decided, and the flight is a drawing on its own clock beside playback.
type handBanner struct {
	// name is the exclamation as it will be read — `PAIR!` — captured at DUEL! from the same
	// `handShout` the fired event goes through, so the word cannot change as it travels.
	name string

	// mult is the second line — `1.15x` — and it travels with the name as one object.
	//
	// **It is what the name is worth** *(2026-08-19, owner's call)*. The hand's name alone says
	// which rung of the ladder was built and says nothing about what building it bought, so the
	// multiplier was a number the player first met when it flew out of the word several beats
	// after the round was committed. Naming it while the hand is still being chosen is what makes
	// the ladder something to play toward rather than something to be told about afterwards.
	//
	// It is a string captured at DUEL! for the reason `name` is: the figure cannot change as it
	// travels, and nothing here formats a second opinion of it — `handMultiplierText` is the same
	// formatting every line's own term goes through.
	//
	// **It is not replaced by the lines' copies, it becomes them.** The line rests under the name
	// until the hits start, and the banner is cleared on the frame the first line's multiplier sets
	// off from exactly this spot at exactly this size — see `handMultiplierOrigin`
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
	// so between the journey and the line there is a stretch of shields going up during which the
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

// mathColumn is one hit's line of arithmetic, run on its own cursor under the card that threw it.
type mathColumn struct {
	// hit is the term on the event, and seat the played card that threw it.
	hit, seat int

	items []mathItem

	// at is the item this line is running, and equals len(items) once they are all up.
	at int

	// cardAt is the card's own figure and multAt the hand's multiplier, as indices into items.
	// **cardAt counts the hit for the relic row**; multAt tells the banner its figure has set off.
	cardAt, multAt int

	// logAt is where this hit's outcome sits in the resolved log, and -1 for a hit that was never
	// thrown because the target fell to an earlier one. See hitOutcomes.
	logAt int

	// thrown says the finished line has been handed on — its figure flown into the target, or
	// marked as a miss or a block — so it happens once. spent says the total has left the line.
	thrown bool
	spent  bool

	// verdict is the word drawn over a line whose hit did not land: MISS, BLOCKED, or "" for a hit
	// that landed. unthrown dims a line whose hit never happened.
	verdict  string
	verdictT ui.Travel
	unthrown bool

	// begun says the line has started running. **A card's second and third hits wait for the one
	// above them** — see Tick — so a line can sit unbegun, and draws nothing, while another is
	// being worked out.
	begun bool
}

// done reports whether every item of the line is up.
func (c mathColumn) done() bool { return c.at >= len(c.items) }

// total is the line's last item: the hit's figure.
func (c mathColumn) total() mathItem { return c.items[len(c.items)-1] }

// mathRef is one item of one line.
type mathRef struct{ col, item int }

// handMathBox is the whole dialog: a shout, a line per hit, and a hold at the end.
//
// **It holds no arithmetic and no rules.** Everything in its lines is a string formatted once from
// the event; nothing here can disagree with the resolver, because nothing here computes.
type handMathBox struct {
	active bool

	// shout is the hand's name — `PAIR!` — and shoutAt its center. Empty when the banner is already
	// carrying the word.
	shout   string
	shoutAt image.Point
	shoutT  ui.Travel

	columns []mathColumn

	// Hold is the pause once every line is finished, before the box clears and playback resumes.
	Hold ui.Travel

	// side is whose turn this is, and grown[t] is what that duelist's worn relics had accumulated
	// after hit t — both copied straight off the event. **This is what makes the relic badges move
	// while the lines are read**: a growing relic steps between the hits of one turn, so the row has
	// a different number to show as they land.
	side  combat.Side
	grown [][]int

	// freshShakes and freshSignals are every item that started since each was last read, for the
	// shakes and the signals to act on once. **Two lists because two readers drain them** on their
	// own schedules — the shakes from Update, the signals from playback.
	freshShakes  []mathRef
	freshSignals []mathRef
}

// growthNow is the accumulators one side's relics have reached at this point, and false when the
// box is not running that side's turn or no hit has been counted yet.
//
// **The furthest hit counted is the one that decides.** The lines run in parallel, so hits are
// counted out of order; the latest of them is the one whose figure the relic row shows.
func (b handMathBox) growthNow(side combat.Side) ([]int, bool) {
	if !b.active || b.side != side {
		return nil, false
	}
	best := -1
	for _, c := range b.columns {
		if c.at > c.cardAt && c.hit > best {
			best = c.hit
		}
	}
	if best < 0 || best >= len(b.grown) {
		return nil, false
	}
	return b.grown[best], true
}

// takeShakes is what the items started since the last call set moving: the worn seats whose cards
// shake, and the played cards that do.
//
// **Every figure is accompanied by its own card shaking** *(owner's call)*. A card's figure shakes
// the card, a relic's multiplier shakes that relic, and an echo's extra hit shakes the relic that
// bought the landing even though it has no figure of its own on the line.
func (b *handMathBox) takeShakes(side combat.Side) (relics []bool, cards []int) {
	if !b.active || b.side != side {
		b.freshShakes = nil
		return nil, nil
	}
	for _, ref := range b.freshShakes {
		it := b.columns[ref.col].items[ref.item]
		for seat, on := range it.shakeRelics {
			if on {
				relics = growTo(relics, seat)
				relics[seat] = true
			}
		}
		if seat := it.relicSeat; seat > 0 {
			relics = growTo(relics, seat-1)
			relics[seat-1] = true
		}
		if it.cardSeat > 0 {
			cards = append(cards, it.cardSeat-1)
		}
	}
	b.freshShakes = nil
	return relics, cards
}

// growTo lengthens a seat row so `seat` can be written.
func growTo(row []bool, seat int) []bool {
	for len(row) <= seat {
		row = append(row, false)
	}
	return row
}

// startHandMath builds the dialog for one KindHand event and starts it running.
//
// **The layout needs the screen and `applyEvent` does not have it** — playback runs from
// `Update`, which does — so this takes `gs` and is called from there. `at` is where the event sits
// in the log, which is where each line's outcome is read ahead from.
func (s *CombatScene) startHandMath(gs *state.GlobalState, e combat.Event, at int) {
	if e.Kind != combat.KindHand || e.HandCardCount < 1 {
		return
	}

	box := handMathBox{
		active: true,
		shout:  shoutFor(e),
		shoutT: ui.NewTravel(0, mathShoutTicks()),
		Hold:   ui.NewTravel(0, mathHoldTicks()),
		side:   e.Side,
		grown:  append([][]int{}, e.HandGrown[:e.HandCardCount]...),
	}

	// **The name snaps on this beat** *(owner's call)*. It flew to the hand row at DUEL! and has
	// been resting there through the defend phase, so without this the word is on screen for the
	// whole of the shields going up and then simply is there when the hits start. The flash is what
	// says *now it matters*.
	s.Theater.banner.flashNow()

	// **The banner is already saying it, so the box does not say it again.** The box keeps its own
	// shout for a hand it did not carry down — an opponent's, which nothing produces today but
	// which the engine can still emit.
	if s.Theater.banner.showing(box.shout) {
		box.shout = ""
	} else if s.Theater.banner.flying {
		// **The announcement wins over the banner it disagrees with.** The two can only differ if
		// the hand that fired is not the hand that was planned, and then the truth is the event.
		s.Theater.banner.Clear()
	}

	outcomes := hitOutcomes(s.log, at)
	for i := 0; i < e.HandCardCount; i++ {
		col := mathColumn{hit: i, seat: e.HandCards[i], items: hitScript(e, i, i == 0), logAt: -1}
		if k, ok := outcomes[i]; ok {
			col.logAt = k
		}
		s.placeFigures(gs, e, &col)
		box.columns = append(box.columns, col)
	}

	s.layOutMath(gs, &box)
	box.shoutAt = s.handShoutAt(gs)

	s.Theater.mathBox = box

	// **Everything the hand kept back fires now, before the box has run a frame** *(owner's
	// call)*. The lines do not begin until those figures have finished their journey —
	// `advancePlayback` freezes the box while any signal is up. **Together, because what the turn
	// kept back is one fact about the turn.**
	s.releaseHeldSignals()

	// **And the rung relic pays into the duelist before the hits are worked out**, on the same beat
	// and for the same reason: the DMG every line is about to use is the figure this raises.
	s.raiseDMGSignal(e)
}

// placeFigures says where each flying item of one line sets off from and what color it is: the
// half of the box that knows the table.
func (s *CombatScene) placeFigures(gs *state.GlobalState, e combat.Event, col *mathColumn) {
	raised := col.hit != 0
	for i := range col.items {
		it := &col.items[i]
		switch {
		case it.cardTerm:
			col.cardAt = i
		case it.handMult:
			col.multAt = i
		}
		if !it.fly {
			continue
		}
		switch {
		case it.relicSeat > 0:
			// **A relic's figure sets off from the relic.**
			it.from = s.relicCardCenter(gs, it.relicSeat-1)
		case it.fromDuelist:
			// **The DMG figure comes off the duelist's own card**, where a rung relic has just
			// raised it. **The rung relic shakes on the first hit's DMG only** — the same figure
			// leads every line, and a relic shaking once per hit would read as that many raises.
			it.from = s.fighterCardMid(gs, e.Side)
			if !raised {
				it.shakeRelics = dmgRaiseSeats(e)
				raised = true
			}
		case it.cardTerm:
			// **A figure is drawn in the color of whatever produced it**, and a card's figure is
			// produced by the card — so it wears that card's element, the color of its border.
			it.cardSeat = col.seat + 1
			it.shakeRelics = e.HandLanding[col.hit]
			it.from = s.handCardCenter(gs, e.Side, col.seat)
			it.tint = s.handCardInk(e.Side, col.seat)
		case it.cardRider:
			// **Out of the card that carries the rider**, and the card shakes as it goes — the rule
			// every figure on the line keeps: whatever produced it is what moves.
			it.cardSeat = col.seat + 1
			it.from = s.handCardCenter(gs, e.Side, col.seat)
		case it.handMult:
			it.from = s.handMultiplierOrigin(gs, e)
		}
	}
}

// hitOutcomes reads ahead from a hand event to the hits it threw, and says where each hit's outcome
// sits in the log, keyed by the hit.
//
// **The hits follow the hand event directly** and are everything the resolver writes for them — the
// outcome, what it drained, what it landed, and a fall — so the walk stops at the first event that
// is none of those. A hit that has no outcome was never thrown.
func hitOutcomes(log []combat.Event, at int) map[int]int {
	out := map[int]int{}
	for k := at + 1; k < len(log); k++ {
		switch log[k].Kind {
		case combat.KindDamage, combat.KindMissed, combat.KindFizzled, combat.KindBlocked:
			if _, seen := out[log[k].Hit]; !seen {
				out[log[k].Hit] = k
			}
		case combat.KindDrained, combat.KindStatus, combat.KindDefeated:
		default:
			return out
		}
	}
	return out
}

// raiseDMGSignal throws the figure a rung relic put into this turn's DMG, out of the relic and onto
// the duelist card's DMG row.
//
// **The climb is what it is worth to this turn, not the relic's own number** — see
// combat.Event.DMGRaise, which is the distance the riders left between the two figures.
func (s *CombatScene) raiseDMGSignal(e combat.Event) {
	raise := e.DMGRaise()
	if raise <= 0 {
		return
	}
	seat := firstSeat(dmgRaiseSeats(e))
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

// hitScript is one hit's line as a list of things to write, in the order they appear: a plus in
// front of every line but the first, the card's term, the hand's multiplier, any relic that
// scales the hand, then the hit's figure.
//
// **It takes no screen and computes no arithmetic**, which is what makes it the testable half of
// the box. Every string in it is formatted from a field the resolver already filled.
func hitScript(e combat.Event, i int, first bool) []mathItem {
	var items []mathItem

	// **The plus sits to the left of the line rather than on a line of its own** *(owner's call)*:
	// the hits add up across the table, and a row given over to one symbol is a row taken from the
	// arithmetic.
	if !first {
		items = append(items, mathOperator("+"))
	}
	items = append(items, termItems(e, i)...)

	// **The multiplier is always shown, the identity included** *(owner's call)*. Hands are going to
	// be upgradable, so the No Hand's 1 is a number that will change, and a term that appeared only
	// once it stopped being 1 would make an upgrade look like a new rule rather than a bigger figure.
	items = append(items, wide(mathOperator("x")), mathItem{
		text: ui.HandMultiplierText(e.Multiplier),
		size: mathTermSize,
		// The hand's own color, because the hand is what produced it.
		tint: handNameInk,
		fly:  true,
		// **It sets off at its own size**: it is already on the screen as the banner's second line,
		// so growing into place would make it a new number appearing.
		fromScale: 1,
		handMult:  true,
		t:         ui.NewTravel(0, mathTermTicks()),
	})

	// **A hand relic multiplies after the hand's own** — the HAND RELICS step — in the pane's pink
	// and flying out of the relic that paid.
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
		text: strconv.Itoa(e.HitAmounts[i]),
		size: mathTotalSize,
		tint: ui.VerbInkFor(combat.CategoryAttack),
		t:    ui.NewTravel(0, mathTotalTicks()),
	})
}

// termItems is one hit's card term, written as the product the game actually worked out:
// `12 x 3` — the DMG the whole hand was swung at, times the card's own multiplier — with every
// relic that priced it as a further factor in the same product.
//
// **The product is the point** *(owner's call)*: a relic that raised DMG by two is visible where it
// happened — the duelist's own figure — and the card's contribution reads as the two things it is
// made of. **It needs no brackets**: the term is a row of its own and everything that multiplies
// the whole of it is on the rows below — see layOutMath.
//
// **The flat form survives for the term the split cannot describe.** See combat.Event.TermSplit:
// an echo's fraction and CardDamage's floor can each put the product a point off the figure that
// landed, and a product coming to the wrong number is worse than a bare figure.
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
			tint:     ui.GroundInk,
			fly:      true,
			cardTerm: true,
			t:        ui.NewTravel(0, mathTermTicks()),
		}}
		out = append(out, playRiderItems(e, i)...)
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
		cardTerm:  true,
		t:         ui.NewTravel(0, mathTermTicks()),
	}

	out := []mathItem{
		{
			// **The duelist's figure, flying off the duelist's card.** It is the same number in
			// every hit of the turn, which is the fact the line is being rewritten to show: one
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
	// **DUELIST, CARD, CARD RELICS**: the duelist's DMG times the card's own multiplier, then the
	// card's own riders, then the relics that priced what the card came to.
	out = append(out, playRiderItems(e, i)...)
	return append(out, relicFactorItems(e, i)...)
}

// playRiderItems is what the hit's own card's riders did to its term, each a row of its own after
// the card's own multiplier and before the relics: the flat DMG first, then the percentage, the
// order the engine applies them in.
//
// **A played card's rider is a step in its own hit's working** *(owner's call, 2026-09-26)*. It
// used to be folded into the DMG every line starts from, so a Hunger doubled the whole turn and no
// line said so; it prices only its own card now, and the row is where the player sees it.
func playRiderItems(e combat.Event, i int) []mathItem {
	var out []mathItem
	rider := func(text string, kind combat.RiderKind) mathItem {
		return mathItem{
			text:      text,
			size:      mathTermSize,
			tint:      systems.UpgradeTint(ui.UpgradeForRider[kind]),
			fly:       true,
			cardRider: true,
			t:         ui.NewTravel(0, mathTermTicks()),
		}
	}
	if add := e.HandPlayAdd[i]; add != 0 {
		out = append(out, wide(mathOperator("+")), rider(strconv.Itoa(add), combat.RiderDamageOnPlay))
	}
	if pct := e.HandPlayPct[i]; pct != 0 && pct != 100 {
		out = append(out, wide(mathOperator("x")), rider(ui.HandMultiplierText(pct), combat.RiderScaleInCombo))
	}
	return out
}

// relicFactorItems is every relic that priced one term, in worn order — which is firing order.
//
// **A factor rather than a note** *(owner's call, 2026-09-19)*: the relic is an `x` like the
// card's own multiplier is. A product would be wrong: one figure per relic is what says which of
// five fingers did what.
//
// **Each relic's factor is a row of its own** *(owner's call)*, under the card's term and above
// the hand's multiplier — so a Cleaver reads `12 x 3`, then `x 4`, then the hand's `x 1.5`, and
// the relic's contribution is a step in the working rather than a figure squeezed into the term.
// Its `x` is therefore the line's own operator rather than the smaller one inside a term.
func relicFactorItems(e combat.Event, term int) []mathItem {
	var out []mathItem
	for seat, pct := range e.HandRelicScale[term] {
		if g := relicNote(pct, seat); g != nil {
			out = append(out, wide(mathOperator("x")), *g)
		}
	}
	return out
}

// mathInnerInk is what the `x` *inside* a term is drawn in: a step quieter again than the line's
// own operators, which are already a step quieter than its figures.
//
// **Punctuation that is as loud as a figure is a figure.** Three levels of quiet is what keeps the
// numerals the thing the eye lands on while the shape of the working still holds together.
func mathInnerInk() color.RGBA { return systems.ColorToward(ui.GroundInk, ui.ScreenGround, 64) }

// firstSeat is the leftmost worn seat in a set of contributors, as a 1-based relicSeat, or 0.
func firstSeat(paid []bool) int {
	for seat, did := range paid {
		if did {
			return seat + 1
		}
	}
	return 0
}

// dmgRaiseSeats is every worn seat that raised the DMG this blow was swung at — a rung relic, the
// cards kept back or the purse — which is what shakes as the DMG figure leaves the duelist card.
func dmgRaiseSeats(e combat.Event) []bool {
	var out []bool
	for _, paid := range [][]bool{e.HandBonusSeats, e.HeldDMGSeats, e.VitaeDMGSeats} {
		for len(out) < len(paid) {
			out = append(out, false)
		}
		for i, did := range paid {
			out[i] = out[i] || did
		}
	}
	return out
}

// relicNote is the little multiplier one relic put on one term, or nil when that relic did not fire.
//
// **It is here rather than on the card** *(owner's call, 2026-08-26)*. Nothing a relic does reaches a
// card's printed damage any more: a growing relic's figure moves between the cards of one turn —
// the first fire card steps it and the second is counted at the bigger number — so a figure printed
// on the card would be right in one queue position and wrong in every other. A hit's line is where
// every relic is accounted for, on the beat the term lands, and the relic's own card bounces with it.
//
// **In the relic pink and smaller than the figure it annotates.** The pink already means "a relic did
// this" everywhere else on screen — see boostInk — and the size is what keeps it a label on the term
// rather than a second term in a line.
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
		// thing paying into this turn — so the eye should not have to learn two gestures for it.
		fromScale: signalFigureSize / mathGrowthSize,
		t:         ui.NewTravel(0, signalFlyTicks()),
		// **The `x` is a separate operator, like every other factor in the term** *(2026-09-19)*:
		// in a row that shows the product being formed, a second way of writing a multiplication
		// is two notations for one thing.
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
	it.tint = mathInnerInk()
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
// hits land. What it does not get is the lift; see builtARung.
//
// **The name comes from the catalog**, like every other place the screen names a hand, so a hand
// renamed in `data/hands.json` is renamed once. An event naming no hand at all still shouts
// nothing; nothing in the game emits one, since a turn with an attack in it always produces a
// turn. Caps are safe at this size — the kubasta note in CLAUDE.md is about small text, where
// `VITAE` renders as `VITRE`.
func shoutFor(e combat.Event) string {
	hand, ok := combat.HandByID(e.Hand)
	if !ok {
		return ""
	}
	return handShout(hand.Name)
}

// builtARung reports whether this turn built a rung of two cards or more.
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

// handMathRect is the strip above the hand at the table's width: where the tutorial points when it
// talks about the arithmetic. The hits themselves are placed by layOutMath.
func (s *CombatScene) handMathRect(gs *state.GlobalState) image.Rectangle {
	bottom := handTop(gs) - mathBandGapAboveCards
	return image.Rect(tableInset, bottom-mathBandHeight, gs.ScreenWidth-tableInset, bottom)
}

// layOutMath measures every hit's working and stacks it under the card that threw the hit.
//
// **A hit is written as a column of rows, not as one line** *(owner's call)*: the card's term
// on a row of its own, then each factor that multiplies everything above it — every relic's, then
// the hand's — a row apiece, and the answer at the bottom, so the working reads downward into its
// result and stays about a card wide. A row starts at every operator marked `wide`; see mathRows.
//
// **A card that lands several times stacks its hits under each other**, each one starting below
// the last row of the one before. **Nothing is shrunk** — see the file comment.
func (s *CombatScene) layOutMath(gs *state.GlobalState, box *handMathBox) {
	below := map[int]float64{}
	for c := range box.columns {
		col := &box.columns[c]
		card := s.handCardCenter(gs, box.side, col.seat)
		y, ok := below[col.seat]
		if !ok {
			y = float64(card.Y + cardHeight/2 + mathLineGap)
		} else {
			y += mathHitGap
		}
		for _, row := range mathRows(col.items) {
			h := mathRowHeight(row)
			layOutLine(gs, row, card.X, int(y+h/2))
			y += h + mathRowGap
		}
		below[col.seat] = y - mathRowGap
	}
}

// mathRows cuts one hit's items into the rows they are stacked in: a new row before every `wide`
// operator — each relic's factor, the hand's multiplier, and the equals in front of the answer.
//
// **The rows are slices of the column's own items**, so laying one out writes the positions the
// drawing reads.
func mathRows(items []mathItem) [][]mathItem {
	var rows [][]mathItem
	start := 0
	for i := 1; i < len(items); i++ {
		if items[i].widePrev {
			rows = append(rows, items[start:i])
			start = i
		}
	}
	if start < len(items) {
		rows = append(rows, items[start:])
	}
	return rows
}

// mathRowHeight is how tall a row stands: its biggest item's figure height with room for the
// drop shadow under the digits.
func mathRowHeight(row []mathItem) float64 {
	size := 0.0
	for _, it := range row {
		size = math.Max(size, it.size)
	}
	return figureHeight(size) * mathRowShare
}

// layOutLine measures one line and centers it on a point.
func layOutLine(gs *state.GlobalState, items []mathItem, cx, cy int) {
	widths := make([]float64, len(items))
	total := 0.0
	for i := range items {
		widths[i] = mathWidth(gs, items[i].text, items[i].size)
		total += widths[i]
		if i > 0 {
			total += gapBefore(items, i)
		}
	}

	x := float64(cx) - total/2
	for i := range items {
		if i > 0 {
			x += gapBefore(items, i)
		}
		items[i].at = image.Pt(int(x+widths[i]/2), cy)
		x += widths[i]
	}
}

// gapBefore is the air between one item and the one before it: the row's own gap, unless the item
// asked to sit tight against a product or wide of what it multiplies.
//
// **Measured and laid out by the same function**, which is the whole of why it is one: the row is
// centered on its card, so a gap the measurer did not know about puts every figure half a gap off
// where it was measured to be.
func gapBefore(items []mathItem, i int) float64 {
	switch {
	case items[i].widePrev:
		return mathWideGap
	case items[i].tightPrev:
		return mathTightGap
	}
	return mathItemGap
}

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
// hand's lead card and every line has its own card. Nothing here is arithmetic — the amount is
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
// carrying `1.15x` down from DUEL!: a figure leaving the name while the same figure sat
// untouched a line below it is two numbers, not one moving.
//
// **It is the `1.15` inside `1.15x` that flies, so the origin is that run's own center** and
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

	line := mathWidth(gs, s.Theater.banner.mult, mathMultLineSize)
	figure := mathWidth(gs, ui.HandMultiplierText(e.Multiplier), mathMultLineSize)
	at.X += int(figure/2 - line/2)
	return at
}

// --- the clock ---------------------------------------------------------------------------

// Running reports whether the box is holding the round. **Playback does not advance while this is
// true**, which is what makes the dialog a beat of the round rather than something drawn over one.
func (b *handMathBox) Running() bool {
	if !b.active {
		return false
	}
	if !b.shoutT.Done() || !b.Hold.Done() {
		return true
	}
	for _, c := range b.columns {
		if !c.done() {
			return true
		}
	}
	return false
}

// Tick runs one frame: the announcement, then the lines, then the hold.
//
// **Different cards' lines advance on the same frame and none waits for another** *(owner's call)*.
// Within a line the items still run one at a time — a figure arrives, is read, and is joined by an
// operator — so a line with fewer terms is finished sooner, and it is thrown the moment it is.
//
// **A card that lands several times builds its hits up one after another** *(owner's call)*: an
// echoed card's second hit does not begin until its first is totaled, and its third waits for the
// second. Three hits of one card running at once is three totals arriving together under one card,
// which reads as one sum written three times rather than a card landing again. See waitsOn.
//
// **The announcement beat runs whether or not the box owns the word.** The box shouts into it when
// it has a word, and the banner flashes into it when it has not; every rung gets it, the No Hand
// included.
func (b *handMathBox) Tick() {
	if !b.active {
		return
	}
	if !b.shoutT.Done() {
		b.shoutT.Tick()
		if b.shoutT.Done() {
			b.begin()
		}
		return
	}

	all := true
	for c := range b.columns {
		col := &b.columns[c]
		col.verdictT.Tick()
		if col.done() {
			continue
		}
		all = false
		if !col.begun {
			continue
		}
		col.items[col.at].t.Tick()
		if col.items[col.at].t.Done() {
			col.at++
			if !col.done() {
				b.started(c, col.at)
			}
		}
	}
	if all {
		b.Hold.Tick()
	}
	b.begin()
}

// begin starts every line whose card has no unfinished hit above it.
func (b *handMathBox) begin() {
	for c := range b.columns {
		if !b.columns[c].begun && !b.waitsOn(c) {
			b.columns[c].begun = true
			b.started(c, 0)
		}
	}
}

// waitsOn reports whether a line is held behind an earlier hit of the same card that is not yet
// totaled.
func (b *handMathBox) waitsOn(c int) bool {
	for k := 0; k < c; k++ {
		if b.columns[k].seat == b.columns[c].seat && !b.columns[k].done() {
			return true
		}
	}
	return false
}

// started records that one item of one line has begun, for the shakes and the signals.
func (b *handMathBox) started(col, item int) {
	if item >= len(b.columns[col].items) {
		return
	}
	ref := mathRef{col, item}
	b.freshShakes = append(b.freshShakes, ref)
	b.freshSignals = append(b.freshSignals, ref)
}

// counting reports whether the box has moved past the announcement into the lines. **It is what
// puts the hand's cards back down**: they are raised to say which cards the announcement is about,
// and the lines that follow need no card raised — the figures fly out of the cards where they stand.
func (b *handMathBox) counting() bool { return b.active && b.shoutT.Done() }

// multStarted reports whether any line's multiplier has set off, which is when the banner — whose
// second line that figure is — is taken down.
func (b *handMathBox) multStarted() bool {
	if !b.active {
		return false
	}
	for _, c := range b.columns {
		if c.at >= c.multAt && c.multAt > 0 {
			return true
		}
	}
	return false
}

// finished is every line that is done and has not been thrown yet.
func (b *handMathBox) finished() []int {
	var out []int
	for c, col := range b.columns {
		if col.done() && !col.thrown {
			out = append(out, c)
		}
	}
	return out
}

// takeSignalSeats hands back the played seat of every card whose own figure has started since the
// last call, once each, so whatever its riders parked is thrown on the beat that card's figure sets
// off. **Asked when the item starts rather than when it ends**, so a card's signals and its figure
// leave together.
func (b *handMathBox) takeSignalSeats() []int {
	var seats []int
	for _, ref := range b.freshSignals {
		it := b.columns[ref.col].items[ref.item]
		if it.cardSeat > 0 {
			seats = append(seats, it.cardSeat-1)
		}
	}
	b.freshSignals = nil
	return seats
}

// Clear takes the box down. Called when the lines finish and whenever a round or a fight starts,
// so a box left up by a screen change cannot outlive the round it describes.
func (b *handMathBox) Clear() { *b = handMathBox{} }

// --- drawing -----------------------------------------------------------------------------

// drawPlannedHand writes the name of the hand the current selection has already formed, in the
// middle of the player's half of the table.
//
// **The name goes where the cards it names are about to land** *(2026-08-19, owner's call)*. The
// half is empty for the whole of the planning phase and fills at DUEL!, so the words are standing
// in the space their own cards fly into — and being on the table rather than in the band above the
// hand leaves the lines clear to arrive into.
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

// handMultiplierLine is the multiplier written as what it does: `1.15x`.
//
// **The figure goes through `handMultiplierText`**, the same formatting the line's own multiplier
// term uses, so the number the player reads while planning is character-for-character the one that
// flies into the line when the hand fires. **It is all figure glyphs**, so it is drawn in the same
// figures as the arithmetic it later flies into — a word after it would put type from the font
// beside a figure from the sheets.
func handMultiplierLine(pct int) string { return ui.HandMultiplierText(pct) + "x" }

// drawHandName draws the hand's name and the multiplier under it as **one object**: two lines that
// breathe, fade and travel together.
//
// **Only the name grows.** The multiplier is written at `mathMultLineSize` wherever it is drawn,
// because it is the figure that later flies into its line and it has to arrive there the size it
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
	multH := figureHeight(mathMultLineSize) * mathRowShare
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

// drawHandMath draws the shout and however much of every line has been revealed.
//
// **Items past a line's cursor are not drawn at all**, rather than drawn transparent. A figure
// fading up from nothing before its turn would make the line look pre-written, which is exactly the
// impression the box exists to break.
func (s *CombatScene) drawHandMath(gs *state.GlobalState, screen *ebiten.Image) {
	b := &s.Theater.mathBox
	if !b.active {
		return
	}

	if b.shout != "" {
		// **The pop and the breath multiply rather than take turns**, so there is no step in the
		// middle of the only thing moving.
		drawMathText(gs, screen, b.shout, mathNameSize, handNameInk, b.shoutAt,
			popScale(mathShoutPopScale, b.shoutT)*mathBreath(gs), alphaOf(b.shoutT), true)
	}

	for c := range b.columns {
		col := &b.columns[c]
		if !col.begun {
			continue
		}
		alpha := float32(1)
		switch {
		case col.unthrown:
			alpha = mathUnthrownAlpha
		case col.verdict != "":
			alpha = mathVerdictDim
		}
		for i := range col.items {
			it := &col.items[i]
			switch {
			case i < col.at:
				// **The total leaves the line when it is thrown**, so it is not drawn twice: the
				// figure in flight is it.
				if col.spent && i == len(col.items)-1 {
					continue
				}
				drawMathText(gs, screen, it.text, it.size, it.tint, it.at, 1, alpha, false)
			case i == col.at:
				drawArrivingMathItem(gs, screen, it)
			}
		}
		if col.verdict != "" {
			at := col.total().at
			drawMathText(gs, screen, col.verdict, mathTermSize, ui.VerbInkFor(combat.CategoryAttack), at,
				popScale(mathPopFromScale, col.verdictT), alphaOf(col.verdictT), true)
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
	if systems.FigureCovers(str) {
		sheet, ink := figureSheetFor(tint)
		systems.DrawFigure(screen, str, sheet, ink, float64(at.X), float64(at.Y),
			figureHeight(size), scale, alpha)
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
