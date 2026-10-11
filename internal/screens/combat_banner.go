package screens

// combat_banner.go — the hand's name and its multiplier, carried from the planning seat down to the
// hand row at DUEL! and flashed on the beat the hand scores. One object with its own clock, beside
// the hand dialog it hands its multiplier to — see combat_mathbox.go.

import "github.com/curiousjc/ascend-duel/internal/ui"

// bannerFlyTicks() is the hand's name traveling from the planning seat to the hand row when
// DUEL! is pressed — a shade longer than a card's own flight, because it crosses more screen
// and grows by half again while it does it.
func bannerFlyTicks() int { return ui.Beat(1, 1) }

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
