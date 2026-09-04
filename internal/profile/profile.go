package profile

// **The player, as opposed to the run: what has been achieved, what has been unlocked, and whether
// the tutorial has been watched.**
//
// See TODO.md's profile entry for the shape this is filling in, and doc.go for why it is its own
// file rather than a field on the run.

import "sort"

// profileFile is the profile's name inside the store's directory.
const profileFile = "profile.json"

// Version is the profile format this build writes.
//
// **It earns its place before there is anything to migrate**, because the alternative is adding it
// to a format players already have on disk, which is the one migration that cannot be written. One
// integer now, and the rule below, buys every later change.
const Version = 1

// Profile is everything that survives a run.
//
// **Every collection is a sorted list of keys.** Sorted because map iteration order decides nothing
// in this repo and a file that reordered itself on every save would be undiffable; keys because an
// ordinal in a saved file is a bug waiting for the next enum insertion — see doc.go.
type Profile struct {
	// Version is the format this file was written by. See loadProfile for what a future one does.
	Version int `json:"version"`

	// TutorialSeen is whether the teaching run has been finished. **It is the only field the game
	// currently reads**; the rest are recorded and not yet consulted.
	TutorialSeen bool `json:"tutorialSeen"`

	// Achievements is what has been achieved, by key, sorted. **An achievement changes nothing
	// about the game** — it is a record, and one day the thing that gets mirrored to Steam, which
	// is why its keys are an external contract and must not be renamed once shipped.
	Achievements []string `json:"achievements"`

	// Unlocks is what the player has opened up, by key, sorted. **An unlock is an input to the
	// rules**, unlike an achievement: it gates something. Nothing writes to it yet — see TODO.md,
	// where what actually unlocks is still undecided — and it is here so that the file's shape is
	// settled before players have one on disk.
	Unlocks []string `json:"unlocks"`

	// HandsDiscovered is which rungs of the hand ladder have been found, by hand key, sorted.
	// MECHANICS.md has hands as discovered rather than given, persisting on the profile. **Nothing
	// gates the table on it yet**, deliberately: gating it is a balance change and belongs in its
	// own commit where its effect can be seen.
	HandsDiscovered []string `json:"handsDiscovered"`

	// Settings is what the player has chosen about the program rather than about a run: how loud
	// the score is and how fast the game moves. **They are on the profile rather than in a file
	// of their own** *(owner's call, 2026-08-27)* — the profile is already the per-user file the
	// game writes, and a second one would double the migration policy above for two numbers.
	//
	// **Its zero value is not its default**, which is the one thing to know before reading it: a
	// speed of 0 would stop every clock in the game. LoadProfile normalises, and Defaults says
	// what a fresh player gets.
	Settings Settings `json:"settings"`

	// unknown is every field this build did not recognise, kept byte-for-byte so that saving a
	// profile written by a newer build cannot delete what that build recorded. See loadProfile.
	unknown map[string]any
}

// Settings is the pair of preferences the settings screen owns.
//
// **Both are stored as the thing the player set, not as the thing the game uses.** Music is 0..1
// against whatever ceiling internal/music decides is full; speed is a multiplier on the game's one
// clock. Storing a tick count or a device volume would put a tuning decision in a file that
// outlives the build that made it.
type Settings struct {
	// MusicVolume is how loud the score is, 0 (silent) to 1.
	//
	// **A fresh profile is silent.** Music that begins on its own is the first thing a new player
	// reaches for a control to stop, and that rule predates there being anywhere to stop it from.
	MusicVolume float64 `json:"musicVolume"`

	// Fullscreen is whether the game takes the whole display rather than a window.
	//
	// **Its zero value is the right default, unlike the two below it.** A game that seizes the
	// screen on a first launch is one the player has to find a way out of before they have found
	// anything else, and the window it opens in instead is deliberately smaller than the internal
	// resolution — see main. So false means windowed and a fresh profile is windowed.
	//
	// **It is the one setting that changes how the game is drawn rather than what it does.** At
	// 1920x1080 internal, a window is scaled by whatever fraction of the display it occupies and
	// fullscreen on a 1080p panel is the only 1:1 case there is — which is the case the pixel art
	// is authored for. It still may not change an outcome: Layout reports the same two numbers
	// either way.
	Fullscreen bool `json:"fullscreen"`

	// Speed is the game-speed multiplier: 1 is the speed every duration in the game was tuned at,
	// above 1 is faster, below is slower. See internal/screens/clock.go, which is the one clock it
	// scales.
	//
	// **It may never change an outcome.** A whole round is resolved before playback begins, so
	// this moves pictures and nothing else — the same constraint the debug flags are under.
	Speed float64 `json:"speed"`
}

// Speed limits. **A slider that can reach zero would stop the game**, and one that can reach ten
// would make a duel unreadable, so the travel is bounded here rather than by whichever scene draws
// the bar.
const (
	SpeedMin = 0.5
	SpeedMax = 2.0
)

// Defaults is what a player who has never opened the settings screen is playing at.
func Defaults() Settings { return Settings{MusicVolume: 0, Speed: 1} }

// normalise brings a settings block read off disk into range.
//
// **A missing field reads as zero, and zero speed is not a speed.** An older profile has no
// settings block at all, so this is the path every existing profile takes; a hand-edited or
// corrupt one takes it too. Clamping rather than rejecting keeps the rule that nothing about a
// save file may ever fail a launch.
func (s Settings) normalise() Settings {
	if s.Speed == 0 {
		s.Speed = Defaults().Speed
	}
	s.Speed = clamp(s.Speed, SpeedMin, SpeedMax)
	s.MusicVolume = clamp(s.MusicVolume, 0, 1)
	return s
}

func clamp(v, lo, hi float64) float64 {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	default:
		return v
	}
}

// AchievementFirstSteps is awarded for winning a duel — the first enemy of any run, at which point
// it is on the profile for good.
//
// **The key is what is written to disk, so it is fixed from here on.** The name shown to a player
// can change freely; this cannot, without orphaning every profile that already holds it.
const AchievementFirstSteps = "first-steps"

// Has reports whether an achievement has been awarded.
func (p *Profile) Has(key string) bool { return contains(p.Achievements, key) }

// Award records an achievement, and reports whether this was the first time.
//
// **The report is the point.** Awarding is idempotent — it fires on every win and the profile holds
// one — so the boolean is how a caller knows whether anything actually happened, which is what a
// toast or a Steam call will need when either exists.
func (p *Profile) Award(key string) bool { return insert(&p.Achievements, key) }

// Unlocked reports whether something has been unlocked, and Unlock records it.
func (p *Profile) Unlocked(key string) bool { return contains(p.Unlocks, key) }
func (p *Profile) Unlock(key string) bool   { return insert(&p.Unlocks, key) }

// Discovered reports whether a hand has been found, and Discover records it.
func (p *Profile) Discovered(key string) bool { return contains(p.HandsDiscovered, key) }
func (p *Profile) Discover(key string) bool   { return insert(&p.HandsDiscovered, key) }

// insert adds a key to a sorted set and reports whether it was new.
func insert(set *[]string, key string) bool {
	if key == "" || contains(*set, key) {
		return false
	}
	*set = append(*set, key)
	sort.Strings(*set)
	return true
}

func contains(set []string, key string) bool {
	for _, k := range set {
		if k == key {
			return true
		}
	}
	return false
}

// LoadProfile reads the player's profile, or hands back a fresh one.
//
// **A missing file, a corrupt file and a file from the future all produce a usable profile**, and
// the second return says whether what came back may be written over. That flag is the whole
// migration policy:
//
//   - **A file this build wrote, or an older one: writable.** An older file is read by the same
//     unmarshal — a field this build has and that file has not is simply the zero value, which is
//     what a new field should mean.
//   - **A file from a newer build: read-only.** It may hold fields this build has never heard of,
//     and while `unknown` carries them through a re-save, its *meaning* is not knowable — a newer
//     build could have changed what an existing field means as well as added one. Refusing to write
//     costs this session's progress; writing costs everything the newer build recorded.
//   - **A corrupt file: read-only too**, so that a launch cannot overwrite a file someone might yet
//     recover by hand. The error is returned for logging and is never fatal.
func LoadProfile(s Store) (*Profile, bool, error) {
	// Unmarshalled twice: once into the struct the game uses, once into a bag that keeps every
	// field verbatim. Two passes over a file of a few hundred bytes, and the alternative is a
	// hand-written UnmarshalJSON that has to be kept in step with the struct above it.
	var raw map[string]any
	if _, err := s.read(profileFile, &raw); err != nil {
		return freshProfile(), false, err
	}

	var p Profile
	found, err := s.read(profileFile, &p)
	if err != nil {
		return freshProfile(), false, err
	}
	if !found {
		// **An inert store is not writable**, even though there is nothing there to conflict with:
		// "writable" is what a caller checks before spending effort saving, and a store with
		// nowhere to write would otherwise fail every save and log a line each time.
		return freshProfile(), s.Dir() != "", nil
	}

	p.unknown = unrecognised(raw)
	p.Settings = p.Settings.normalise()
	if p.Version > Version {
		return &p, false, nil
	}
	p.Version = Version
	return &p, true, nil
}

// freshProfile is the profile of a player the game has never met.
func freshProfile() *Profile { return &Profile{Version: Version, Settings: Defaults()} }

// SaveProfile writes the profile.
func SaveProfile(s Store, p *Profile) error {
	p.Version = Version
	return s.write(profileFile, p.merged())
}

// merged is the profile as it goes to disk: this build's fields, plus every field it did not
// recognise, put back exactly as it found them.
func (p *Profile) merged() map[string]any {
	out := map[string]any{}
	for k, v := range p.unknown {
		out[k] = v
	}
	out["version"] = p.Version
	out["tutorialSeen"] = p.TutorialSeen
	out["achievements"] = nonNil(p.Achievements)
	out["unlocks"] = nonNil(p.Unlocks)
	out["handsDiscovered"] = nonNil(p.HandsDiscovered)
	out["settings"] = p.Settings
	return out
}

// known is every field name this build writes, which is how unrecognised tells the two apart.
var known = map[string]bool{
	"version": true, "tutorialSeen": true,
	"achievements": true, "unlocks": true, "handsDiscovered": true,
	"settings": true,
}

func unrecognised(raw map[string]any) map[string]any {
	var out map[string]any
	for k, v := range raw {
		if known[k] {
			continue
		}
		if out == nil {
			out = map[string]any{}
		}
		out[k] = v
	}
	return out
}

// nonNil writes an empty set as `[]` rather than `null`, so the file reads as a list nobody has
// added to rather than as a field nobody has written.
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
