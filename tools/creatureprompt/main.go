// Command creatureprompt assembles the brief for one creature's picture and prints it.
//
//	go run ./tools/creatureprompt -record goblins-outer-bomber -element ice
//	go run ./tools/creatureprompt -motif goblins            # every record, every element it takes
//	go run ./tools/creatureprompt -gaps                     # what is still unwritten, roster-wide
//
// **A picture is briefed from four layers and only one of them is in the prompt file.** The style
// and composition block is `docs/art/creature_art_prompt.MD` and is true of every creature; the
// other three are authored in the motif file — what the motif shares, what the element does to
// *this* motif, and what this particular record is. Assembling them by hand for every record at
// every element is where they would quietly stop agreeing, so one command does it.
//
// **It prints and never writes a file.** What comes back from a generator is installed with the
// `art-batch` procedure; this end of it is a thing to paste.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
)

// promptFile is the style and composition layer, named rather than inlined: it is one file every
// creature shares and a copy here would be a second one to keep in step.
const promptFile = "docs/art/creature_art_prompt.MD"

func main() {
	record := flag.String("record", "", "one record key, e.g. goblins-outer-bomber")
	element := flag.String("element", "", "which element to brief it as; empty means all it can take")
	motif := flag.String("motif", "", "every record of one motif")
	gaps := flag.Bool("gaps", false, "list what is still unwritten instead of printing briefs")
	flag.Parse()

	motifs := data.LoadMotifs()

	if *gaps {
		reportGaps(motifs)
		return
	}

	switch {
	case *record != "":
		m, rec, ok := find(motifs, *record)
		if !ok {
			fmt.Fprintf(os.Stderr, "no record %q in any motif file\n", *record)
			os.Exit(1)
		}
		printRecord(m, rec, *element)
	case *motif != "":
		m, ok := motifs[*motif]
		if !ok {
			fmt.Fprintf(os.Stderr, "no motif %q\n", *motif)
			os.Exit(1)
		}
		for _, rec := range m.Records {
			printRecord(m, rec, *element)
		}
	default:
		fmt.Fprintln(os.Stderr, "name a -record or a -motif, or pass -gaps")
		flag.Usage()
		os.Exit(2)
	}
}

// find is the record with this key and the motif it belongs to.
func find(motifs map[string]data.MotifData, key string) (data.MotifData, data.MotifRecord, bool) {
	for _, name := range data.MotifOrder(motifs) {
		for _, r := range motifs[name].Records {
			if r.Record == key {
				return motifs[name], r, true
			}
		}
	}
	return data.MotifData{}, data.MotifRecord{}, false
}

// printRecord writes one brief per element the record can be dealt as, or one for the element
// asked for.
//
// **An element the record cannot take is refused rather than briefed**, because a picture of a
// creature the game will never deal is a picture nothing can use.
func printRecord(m data.MotifData, r data.MotifRecord, only string) {
	elements := r.Affinities
	if only != "" {
		if !r.HasAffinity(only) {
			fmt.Fprintf(os.Stderr, "%s is never dealt as %s — it takes %s\n",
				r.Record, only, strings.Join(r.Affinities, ", "))
			os.Exit(1)
		}
		elements = []string{only}
	}

	for _, e := range elements {
		fmt.Printf("=== %s — %s\n", r.ArtKey(e), r.FullName())
		fmt.Printf("Paste %s first, then this.\n\n", promptFile)

		parts := m.Brief(r, e)
		if len(parts) == 0 {
			fmt.Printf("(nothing written yet: %s has no Draw, no ElementDraw for %s, and %s has no Draw)\n\n",
				m.Motif, e, r.Record)
			continue
		}
		fmt.Println(strings.Join(parts, "\n\n"))
		fmt.Printf("\nName it %s.png.\n\n", r.ArtKey(e))
	}
}

// reportGaps is which of the three authored layers is still unwritten, motif by motif.
//
// **It counts rather than refusing.** An unwritten brief is work nobody has done, not a broken
// file — the loader lets it through and this is where it is seen.
func reportGaps(motifs map[string]data.MotifData) {
	var motifsShort, elementsShort, recordsShort int

	for _, name := range data.MotifOrder(motifs) {
		m := motifs[name]
		var lines []string

		if m.Draw == "" || m.Draw == data.DrawUnwritten {
			lines = append(lines, "  the motif itself has no art direction")
			motifsShort++
		}

		var missing []string
		for _, e := range data.AffinityElements {
			if v := m.ElementDraw[e]; v == "" || v == data.DrawUnwritten {
				missing = append(missing, e)
				elementsShort++
			}
		}
		if len(missing) > 0 {
			lines = append(lines, "  no element direction for "+strings.Join(missing, ", "))
		}

		var bare []string
		for _, r := range m.Records {
			if r.Draw == "" || r.Draw == data.DrawUnwritten {
				bare = append(bare, r.Record)
				recordsShort++
			}
		}
		sort.Strings(bare)
		if len(bare) > 0 {
			lines = append(lines, fmt.Sprintf("  %d records with no brief: %s",
				len(bare), strings.Join(bare, ", ")))
		}

		if len(lines) == 0 {
			fmt.Printf("%-14s complete\n", m.Motif)
			continue
		}
		fmt.Printf("%-14s\n%s\n", m.Motif, strings.Join(lines, "\n"))
	}

	fmt.Printf("\n%d motifs without direction, %d motif-elements without direction, %d records without a brief\n",
		motifsShort, elementsShort, recordsShort)
}
