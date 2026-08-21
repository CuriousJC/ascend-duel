package session

// **Where the run is in its loop, and the one place that moves it on.**
//
// The loop is: fight a duel, take a reward, visit the shop, choose the room ahead, fight again.
// Every scene in it used to name its own successor — the combat screen set PostBattle, the
// post-battle screen set Combat — which meant the shape of the game was four hardcoded jumps in
// four files and nothing anywhere answered "where is this run". Inserting a fifth scene meant
// finding and editing the two either side of it.
//
// **A phase belongs to the run, not to a screen.** It survives a fight the way the deck and the
// purse do, it is what a save file would have to record, and it is the thing the tower's shape is
// read against. So it is here, and `internal/screens` maps a phase onto the scene that draws it —
// see screens/flow.go. The mapping is deliberately that way round: this package must not know
// that a screen exists.

// Phase is one station of the between-fight loop.
//
// **Append at the end, and never renumber.** A phase is a candidate for a save file, and the rule
// that applies to every other ordinal in this game applies here: what gets written down is a name,
// never a number. Until something is written down, the cost of getting this wrong is only that a
// run resumes in the wrong room.
type Phase int

const (
	// PhaseFight is the duel. A run opens here — the first thing that happens is a fight.
	PhaseFight Phase = iota

	// PhaseReward is the worm and the card it eats: the post-battle screen.
	PhaseReward

	// PhaseShop is spending what the fight paid. Not built yet; Advance already routes through
	// it, so the scene arriving is one entry in the screens table rather than a change here.
	PhaseShop

	// PhaseChoice is picking the room ahead, which shapes the opponent in it. Not built yet.
	PhaseChoice
)

// phaseNames is what each phase is called, for traces and for the day a run is written down.
var phaseNames = [...]string{"fight", "reward", "shop", "choice"}

func (p Phase) String() string {
	if p < 0 || int(p) >= len(phaseNames) {
		return "unknown"
	}
	return phaseNames[p]
}

// order is the loop, written once. Advance walks it and wraps.
//
// **The loop is data rather than a switch** so that adding a station is one entry here — which is
// the whole reason the phase left the screens. A scene that is not built yet still holds its place
// in the order; what decides whether it is *shown* is whether `screens` has one registered for it.
var order = []Phase{PhaseFight, PhaseReward, PhaseShop, PhaseChoice}

// Phase is where the run is.
func (s *Session) Phase() Phase { return s.phase }

// Advance moves the run to the next station of the loop.
//
// **It does not decide anything about the fight.** Winning is `WonFight`, which is what moves the
// room counter and pays out; this only says which screen comes next. The two are separate because
// losing advances the phase without advancing the room — a defeat puts the same opponent back up.
func (s *Session) Advance() {
	for i, p := range order {
		if p == s.phase {
			s.phase = order[(i+1)%len(order)]
			return
		}
	}
	s.phase = order[0]
}

// SetPhase puts the run at a named station. **For the title screen starting a run and for tests**
// — nothing in the loop itself should call it, because a scene that jumps the run to a phase of
// its choosing is the hardcoded-successor problem with an extra step.
func (s *Session) SetPhase(p Phase) { s.phase = p }

// PhaseCount is how many stations the loop has. It is the bound a caller walking the loop uses,
// so that a station nothing draws cannot turn a walk into a spin.
var PhaseCount = len(order)
