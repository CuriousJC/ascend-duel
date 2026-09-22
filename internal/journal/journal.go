package journal

// **The file itself: when it is opened, when it is truncated, and what happens when it cannot be
// written.**

import (
	"runtime"
	"time"

	"github.com/curiousjc/ascend-duel/internal/profile"
)

// Schema is the journal format this build writes. See Header.
const Schema = 1

// FileName is the one journal, in the profile's own directory.
//
// **A fixed name rather than one per run**, which is the whole of the retention policy: starting a
// run truncates it, so there is nothing to sweep up and nothing that grows without bound. What
// makes that safe is the copy a crash takes — see internal/crashlog, which is the only other thing
// that names this file.
const FileName = "journal.jsonl"

// Journal is the run's choices on their way to the disk.
//
// **A value rather than a package-level singleton**, exactly as profile.Store is one and for the
// same reason: a test writes into a temp directory without a global to put back afterwards. It is
// carried on state.GlobalState beside the store it writes through.
//
// **Every method is safe on a nil receiver**, which is what lets a scene write a line without
// asking whether there is a journal. A test scene, a review tool and a build that never opened one
// all get a journal that records nothing rather than a nil check at forty call sites.
type Journal struct {
	store profile.Store

	// onFail is what a caller wants said when the journal gives up, or nil for a journal that
	// gives up quietly.
	//
	// **A callback rather than a call into internal/crashlog**, because a crash report takes a copy
	// of this file and so crashlog has to be able to name it — one of the two has to point at the
	// other, and the one that knows what a player should be told is the caller. See main, which is
	// where the wording lives.
	onFail func(error)

	// tick is what the next record is stamped with, set once a frame by At.
	tick int

	// open says a header has been written, so a stray choice cannot start a headerless file. It
	// goes false again on a write that failed for good.
	open bool

	// told is whether onFail has already been called.
	//
	// **Once a session, not once per failure.** A journal stops writing the moment it fails, so
	// there is one failure per attempt — but a new run attempts again, and a broken directory would
	// otherwise put a box in front of the player at the top of every climb. The first one has
	// already said everything the second would.
	told bool
}

// New is a journal writing into a store, telling onFail once if it ever gives up.
//
// **It touches no file**: a journal that created its file at startup would leave one behind for
// every player who launched the game and never played.
func New(s profile.Store, onFail func(error)) *Journal {
	return &Journal{store: s, onFail: onFail}
}

// At sets the tick every record from here on is stamped with.
//
// **Set once a frame rather than passed to each call**, exactly as trace.Tick and crashlog.Tick
// are, so a scene recording a click does not have to be holding the game's counter to say when it
// happened.
func (j *Journal) At(tick int) {
	if j == nil {
		return
	}
	j.tick = tick
}

// Begin opens a journal for a run that is starting: the old file goes and this header is the first
// line of a new one.
//
// **Starting a run truncates it.** There is one journal and it belongs to the run being played; a
// climb that ended without going wrong is one nobody is going to ask about.
func (j *Journal) Begin(h Header) {
	if j == nil || j.store.Dir() == "" {
		return
	}
	if err := j.store.RemoveFile(FileName); err != nil {
		j.fail(err)
		return
	}
	j.open = true
	h.Resumed = false
	j.header(h)
}

// Resume opens a journal for a run that was already being played: the file stays and this header is
// appended to it.
//
// **A Continue that cleared the file would throw away everything before the last launch**, which is
// most of the run it is about. Two headers in one file is one climb played across two sessions, and
// Resumed is what says so.
func (j *Journal) Resume(h Header) {
	if j == nil || j.store.Dir() == "" {
		return
	}
	j.open = true
	h.Resumed = true
	j.header(h)
}

// header stamps and writes one.
func (j *Journal) header(h Header) {
	h.T, h.Kind, h.Schema = j.tick, KindHeader, Schema
	h.Platform = runtime.GOOS + "/" + runtime.GOARCH
	h.UTC = time.Now().UTC().Format(time.RFC3339)
	j.append(h)
}

// Write puts one choice in the file.
//
// **A record before a header is dropped**, rather than starting a file nothing can say which build
// or which run wrote. That is not a state the game reaches — main opens the journal before the
// first scene draws — and it is guarded because the failure is silent: a headerless journal reads
// as a journal until somebody tries to use it.
func (j *Journal) Write(r Record) {
	if j == nil || !j.open {
		return
	}
	r.T = j.tick
	j.append(r)
}

// append is the one door to the disk, and the one place a failure is handled.
func (j *Journal) append(v any) {
	if err := j.store.AppendLine(FileName, v); err != nil {
		j.fail(err)
	}
}

// fail gives up on the journal and reports it once.
//
// **It stops writing rather than retrying.** Everything that fails here fails for a reason that
// does not go away inside one session — a read-only directory, a full disk — so a journal that kept
// trying would be a failed write per click for the rest of the run.
func (j *Journal) fail(err error) {
	j.open = false
	if j.onFail == nil || j.told {
		return
	}
	j.told = true
	j.onFail(err)
}
