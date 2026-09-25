package main

// The archive page: the relics taken out of the game, drawn the way the catalog is.
//
// **Same cards, same plates, same family grouping**, so an archived relic reads exactly as it did
// on the live sheet and one moved back lands looking the way it looked here. What it leaves out is
// the shelf — the tier shares, the price, the sell-back and the accumulator badge — because every
// one of those is a fact about a relic the game has registered, and nothing here is.
//
// **Read off disk rather than out of the embed**, because the archive is not in the binary: see
// data/archive.go. The pictures come from `assets/archive/relic/`, and a record with no Art of its
// own draws the embedded default face as it would in the game.
//
// **A record that would not load is drawn anyway**, with the reason in pink on its plate. The
// catalog's sheet refuses to start on a bad relic because the game would; the archive's job is to
// show what is there, and `TestEveryArchivedRelicWouldLoad` is the gate that fails.

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/tools/sheetfilter"
)

func runArchive(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}

	faces, err := cards.NewFaces(assets.LoadFontData()["kubasta"])
	if err != nil {
		return err
	}

	raw, err := os.ReadFile(data.ArchivedRelicsFile)
	if err != nil {
		return fmt.Errorf("reading %s — run from the repository root: %w", data.ArchivedRelicsFile, err)
	}
	records, order := data.ParseRelics(raw, data.ArchivedRelicsFile)

	page := page{
		Title:   "Relic archive",
		Archive: true,
		Ground:  ground,
		Style:   styleFacts(cards.RelicStyle),
		Count:   len(records),
	}

	var plates []plate
	for _, key := range order {
		record := records[key]

		art, err := archivedArtwork(record)
		if err != nil {
			return err
		}
		spec := cards.Spec{
			Name:    record.Name,
			Element: cards.Relic,
			Rarity:  record.Rarity,
			Art:     art,
			Enabled: true,
		}
		cell, err := write(dir, faces, spec, cards.RelicStyle, "relic-"+key+".png", record.Name)
		if err != nil {
			return err
		}

		p := plate{
			Cell:    cell,
			Record:  key,
			Name:    record.Name,
			Text:    record.Text,
			Rarity:  string(record.Rarity),
			Family:  record.Family,
			Art:     record.Art,
			Draw:    record.Draw,
			Default: record.Art == "",
			Rules:   ruleLines(record),
		}
		if err := session.CheckRelicRecord(record); err != nil {
			p.Problem = err.Error()
			page.Broken++
		}
		plates = append(plates, p)
		if record.Art == "" {
			page.Undrawn++
		}
		if record.Draw == "" {
			page.Unwritten++
		}
	}

	if err := prune(dir, plates); err != nil {
		return err
	}
	page.Families = groupByFamily(plates)
	page.Filters = sheetfilter.Bar(relicFacets(groupByRarity(plates), page.Families))

	out := filepath.Join(dir, "index.html")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, page); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	fmt.Printf("wrote %s and %d PNGs — %d archived relics, %d of which would not load\n",
		out, len(plates), page.Count, page.Broken)
	return nil
}

// archivedArtwork is an archived record's picture, out of the archive's own art directory — or the
// embedded default face, for a record that never had one.
func archivedArtwork(r data.RelicData) (image.Image, error) {
	if r.Art == "" {
		return artwork(data.DefaultRelicArt)
	}
	path := filepath.Join(filepath.FromSlash(data.ArchivedRelicArtDir), r.Art+".png")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s draws %q: %w", r.RelicRecord, r.Art, err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", path, err)
	}
	return img, nil
}

// prune deletes every relic picture in dir that this run did not write.
//
// **A sheet's directory is the catalog it draws and nothing else.** A relic moved into the archive,
// or back out of it, leaves its PNG behind on the page it left — linked by nothing, and committed
// all the same. Only `relic-*.png` is touched, so the state row and the page itself are left alone.
func prune(dir string, plates []plate) error {
	wrote := make(map[string]bool, len(plates))
	for _, p := range plates {
		wrote[p.Cell.File] = true
	}
	stale, err := filepath.Glob(filepath.Join(dir, "relic-*.png"))
	if err != nil {
		return err
	}
	for _, path := range stale {
		if wrote[filepath.Base(path)] {
			continue
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing %s: %w", path, err)
		}
		fmt.Printf("removed %s, which no relic here draws\n", path)
	}
	return nil
}
