# Ascending Duel

Ascending Duel is a roguelike engine-building card game for dueling up a tower.

The duelist draws eight cards and can play up to five cards within their AP budget. They can spend AP to defend first or they can attack. Cards played create a poker-like hand across the elements, attack type, or specifc attacks to get multipliers.

A first-time player is taught the game by a short scripted tutorial on their opening
fight, and a run is written to disk between rooms, so quitting and relaunching picks the
climb back up.

## Play it

Grab a build from [Releases](https://github.com/CuriousJC/ascend-duel/releases).

- **Windows** — download the `.exe` and run it. Everything is embedded in the binary;
  there is nothing to install and nothing to unpack. Windows will warn you about an
  unsigned executable, because it is one.
- **Linux** — download the `.tar.gz` and extract it. It ships as a tar because GitHub
  release assets carry no file permissions, so a bare binary would arrive without its
  execute bit.

Saves live in your platform's config directory — `%APPDATA%\ascend-duel` on Windows,
`~/.config/ascend-duel` on Linux — never beside the executable. `profile.json` is you
(tutorial seen, achievements); `run.json` is the climb in progress. Deleting either is
safe: a missing file is a new player.

Every run has a **six-character code** you can write down. The alphabet is Crockford
base32, so there is no `I`, `L`, `O` or `U` to mistype.

## Build it

Go 1.25 or newer, and that is the whole toolchain — there is no asset pipeline and no
build step beyond `go build`.

```sh
go run .            # build and launch the game window
go build .          # produce the binary
go test ./...
```

On Linux, [Ebitengine](https://ebitengine.org/) needs cgo and a few development headers
(X11, GL, ALSA); the exact package list is in
[.github/workflows/ci.yml](.github/workflows/ci.yml). On Windows it is pure Go and needs
nothing.

## What is in it

**Duelist cards** in four forms — stab, slash, crush and defend — across five elements.
**Hands** on three axes, concept, form and element, wearing poker's names.

Then everything a run picks up, each touching a different thing. A **relic** is worn, a **stone**
raises a rung, an **essence** eats a card, a **potion** changes the duelist, and a **rune** is the
only one spent during a fight, between the turns of a round. A **sealed good** is the shop's
gamble: paid for and _then_ read, four stones or four essences inside and you keep one.

Against all that, **creatures** sorted into floor bands, each with a deck of its own, and
**bosses** guarding every stairway.

Every one of those is a file in [data/](data/), and they are still being authored.

**Every catalog is browsable without launching the game.** [docs/sheets/](docs/sheets/)
holds a rendered page per catalog — the cards, relics, essences, runes, stones, hands, upgrades,
creatures and bosses, as the game actually draws them, with the rules that fire beside the text a
player reads. Open [docs/sheets/index.html](docs/sheets/index.html).

## What it is built from

- **[Ebitengine v2](https://ebitengine.org/)** for the window, input and drawing. No UI
  toolkit — the widgets are hand-rolled.
- **Generated art and a synthesized score.** The pictures are generated from prompts this
  repository owns — they live in [docs/art/](docs/art/) — and committed as ordinary PNGs. The
  score is a Standard MIDI file synthesized to audio at startup rather than a shipped recording,
  so the whole tune is a kilobyte of notes that can be read and edited in a diff.
- **No `math/rand` globals, no `time.Now()` in the rules.** A run is meant to be
  reproducible from its code, so every roll comes off an explicit stream salted from the
  run seed.

## Where things are written down

| File                               | What it holds                                                  |
| ---------------------------------- | -------------------------------------------------------------- |
| [MECHANICS.md](MECHANICS.md)       | what the game _is_ — elements, cards, hands, relics, the tower |
| [TODO.md](TODO.md)                 | what to build next — open work only                            |
| [ideas.md](ideas.md)               | the unfiltered inbox                                           |
| [CLAUDE.md](CLAUDE.md)             | how the code is organized and the conventions it follows       |
| [CONTRIBUTING.md](CONTRIBUTING.md) | the contributor license grant                                  |

## License

Source-available, not open source. Ascending Duel is licensed under the
[PolyForm Noncommercial License 1.0.0](LICENSE).

You are free to read the source, build it, modify it, and share your changes for
any **noncommercial** purpose — personal use, study, hobby projects, and by
charitable, educational, and government organizations. Selling the game or any
derivative of it is reserved to the copyright holders.

**Streaming and video are fine, including monetized.** Record it, stream it, put
ads on it — that is explicitly permitted and needs no permission from us. See
Additional Permissions at the top of [LICENSE](LICENSE).

Contributions are welcome, but require a license grant so the game remains sellable
by its authors — see [CONTRIBUTING.md](CONTRIBUTING.md).

## Credit

Programming by: CuriousJC and KingSherman1820
Art by: CuriousJC and KingSherman1820

### Third-party assets

- **Fonts** — `Kubasta.ttf` (CC0, via FontStruct), Fira Sans, and Roboto Flex.
- **Music loops** — from Humble Bundle. They are not in this repository; see
  [privateassets/README.md](privateassets/README.md). A clone without them builds and plays the
  synthesized score everywhere.

Everything else under [assets/](assets/) is first-party: generated from the prompts in
[docs/art/](docs/art/), or generated at runtime.

## References

Examples:
https://github.com/hajimehoshi/ebiten/blob/main/examples/fullscreen/main.go
