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

	// unknown is every field this build did not recognise, kept byte-for-byte so that saving a
	// profile written by a newer build cannot delete what that build recorded. See loadProfile.
	unknown map[string]any
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
		return &Profile{Version: Version}, false, err
	}

	var p Profile
	found, err := s.read(profileFile, &p)
	if err != nil {
		return &Profile{Version: Version}, false, err
	}
	if !found {
		// **An inert store is not writable**, even though there is nothing there to conflict with:
		// "writable" is what a caller checks before spending effort saving, and a store with
		// nowhere to write would otherwise fail every save and log a line each time.
		return &Profile{Version: Version}, s.Dir() != "", nil
	}

	p.unknown = unrecognised(raw)
	if p.Version > Version {
		return &p, false, nil
	}
	p.Version = Version
	return &p, true, nil
}

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
	return out
}

// known is every field name this build writes, which is how unrecognised tells the two apart.
var known = map[string]bool{
	"version": true, "tutorialSeen": true,
	"achievements": true, "unlocks": true, "handsDiscovered": true,
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
