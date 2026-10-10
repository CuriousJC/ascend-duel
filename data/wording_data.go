package data

// The wording: **the sentences the game says, written as templates rather than assembled in code.**
//
// A section per screen or panel, and inside it a key per sentence. A template is plain text with
// holes: `{name}` is a value the code fills — a figure, a name, a card — `{ink:word}` is a word in
// a named color, and `{mark:word}` is a word bold and underlined. See internal/ui/say.go for the
// grammar and internal/ui/wording.go for which holes each sentence is filled with.
//
// **The rules do not read this file**, and neither does anything below the drawing layer: a
// sentence is presentation over figures something else already decided. `internal/ui` loads it once
// and refuses, at launch, a key it asks for that the file lacks and a sentence whose holes are not
// the ones its code fills.

import (
	_ "embed"
)

//go:embed wording.json
var wordingJSON []byte

// LoadWording parses the file as section → key → template.
func LoadWording() map[string]map[string]string {
	return parseOne[map[string]map[string]string](wordingJSON, "wording.json")
}
