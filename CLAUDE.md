# CLAUDE.md — working rules for this repository

## Your role

You are a world-class Go engineer and software architect: you design before you type, and you write code meant to be read and maintained for years.

That bar is concrete, not a slogan. Before you call anything done:

- the change reads top to bottom — no jumping between files to follow one thought;
- names state intent, so comments are rarely needed;
- each package has one reason to change;
- a failing test names exactly what broke.

Clean Architecture, SOLID and KISS are the defaults, and they serve readability — not the other way around. When a principle would add a layer nobody needs, simplicity wins and you say why. When a task pushes you toward "good enough for now", say so out loud instead of quietly shipping it.

## The project

MathTrail Canvas (a working name, О-9) is a maths tutor for children in grades 1–6. The child solves a task by hand, with a stylus or a finger, on a virtual sheet of paper, and the service checks every step of the solution, not only the answer. When it finds a mistake it never gives the answer: it highlights the place over the child's writing and asks one short guiding question. A correct step passes in silence. `canvas-api` is the name of the service, not of the product.

The MVP is a closed, free beta for families (R05, R06). The parent owns the account and is always the subject of access; the child works on the parent's device, in child mode behind a PIN (R07). There are two static tasks, one for each way of checking (R03, R04):

- path A — arithmetic and equations: each step is a ruled line, which Mathpix recognizes and a deterministic engine checks against the last correct line;
- path B — olympiad-style diagrams: the child connects objects with arrows, and a program reads each connection from the stroke's ends and the known layout of the objects.

A false alarm is worse than no check: when recognition is unsure or a provider fails, the child gets silence or a neutral line — never a mistake (product 5.6, R20).

Technical shape (spec 1–2):

- one Go service, `canvas-api`, on Google Cloud Run, with OpenFGA as a sidecar on `localhost` (R09); managed services only — no Kubernetes, no queues (R01) — and the services scale to zero when idle;
- Centrifugo v6, a Cloud Run service of its own, pushes events to the client over WebSocket and keeps the channels' history in Upstash Redis;
- Neon Postgres holds the application's data and the OpenFGA store;
- the parent signs in with Firebase Auth (Google or Apple), and every API call but `/health` and the CORS preflight also carries a Firebase App Check token (spec 8.1);
- Mathpix recognizes handwritten lines; Gemini, on Gemini Enterprise Agent Platform (formerly Vertex AI), writes the hints and reads ambiguous drawings;
- the SPA — React and Konva, a canvas of three layers — lives in `web/` and is served by Firebase Hosting (R10, R13);
- the API contract is protobuf, package `canvasapi.v1` in `proto/`, from which buf generates Go and TypeScript (R12);
- deterministic first: the engine checks every step and a program decides every verdict; a model only words the hint and, on path B, reads what an ambiguous stroke connects (spec 1, R21);
- the MVP has one environment, prod, and every merge to `main` ships to it once CI is green (R32, R33).

## Documents

The documents under `docs/` and RUN.md are in Russian (R25); they are cited by section: "spec 8.4", "product 5.4", "privacy 7".

- [docs/product.md](docs/product.md) — what we build and why: value, the MVP's scope, scenarios, unit economics, metrics and acceptance criteria. Open questions О-… are in section 15.
- [docs/spec.md](docs/spec.md) — how it is built: stack, architecture, the canvas model, the engine, the AI tutor, the API, data, security, deployment, observability, tests and the repository's layout. Open questions Т-… and the spikes S1–S4 are in section 20.
- [docs/privacy.md](docs/privacy.md) — children's data and the law: what we collect and keep, what the external AI services receive, consent, deletion and retention. Open questions П-… are in section 12.
- [docs/decisions.md](docs/decisions.md) — the decision log, R01…: what was decided, why, and what was rejected. Read it before proposing a design change.
- [RUN.md](RUN.md) — the implementation plan: tasks T00–T58, run one at a time. Full executor rules are in its «Как пользоваться» section.
- `docs/live/` — reports of the spikes and of live runs in the cloud (from T12).
- [draft/ui/](draft/ui/README.md) — a clickable mockup of the canvas in child mode. Its README separates what comes from the docs from what is only proposed; where the two differ, the docs win.

Decisions of mathtrail-standalone are cited as "R38 standalone", or "standalone: R26, R29" in a list, as RUN.md does. The two logs are numbered independently: R32 here and R32 standalone are different decisions.

Project context lives in these files, not in chat history.

## How we work

- The author runs tasks one at a time: «Выполни задачу Txx из RUN.md». Do only that task; do not touch files that belong to other tasks unless the task says so.
- Before starting, read the RUN.md task, the product, spec and privacy sections it cites, and `docs/decisions.md`; a DevX task also reads the files of the sample it names under `reference/mathtrail-standalone/`. If the task contradicts them or something is missing, stop and ask. Do not silently fill gaps; record them as open questions under the next free number, which the section's opening names — О- in product 15 (the open ones are in 15.2), Т- in spec 20, П- in privacy 12 — or in the «Замечания» section of the document being written.
- Before calling any change done — code, config, docs or content — run `/code-review high` on it. Skip it only when the author asks, for that change alone, and say in the report that it did not run.
  - Run it once the change is written and its checks — `just ci-lint` and `just ci-test` (from T04), the others RUN.md rule 7 names for what the change touches, whatever else the task names — are green.
  - It reads the working tree plus the branch's commits — those not yet pushed, or all since `main` without an upstream — but not untracked files. Mark new files with `git add -N` for the review and unmark them with `git reset -- <file>` afterwards: the mark makes `git stash` fail. Pass no path — a path is reviewed instead of the diff, not beside it.
  - Verify each finding against the repository before acting on it; do not use `--fix`. Fix a real defect your uncommitted work introduced, wherever it shows up, with a test that fails without the fix when the defect is in code. A defect your work did not introduce — in an earlier commit, in code your work does not touch, or in another session's uncommitted work in the same tree — goes to the report as open.
  - Drop a wrong finding with a one-line reason, and one that re-proposes a rejected alternative with no new evidence by naming the decision. A finding that questions product, spec, privacy or a decision for a reason that holds goes to the author as a question.
  - If anything was fixed, run the checks and the review once more. That second round is the last: its findings are handled the same way, and its fixes get the checks but no third review. The report says what was fixed, what was dropped and why, and what is left open.
  - `/code-review ultra` is billed and started only by the author; this rule does not cover it.
- When done, mark the task in the RUN.md summary table: `[ ]` → `[x]`.
- Finish with a short report in Russian: what was done, which files, how to check it (commands), what the review found, what is still open.
- Do not commit: the author commits after review.
- Mathpix, Gemini, Google Cloud, Firebase and Apple Developer cost money. Before a live run, a spike or any action in the cloud, name its expected cost and wait for the author's confirmation.
- Do not re-propose alternatives rejected in `docs/decisions.md`, product, spec or privacy without new evidence. If a decision has to change, ask the author and update the log.
- Do not write SDK, protocol or API code from memory: check the docs of the pinned version first (Firebase Admin SDK, OpenFGA go-sdk, `google.golang.org/genai`, Mathpix v3/strokes, Centrifugo v6, pgx, goose, buf, Konva, centrifuge-js, Firebase JS SDK, the Terraform providers).
- RUN.md «Как пользоваться» holds the executor rules in full. Beyond the bullets here, it asks you to propose a split before starting a task too big for one review (about 30–60 minutes of reading), to close the previous task's SonarCloud and CodeQL findings first (from T08), and — since every merge ships to prod — to have a task bring its own environment variables, secrets and services in Terraform and CD.

## Hard rules

- **Devcontainer only.** All work happens inside the devcontainer (`.devcontainer/`, from T03): building, tests, linters, mocks, running the server and the SPA, Terraform, gcloud and live runs. Nothing but Docker and VS Code is installed or run on the host. If you are not inside the container (`MATHTRAIL_DEVCONTAINER` is not `1`), stop and ask the author to "Reopen in Container". The exceptions run on the host: T00–T03, the only host-side tasks (T03 creates the container), and the author's refresh of the reference copy (see "Reference copies").
- **Exact versions only.** No `latest`, no floating tags or ranges (spec 1). This covers:
  - Docker images and devcontainer features: exact tag plus digest;
  - Go and its modules: exact versions in `go.mod`; `github.com/tdewolff/canvas` has no tags, so it is pinned by pseudo-version (spec 3.2);
  - npm dependencies: exact versions plus a lockfile;
  - Terraform and its providers: exact versions plus `.terraform.lock.hcl` (spec 15.2);
  - Gemini models: an exact model ID in the configuration, never an alias (spec 7.1, R22);
  - tools and VS Code extensions: exact versions. The one exception: on arm64, SonarQube for IDE downloads a Java runtime of its own, at whatever version is current, because the image installs no Java (R81 standalone);
  - GitHub Actions: pinned by commit SHA.

  Each tool's exact version is pinned directly where it is installed — as an `ARG` in
  `.devcontainer/Dockerfile`, as a literal in `devcontainer.json`. No separate versions file,
  no checksum verification.
- **Language.** Code, every comment in code and config files (Go, TS, SQL, protobuf, YAML, HCL, Dockerfile, justfile, `.env.example`, …), `CLAUDE.md`, `README.md`, `CONTRIBUTING.md` and `SECURITY.md`, the texts of the UI, the hints the child sees and the model's prompts are in English (R02).
  - RUN.md and every document under `docs/` are in Russian (R25).
  - Reports to the author in chat are in Russian.
- **Children's data** (privacy 1, 4, 5).
  - About the child we keep only a pseudonym, an avatar from the set and the grade — never a name, photo, birth date, school or contacts.
  - Logs, metrics and traces never carry a direct identifier (email, name), the pseudonym, LaTeX, hint text or a token; an ID in them is a pseudonymous one — a UUID or the parent's Firebase UID (spec 13, 16; privacy 4).
  - External AI services get the maths and nothing else: no identifier, no pseudonym, no parent's data. Mathpix receives the integer coordinates of one line's strokes, with `metadata.improve_mathpix: false` (privacy 5.1). Gemini receives what privacy 5.2 lists and no more — the task's text and grade, the steps as LaTeX, the mistake's code and the hint's level, or on path B a PNG of the task's area with its list of objects. Its caching is switched off for the project, and none of its features that keep data is used: context cache, grounding with Google Search, session resumption in the Live API.
  - Children's handwriting never enters the repository or `testdata/`: tests use synthetic data and adults' samples only (spec 17), and a child's sample is never sent to Gemini.
  - No child's data exists before the operator has `verified` the parent's consent: until then, child profiles and canvases are refused with `CONSENT_REQUIRED` (R26).
- **The answer stays hidden** until the task is solved. `GET /v1/tasks`, `POST /v1/canvases` and `GET /v1/canvases/{id}` return the task without `answer` (spec 8.6, 13), and the answer never goes into a vision request (R21) or a log line (spec 16). Every hint passes the leak check before it is shown; one that fails it is replaced by a template (spec 7.3).
- **Secrets** live only in Secret Manager, environment variables or a local `.env` (never committed). No keys, tokens or client secrets in the repository: `prod.auto.tfvars` and `backend.hcl` are committed and public (spec 15.2, T10). A value reaches Secret Manager through `ci-tf-apply` — from the repository's secrets, or generated by the recipe itself — never by hand after a merge (R34; RUN.md, rule 13).
- **Licenses.** The code and the tasks, their images included, are MIT. Only add dependencies with MIT-compatible licenses, and keep the third-party license list current.

## Architecture and code standards

These are the sample's standards, applied to canvas-api; R11 reuses the sample's skeleton as well. The sample checked them against `mentor-api`, the platform's reference Go service, and where it differs from mentor-api on purpose, the reason is spelled out. mentor-api has no copy here: where a rule names it, it names the convention mentor-api set. What canvas-api changes cites the document behind the change.

### Layout

```
cmd/
  server/         the canvas-api entry point
  migrate/        goose migrations, a Cloud Run job that runs before the service is updated
  admin/          invites and consent verification, run by the operator as a Cloud Run job
  cleanup/        the daily cleanup, a Cloud Run job
internal/
  app/ config/ logger/ telemetry/ apierror/ version/   the skeleton, as in the sample (R11)
  domain/         pure logic — no I/O, no SDKs, no transport
    canvas/       page, strokes, lines, segmentation, coordinates
    mathstate/    parser, exact arithmetic, equivalence, the library of mistakes (path A)
    diagram/      connections between objects (path B)
    tutor/        routing, the hint ladder, the hint's checks, templates, prompts
  infra/          adapters to the outside world
    firebase/ openfga/ postgres/ mathpix/ gemini/ centrifugo/ render/
  transport/http/ handlers; middleware in this order: CORS, App Check, ID token, limits
proto/canvasapi/v1/  the API contract (buf)
gen/              generated Go
tasks/            the static tasks: JSON, embedded with go:embed
migrations/       goose SQL migrations
deploy/
  openfga/model.fga        the access model
  centrifugo/config.json   Centrifugo's config
infra/terraform/  the Google Cloud project as code — not to be confused with internal/infra/
web/              the SPA: React and Konva (R13); src/gen/ holds the generated TypeScript
docs/             the documents; live/ holds the reports of spikes and live runs
draft/            the canvas mockup: a proposal, not a spec
.devcontainer/ justfile .github/workflows/
```

This is the layout of spec 18, plus the mockup in `draft/`. The grouping — `domain/`, `infra/`, `transport/` — is mentor-api's and the sample's, without mentor-api's `repository.go` and `handler.go` inside each domain package, which make its domain import gin and pgx. Our domains are computation — the page's geometry, the engine, the reading of a diagram, the tutor's rules — and they stay free of I/O. That is what lets the engine be tested with tables and properties alone, with no mocks at all (spec 17). The HTTP router is Т-1's to settle, in T13; the sample and mentor-api use gin.

### The dependency rule

Dependencies point inward: `app` → `transport` → `domain`, and `infra` implements what `domain` declares. Nothing in `internal/domain/` may import `net/http`, the router, the Firebase, OpenFGA or Centrifugo clients, `genai`, pgx, `database/sql` or the renderer; pgx itself is imported only by `internal/infra/postgres` and `cmd/migrate`.

The domain reaches the outside world only through its ports (spec 1): `Recognizer`, `Tutor`, `Authorizer`, `Notifier` and `Store` — the last one, in practice, several narrow storage interfaces, one per consumer, rather than one for everything. Their adapters live in `internal/infra/`: Mathpix, Gemini, OpenFGA, Centrifugo and Postgres, plus Firebase for sign-in, App Check and deleting an account. The ports are also what keeps a move to the MathTrail platform (k8s + EDA) a change of adapters rather than of the domain (spec 19).

A domain package reaching for infrastructure is not a style problem — it means the code is in the wrong package. Move the I/O out to `infra/` or `transport/` and leave the decision-making behind. The engine does not call Mathpix:

```go
// wrong — the engine now needs Mathpix, a network and a key to be tested
func (e *Engine) CheckLine(ctx context.Context, line canvas.Line) (Verdict, error) {
    rec, err := e.mathpix.Recognize(ctx, line.Strokes) // the engine doing I/O
    ...
}

// right — a pure function over what was recognized; the caller gets the LaTeX
// through the Recognizer port and hands it in
func CheckStep(lastCorrect, step Expr) Verdict
```

### Interfaces

As in `mentor-api`: `NewX(...)` returns the interface, and the implementation stays unexported. An interface lives in the package that provides it — except a port, which the domain package that needs it declares; the adapter's constructor then returns the port, the way the sample's `starlark.New` returns `solver.Runner`.

```go
// internal/domain/canvas, say — the port, declared by the side that needs it
type Recognizer interface {
    Recognize(ctx context.Context, line Line) (Recognition, error)
}

// internal/infra/mathpix — the adapter
type client struct{ http *http.Client }                // unexported
func New(c *http.Client, key AppKey) canvas.Recognizer // the constructor returns the port
```

This is deliberately *not* the usual Go advice of "accept interfaces, return structs": we keep the platform's convention so that every service in MathTrail reads the same way and mockery always has an interface to generate from. Do not "fix" it back.

Keep them narrow — every interface in `mentor-api` has one to three methods. Past five, split it: an interface that large usually means the type does several jobs.

Extend by adding an implementation, not another branch. A second `switch` over kinds of recognizer or kinds of notifier is the signal to reach for an interface instead.

Implementations of the same interface must be interchangeable, and that is worth proving in code: when a port gets a second implementation — Gemini behind `Recognizer`, should spike S1 overturn R19 — both run one shared contract test set.

### Simplicity beats purity

- Introduce an abstraction when there is a second consumer, not in anticipation of one. Three similar lines beat a premature interface.
- No layer that only forwards calls. If a type's whole body is `return s.next.Do(x)`, delete it.
- If explaining the design takes longer than reading the code, the design is wrong.
- When simplicity and a principle collide, simplicity wins — and the trade-off goes into `docs/decisions.md` so nobody relitigates it later.

### Naming and size

- Packages are named after the domain (`canvas`, `mathstate`, `diagram`, `tutor`). Never `utils`, `helpers`, `common`, `misc` — those names attract anything and explain nothing.
- A function does one thing. If you have to scroll back to remember what it was doing, split it.
- A file that cannot be taken in at a glance usually means the package mixes topics.
- Errors read `fmt.Errorf("mathpix: recognize line: %w", err)`: package, action, wrapped cause.

### Wiring

- A hand-written container, `app.NewContainer(ctx, cfg, logger)` — the same shape as `mentor-api` and the sample. No DI framework.
- Resources close in reverse order, including when construction fails halfway.
- `signal.NotifyContext` plus `errgroup` in `main`; graceful shutdown uses `context.WithoutCancel(ctx)` with a timeout.

### Config

- Environment variables only, read in `internal/config`, nowhere else. Viper does the reading, as in `mentor-api`: one declaration per variable carries its name, its default and its type, and no file or remote source is registered.
- Every variable of ours is prefixed `CANVAS_` (spec 15.4); `CANVAS_ENV` is `local` or `prod` (spec 8.1). The listen port comes from `PORT`, as Cloud Run sets it.
- Defaults as named constants; `Validate()` returns errors that name the offending variable.
- Dev-only switches — the stub for sign-in and App Check, JSON bodies (spec 8.1), faked providers — refuse to start when `K_SERVICE` is set, whatever `CANVAS_ENV` says: Cloud Run sets that variable on every service, so no value of ours can open a switch there. A Cloud Run job gets `CLOUD_RUN_JOB` instead, and a switch that a job can reach checks that one as well.

### Logging

- `go.uber.org/zap`, injected into constructors, never global.
- JSON encoder with the Cloud Run keys `severity`, `message` and `time`. (`mentor-api` logs `ts` in ISO8601 — different runtime, different keys.)
- Lowercase messages, snake_case keys.
- A line carries IDs, codes, timings, `prompt_version` and the model's ID — nothing that "Children's data" above keeps out (spec 16). Every model call logs its `prompt_version` and model ID (spec 7.4).

### Errors

- Wrap with short context; use sentinel errors where callers branch, checked with `errors.Is`/`errors.As`.
- Log an error once, at the boundary.
- An HTTP error is the flat `apierror` body — `{code, message}` with an optional `details` — carrying a code from spec 8.3; the request's ID travels in the `X-Request-Id` header, not in the body (R11). Never send internal error text to the client.
- A failure of Mathpix, Gemini or the engine is not an HTTP error: it becomes an `UNCHECKED` verdict or a template hint (spec 9.9). The engine's panic is recovered at its boundary into `UNCHECKED` with the reason `engine_error`; a failed publish to Centrifugo is logged, and the HTTP response still carries the events.

### Context and docs

- `ctx context.Context` is the first parameter of anything that blocks or does I/O; slow calls get their own child timeout — Mathpix 3 s (spec 9.9), Gemini 4 s for a hint and 8 s for a drawing (spec 7.1).
- A request finishes its work before it answers: with request-based billing, Cloud Run gives almost no CPU to a goroutine left running after the response (R23).
- Every package has a package comment; every exported identifier has a doc comment starting with its name. Comments explain *why*.
- `internal/version` variables are set through `-ldflags -X` in both the Dockerfile and CI builds.

### Tests

- Standard `testing` with `t.Fatalf`/`t.Errorf` in "got …, want …" form — the house style in `mentor-api`, where testify appears only as `mock.Anything` in generated mocks. That is the sense in which spec 3.2 lists testify "as in the ecosystem".
- Table-driven subtests (`t.Run`) for similar cases, with `t.Parallel()` where a case shares no state; external `_test` packages for public behaviour, internal ones for unexported helpers.
- Test helpers take `*testing.T` and start with `t.Helper()`, so a failure points at the test that broke, not at the helper.
- Anything that parses what came from outside gets a fuzz test (`FuzzXxx(f *testing.F)`): the LaTeX that Mathpix returns, the model's JSON, stroke deltas, the task JSON — and the responses and request bodies that carry them. That input is untrusted, a panic there takes a request handler down with it, and fuzzing is how those inputs get found before a child's handwriting finds them.
- Computation that holds an invariant gets a property test with `gopter`, the engine first of all (spec 17): random linear equations under valid transformations stay `CORRECT`, an injected mistake of each type gets its own code, no valid transformation gives `ERROR`, and no division, by zero included, panics. The test names the invariant and the library looks for the input that breaks it. A property replaces neither the worked examples of spec 5.6 nor a fuzz test; it is what catches the case nobody thought of.
- A new check counts as covered only when its test fails with the check switched off. Break it, watch the test go red, put it back — that is how a test that passes by accident is found, and it is cheaper than any tool that does the same thing by rewriting the code.
- `go test -race` always; fixtures under `testdata/`.
- Recorded responses of Mathpix and Gemini in `testdata/` are synthetic — never a child's sample (spec 17). They are the reference a test is checked against, so no test rewrites them. A snapshot the service itself owns, such as the path B render, may have `-update`, and then the regenerated diff is part of the review.
- No network in unit tests: Firebase, Mathpix, Gemini and Centrifugo are faked with `httptest`.
- Integration tests run Postgres and OpenFGA in containers, under a build tag of their own (`just ci-integration`, from T20).
- Behind an interface stands the real implementation, whenever that implementation does no I/O, and a small hand-written fake where a test has to force it to behave a particular way. That is what proves a property rather than the fact of a call, and it keeps what a test assumes in one readable place.
- A mock earns its place when the real implementation needs the network, a clock, or is simply slow — and when what is being checked is that a call happened, with these arguments. It is then generated by mockery v3 into a `mocks/` folder next to the interface and committed; CI checks it is up to date. Config lives in `.mockery.yaml`, with expecters on, mirroring `mentor-api`, and its list of packages is empty until an interface of that kind exists.

### Tooling and images

- `gofmt -s`, plus `golangci-lint` with a committed `.golangci.yml` and a pinned version. (`mentor-api` has no lint config checked in — this is one of the gaps we close.)
- The SPA in `web/`: Biome formats and lints it (`biome.json`), TypeScript checks its types and compiles nothing, Vitest runs its tests in happy-dom, and Vite builds it (R84 standalone). Firebase Hosting serves the build, so the service's binary and image carry none of it (R10).
- buf lints `proto/`, holds it to backward compatibility against `main`, and generates Go into `gen/` and TypeScript into `web/src/gen/`; the generated code is committed, and CI checks it is current (R12).
- A pre-commit hook in `.githooks/`, enabled by `post-start`; `govulncheck`, `npm audit` and `gitleaks` in CI.
- The justfile is the entry point for everything; `ci-*` recipes are what CI calls — same recipe names as `mentor-api` (`fmt`, `fmt-check`, `build`, `test`, `mocks`, `ci-lint`, `ci-test`, `ci-mocks-check`).
- Multi-stage Dockerfile, minimal non-root runtime image pinned by digest, `CGO_ENABLED=0`, `-trimpath`.
- Services and tools that are not part of the build run from images pinned by digest instead of being installed into the devcontainer: Postgres, OpenFGA, Centrifugo and Redis for local runs and integration tests, the OpenFGA CLI for the model's tests, Playwright for the end-to-end tests. The SPA's own packages are the exception, because they are part of the build: they install into `web/node_modules` from its lockfile.

## Reference copies

`reference/mathtrail-standalone/` is a local, gitignored snapshot of the working tree of `mathtrail-standalone`, the sibling repository these rules and the DevX come from. The devcontainer cannot see a sibling directory of the host, so a task reads the snapshot instead.

- Read-only: never edit it, and never commit anything from it.
- It is what R11 ports from: the skeleton (`app`, `config`, `logger`, `telemetry`, `apierror`, `version`), the devcontainer, the justfile, CI/CD, Terraform and the quality settings — each changed only where canvas-api needs it. "As in standalone" in RUN.md means: take the named file from the snapshot and change only what the task names.
- Its rules file is renamed to `CLAUDE.standalone.md`. Under its own name, Claude Code would load it whenever a file of the snapshot is read, and its rules do not apply here: it is the sample this file follows, section by section.
- Left out of it: the sample's `.git`, its installed packages, binaries, coverage profile and Terraform providers (`node_modules`, `bin/`, `dist/`, `coverage.out`, `.terraform/`), its own reference material (`prototype/`, `reference/`, `research/`, `draft/`) — so mentor-api, which the sample keeps under `reference/`, is not here — and what is local to the sample's machine: a `.env` with its secrets and a `CLAUDE.local.md`, which Claude Code would load just like a `CLAUDE.md`.
- Search tools that respect `.gitignore` (ripgrep) skip it. Use explicit paths (`Read`, `ls`, `grep -r reference/mathtrail-standalone/...`).
- The author refreshes it on the host, from the root of this repository:

  ```sh
  mkdir -p reference && rsync -a --delete \
    --exclude=.git --exclude=node_modules \
    --exclude=/prototype/ --exclude=/reference/ --exclude=/research/ --exclude=/draft/ \
    --exclude=bin/ --exclude=dist/ --exclude=coverage.out --exclude=.terraform/ \
    --exclude=.env --exclude=CLAUDE.local.md \
    ../mathtrail-standalone/ reference/mathtrail-standalone/ \
    && mv reference/mathtrail-standalone/CLAUDE.md reference/mathtrail-standalone/CLAUDE.standalone.md
  ```

## Environment

Open the repository in VS Code and choose "Reopen in Container" (`.devcontainer/`).

The host needs Docker and VS Code and nothing else: no builder plugin, no daemon settings, nothing to edit as root (R36 standalone). The image is written for any builder, and the network the container runs on, `canvas-api`, is created before it starts by `.devcontainer/host-network.sh`, with the MTU of the interface the host reaches the internet by — a tunnel on the host would otherwise cut every large download inside the container short with an error that never mentions the network. That script runs in `bash`, which Linux and macOS have; on Windows, the repository is opened from WSL. `post-start` mirrors that same value onto the docker running inside the container; that docker reads it when it starts, so after a rebuild behind a tunnel the container is restarted once.

Inside the container:

- **Pinned versions.** No separate versions file: each tool's exact version is an `ARG` default in `.devcontainer/Dockerfile`, repeated as a literal in `.devcontainer/devcontainer.json` where needed.
  - The Dockerfile has two stages. `toolchain` installs Go, Node.js, just, golangci-lint and mockery — what it takes to build and check the code; `devcontainer` adds gopls, dlv, cloudflared, gh, the Docker Compose plugin, Terraform, gcloud and Claude Code on top, and is the stage the container is built from. cloudflared is the tunnel that opens the SPA served from here on a real tablet. gcloud is installed on x86_64 only, because its archive for arm carries no Python interpreter; there the same commands come from Google's own container image (R28 standalone).
  - Everything is installed straight from the Dockerfile, and downloads refuse a protocol downgrade (R35 standalone). What is verified is verified by the source it comes from: the Go checksum database for `go install`, the registry hash for the npm package; the tarballs fetched with `curl` carry no checksum of their own. Of the npm packages only the Claude Code CLI may run its own install step, named explicitly, because that step is what puts its native binary in place.
  - Docker (docker-in-docker, Moby engine and buildx) comes from a devcontainer feature pinned in `devcontainer.json` and `devcontainer-lock.json`.
  - To change a version: edit the literal everywhere it appears (Dockerfile, `devcontainer.json` for Docker or the Claude Code extension), rebuild the container. The Claude Code CLI and its editor extension are one version in two files, so `just claude-update` raises both at once — with a version, or to the newest published one — and prints what it changed.
- **Container marker.** `MATHTRAIL_DEVCONTAINER=1` is set only inside the container. A session that does not see it is on the host and must stop (see "Devcontainer only").
- **Claude Code.** Its config lives in the `canvas-api-claude` volume, so the login survives rebuilds. Auto-update is off; the version is pinned.
- **`gh`.** What a change looks like on github.com is otherwise invisible from in here, and two parts of it are asked for by name: the alerts of the code scan, which a task closes before the next one starts, and whether the delivery that carried the change ran. `gh auth login` is answered once: the credentials live in the `canvas-api-gh` volume, as gcloud's live in `canvas-api-gcloud`, where Terraform reads them.
- **Ports.** 8080 (the API) and 5173 (Vite's dev server for the SPA) are forwarded to the host.
- **Just recipes.** `just --list` shows all of them.
- **Build context.** The devcontainer image is built with the repository root as context, though it copies nothing from it; `.dockerignore` keeps `.git`, `.env`, `.devcontainer/`, `reference/`, `docs/`, `draft/` and the Markdown files at the root out of every image built from there.
