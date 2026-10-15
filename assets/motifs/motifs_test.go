package motifs

import (
	"testing"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
)

// TestImportingItFilesTheTree holds the one thing this package is for: imported, the motif
// pictures are in assets.LoadImageData, and nothing that is not a picture is.
func TestImportingItFilesTheTree(t *testing.T) {
	images := assets.LoadImageData()
	if _, ok := images[data.DefaultBackgroundArt]; !ok {
		t.Fatalf("%s is not in LoadImageData, so the tree was never filed", data.DefaultBackgroundArt)
	}
	for _, stray := range []string{"doc.go", "embed.go", "doc", "embed"} {
		if _, ok := images[stray]; ok {
			t.Errorf("%q is filed as a picture", stray)
		}
	}
}

func TestEveryBackdropDrawsSomething(t *testing.T) {
	// The backdrop half of the relic catalog's TestEveryRelicDrawsSomething, plus the default
	// itself: a realm whose element has no backdrop draws the default, so a default that is not
	// embedded is a duel on the bare gradient with nothing failing.
	images := assets.LoadImageData()
	if _, ok := images[data.DefaultBackgroundArt]; !ok {
		t.Errorf("the default backdrop %q is not an embedded image", data.DefaultBackgroundArt)
	}
	// An authored room with no picture yet is the normal state, so an absent one is not a
	// failure. What is: a picture filed under a room's family for an element the room is never
	// drawn in — art that ships and that nothing can ever show.
	motifs := data.LoadMotifs()
	for _, name := range data.MotifOrder(motifs) {
		for _, b := range motifs[name].Backdrops {
			for _, e := range data.AffinityElements {
				key := b.Art + "-" + e
				if _, ok := images[key]; ok && !b.HasAffinity(e) {
					t.Errorf("%s is embedded, and %s is never drawn in %s", key, b.Backdrop, e)
				}
			}
		}
	}
}

func TestEveryOpponentHasSomethingToDraw(t *testing.T) {
	// **Nearly every record's own picture is still to be generated**, so what this holds is the
	// fallback rather than the pictures: a record whose art key names no file has to land on the
	// placeholder, because a card with a hole in it reads as a bug where a placeholder reads as
	// art nobody has made yet.
	//
	// **The key is the filename stem** — see this package's doc — so renaming a file is what this
	// catches once the pictures exist.
	images := assets.LoadImageData()
	if _, ok := images[data.DefaultEnemyArt]; !ok {
		t.Fatalf("%s is not embedded, so a record with no picture draws nothing at all",
			data.DefaultEnemyArt)
	}

	motifs := data.LoadMotifs()
	if len(motifs) == 0 {
		t.Fatal("no motifs loaded — this test is checking nothing")
	}
	for _, key := range data.MotifOrder(motifs) {
		for _, r := range motifs[key].Records {
			for _, e := range r.Affinities {
				if _, ok := images[r.ArtKey(e)]; ok {
					continue
				}
				// Not an error: it is the state the whole roster is in. The check that matters is
				// that the key is well formed and the placeholder is there to take it.
				if r.ArtKey(e) == "" {
					t.Errorf("%s dealt as %s names no picture at all", r.Record, e)
				}
			}
		}
	}
}

func TestNoTwoRecordsDrawTheSamePicture(t *testing.T) {
	// Every picture lands in one flat map of images, so two records claiming one key is one
	// lookup with two answers — and the wrong creature drawn, silently and correctly as far as
	// every lookup is concerned.
	motifs := data.LoadMotifs()
	taken := map[string]string{}
	for _, key := range data.MotifOrder(motifs) {
		for _, r := range motifs[key].Records {
			for _, e := range r.Affinities {
				art := r.ArtKey(e)
				if other, clash := taken[art]; clash {
					t.Errorf("%s and %s both draw %q", r.Record, other, art)
				}
				taken[art] = r.Record
			}
		}
	}
}
