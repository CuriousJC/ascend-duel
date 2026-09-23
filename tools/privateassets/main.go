// Command privateassets reads privateassets/audio/ and its manifest, and reports, records or
// enforces what is in there.
//
//	go run ./tools/privateassets            # report: what is present, what matches, what is missing
//	go run ./tools/privateassets -write     # rewrite manifest.json from what is on disk
//	go run ./tools/privateassets -require   # exit non-zero unless every manifested file is present and attributed
//
// # Why it exists
//
// privateassets/ holds the music loops, which are not in git — they are synced from a private
// bucket instead. That buys a problem the rest of the catalog does not have — **the thing being shipped is not the
// thing being reviewed.** A picture is in the diff; a sound is on whichever machine synced the
// bundle last, and an audio file is the one asset where "is this the file I think it is" cannot
// be answered by looking at it.
//
// So the manifest is the reviewable artefact, and this is what keeps it honest in both
// directions: -write records what a build machine has, and -require refuses a build against
// anything else.
//
// # The three things -require refuses, and why each one is worth a failed release
//
//   - **A file the manifest names and the directory does not hold.** This is the silent failure
//     the whole arrangement exists to prevent: the game builds fine without its sounds, so a
//     release cut on a machine that never synced the bundle ships quiet and nothing goes red.
//   - **A file whose bytes do not hash to what the manifest recorded.** A bundle prefix is meant
//     to be immutable, and an overwritten one means two releases claiming one bundle shipped
//     different audio.
//   - **A file belonging to no pack.** A pack is the batch a file arrived in, and it is the only
//     thing the manifest says about where a file came from — so a file in no pack is one nothing
//     can be traced back through.
//
// An **empty** manifest passes -require, deliberately. "This game has no sound effects yet" is a
// true state and was the state for the whole of its development; a tool that failed on it would
// have to be disabled to be adopted.
//
// # What it deliberately does not do
//
// **It does not talk to S3.** Syncing is `aws s3 sync`, which is one line, already installed on
// both runners and on any machine with credentials, and wrapping it would mean an AWS SDK in
// go.mod for a job a CLI already does. This tool reads a directory and hashes files; it holds no
// credentials and opens no socket, which is what makes it safe to run in any context.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// dir is the directory this tool is about, relative to the repository root, which is where every
// tool in this repo is run from.
const dir = "privateassets/audio"

// manifestName sits inside dir and is committed; see privateassets/README.md.
const manifestName = "manifest.json"

// audioExts is the set of extensions the game can actually decode. It matches the filter in
// privateassets.Audio and the two have to stay in step: a file this tool records and the
// loader skips would be a manifested sound that is not in the binary.
var audioExts = map[string]bool{".ogg": true, ".wav": true, ".mp3": true}

// Manifest is the committed record of what the bundle holds.
//
// **Bundle is the S3 prefix the files came from**, and moving it is how a new set of sounds is
// adopted — never by overwriting a prefix in place, which would leave two releases claiming one
// bundle and shipping different audio.
type Manifest struct {
	Bundle string `json:"Bundle"`
	Packs  []Pack `json:"Packs"`
	Files  []File `json:"Files"`
}

// Pack is where a file came from — one batch, named. Nothing beyond the name is recorded here:
// a file belongs to a pack, and that is the whole of what the manifest says about its origin.
type Pack struct {
	Name string `json:"Name"`
	Note string `json:"Note,omitempty"`
}

// File is one shipped sound. Name is the filename as it sits in the directory, extension
// included — the loader keys by stem, but the manifest is about files on disk.
type File struct {
	Name   string `json:"Name"`
	Pack   string `json:"Pack"`
	Bytes  int64  `json:"Bytes"`
	SHA256 string `json:"SHA256"`
}

func main() {
	write := flag.Bool("write", false, "rewrite manifest.json from what is on disk, preserving pack attributions")
	require := flag.Bool("require", false, "exit non-zero unless every manifested file is present, matching and attributed")
	flag.Parse()

	if *write && *require {
		fmt.Fprintln(os.Stderr, "-write and -require are alternatives: one records what is here, the other refuses it")
		os.Exit(2)
	}

	man, err := readManifest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", filepath.Join(dir, manifestName), err)
		os.Exit(1)
	}

	onDisk, err := scan()
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", dir, err)
		os.Exit(1)
	}

	if *write {
		if err := rewrite(man, onDisk); err != nil {
			fmt.Fprintf(os.Stderr, "writing %s: %v\n", filepath.Join(dir, manifestName), err)
			os.Exit(1)
		}
		return
	}

	problems := report(man, onDisk)
	if *require && len(problems) > 0 {
		fmt.Fprintf(os.Stderr, "\n%d problem(s) with %s; refusing the build\n", len(problems), dir)
		os.Exit(1)
	}
}

// readManifest reads the committed manifest. A missing one is an error rather than an empty
// manifest: the file is committed, so its absence means the directory is not what this tool
// thinks it is, and inventing one would write a fresh manifest over whatever is actually wrong.
func readManifest() (Manifest, error) {
	var man Manifest
	raw, err := os.ReadFile(filepath.Join(dir, manifestName))
	if err != nil {
		return man, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	// An unknown field is a manifest written by a newer build, or a typo. Either way, reading it
	// as if it were understood is how a field that was meant to say something gets silently
	// dropped by the next -write.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&man); err != nil {
		return man, err
	}
	return man, nil
}

// scan hashes every decodable audio file in the directory. Anything else — the README, the
// manifest, an original somebody left beside its converted copy — is not a shipped sound and is
// skipped, exactly as privateassets.Audio skips it.
func scan() (map[string]File, error) {
	found := make(map[string]File)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !audioExts[strings.ToLower(filepath.Ext(e.Name()))] {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(raw)
		found[e.Name()] = File{
			Name:   e.Name(),
			Bytes:  int64(len(raw)),
			SHA256: hex.EncodeToString(sum[:]),
		}
	}
	return found, nil
}

// rewrite records what is on disk, carrying each file's existing Pack across so that re-running
// this after a sync does not wipe the attributions that are the point of the manifest. A file
// that is new to the manifest has no pack and is reported as such — it is the one thing this
// tool cannot work out for itself.
func rewrite(man Manifest, onDisk map[string]File) error {
	packOf := make(map[string]string, len(man.Files))
	for _, f := range man.Files {
		packOf[f.Name] = f.Pack
	}

	names := make([]string, 0, len(onDisk))
	for name := range onDisk {
		names = append(names, name)
	}
	sort.Strings(names)

	man.Files = nil
	var unattributed []string
	for _, name := range names {
		f := onDisk[name]
		f.Pack = packOf[name]
		if f.Pack == "" {
			unattributed = append(unattributed, name)
		}
		man.Files = append(man.Files, f)
	}
	if man.Files == nil {
		// An empty list rather than null, so the committed file reads the same whether or not
		// anything has been filed yet.
		man.Files = []File{}
	}
	if man.Packs == nil {
		man.Packs = []Pack{}
	}

	raw, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, manifestName), append(raw, '\n'), 0o644); err != nil {
		return err
	}

	fmt.Printf("recorded %d file(s) in %s\n", len(man.Files), filepath.Join(dir, manifestName))
	if len(unattributed) > 0 {
		fmt.Printf("\n%d file(s) belong to no pack. Set their Pack field and describe the pack in Packs:\n", len(unattributed))
		for _, name := range unattributed {
			fmt.Printf("  %s\n", name)
		}
		fmt.Println("\nA release refuses to build until they are attributed.")
	}
	return nil
}

// report prints the state of the directory and returns one line per problem. The same reading
// serves the bare run and -require, so what a developer is shown and what a release refuses
// cannot drift apart.
func report(man Manifest, onDisk map[string]File) []string {
	packs := make(map[string]bool, len(man.Packs))
	for _, p := range man.Packs {
		packs[p.Name] = true
	}

	fmt.Printf("%s — bundle %s, %d file(s) manifested, %d present\n", dir, man.Bundle, len(man.Files), len(onDisk))

	var problems []string
	seen := make(map[string]bool, len(man.Files))
	for _, want := range man.Files {
		seen[want.Name] = true
		got, present := onDisk[want.Name]
		switch {
		case !present:
			problems = append(problems, fmt.Sprintf("%s: manifested but not present — sync bundle %s", want.Name, man.Bundle))
		case got.SHA256 != want.SHA256:
			problems = append(problems, fmt.Sprintf("%s: present, but its bytes are not the manifested ones", want.Name))
		case want.Pack == "":
			problems = append(problems, fmt.Sprintf("%s: belongs to no pack — nothing records where it came from", want.Name))
		case !packs[want.Pack]:
			problems = append(problems, fmt.Sprintf("%s: names pack %q, which Packs does not describe", want.Name, want.Pack))
		}
	}
	for name := range onDisk {
		if !seen[name] {
			problems = append(problems, fmt.Sprintf("%s: present but unmanifested — run -write, then record its pack", name))
		}
	}

	if len(problems) == 0 {
		// An empty manifest reaches here too, and says so honestly rather than reporting a clean
		// bill of health for a directory holding nothing.
		if len(man.Files) == 0 {
			fmt.Println("no sounds are manifested; the game builds and plays silent")
		} else {
			fmt.Println("every manifested file is present, matching and attributed")
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
