# MathTrail Canvas

[![CI](https://github.com/MathTrail/canvas-api/actions/workflows/ci.yml/badge.svg)](https://github.com/MathTrail/canvas-api/actions/workflows/ci.yml)
[![CodeQL](https://github.com/MathTrail/canvas-api/actions/workflows/codeql.yml/badge.svg)](https://github.com/MathTrail/canvas-api/actions/workflows/codeql.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/MathTrail/canvas-api)](https://github.com/MathTrail/canvas-api/blob/main/go.mod)
[![codecov](https://codecov.io/gh/MathTrail/canvas-api/branch/main/graph/badge.svg)](https://codecov.io/gh/MathTrail/canvas-api)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/MathTrail/canvas-api/badge)](https://scorecard.dev/viewer/?uri=github.com/MathTrail/canvas-api)
[![License: MIT](https://img.shields.io/github/license/MathTrail/canvas-api)](LICENSE)

[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=MathTrail_canvas-api&metric=code_smells)](https://sonarcloud.io/summary/new_code?id=MathTrail_canvas-api)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=MathTrail_canvas-api&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=MathTrail_canvas-api)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=MathTrail_canvas-api&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=MathTrail_canvas-api)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=MathTrail_canvas-api&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=MathTrail_canvas-api)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=MathTrail_canvas-api&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=MathTrail_canvas-api)

MathTrail Canvas is a math tutor for children in grades 1–6. A child solves a task by hand, with a stylus or a finger, on a squared draft sheet and writes the final answer in an answer box. On **Check**, an AI reads the answer and a program compares it with the task's key. When the answer is wrong, the tutor looks at the draft, frames the place where the reasoning went wrong and asks one short guiding question. It never gives the answer.

What the child sees on the screen:

- **The task** at the top. It never changes.
- **The draft sheet** — squared paper for working things out. The child writes, draws and erases whole strokes; the draft stays on the device.
- **The answer box** — the child writes the final answer there and presses **Check**.
- **The hint layer** over the draft: a soft frame around the child's own strokes and one question. It never covers the child's writing.

After **Check**, the child sees one of these:

- "Correct! I read “7”." with a short explanation of the solution;
- "Not yet. I read “6”.", then a frame and a question. The questions get more specific with every wrong answer, up to three, and then the tutor suggests asking a grown-up or starting the task again;
- a request to rewrite the answer, if it was hard to read. Doubt is never shown as a mistake;
- a neutral line if the checker cannot be reached, or, once the day's help is used up, a line saying the child can keep going alone.

The MVP is a closed, free beta for families, with five static tasks and a daily limit on the tutor's help. A clickable mockup of the screen is in [draft/](draft/README.md).

## Documents

The documents are in Russian.

- [docs/product.md](docs/product.md) — what we build and why: the scope of the MVP, scenarios, unit economics, metrics and acceptance criteria.
- [docs/spec.md](docs/spec.md) — how it is built: architecture, the sheet and the check images, the answer check, the draft analysis, the AI tutor, the API, data, security and deployment.
- [docs/privacy.md](docs/privacy.md) — children's data: what is collected and kept, what the external AI service receives, parental consent, deletion and retention.
- [docs/decisions.md](docs/decisions.md) — the decision log: what was decided, why, and what was rejected.
- [docs/stack.html](docs/stack.html) — the technology stack, layer by layer, with versions and what is still to be chosen. Open the file from a checkout in a browser: GitHub shows only its source.
- [RUN.md](RUN.md) — the implementation plan, in epics: the current epic broken into small tasks, run and reviewed one at a time, and the later epics.

## Development

All development happens in the devcontainer; nothing but Docker and VS Code is needed on the host.

1. Install Docker and VS Code with the Dev Containers extension.
2. Open the repository — on Windows, from WSL — and choose "Reopen in Container". The first build downloads the pinned toolchain and takes a few minutes.
3. `just --list` shows the available recipes.

The environment, its pinned versions and how to change one are described in [CLAUDE.md](CLAUDE.md#environment). What a change has to satisfy is in [CONTRIBUTING.md](CONTRIBUTING.md); a vulnerability is reported privately, as [SECURITY.md](SECURITY.md) describes.
