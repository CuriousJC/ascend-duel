package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// root is the repository root, from this package's directory — where `go test` runs it.
const root = "../.."

func archivedRelics(t *testing.T) (map[string]data.RelicData, []string) {
	t.Helper()
	path := filepath.Join(root, data.ArchivedRelicsFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the relic archive: %v", err)
	}
	return data.ParseRelics(raw, path)
}

// pictureStems is every PNG in a directory, by filename stem — the key a record's Art writes.
func pictureStems(t *testing.T, dir string) map[string]bool {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	out := map[string]bool{}
	for _, e := range ents {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".png") {
			out[strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))] = true
		}
	}
	return out
}

// TestEveryArchivedRelicWouldLoad holds the archive to the grammar the live catalog is under. An
// archived relic is kept so it can be moved back, and one whose words the rules have stopped
// knowing is a relic that would refuse to load the day it is wanted. **The fix is to update the
// archived record to the current vocabulary**, not to loosen this.
func TestEveryArchivedRelicWouldLoad(t *testing.T) {
	records, order := archivedRelics(t)
	for _, key := range order {
		if err := CheckRelicRecord(records[key]); err != nil {
			t.Errorf("archived relic would not load: %v", err)
		}
	}
}

// TestNoRelicIsBothLiveAndArchived: a key in both files is two records claiming one identity, and
// moving either one back would collide with the other.
func TestNoRelicIsBothLiveAndArchived(t *testing.T) {
	records, _ := archivedRelics(t)
	for key := range records {
		if _, live := registeredRelics[key]; live {
			t.Errorf("%s is in both %s and data/relics.json", key, data.ArchivedRelicsFile)
		}
	}
}

// TestArchivedRelicArtIsInTheArchive holds the pictures to the records. An archived record's Art has
// to name a file in the archive's own art directory — one left in `assets/relic/` still ships in
// the binary — and a picture in the archive directory has to belong to an archived record, or it is
// a file nothing will ever draw.
func TestArchivedRelicArtIsInTheArchive(t *testing.T) {
	records, order := archivedRelics(t)
	archived := pictureStems(t, filepath.Join(root, data.ArchivedRelicArtDir))
	live := pictureStems(t, filepath.Join(root, "assets", "relic"))

	named := map[string]bool{}
	for _, key := range order {
		art := records[key].Art
		if art == "" {
			continue
		}
		named[art] = true
		if !archived[art] {
			t.Errorf("%s draws %q, which is not in %s", key, art, data.ArchivedRelicArtDir)
		}
		if live[art] {
			t.Errorf("%s draws %q, which is still in assets/relic and ships in the game", key, art)
		}
	}
	for stem := range archived {
		if !named[stem] {
			t.Errorf("%s/%s.png belongs to no archived relic", data.ArchivedRelicArtDir, stem)
		}
	}
}
