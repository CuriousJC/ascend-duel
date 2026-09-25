package data

// The archive: records taken out of the game and kept in the repository.
//
// **An archived record is a parallel path, not a deletion.** It lives in a file under
// `data/archive/` named after the catalog it came out of, and its picture lives under
// `assets/archive/` in a directory named after the asset family it came out of — so the relic
// archive is `data/archive/relics.json` and `assets/archive/relic/`. Restoring a record is moving
// it back: the record into the live file, the picture into the live directory. Nothing is renamed
// on the way in either direction.
//
// **Nothing here is embedded**, and that is what takes a record out of the game. The live catalogs
// are `//go:embed` and ship in the binary; the archive is read off disk by the review sheet and by
// the test that holds it to the grammar, both of which run from a checkout. A release carries
// neither the records nor their pictures.
//
// **Only the relics have an archive today.** Another catalog gets one by the same shape: a file
// here beside `relics.json`, an art directory beside `assets/archive/relic/`, a pair of paths below,
// a test in the package that validates the live catalog, and a `-archive` mode on its sheet.

// ArchivedRelicsFile is the relic archive, as a path from the repository root.
const ArchivedRelicsFile = "data/archive/relics.json"

// ArchivedRelicArtDir is where an archived relic's picture lives, as a path from the repository
// root. A record's `Art` names a file here exactly as it names one under `assets/relic/`.
const ArchivedRelicArtDir = "assets/archive/relic"

// ParseRelics reads a relic file that is not the embedded catalog — the archive — into a map keyed
// by RelicRecord and the keys in the order the file writes them.
//
// **The caller reads the file**, because this package embeds and never opens one: the archive's
// two readers each know where the repository root is and this package does not. A malformed file
// or a key written twice panics with the filename, as every catalog here does.
func ParseRelics(raw []byte, file string) (map[string]RelicData, []string) {
	key := func(r RelicData) string { return r.RelicRecord }
	return keyed(raw, file, key), fileOrder(raw, file, key)
}
