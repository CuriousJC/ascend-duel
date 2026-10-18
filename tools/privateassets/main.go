// Command privateassets holds privateassets/audio/ against data/music.json, and reports, records or
// enforces what is in there.
//
//	go run ./tools/privateassets            # report: what is present, what matches, what is missing
//	go run ./tools/privateassets -write     # record each game-ready file's Bytes and SHA256 in music.json
//	go run ./tools/privateassets -require   # exit non-zero unless the directory is exactly what music.json names
//
// # Why it exists
//
// The audio is not in git — it is built into a game-ready folder outside the repository, mirrored
// into privateassets/audio/ and synced from a private bucket. That buys a problem the rest of the
// catalog does not have — **the thing being shipped is not the thing being reviewed.** A picture is
// in the diff; a sound is on whichever machine synced last, and an audio file is the one asset where
// "is this the file I think it is" cannot be answered by looking at it.
//
// So data/music.json is the reviewable artefact, and this is what keeps it honest in both
// directions: -write records what a machine built, and -require refuses a build against anything
// else.
//
// # What -require refuses, and why each one is worth a failed release
//
//   - **A track music.json names and the directory does not hold.** The game builds fine without its
//     sounds, so a release cut on a machine that never synced ships quiet and nothing goes red.
//   - **A file whose bytes do not hash to what music.json recorded**, or a track never recorded at
//     all. The bucket is the current set, so a mismatch is a file re-made and pushed with nobody
//     committing its record, or a record committed with nothing pushed — either way the release
//     would ship a sound nobody reviewed.
//   - **A file music.json does not name**, or one in a folder. Only what the catalog names is
//     game-ready, and the embed would compile anything in the directory into the binary.
//
// Everything about a record itself — a pack nobody credits, a missing gain — is refused by
// data.ParseMusic, which the game's own tests read through, so the tool and the game agree.
//
// An **empty** catalog with an empty directory passes, deliberately: a game with no sound is a true
// state, and a tool that failed on it would have to be disabled to be adopted.
//
// # What it deliberately does not do
//
// **It does not talk to S3 and it does not convert.** Syncing is `aws s3 sync` and converting is
// ffmpeg, both driven from privateassets/mgmt/. This tool reads a directory, hashes files and
// writes one JSON file; it holds no credentials and opens no socket, which is what makes it safe to
// run in any context.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
)

// dir is the directory this tool is about, relative to the repository root, which is where every
// tool in this repo is run from.
const dir = "privateassets/audio"

// catalogPath is the record of what dir should hold.
const catalogPath = "data/music.json"

// onDisk is one decodable file found in dir.
type onDisk struct {
	bytes  int64
	sha256 string
}

func main() {
	write := flag.Bool("write", false, "record each game-ready file's Bytes and SHA256 in music.json")
	require := flag.Bool("require", false, "exit non-zero unless the directory is exactly what music.json names")
	flag.Parse()

	if *write && *require {
		fmt.Fprintln(os.Stderr, "-write and -require are alternatives: one records what is here, the other refuses it")
		os.Exit(2)
	}

	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", catalogPath, err)
		os.Exit(1)
	}
	cat := data.ParseMusic(raw)

	found, err := scan()
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", dir, err)
		os.Exit(1)
	}

	if *write {
		if err := record(cat, found); err != nil {
			fmt.Fprintf(os.Stderr, "writing %s: %v\n", catalogPath, err)
			os.Exit(1)
		}
		return
	}

	problems := report(cat, found)
	if *require && len(problems) > 0 {
		fmt.Fprintf(os.Stderr, "\n%d problem(s) with %s; refusing the build\n", len(problems), dir)
		os.Exit(1)
	}
}

// scan hashes every decodable audio file under the directory, keyed by its path under it. A file in
// a folder is still found, so that it can be refused: game-ready is flat.
//
// **It skips what go:embed skips**: a name starting with `.` or `_`, file or folder, is not in the
// binary. Anything that is not playable audio — the directory's README — is skipped as
// privateassets.Audio skips it.
func scan() (map[string]onDisk, error) {
	found := make(map[string]onDisk)
	exts := make(map[string]bool, len(data.MusicExts))
	for _, e := range data.MusicExts {
		exts[e] = true
	}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != dir && (strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "_")) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !exts[strings.ToLower(filepath.Ext(d.Name()))] {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		found[filepath.ToSlash(rel)] = onDisk{bytes: int64(len(raw)), sha256: hex.EncodeToString(sum[:])}
		return nil
	})
	return found, err
}

// record writes each present track's Bytes and SHA256 into music.json and leaves every other field
// as it was. **A track whose file is absent keeps what it had**, so recording on a machine that has
// only built some of the catalog does not wipe the rest. A file no record names is reported: a
// record is authored first — it is what the file was built from — so this cannot invent one.
func record(cat data.MusicCatalog, found map[string]onDisk) error {
	named := make(map[string]bool, len(cat.Tracks))
	changed := 0
	for i, t := range cat.Tracks {
		named[t.File] = true
		got, ok := found[t.File]
		if !ok {
			continue
		}
		if t.Bytes != got.bytes || t.SHA256 != got.sha256 {
			changed++
		}
		cat.Tracks[i].Bytes = got.bytes
		cat.Tracks[i].SHA256 = got.sha256
	}

	// No HTML escaping: a pack credited "Tao & Sound" is written as it reads, not as &.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cat); err != nil {
		return err
	}
	if err := os.WriteFile(catalogPath, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("recorded %d track(s) in %s, %d changed\n", len(cat.Tracks), catalogPath, changed)

	var stray []string
	for name := range found {
		if !named[name] {
			stray = append(stray, name)
		}
	}
	sort.Strings(stray)
	if len(stray) > 0 {
		fmt.Printf("\n%d file(s) in %s that no track names; a release refuses them:\n", len(stray), dir)
		for _, name := range stray {
			fmt.Printf("  %s\n", name)
		}
	}
	return nil
}

// report prints the state of the directory and returns one line per problem. The same reading
// serves the bare run and -require, so what a developer is shown and what a release refuses cannot
// drift apart.
func report(cat data.MusicCatalog, found map[string]onDisk) []string {
	fmt.Printf("%s: %d track(s) in %s, %d file(s) present\n", dir, len(cat.Tracks), catalogPath, len(found))

	var problems []string
	named := make(map[string]bool, len(cat.Tracks))
	for _, t := range cat.Tracks {
		named[t.File] = true
		got, present := found[t.File]
		switch {
		case !present:
			problems = append(problems, fmt.Sprintf("%s: named but not present; sync the sounds", t.File))
		case t.SHA256 == "":
			problems = append(problems, fmt.Sprintf("%s: present and never recorded; run -write and commit music.json", t.File))
		case got.sha256 != t.SHA256:
			problems = append(problems, fmt.Sprintf("%s: present, but its bytes are not the recorded ones", t.File))
		}
	}
	for name := range found {
		switch {
		case strings.Contains(name, "/"):
			problems = append(problems, fmt.Sprintf("%s: in a folder; game-ready is flat", name))
		case !named[name]:
			problems = append(problems, fmt.Sprintf("%s: present and named by no track in %s", name, catalogPath))
		}
	}

	if len(problems) == 0 {
		if len(cat.Tracks) == 0 {
			fmt.Println("no tracks are named; the game builds and plays silent")
		} else {
			fmt.Println("every named track is present and matches its record")
		}
		return nil
	}

	sort.Strings(problems)
	fmt.Println()
	for _, p := range problems {
		fmt.Printf("  %s\n", p)
	}
	return problems
}
