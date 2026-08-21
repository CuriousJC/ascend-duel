package screens

// **The one speed every movement in the game is a fraction of.**
//
// `beatTicks` is the number, `beat(num, den)` is how anything else is written, and there is no
// second clock anywhere. A card's flight, a damage figure's journey, each beat of the hand
// dialog, the pause on a won fight, a prize card settling into the middle of the post-battle
// screen and — when it exists — anything the shop moves are all proportions of the same figure.
//
// **It lived on the combat screen until 2026-08-21**, which meant the between-fight screens were
// outside it: `postbattle.go` was written before `beat` existed and carried raw tick counts, so
// there were two clocks in the game and turning the speed down would have sped up a duel and left
// the reward screen exactly as slow as it was. That is the failure this file exists to prevent,
// and it is the same failure `beat` was introduced to fix inside the duel — fifteen independent
// constants tuned against a speed that then moved.
//
// **Playback is a consumer of this, not the owner of it.** The combat screen's per-event pacing is
// `eventDwells` in combat.go: a multiplier per event kind on top of this number, so "everything is
// too slow" and "prepares are too slow" stay two different edits.
//
// **This is the game-speed setting when there is one.** Scaling this one constant scales
// everything that moves, which is exactly what such a setting has to do — and none of it may
// change an outcome. A whole round is resolved before playback begins, so pacing is something to
// look at.

const (
	// ticksPerSecond is the fixed simulation rate. Ebitengine's Update runs at 60 TPS and Layout
	// pins the resolution, so a tick is a real unit rather than a frame that might not come.
	ticksPerSecond = 60

	// beatTicks is **the game's one speed**: how long an ordinary beat of anything lasts.
	//
	// **Five twelfths of a second since 2026-08-19** *(owner's call, from playing it)*, where it
	// was a second and a quarter. History, because the number has moved for real reasons three
	// times: three seconds an action originally; halved on 2026-08-02 because a six-action round
	// took twenty seconds and the pause between a move and its consequence read as the game
	// hesitating; up to 75 on 2026-08-07 because the live feed had made every beat a sentence to
	// read. **That last reason is gone** — the feed went behind a button on 2026-08-18 and a round
	// narrates itself in pictures now, which is what a quarter of a second a beat is for.
	//
	// A per-kind table of durations was tried and removed on 2026-08-19. Three dwells selected by
	// a switch with a `default` arm meant every event kind added after that switch was written
	// inherited the shortest of them without anyone choosing it — a Defend blunting a Heavy went
	// past faster than the round-start beat. That is not a tuning mistake, it is a shape that
	// produces tuning mistakes. One number and a table of *proportions* is what replaced it.
	beatTicks = 25
)

// beat scales the game's speed by a fraction, and it is **how every clock in the game that is not
// the speed itself is written** *(2026-08-19, owner's call; made the game's rather than the combat
// screen's on 2026-08-21)*.
//
// **Never less than a tick**, or a small enough fraction of a slow enough speed becomes a movement
// that does not happen.
//
// **What is deliberately not written this way is `mathBreathTicks`** — see it for why.
func beat(num, den int) int {
	if ticks := beatTicks * num / den; ticks > 1 {
		return ticks
	}
	return 1
}
