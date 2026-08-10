# caveman — where it came from, and what was left behind

`SKILL.md` is a **verbatim copy** of `skills/caveman/SKILL.md` from
[JuliusBrussee/caveman](https://github.com/JuliusBrussee/caveman), fetched 2026-08-10.
Upstream is **MIT**, which is permissive and therefore compatible with selling this game —
see the licensing section of `CLAUDE.md`. Do not edit `SKILL.md` in place without noting the
divergence here; the point of a verbatim copy is that it can be re-fetched and diffed.

## Why the file and not the plugin

Upstream's install is `claude plugin install caveman@caveman`, which is **global to every
repo** and registers two Node hooks — `SessionStart` and `UserPromptSubmit` — that run on
every session start and every prompt you submit, in every project. This is a trial in one
repo, so it is installed as one Markdown file with no executable parts and nothing outside
`c:\repos\ascend-duel`.

What that costs, concretely:

- **No auto-activation.** The hook is what turns the mode on at session start and remembers
  the level across turns. Without it, the skill loads when it is invoked or when the request
  matches its description ("be brief", "less tokens"), and a `/clear` forgets it.
- **`/caveman <level>` is not wired up.** Level switching is a slash command in
  `commands/caveman.md` upstream, backed by the mode-tracker hook. Ask for a level in words
  instead.
- The sibling skills — `caveman-compress`, `caveman-stats`, `caveman-review`,
  `cavecrew` — are not here. `caveman-compress` ships Python that rewrites files; it was
  left out deliberately.

## What it does not apply to

The skill's own Boundaries section says anything persisted outside the chat stays normal
prose — code, comments, commit messages, PR bodies, docs, memory files. That matters here
more than usual: `MECHANICS.md`, `TODO.md` and `CLAUDE.md` are the design record, and their
value is the reasoning written out longhand. **Compression applies to replies in the
terminal, nothing else.**

## The honest arithmetic

Upstream measures **65% fewer output tokens on chat-style prompts, but 8.5% on real agentic
runs**, and the skill file itself adds roughly 1.5k input tokens to a turn once loaded
(`SKILL.md` is 6.2 KB). Sessions in this repo are input-heavy — `CLAUDE.md` alone is around
8k tokens every session, before any file is read — so the saving is taken from the smaller
half of the bill. Upstream's own `docs/HONEST-NUMBERS.md` says net savings can go negative on
already-terse workloads.

Judge it on whether the replies are still worth reading, not on the 65%.

## Removing it

Delete this directory. Nothing else references it.
