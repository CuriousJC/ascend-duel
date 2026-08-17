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

### The one exception, and how narrow it is

The owner does sometimes delegate exactly this — *"you are empowered to do everything you
need to get that out there, even forcing through an update into main"* — and that is real
authorization, not a figure of speech. `--admin` is allowed while it stands.

**It is scoped to the window they said they would be away for, and it lapses the moment they
are back.** Used once on 2026-08-06 to land #22 during an unattended overnight session, and
revoked the next morning: *"we'll go back to me being aware of any pushes to main now that
I'm back."*

- **The default is always "open the PR and stop".** Assume no standing grant.
- A grant requires the owner to be **explicit that they are unavailable**. "Go ahead and
  merge it" from someone sitting at the keyboard is a request to *merge*, not to bypass
  review — ask them to click it.
- **A grant never carries into the next session**, and never past their return. Do not reason
  from "this was allowed last time".
- While a grant is live, say which control is being bypassed and why, as it happens.

## After a merge — the squash leaves a mess, and this is the cleanup

**`git branch -d` will always refuse a merged feature branch.** The squash creates a new
commit, so the branch tip is never an ancestor of `main`. This is expected, not a warning to
investigate.

**Two branches exist at a time: `main` and one feature branch. Everything else is rubbish,
and clearing it out is a standing instruction rather than a question.** Stated by the owner
on 2026-08-17, after a cleanup turned up nine stale remote branches and two stale local ones,
every one of them a PR merged days earlier. Delete them — local and remote, in the same pass
as the new branch — and report what went, rather than listing them and waiting.

**This is the one deliberate exception to "stop before anything reaches the remote".** The
general rule is about publishing *content*; deleting a branch whose content is already on
`main` publishes nothing, and the proof below is what makes that true. Pushes, PRs and
releases still stop and wait.

```powershell
# 1. confirm the content actually landed - an EMPTY diff is the check,
#    not whether -d succeeds
git diff origin/main game-updates-33 --stat

# 2. move to the next branch off the new main
git checkout main; git pull; git checkout -b game-updates-34

# 3. sweep every branch that is not main or the new one, both copies
git branch -D game-updates-33 combat-rework game-updates-28
git push origin --delete game-updates-23 game-updates-24 game-updates-33
git fetch --prune
```

**Each branch needs its own proof before it is deleted, and there are two kinds.** Neither
takes long and skipping them is how unmerged work disappears:

- **The branch just merged**: `git diff origin/main <branch> --stat` is empty. Diff against
  `origin/main`, not `main` — a local `main` that has not been pulled yet reports a
  difference that is only its own staleness.
- **An older stale branch**: that diff is *not* empty and cannot be, because `main` has moved
  on since. Ask the other question instead — `git rev-list --count origin/main..<branch>`
  is `0`, meaning it holds no commit `main` has not got. For a squash-merged branch that
  count is non-zero, so confirm it through its PR: `gh pr list --state merged --limit 20
  --json number,headRefName,mergedAt` names the branch each merged PR came from.

**A branch matching no merged PR and holding commits `main` has not got is the one case that
stops and asks.** `combat-rework` looked like that and turned out to be `rev-list` 0 —
already contained — so it went with the rest.

## Gotchas that have cost time

- **`.gitignore` patterns without a leading slash match at any depth.** `trace/` silently
  ignored `internal/trace/` — the *source package* — as well as the intended output
  directory. Anchor with `/trace/`. Verify with `git check-ignore -v <path>`, both for the
  path that should be ignored and one that should not.
- **`git status --short` not listing an expected `??` entry means it is being ignored**, not
  that it does not exist. That is how the above was caught.
- **Build tags select different files.** `internal/trace` has `trace_on.go` (`debugtrace`)
  and `trace_off.go` (`!debugtrace`); `internal/idle` has the same shape on `idleexit`. One
  configuration can compile while another does not. Vet and build **all** of them before
  committing anything that touches either:

  ```powershell
  go vet ./...; go vet -tags debugtrace ./...; go vet -tags idleexit ./...
  go build ./...; go build -tags debugtrace ./...; go build -tags idleexit ./...
  ```

- **A stacked PR conflicts with `main` the moment the one below it is squashed.** Cost time
  on 2026-08-07. Branch B was opened against branch A; A squash-merged into `main`; B
  retargeted to `main` and immediately reported `CONFLICTING` on 22 files.

  Nothing is actually wrong. The squash is a *new* commit that is not an ancestor of B, so
  the merge base falls back to before A branched and git sees both sides editing the same
  regions. **Do not resolve those conflicts by hand** — you would be re-resolving work that
  already landed.

  Drop the duplicated commit instead, after proving the content is identical:

  ```powershell
  git fetch origin
  git diff <A's last commit> origin/main --stat     # MUST be empty
  git rebase --onto origin/main <A's last commit> <B>
  git push --force-with-lease origin <B>
  ```

  The empty diff is the proof that replaying only B's own commits loses nothing. Plain
  `git rebase --onto` is fine here — it is `-i` that this environment cannot run.

  **Retargeting is manual if the base branch still exists.** GitHub only auto-retargets a
  stacked PR when the branch under it is deleted, so `gh pr merge --delete-branch=false`
  leaves you to run `gh pr edit <B> --base main` yourself.

## Before committing anything

```powershell
gofmt -l .        # must print nothing
go vet ./...
go build ./...
go test ./...
```

Report failures with the output. Do not describe work as done if a step was skipped.
