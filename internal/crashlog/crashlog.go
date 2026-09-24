package crashlog

// **The report itself: what a panic leaves behind on the disk.**

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// Schema is the report format this build writes.
//
// **It earns its place before there is anything to migrate**, on the profile's argument: the one
// version number that cannot be added later is the one a reader already has files without. A crash
// report is read by whatever is on the other end of a bug report, which will not be this build.
const Schema = 1

// The file's name. **The UTC time leads so the directory sorts by when**, and the run code follows
// so a report names a run a player can read off the screen — a code is not unique, since a pinned
// seed deals the same one every launch, so it cannot lead.
const (
	filePrefix = "crash-"
	fileSuffix = ".json"

	// shotSuffix is the screen beside the report. It shares the report's base name, exactly as a
	// companion does, so the two sort together and are pruned together.
	shotSuffix = ".png"

	// stampFormat is the time in the name: compact, sortable, and with nothing in it that a file
	// system objects to. A colon is not a legal character in a Windows file name.
	stampFormat = "20060102T150405Z"
)

// keep is how many reports survive a prune.
//
// **Pruning is not optional.** A config directory that grows without bound is a bug that shows up
// only on the machine of the player who plays most — and a player whose game crashes often is
// exactly that player.
const keep = 8

// State is everything the game knows about itself at the moment it went wrong.
//
// **The caller assembles it rather than this package reaching for it**, which is what keeps the
// dependency arrow pointing down: a crash report is written from internal/game, which can already
// see the whole state, and this package never learns what a scene is.
type State struct {
	// Version is the build, from the linker. Screen is what was being drawn and Phase where the run
	// stood, both as words rather than as the ordinals they are held as.
	Version string
	Screen  string
	Phase   string

	// Tick is the simulation counter, GlobalState.Count.
	Tick int

	// RunSeed is the run, written into the report as its code.
	RunSeed int64

	// InstallID groups several reports from one player and identifies nobody. See profile.Profile.
	InstallID string

	// Run is the run as it would have been saved, and Ledger its account of itself so far. Both may
	// be nil: a panic on the title screen has neither.
	Run    *profile.RunSnapshot
	Ledger []session.LedgerFight

	// Scene is whatever the screen that panicked had to say about itself, or nil for a screen that
	// says nothing. **Assembled by the caller like every other tier**, which is what keeps this
	// package ignorant of what a scene is — see ui.Reporter for the interface a screen opts into
	// and internal/game for the guarded call.
	Scene map[string]any
}

// Report is one crash, as it goes to the file.
//
// **The tiers are in the order the plan puts them in**, outermost first, so a report truncated by
// anything at all still opens with the identity a bug report needs most.
type Report struct {
	Schema   int    `json:"schema"`
	Version  string `json:"version"`
	RunCode  string `json:"runCode"`
	Install  string `json:"installId,omitempty"`
	UTC      string `json:"utc"`
	Platform string `json:"platform"`
	Screen   string `json:"screen"`
	Phase    string `json:"phase,omitempty"`
	Tick     int    `json:"tick"`

	// Panic is the value that was recovered, worded, and Stack the goroutine stack taken at the
	// moment of recovery. **The stack is a string rather than a list of frames**: nothing reads it
	// but a person, and a parsed one would be this package claiming to know what a frame is.
	Panic string `json:"panic"`
	Stack string `json:"stack"`

	Run      *profile.RunSnapshot  `json:"run,omitempty"`
	Ledger   []session.LedgerFight `json:"ledger,omitempty"`
	Problems []Problem             `json:"problems,omitempty"`

	// Scene is the screen's own account of itself. **Flat named values rather than a type**,
	// because no two screens describe the same thing and a struct here would be this package
	// learning what each of them is. See ui.Reporter.
	Scene map[string]any `json:"scene,omitempty"`

	// Screenshot is the name of the picture filed beside this report, and empty when there is
	// none. **An absence is not an error**: a panic inside Update happens between two frames, so
	// there is no half-drawn screen to read and the report simply has no picture.
	Screenshot string `json:"screenshot,omitempty"`

	// Shed names the tiers left out to get the document under its ceiling, in the order they went.
	// **A report says when it is partial** rather than leaving a reader to guess whether a missing
	// ledger means a quiet run or a fat one. See Encode.
	Shed []string `json:"shed,omitempty"`
}

// Build assembles a report. **Separate from writing it** so a test can read what a panic produced
// without a directory, and so the crash screen can be handed the same value the file holds.
func Build(st State, cause any, stack []byte) Report {
	return Report{
		Schema:   Schema,
		Version:  st.Version,
		RunCode:  seeds.Code(st.RunSeed),
		Install:  st.InstallID,
		UTC:      time.Now().UTC().Format(time.RFC3339),
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
		Screen:   st.Screen,
		Phase:    st.Phase,
		Tick:     st.Tick,
		Panic:    fmt.Sprint(cause),
		Stack:    string(stack),
		Run:      st.Run,
		Ledger:   st.Ledger,
		Problems: Problems(),
		Scene:    st.Scene,
	}
}

// Stack is the goroutine stack, for a caller that has just recovered.
//
// **Taken at the recover rather than inside Build**, because by the time a report is being
// assembled the frames that panicked have already unwound.
func Stack() []byte { return debug.Stack() }

// Write puts a report in the store, files the screen beside it, takes a copy of every file named
// beside that, and hands back where the report went.
//
// **A screenshot is not a companion**, which is why it is a parameter rather than another name in
// the list. A companion is a file the game was already writing and this one copies; the picture is
// bytes that exist only because a panic happened, so there is nothing to copy and the caller hands
// over what it captured. It may be nil, and usually is — see internal/game, which captures only on
// a panic raised while the screen was being drawn.
//
// **A companion is a file the game was already writing that the report would be poorer without**,
// and today that is the journal: there is one of it and the next run truncates it, so the run that
// blew up — which is exactly the run worth retracing — would otherwise be overwritten by the next
// launch. See internal/journal.
//
// **The names are the caller's**, on the rule the whole package is under: this one is written from
// internal/game, which can already see the whole state, and nothing here learns what a journal is.
// A copy lands under the report's own base name with the companion's own extension, so the two
// travel together and sort together.
//
// **It prunes first.** A prune afterwards would be a sweep that never runs on the one launch that
// mattered — the game is about to be quit — and a report deleted by its own tidying is worse than
// a directory one file over its allowance.
//
// **Nothing here is fatal and the error is for logging only.** A machine that cannot write a crash
// report is a machine whose crash is not recorded, which is the rule internal/profile is under; the
// crash screen still comes up and still says what happened, with nowhere to point at. **A
// companion that will not copy costs the report nothing**, for the same reason a tier that cannot
// be read is left out of one.
func Write(s profile.Store, r Report, shot []byte, companions ...string) (string, error) {
	if s.Dir() == "" {
		return "", fmt.Errorf("crashlog: nowhere to write to")
	}
	Prune(s)

	base := filePrefix + time.Now().UTC().Format(stampFormat) + "-" + r.RunCode

	// **The picture goes first, so the report can name it.** A report saying where its screenshot
	// went and a screenshot nobody wrote is worse than either on its own, and a picture filed
	// beside a report that never appeared is an orphan the prune would never sweep.
	if len(shot) > 0 {
		name := base + shotSuffix
		if _, err := s.WriteBytes(name, shot); err != nil {
			Note("could not keep the screenshot beside the crash report: %v", err)
		} else {
			r.Screenshot = name
		}
	}

	raw, err := Encode(r)
	if err != nil {
		return "", err
	}
	path, err := s.WriteBytes(base+fileSuffix, raw)
	if err != nil {
		return path, err
	}

	for _, name := range companions {
		if _, err := s.CopyFile(name, base+extOf(name)); err != nil {
			Note("could not keep %s beside the crash report: %v", name, err)
		}
	}
	return path, nil
}

// extOf is a file name's extension, including the dot, or "" if it has none.
//
// **Written here rather than taken from `path/filepath`**, on the storage-boundary rule: this
// package deliberately knows nothing about paths, and an extension is the one thing about a name it
// does need. See doc.go.
func extOf(name string) string {
	if i := strings.LastIndex(name, "."); i > 0 {
		return name[i:]
	}
	return ""
}

// Prune deletes the oldest reports until there is room for one more, and takes each one's
// companions with it.
//
// **It sorts by name, which is what the name's shape is for**: the UTC stamp leads, so the
// alphabetical order and the chronological one are the same and nothing has to open a file to know
// how old it is.
//
// **A report is counted by its own file and nothing else.** Every crash writes a `.json` and may
// write a companion beside it, so counting names would make the allowance depend on how many
// companions a build happens to keep — and shrink the number of crashes kept the day a screenshot
// joined them.
func Prune(s profile.Store) {
	names := s.Names(filePrefix)

	var reports []string
	for _, n := range names {
		if strings.HasSuffix(n, fileSuffix) {
			reports = append(reports, n)
		}
	}

	for len(reports) >= keep {
		base := strings.TrimSuffix(reports[0], fileSuffix)
		for _, n := range names {
			if !strings.HasPrefix(n, base) {
				continue
			}
			if err := s.RemoveFile(n); err != nil {
				// **One failure stops the sweep.** A directory that will not let a file be deleted
				// will not let the next one be deleted either, and a loop that kept trying would be
				// a report spending its last moments on a directory it cannot change.
				return
			}
		}
		reports = reports[1:]
	}
}
