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
// **The table names all of them, including the ones that take music.Score.** This file is the
// answer to "what plays where", and a screen answered by a zero value is a screen this page does
// not mention — which is how the score came to be the music on nearly every screen and the one
// piece not written down anywhere near the decision. A screen missing from the table still gets
// the score rather than silence; that is a floor under a mistake, not a way to spell an
// intention.

import (
	"github.com/curiousjc/ascend-duel/internal/music"
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
}

// The loops, by the names they are loaded under. A track's name is its filename stem, so these
// are the files in privateassets/audio/. music.Score is the third piece and is named by
// internal/music, being the one that is always present.
const (
	darkFantasy01 = "dark-fantasy-01"

	// Loaded and playable, and no screen asks for it yet. **Named here anyway**, because the
	// question this file answers is what music the game has as much as where each piece plays,
	// and a loop mentioned nowhere is a loop nobody knows is going spare.
	darkFantasy02 = "dark-fantasy-02"
)

var screenTunes = map[state.ActiveScreen]tune{
	state.Title:      {track: music.Score},
	state.Ascend:     {track: music.Score},
	state.Combat:     {track: music.Score},
	state.PostBattle: {track: music.Score},
	state.Shop:       {track: darkFantasy01},
	state.RunOver:    {track: music.Score},

	// Opened from anywhere, and back where they came from. See the note above.
	state.Settings:     {keep: true},
	state.Achievements: {keep: true},
	state.Credits:      {keep: true},
	state.Animations:   {keep: true},

	// Opened from the shop's shelf and returns to it, which is the same shape — and the case
	// where cutting the music would be most obviously wrong, since the good was bought on the
	// screen still playing it.
	state.Goods: {keep: true},
}

// updateScore puts the right music on for whatever screen is up.
func (g *Game) updateScore() {
	t, listed := screenTunes[g.GlobalState.ActiveScreen]
	switch {
	case !listed:
		// A screen nobody added to the table. The score rather than silence, and rather than
		// whatever the screen before it happened to be playing.
		music.PlayTrack(music.Score)
	case t.keep:
	default:
		music.PlayTrack(t.track)
	}
}
