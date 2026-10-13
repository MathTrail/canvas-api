# Contributing

Thank you for looking. This is a small project with one maintainer, and it is built for children, so it is worth saying plainly what helps and what will be turned down — before you spend an evening on something.

## Before you write anything

**Open an issue first** for anything beyond a typo. The order of the work is fixed: [`RUN.md`](RUN.md) lists the tasks in the order they are done, and what the product is meant to be is settled in [`docs/product.md`](docs/product.md), [`docs/spec.md`](docs/spec.md) and [`docs/privacy.md`](docs/privacy.md). A change that fits none of them is not rejected because it is bad — it is rejected because the plan is what keeps this finishable by one person.

The documents under `docs/` and `RUN.md` are written in Russian (R25 in [`docs/decisions.md`](docs/decisions.md)); everything else — code, comments, configuration, the texts a child or a parent reads — is in English, and an issue is welcome in English.

If you are reporting a bug, an issue with what you did, what happened and what you expected is already a real contribution.

**A security fault is not an issue.** It goes through the private channel in [`SECURITY.md`](SECURITY.md).

## What will probably be declined

- **A new dependency.** Every one of them has to be read, licensed compatibly with MIT, and kept current by one person.
- **A feature the documents do not describe.** Section 7 of `docs/product.md` lists what is deliberately left out of the MVP, and section 14 what comes after it.
- **A change that makes the service keep more about a child.** About a child it keeps a pseudonym, an avatar from a set and the grade, and the handwriting stays on the device: that is a design decision rather than an unfinished piece.
- **Children's handwriting in the repository**, as a test fixture or anywhere else. Tests use synthetic data and samples written by adults.

## Setting up

Everything happens inside the development container. The host needs Docker and VS Code and nothing else:

```
git clone https://github.com/MathTrail/canvas-api
code canvas-api      # then: Reopen in Container
```

Inside it, `just --list` shows every command. The everyday ones:

```
just fmt        # format the Go code
just test       # the tests
just run        # the server, on port 8080
just web-dev    # the SPA, on port 5173
just web-fmt    # format the SPA
```

## What a change has to satisfy

The conventions are written down in [`CLAUDE.md`](CLAUDE.md) — it is addressed to an AI assistant, but it is the house style for everybody, and a reviewer will hold a change to it.

The short version:

- **The full checks are green**: `just ci-lint`, `just ci-test`, `just ci-mocks-check`, `just ci-licenses`, `just ci-vuln`, `just ci-secrets`, `just ci-image-smoke` and `just ci-image-scan`, and `just ci-web` too when the SPA in `web/` changed. They are what the pull request checks run — in the same toolchain image, but for the two image checks, which need Docker and run beside it — so a failure here is a failure there.
- **Tests come with the change.** A pull request whose changed lines are less than 90% covered cannot be merged. A new check is not considered covered until its test has been seen to fail with the check switched off.
- **Nothing personal reaches a log.** Log lines, metrics and traces carry IDs, codes and timings — never an email, a name, a pseudonym, a check's picture, a read answer or the text of a hint.
- **The task's answer stays hidden** until the task is solved, and every hint is checked for leaking it.
- **No secrets in the repository.** The whole history is scanned on every pull request.
- **Exact versions only**: every image, module, package, action and tool is pinned to an exact version.

One commit or several, as suits the change; the title of the pull request is what a reader sees in the history, so make it a sentence about what changed.

## Licensing

The code and the tasks, their pictures included, are MIT, and a contribution arrives under the same licence — by opening a pull request you are offering it on those terms.
