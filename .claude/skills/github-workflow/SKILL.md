---
name: github-workflow
description: Git and GitHub procedure for this repo - branching, committing, opening PRs, merging, and cleaning up afterwards. Load this before any git or gh work: branching, committing, pushing, opening or merging a PR, deleting branches, or when a merge is refused. Covers the squash-merge consequences, the review ruleset on main, and the branch numbering scheme.
---

# Git and GitHub in this repo

Read this **before** running `git` or `gh`, not after something is refused. Most of what
follows exists because a specific thing went wrong once.

## The owner reviews diffs in VS Code

**Leave work unstaged.** No `git add`, no `git commit`, no `git push` unless asked for that
specific step. Staging hides changes from where the owner reads them.

- Asked to commit? Commit, then **stop** and show the subject and full body. Do not
  continue into a push.
- Before anything reaches a remote — push, PR, release, workflow trigger — **stop and wait**,
  unless told explicitly to proceed ("open the PR", "send it in without my say-so"). An
  explicit instruction is the authorization; a standing default is not a veto over it.
- Deciding on your own that work is ready to be recorded is not wanted. Writing the message
  when asked is.

## Branching

**Work on `game-updates-N`.** One numbered branch, not a branch per feature. This is a
one-author project, so branch names buy no coordination, and **every PR is squash merged, so
the branch name never reaches `main`'s history at all** — the PR title becomes the commit.

- **Increment the number every PR. Never reuse a name.** A reused branch diverges from its
  own remote after the squash (one ahead, one behind, identical content) and the next push
  is refused until it is force-pushed.
- A PR may cover several unrelated things. What is not acceptable is a branch open across
  many sessions. **Land it while you can still describe it honestly** — if the PR
  description will not fit one clear paragraph without becoming a list of unrelated
  headings, the branch has been open too long.
- **Say so out loud before switching branches.** Never as a side effect of another task.
- `git checkout main` **before** `git pull`. Pulling on a feature branch drags `main`'s
  history onto it.

## Committing

- `git commit -s` — sign-off is required by `CONTRIBUTING.md`.
- Commits use the **GitHub noreply identity**, not a personal address. Already configured;
  check with `git log -1 --format='%an <%ae>'` if a commit looks wrong.
- Interactive flags do not work in this environment: no `git rebase -i`, no `git add -i`.
- Prefer a new commit over amending.
- Never `--no-verify` or `--no-gpg-sign` unless asked. A failing hook is a thing to fix.

## Opening a PR

Use `gh`. The **PR description is the artefact that lasts** — the branch name is discarded
by the squash and the title becomes the commit subject, so the body is the only place the
reasoning survives.

Write it the way the commit messages in this repo read: what changed, and *why*, including
the thing that forced the design. Flag anything left open.

## Merging — read this before running `gh pr merge`

**`main` has a ruleset requiring a pull request review.** `gh pr merge` will fail with:

```
X Pull request #NN is not mergeable: the base branch policy prohibits the merge.
```

- `gh pr view NN --json reviewDecision` reports `REVIEW_REQUIRED`.
- `gh api repos/CuriousJC/ascend-duel/branches/main/protection` returns **404 "Branch not
  protected"** — misleading. It is a *ruleset*, not classic branch protection, and the
  protection API does not see rulesets. Do not conclude the branch is unprotected.

**Do not pass `--admin`.** It bypasses the rule outright. Merging when asked is fine;
forcing past the one control that stops a change landing unlooked-at is the owner's call,
not yours. `--auto` is equally useless here — it queues a merge that will never fire,
because no review is coming.

**You cannot approve it yourself.** `gh` authenticates as the repo owner, who is also the PR
author, and GitHub refuses self-approval.

So: open the PR, report that it needs one approval, and stop. Even when told to merge
without asking — that instruction authorizes *merging*, not *bypassing the review rule*.
Say which one is blocking and let the owner click.

## After a merge — the squash leaves a mess, and this is the cleanup

**`git branch -d` will always refuse a merged feature branch.** The squash creates a new
commit, so the branch tip is never an ancestor of `main`. This is expected, not a warning to
investigate.

```powershell
# 1. confirm the content actually landed - an EMPTY diff is the check,
#    not whether -d succeeds
git diff main game-updates-2 --stat

# 2. move to the next branch off the new main
git checkout main; git pull; git checkout -b game-updates-3

# 3. clean up both copies of the old one
git branch -D game-updates-2
git push origin --delete game-updates-2
```

`git diff main <branch>` returning nothing is the proof. Only then `-D`.

## Gotchas that have cost time

- **`.gitignore` patterns without a leading slash match at any depth.** `trace/` silently
  ignored `internal/trace/` — the *source package* — as well as the intended output
  directory. Anchor with `/trace/`. Verify with `git check-ignore -v <path>`, both for the
  path that should be ignored and one that should not.
- **`git status --short` not listing an expected `??` entry means it is being ignored**, not
  that it does not exist. That is how the above was caught.
- **Build tags select different files.** `internal/trace` has `trace_on.go` (`debugtrace`)
  and `trace_off.go` (`!debugtrace`), so one configuration can compile while the other does
  not. Vet and build **both** before committing anything that touches traced code:

  ```powershell
  go vet ./...; go vet -tags debugtrace ./...
  go build ./...; go build -tags debugtrace ./...
  ```

## Before committing anything

```powershell
gofmt -l .        # must print nothing
go vet ./...
go build ./...
go test ./...
```

Report failures with the output. Do not describe work as done if a step was skipped.
