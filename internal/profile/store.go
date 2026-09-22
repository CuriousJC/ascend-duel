package profile

// **Where the files live, and how one is read and written safely.**
//
// Everything in this file is about the disk rather than about the game, which is what keeps the two
// records above it — the profile and the run snapshot — as plain structs with no opinion about
// where they end up.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DirEnv is the environment variable that moves the whole directory.
//
// **It overrides the directory rather than a file**, so one variable moves the profile and the run
// together and they cannot be pointed at different places by accident. It exists for the reason
// `ASCEND_DUEL_SCENARIO` and `ASCEND_DUEL_IDLE_SECONDS` do: without it, "start again as a new
// player" means going and finding a file under AppData by hand.
const DirEnv = "ASCEND_DUEL_PROFILE"

// dirName is the directory the game keeps its files in, under the platform's config root:
//
//	Windows   %APPDATA%\ascend-duel
//	Linux     ~/.config/ascend-duel
//	macOS     ~/Library/Application Support/ascend-duel
const dirName = "ascend-duel"

// Store is one directory holding one player's files.
//
// **A value rather than a package-level singleton**, so a test writes into a temp directory
// without touching the machine's real profile and without a global to put back afterwards.
type Store struct {
	// dir is where the two files go. Empty means the store could not work out where to write and
	// is *inert*: every load reports nothing and every save reports an error, which is the
	// unwritable-machine case rather than a state to guard against at each call site.
	dir string
}

// Open is the store for this machine, honoring DirEnv.
//
// **It does not create the directory and it does not fail.** A machine whose config root cannot be
// determined gets an inert store, so a launch is never held up by a question about the filesystem;
// the directory itself is created lazily by the first save that has something to write. That
// ordering matters: creating a directory at startup would leave one behind for every player who
// launched the game once and never finished anything.
func Open() Store {
	if dir := os.Getenv(DirEnv); dir != "" {
		return Store{dir: dir}
	}
	root, err := os.UserConfigDir()
	if err != nil {
		return Store{}
	}
	return Store{dir: filepath.Join(root, dirName)}
}

// At is a store rooted at a named directory. For tests, and for anything that wants to be explicit.
func At(dir string) Store { return Store{dir: dir} }

// Dir is where this store writes, or "" for an inert one.
func (s Store) Dir() string { return s.dir }

// path is one file inside the store.
func (s Store) path(name string) string { return filepath.Join(s.dir, name) }

// read unmarshals one file into v.
//
// **A missing file and a corrupt file are the same answer to the caller**: `false, nil` for the
// first and `false, err` for the second, and both mean "carry on as if there were nothing here".
// The error is handed back so a caller can log it — a corrupt file is worth a line in the log,
// because it is the difference between a player who never saved and a player whose save was eaten.
func (s Store) read(name string, v any) (bool, error) {
	if s.dir == "" {
		return false, nil
	}
	raw, err := os.ReadFile(s.path(name))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return false, fmt.Errorf("%s is not readable: %w", s.path(name), err)
	}
	return true, nil
}

// write puts v in a file, atomically.
//
// **Temp file and rename, the way `trace_on.go` writes a capture.** A crash or a power cut partway
// through a plain write leaves a half-written file, and a half-written save is worse than no save:
// it parses as far as it goes and then fails, so the player loses the run *and* cannot be told why.
// A rename is atomic on both platforms the game ships to, so the file on disk is only ever the old
// one or the new one.
//
// **Indented on purpose.** These files are what a bug report will paste, and a diffable one is
// worth the handful of bytes.
func (s Store) write(name string, v any) error {
	if s.dir == "" {
		return errors.New("profile: nowhere to save to")
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	final := s.path(name)
	tmp := final + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, final); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// remove deletes one file, and is content for it not to be there.
func (s Store) remove(name string) error {
	if s.dir == "" {
		return nil
	}
	if err := os.Remove(s.path(name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// checkName refuses a file name that is not one.
//
// **This is the one door out of the store that takes a name from further up**, so a name carrying
// a separator or a `..` would be a way to write anywhere on the machine from a panel button. It is
// refused rather than sanitized — a quietly renamed export is a file nobody can find again.
// **Both separators are refused whatever the platform**, since `filepath` on Linux reads a
// backslash as an ordinary character and would write a file literally called `..\log.json` rather
// than refusing the name a Windows caller meant.
func checkName(name string) error {
	if name == "" || name != filepath.Base(name) || name == "." || name == ".." ||
		strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("profile: %q is not a file name", name)
	}
	return nil
}

// Names lists the files in the store whose names begin with a prefix, sorted.
//
// **Sorted, because the caller is pruning by age and the names lead with a timestamp.** That is
// the whole reason a crash report is named the way it is — see internal/crashlog — and it is what
// lets a directory be swept without reading a single file.
//
// **An inert store and an unreadable directory are both "nothing here".** Listing is something a
// prune does on the way to writing a report, and a prune that failed a crash report would be the
// tidying costing the thing it was tidying up after.
func (s Store) Names(prefix string) []string {
	if s.dir == "" {
		return nil
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

// RemoveFile deletes one named file from the store, and is content for it not to be there.
//
// **The name is checked exactly as WriteExport's is**, and for a sharper reason: this one deletes.
func (s Store) RemoveFile(name string) error {
	if err := checkName(name); err != nil {
		return err
	}
	return s.remove(name)
}

// WriteExport puts one export file in the store's directory and hands back where it went.
//
// **It writes beside the profile rather than beside the executable**, on the rule the whole package
// is under: the install tree is somewhere a shipped game cannot write, and a per-executable
// directory is per-install rather than per-player. So an export lands wherever `ASCEND_DUEL_PROFILE`
// or the platform's config root put the two files the game already keeps.
//
// **The name is the caller's and is checked by checkName**, which is where the rule lives now that
// a crash report deletes by name as well as writing by one.
//
// It is atomic and indented like every other write, for the same two reasons.
func (s Store) WriteExport(name string, v any) (string, error) {
	if err := checkName(name); err != nil {
		return "", err
	}
	if err := s.write(name, v); err != nil {
		return "", err
	}
	return s.path(name), nil
}
