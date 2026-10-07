---
name: secondary
description: How to work as the secondary copy of this repo - the second clone, on the long-lived secondary-work branch, working beside the primary copy. Load the moment the owner says you are the secondary, and before refreshing from main, checking for conflicts, or opening a PR from secondary-work. Covers the branch, the start-of-session refresh that takes main wholesale, the conflict check before a merge, and the parts of github-workflow that do not apply here.
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

## Refreshing from main — main wins, wholesale

**The secondary is a side job taken while the primary is busy, disconnected from it.** So a
session starts by making `secondary-work` *be* main, and then does the owner's task on top. Run it
at the start of every session without asking, and say it happened. **Every conflict resolves to
main's side**, and so does every non-conflicting hunk: the end state is a tree identical to
`origin/main`. A conflict here is not a decision to bring to the owner.

**The one thing to check first is that nothing on this branch is unlanded**, because taking main
wholesale throws it away. It is unlanded only if the working tree is dirty, or there is an open PR
from `secondary-work`, or the branch differs from the squash of its last merged PR:

```powershell
git status --short                              # dirty -> stop and say what is there
gh pr list --head secondary-work --state open   # open PR -> stop; its work has not landed
$sq = gh pr list --head secondary-work --state merged --limit 1 --json mergeCommit --jq '.[0].mergeCommit.oid'
git fetch origin --prune
git diff $sq HEAD --stat                        # must be empty; if not, stop and say what differs
```

Then take main. A plain merge with `-X theirs` is **not** enough — it keeps this side's
non-conflicting hunks, which re-adds anything main deleted since the stale merge base. Reset the
merge's tree to main's outright instead:

```powershell
git log --oneline HEAD..origin/main             # what landed since the last refresh, for the summary
git merge --no-commit -X theirs origin/main
git read-tree -u --reset origin/main            # index and working tree become exactly main
git commit --no-edit
git diff origin/main HEAD --stat                # must be empty
```

**It is a merge, not a reset**, because `secondary-work` is on the remote: a `reset --hard` to main
would leave it behind its own remote and the next push would have to be forced. The merge commit
carries main's tree and keeps the branch fast-forwardable.

Run it again before opening a PR, and there the rule changes: the branch now *has* unlanded work,
so it is an ordinary merge — preview with `git merge-tree --write-tree --name-only origin/main
HEAD`, and read every conflict on both sides.

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
