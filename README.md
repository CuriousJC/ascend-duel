# Ascending Duel

Ascending Duel is a roguelike game where you must duel up a tower and build your skills while collecting magical rings, brands of power, and pets to aid you.

It is an early prototype. The duel itself plays — a hand of cards, an action-point
budget, an order you choose, and four opponents that each punish a different habit — but
the tower around it does not exist yet.

## Play it

Grab a build from [Releases](https://github.com/CuriousJC/ascend-duel/releases).

- **Windows** — download the `.exe` and run it. Everything is embedded in the binary;
  there is nothing to install and nothing to unpack. Windows will warn you about an
  unsigned executable, because it is one.
- **Linux** — download the `.tar.gz` and extract it. It ships as a tar because GitHub
  release assets carry no file permissions, so a bare binary would arrive without its
  execute bit.

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

## What it is built from

- **[Ebitengine v2](https://ebitengine.org/)** for the window, input and drawing. No UI
  toolkit — the widgets are hand-rolled.
- **Generated art and audio.** The glyphs on the cards are drawn by code at run time, and
  the score is a Standard MIDI file synthesised to audio at startup rather than a shipped
  recording. Both choices are about provenance as much as size: generated output has no
  licence question attached to it.

## Where things are written down

| File | What it holds |
|---|---|
| [MECHANICS.md](MECHANICS.md) | what the game *is* — elements, cards, hands, rings, the tower |
| [TODO.md](TODO.md) | what to build next — open work only |
| [ideas.md](ideas.md) | the unfiltered inbox |
| [CLAUDE.md](CLAUDE.md) | how the code is organised and the conventions it follows |
| [CONTRIBUTING.md](CONTRIBUTING.md) | the contributor licence grant |

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

- **Enemy sprites** — [PVGames](https://pvgames.itch.io/), from the Humble *Isometric
  Assets Galore* bundle. Licensed for use inside a game; they may not be redistributed
  as assets, so please do not lift them out of this repository for your own project.
- **Fonts** — `Kubasta.ttf` (CC0, via FontStruct), Fira Sans, and Roboto Flex.

Everything else under [assets/](assets/) is first-party or generated at run time.

## References

Examples:
https://github.com/hajimehoshi/ebiten/blob/main/examples/fullscreen/main.go
