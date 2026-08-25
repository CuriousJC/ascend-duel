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

// Open is the store for this machine, honouring DirEnv.
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
