---
name: secondary
description: How to work as the secondary copy of this repo - the second clone, on the long-lived secondary-work branch, working beside the primary copy. Load the moment the owner says you are the secondary, and before refreshing from main, checking for conflicts, or opening a PR from secondary-work. Covers the branch, the resync after a squash, the conflict check before a merge, and the parts of github-workflow that do not apply here.
---

# The secondary copy

**The owner says so at the start of a session.** Two clones of the one GitHub repo are worked at
once. The **primary** has the owner's main attention; the **secondary** takes work that does not
need it — so far mostly pipeline and art, though nothing restricts it to those. Same `origin`,
separate working trees, separate branches, and neither copy can see the other's uncommitted work.
Everything below follows from that last fact.

**A self-check, not the trigger:** the secondary's working tree is
`C:\repos\ascend-duel-pipeline\ascend-duel`. If the owner says "secondary" and the path is
something else, or the path is that one and nobody said, say so before doing anything.

The `github-workflow` skill still applies — load it too before any `git` or `gh` command. This
skill is the list of places it bends.

## No file is off limits — overlap is what to watch

The split is by the owner's attention, not by package, so any file may be edited here. What
changes is that **the primary may be editing the same file at the same time**, unseen. So:

- **Say early when a task reaches into what the primary is likely working on** — the engine,
  rules, screens it has been building — so the owner can tell you whether it is live over there.
- **The shared documents conflict worst**: `CLAUDE.md`, `MECHANICS.md`, `TODO.md` and the
  skills are edited from both copies, in long paragraphs, and a conflict in one is a prose merge
  rather than a mechanical one. Name every edit to them in the end-of-work summary.
- **Refresh from main often** (below). The longer the branch is open, the more the two copies
  have drifted without either seeing it.

## The branch

**`secondary-work` is long-lived and reused across PRs** — the one deliberate exception to
`github-workflow`'s "never reuse a name". It can be, because main is merged *into* it after every
squash, which puts the reused name back in step with its own remote. Its name is what says it is
off to the side, and `github-workflow`'s sweep leaves it alone for that reason.

- **No `up-N` here.** Those numbers belong to the primary.
- **No branch sweep from this copy.** Cleanup is the primary's; an `up-N` branch seen from here may
  be live work in the other clone.
- **If `secondary-work` has gone** — deleted locally or on the remote — recreate it off
  `origin/main` rather than working on anything else.

## Refreshing from main

Run this at the start of a session and again before opening a PR. Say it is happening; it is a
merge onto the working branch.

```powershell
git status --short                              # must be clean - stop and ask if not
git fetch origin --prune
git log --oneline HEAD..origin/main             # what landed since the last refresh
git merge-tree --write-tree --name-only origin/main HEAD   # exit 0 = clean, 1 = conflicts listed
git merge origin/main                           # only once the preview is understood
```

**After one of our own PRs squashes, this is the resync.** The branch reads as N ahead and one
behind with identical content; merging `origin/main` folds the squash in cleanly, because both
sides made the same change. Prove it first: `git diff origin/main HEAD --stat` is **empty**
before the merge. If it is not, something on this branch did not land — stop and say what.

**Merge, never rebase.** The branch is on the remote and a PR has been opened from it; rewriting
it forces a push for nothing.

## The conflict check, before a PR

A clean `merge-tree` is the textual answer. The other two:

**Overlap — which files both copies touched since they split:**

```bash
base=$(git merge-base HEAD origin/main)
comm -12 <(git diff --name-only $base origin/main | sort) <(git diff --name-only $base HEAD | sort)
```

Any file in that list gets read on both sides, even when it merged clean — two edits to one JSON
record can merge into valid JSON that says something neither copy meant.

**Semantics — what merged clean and still broke:** the primary renames a field, moves a symbol or
retunes a record this branch leans on. Nothing textual catches that, so after the merge:

```powershell
gofmt -l .
go vet ./...; go vet -tags debugtrace ./...; go vet -tags idleexit ./...
go vet -tags demoplay ./...; go vet -tags scenario ./...
go build ./...
go test ./...
```

**`docs/sheets/` is never resolved by hand.** A conflicting PNG or sheet page means both copies
regenerated it; take either side and **regenerate the sheet on the merged tree** with the one tool
that draws it. That is the only version that is a picture of what will ship.

**A motif file conflicting is the case to read closely.** One copy retunes stats and decks while
the other writes `Draw`, `Art` and backdrops, in the same record. The resolution is almost always
both edits kept, field by field — and then `go test ./...`, since the loader validates coverage on
launch.

## Reporting

End every piece of work with which shared files were touched (the four above, plus any file in the
overlap list), whether the last refresh was clean, and how far `origin/main` has moved since it.
The owner merges the two streams; this is what lets them do it without re-deriving it.
