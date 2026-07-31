# Contributing to Ascending Duel

Thanks for your interest. Please read this before opening a pull request — the
licensing here is deliberately different from a typical open source project, and
it affects what you are agreeing to when you contribute.

## This project is source-available, not open source

Ascending Duel is licensed under the
[PolyForm Noncommercial License 1.0.0](LICENSE). Anyone may read, build, modify and
share the code for noncommercial purposes. **The right to sell the game, or anything
derived from it, is reserved to the copyright holders.**

That reservation is the whole point of the arrangement, and it only works if the
copyright holders actually hold the rights to everything in the repository.

## The contributor grant

By opening a pull request against this repository, you agree that:

1. **You wrote it, or you have the right to submit it.** The contribution is your
   original work, or you have permission from whoever owns it to submit it here
   under these terms. It is not copied from a source whose license forbids this.

2. **You grant the copyright holders a broad license to your contribution.**
   Specifically, you grant Justin Crosby and KingSherman1820 a perpetual,
   worldwide, non-exclusive, royalty-free, irrevocable license to use, reproduce,
   modify, adapt, publish, distribute, sublicense, and **commercially exploit** your
   contribution as part of Ascending Duel or any derivative of it, and to relicense
   it under different terms — including selling the game.

3. **You keep your copyright.** This is a license, not an assignment. You still own
   your contribution and may use it elsewhere however you like.

4. **You are not owed payment.** The grant is royalty-free. Contributing does not
   create an ownership stake in the project or entitle you to a share of revenue.

Without point 2, a merged contribution would leave the project unable to be sold,
because the contributor would retain rights the copyright holders do not have. That
is why this document exists.

## How to signal agreement

Add a `Signed-off-by` line to each commit, which you get automatically with:

```
git commit -s
```

This follows the [Developer Certificate of Origin](https://developercertificate.org/)
convention, and in this repository it also indicates agreement with the grant above.

If you would rather not agree to these terms, please open an issue describing the
change instead of a pull request. Bug reports and design discussion are welcome with
no grant attached.

## Practical notes

Before opening a PR:

```powershell
go build ./...
go vet ./...
go test ./...
gofmt -l .
```

`CLAUDE.md` describes the architecture and the conventions worth following —
particularly the model/system split, the `GlobalState` threading, and the rule that
`internal/combat` must never import Ebitengine.

## Questions

There is no contact address yet. Open an issue.

---

*This document sets out the terms under which contributions are accepted. It has not
been reviewed by a lawyer.*
