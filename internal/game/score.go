package game

// score.go — which music plays on which screen.
//
// **The frame decides this and no scene does**, for the reason the settings cog and the ledger
// are here: the score runs for the whole session across every screen, so the thing that says
// which music a screen gets is owned by the frame rather than by a dozen scenes. The alternative
// was a call in each scene's Init plus a matching one to put the score back, which is two edits
// per screen and one of them is the one that gets forgotten — leaving a shop's loop playing over
// a duel.
//
// **It is read every frame rather than on a transition.** music.PlayTrack does nothing when the
// track is already the one sounding, so there is no previous screen to remember and no way for
// the table and what is audible to drift apart. A screen re-entered mid-run, a scene whose Init
// ran twice and a run resumed onto a saved phase all come out the same.
//
// # Two answers, and the second is easy to miss
//
// A screen either **names a track** or **leaves the music alone**. Settings, Achievements,
// Credits, the debug gallery and an opened sealed good are all reached *from* somewhere and go
// back to it — so they are transparent to the music the way they are transparent to the run, and
// a player who opens the volume bar to turn the shop's loop up should not have the shop's loop
// stop to let them.
//
// **The restart falls out of that for free.** Shop to Settings and back asks for a track that is
// already sounding, which PlayTrack ignores, so the loop plays through the visit rather than
// starting over — without anything here knowing what a return is.
//
// **This is deliberately not chromeShowing.** That predicate answers whether the frame draws its
// own controls, and its list is close enough to be tempting and wrong in three places: Title and
// PostBattle stand the chrome down while naming their own music, and RunOver is a destination
// rather than an overlay — the run is already over, so there is nothing to be transparent *to*.
// Two questions that agree about five screens and disagree about three are two tables.
//
// # Every screen is listed
//
// **The table names all of them, including the ones that take music.Default.** This file is the
// answer to "what plays where", and a screen answered by a zero value is a screen this page does
// not mention — which is how the score came to be the music on nearly every screen and the one
// piece not written down anywhere near the decision. A screen missing from the table still gets
// the score rather than silence; that is a realm under a mistake, not a way to spell an
// intention.

import (
	"math/rand"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/music"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// tune is what a screen does about music. **One table rather than a map of tracks beside a set of
// exceptions**, because a screen appearing in both of those would be a contradiction nothing
// catches, and here it cannot be written down.
type tune struct {
	// track names a piece of music. A name nothing was loaded under falls back to the score
	// inside music.PlayTrack, which is what lets this table name a loop on a build that does not
	// have it — see privateassets/README.md.
	track string

	// keep leaves whatever is sounding alone. For a screen that is opened from somewhere and
	// returns there, rather than one the run walks to.
	keep bool

	// pool plays one of the tracks data/music.json lets this screen play, rolled per fight off
	// seeds.CombatMusic. See poolPick.
	pool bool
}

// The loops, by the names they are loaded under. A track's name is its filename stem, so these
// are the tracks data/music.json names, and a screen may only play a track whose `Screens` there
// list it — TestEveryTrackPlaysWhereItMay holds the two together. music.Default is named by
// internal/music.
const (
	darkFantasy01 = "dark-fantasy-01"

	// Loaded and playable, and no screen asks for it yet. **Named here anyway**, because the
	// question this file answers is what music the game has as much as where each piece plays,
	// and a loop mentioned nowhere is a loop nobody knows is going spare.
	darkFantasy02 = "dark-fantasy-02"
)

var screenTunes = map[state.ActiveScreen]tune{
	// The title shares the fight's pool, drawn for the run it would continue — so Continue walks
	// into a fight already playing its track. The studio splash, before this table, plays the
	// default — started by the loader in main.go.
	state.Title: {pool: true},

	// A fight plays one track of its pool, the same one for the whole fight and for a retry of it.
	state.Combat: {pool: true},

	state.PostBattle: {track: darkFantasy01},
	state.Shop:       {track: darkFantasy01},
	state.RunOver:    {pool: true},
	state.Portal:     {pool: true},
	state.Victory:    {track: music.Default},

	// The screen a panic ends on. **The default rather than {keep: true}**, which would leave the
	// shop's loop playing over a crash report — a destination like RunOver, not an overlay the
	// player came to and will go back from. There is nothing to be transparent *to*: whatever
	// was happening is over.
	state.Crashed: {track: music.Default},

	// Opened from anywhere, and back where they came from. See the note above.
	state.Settings:     {keep: true},
	state.Achievements: {keep: true},
	state.Compendium:   {keep: true},
	state.Credits:      {keep: true},
	state.Animations:   {keep: true},

	// Opened from the shop's shelf and returns to it, which is the same shape — and the case
	// where cutting the music would be most obviously wrong, since the good was bought on the
	// screen still playing it.
	state.Goods: {keep: true},
}

// musicPools is every screen's pool, in data/music.json's file order — the order poolPick draws
// against, so it is the file and never a map walk that decides which index is which track.
var musicPools = func() map[string][]string {
	pools := map[string][]string{}
	for _, t := range data.LoadAudio().Music {
		for _, s := range t.Screens {
			pools[s] = append(pools[s], t.Name())
		}
	}
	return pools
}()

// poolPick is the track a pool screen plays in a fight: one draw off seeds.CombatMusic, so a fight
// resumed or retried plays the same track. **Every track in the pool is a candidate whether or not
// it was loaded**, so what a fight plays is a fact about the run and the catalog, not about which
// machine synced the sounds; one that was not loaded falls back to the score inside PlayTrack. An
// empty pool is the score.
func poolPick(screen state.ActiveScreen, runSeed int64, fight int) string {
	pool := musicPools[screen.String()]
	if len(pool) == 0 {
		return music.Default
	}
	rng := rand.New(rand.NewSource(seeds.ForFight(runSeed, seeds.CombatMusic, fight)))
	return pool[rng.Intn(len(pool))]
}

// updateScore puts the right music on for whatever screen is up.
//
// **A pool screen picks once, on the way in**, and holds that track until it is left: the fight
// counter moves on a win while the combat screen is still up, and a pick read every frame would
// change track under the last blow. Coming back from Settings re-enters and picks again, which
// lands on the same track, since the fight has not changed.
func (g *Game) updateScore() {
	screen := g.GlobalState.ActiveScreen
	entered := !g.scoreStarted || screen != g.scoreScreen
	g.scoreScreen, g.scoreStarted = screen, true

	t, listed := screenTunes[screen]
	switch {
	case !listed:
		// A screen nobody added to the table. The score rather than silence, and rather than
		// whatever the screen before it happened to be playing.
		music.PlayTrack(music.Default)
	case t.keep:
	case t.pool:
		if entered {
			fight := 0
			if g.GlobalState.Run != nil {
				fight = g.GlobalState.Run.Fight()
			}
			g.poolTrack = poolPick(screen, g.GlobalState.RunSeed, fight)
		}
		music.PlayTrack(g.poolTrack)
	default:
		music.PlayTrack(t.track)
	}
}
