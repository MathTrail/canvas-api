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

MathTrail Canvas (the name settled by R55, О-9) is a maths tutor for children in grades 1–6. The child solves a task by hand, with a stylus or a finger, on a squared draft sheet and writes the final answer in an answer box. On Check, an AI reads the answer, a program compares it with the task's key, and the child sees what was read. When the answer is wrong, the tutor looks at the draft, frames the place where the reasoning went wrong over the child's own strokes and asks one short guiding question. It never gives the answer. `canvas-api` is the name of the service, not of the product.

The MVP is a closed, free beta for families (R05, R06). The parent owns the account and is always the subject of access; the child works on the parent's device, in child mode behind a PIN (R07). There are five static tasks from the mockup, kept in `tasks/` until a catalog exists (R40).

A false alarm is worse than no check: when the reading is unsure, the child is asked to rewrite the answer, and when a provider fails, the child gets a neutral line — never a mistake (product 5.6, R38).

Technical shape (spec 1–2):

- one Go service, `canvas-api`, on Google Cloud Run, with OpenFGA as a sidecar on `localhost` (R09); managed services only — no Kubernetes, no queues (R01) — and the services scale to zero when idle;
- Centrifugo v6, a Cloud Run service of its own, pushes events to the client over WebSocket and keeps the channels' history in Upstash Redis;
- Neon Postgres holds the application's data and the OpenFGA store;
- the parent signs in with Firebase Auth (Google or Apple), and every API call but `/health` and the CORS preflight also carries a Firebase App Check token (spec 8.1);
- Gemini, on Gemini Enterprise Agent Platform (formerly Vertex AI), makes two calls: it reads the answer box's image on a Check — unless the daily limit is spent or the box is blank — and after a wrong answer it reads the draft's image together with the key and writes the frame and the question (R38, R39);
- the SPA — React and Konva: a draft sheet with a drawing layer and a hint layer, and an answer box — lives in `web/` and is served by Firebase Hosting (R10, R13). The draft lives on the device, in IndexedDB; on Check the SPA sends PNGs drawn from the strokes — the answer box and, unless the sheet is empty, the draft — and the server keeps them only in the request's memory (R42);
- the API contract is protobuf, package `canvasapi.v1` in `proto/`, from which buf generates Go and TypeScript (R12);
- a program decides every verdict: the reading is compared with the task's key, and a model only reads the answer and words the hint (spec 1, R37, R38);
- the MVP has one environment, prod, and every merge to `main` ships to it once CI is green, from a signed release (R32, R33, R45).

## Documents

The documents under `docs/` and RUN.md are in Russian (R25); they are cited by section: "spec 8.4", "product 5.4", "privacy 7".

- [docs/product.md](docs/product.md) — what we build and why: value, the MVP's scope, scenarios, unit economics, metrics and acceptance criteria. Open questions О-… are in section 15.
- [docs/spec.md](docs/spec.md) — how it is built: stack, architecture, the sheet and the check images, the answer check, the draft analysis, the AI tutor, the API, data, security, deployment, observability, tests and the repository's layout. Open questions Т-… and the spikes S1–S4 are in section 20.
- [docs/privacy.md](docs/privacy.md) — children's data and the law: what we collect and keep, what the external AI service receives, consent, deletion and retention. Open questions П-… are in section 12.
- [docs/decisions.md](docs/decisions.md) — the decision log, R01…: what was decided, why, and what was rejected. Read it before proposing a design change.
- [docs/stack.html](docs/stack.html) — the stack by architecture layer: each technology with its version, status and the section, decision or task behind it. A summary, not a source: where it differs from spec, the decision log or RUN.md, they win.
- [RUN.md](RUN.md) — the implementation plan, in epics (R44). Table 1 holds the current epic's tasks, run one at a time; table 2 holds later epics, broken into tasks only when their turn comes; table 3 holds the questions the author has not answered yet, and table 4 the technical debt. Full executor rules are in its «Как пользоваться» section.
- `docs/live/` — reports of the spikes and of live runs in the cloud (from T12). Every report follows one format:
  - its status, in the first line under the title;
  - no identifier of a person or an account — no email, UID, invite code, billing account or token: the repository is public;
  - the version and the commit that ran;
  - the roles: what the author did and what Claude did;
  - the cost, estimated before the run and actual after it;
  - the course of the run, in UTC;
  - «Открыто»;
  - «Как повторить».
- [draft/](draft/README.md) — `MathTrail Minimal.html`, a clickable mockup of the task screen in child mode. Its README lists where the mockup differs from the docs; where the two differ, the docs win.

Decisions of mathtrail-standalone are cited as "R38 standalone", or "standalone: R26, R29" in a list, as RUN.md does. The two logs are numbered independently: R32 here and R32 standalone are different decisions.

Project context lives in these files, not in chat history.

## How we work

- The author runs tasks one at a time: «Выполни задачу Txx из RUN.md». Do only that task; do not touch files that belong to other tasks unless the task says so.
- Before starting, read the RUN.md task, the product, spec and privacy sections it cites, and `docs/decisions.md`; a DevX task also reads the files of the sample it names under `reference/mathtrail-standalone/`. If the task contradicts them or something is missing, stop and ask. Do not silently fill gaps; record them as open questions under the next free number, which the section's opening names — О- in product 15 (the open ones are in 15.2), Т- in spec 20, П- in privacy 12 — or in the «Замечания» section of the document being written.
- Before calling any change done — code, config, docs or content — run `/code-review high` on it. Skip it only when the author asks, for that change alone, and say in the report that it did not run.
  - Run it once the change is written and its checks — `just ci-lint`, `just ci-test` and `just ci-mocks-check` (from T04), `just ci-licenses` and `just ci-vuln` (from T05), `just ci-secrets`, `just ci-image-smoke` and `just ci-image-scan` (from T06), `just release-version` and `just release-artifacts` (from T08), `just tf-check` (from T10), `just ci-workflows` (from T11), the others RUN.md rule 7 names for what the change touches, whatever else the task names — are green.
  - The checks and the review run in a clean copy of the tree that holds this change and nothing else, so that another session's uncommitted work gets into neither. A tree that holds nothing but the change is that copy already. Otherwise the copy is a `git worktree` of `HEAD`, outside the repository, given this change's own edits, new files included, and what its checks need that git does not keep — a link to `reference/`, `web/node_modules` installed from the lockfile — and the checks and the review run from inside it. Fixes are made in the tree and carried into the copy, and `git worktree remove --force` takes the copy away after the review: it holds nothing the tree does not.
  - It reads the working tree plus the branch's commits — those not yet pushed, or all since `main` without an upstream — but not untracked files. Mark new files with `git add -N` for the review and unmark them with `git reset -- <file>` afterwards: the mark makes `git stash` fail. Pass no path — a path is reviewed instead of the diff, not beside it.
  - Verify each finding against the repository before acting on it; do not use `--fix`. Fix a real defect your uncommitted work introduced, wherever it shows up, with a test that fails without the fix when the defect is in code. A defect your work did not introduce — in an earlier commit, in code your work does not touch, or in another session's uncommitted work in the same tree — goes to the report as open.
  - Drop a wrong finding with a one-line reason, and one that re-proposes a rejected alternative with no new evidence by naming the decision. A finding that questions product, spec, privacy or a decision for a reason that holds goes to the author as a question.
  - If anything was fixed, run the checks and the review once more. That second round is the last: its findings are handled the same way, and its fixes get the checks but no third review. The report says what was fixed, what was dropped and why, and what is left open.
  - `/code-review ultra` is billed and started only by the author; this rule does not cover it.
- A finding of a code analyser — CodeQL, SonarCloud, Scorecard — is answered by changing the code. Marking one false is the last resort: the mark gives its reason in words and names the test that proves it (standalone: R35, R50, R111). A finding no change can close, such as Scorecard's ask for a second reviewer, stays open with its reason written down (R51 standalone).
- When done, mark the task in RUN.md's table 1, `[ ]` → `[x]`, and add «Сделано (дата)» and «Открыто» to its body (RUN.md, rule 10).
- Finish with a short report in Russian: what was done, which files, how to check it (commands), what the review found, what is still open.
  - It says how many checks in the code were switched off to prove their tests and whether each one turned its test red, and how long each fuzz test ran.
  - It names every flake — a test that fails under load and passes when run alone — and lists it under «Открыто». A flake is cured in its test, as R207 standalone cured one; the CI job it turned red is not re-run in the hope that it passes, since a pass by chance leaves the flake where it was.
- Do not commit: the author commits after review.
- Gemini, Google Cloud, Firebase and Apple Developer cost money. Before a live run, a spike or any action in the cloud, name its expected cost and wait for the author's confirmation. For the length of a live run, the author gives Claude a narrow permission in `.claude/settings.local.json`, for the run's own commands and nothing more; once the run is over, the file is deleted.
- Do not re-propose alternatives rejected in `docs/decisions.md`, product, spec or privacy without new evidence — nor, for a practice ported from the sample, what the sample's log rejected, such as mutation-testing tools (R39 standalone) or a merge gated on SonarCloud's quality gate (R31 standalone). If a decision has to change, ask the author and update the log.
- A new decision's R number and a new task's T number are the ones still free when they are written down, not when the work began: a parallel session may have taken the next one meanwhile.
- Do not write SDK, protocol or API code from memory: check the docs of the pinned version first (Firebase Admin SDK, OpenFGA go-sdk, `google.golang.org/genai`, Centrifugo v6, pgx, goose, buf, Konva, centrifuge-js, Firebase JS SDK, the Terraform providers), and cite them with the date you read them — in the report, and in any decision or document that rests on them. Where the docs and the pinned version disagree, the library itself is the reference (R06 standalone).
- RUN.md «Как пользоваться» holds the executor rules in full. Beyond the bullets here, it asks you to propose a split before starting a task too big for one review (about 30–60 minutes of reading); before starting, to close what the previous task left — the findings of SonarCloud, CodeQL and Scorecard (from T08) — and, once T11 brings CD, to check with `gh` that the delivery that carried it passed; and, since every merge ships to prod, to have a task bring its own environment variables, secrets and services in Terraform and CD.

## Hard rules

- **Devcontainer only.** All work happens inside the devcontainer (`.devcontainer/`, from T03): building, tests, linters, mocks, running the server and the SPA, Terraform, gcloud and live runs. Nothing but Docker and VS Code is installed or run on the host. If you are not inside the container (`MATHTRAIL_DEVCONTAINER` is not `1`), stop and ask the author to "Reopen in Container". The exceptions run on the host: T00–T03, the only host-side tasks (T03 creates the container), and the author's refresh of the reference copy (see "Reference copies").
- **Exact versions only.** No `latest`, no floating tags or ranges (spec 1). This covers:
  - Docker images and devcontainer features: exact tag plus digest. An image whose tags name no version, such as distroless, is pinned by digest alone;
  - Go and its modules: exact versions in `go.mod`;
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
  - A comment, and any text a person reads from code or config — a description, an error message, a test's name — gives its reason in its own words. It cites no decision, task or section of the documents: no `(R58)`, `(T11)`, `(spec 15.3)`, `(RUN.md, rule 13)`. The documents do not outlive the code.
- **Children's data** (privacy 1, 4, 5).
  - About the child we keep only a pseudonym, an avatar from the set and the grade — never a name, photo, birth date, school or contacts.
  - Logs, metrics and traces never carry a direct identifier (email, name), the pseudonym, a check image, the read answer, hint text or a token; an ID in them is a pseudonymous one — a UUID or the parent's Firebase UID (spec 13, 16; privacy 4).
  - The child's strokes stay on the device (R42). The server holds the two check images only in the request's memory — never in the database, a log or a trace.
  - Gemini receives what privacy 5.1 lists and no more: on a Check that the limit or a blank box does not stop, the answer box's image and the answer's form; after a wrong answer, also the draft's image, the task's text and grade, its key and solution, the read answer, the matched trap, the hint's level and the sheet's earlier questions. We add no identifier, no pseudonym and no parent's data — though an image shows whatever the child wrote, a name included (privacy 5.3). Gemini's caching is switched off for the project, and none of its features that keep data is used (privacy 5.2).
  - Children's handwriting never enters the repository or `testdata/`: tests use synthetic data and adults' samples only (spec 17). Eval sets are built from adults' samples; children's samples join them only if П-10 allows.
  - No child's data exists before the operator has `verified` the parent's consent: until then, child profiles and canvases are refused with `CONSENT_REQUIRED` (R26).
- **The answer stays hidden** until the task is solved. `GET /v1/tasks`, `POST /v1/canvases` and `GET /v1/canvases/{id}` return the task's open view, `TaskView`, without `answer` (spec 8.6, 13); the solution comes only with a `CORRECT` verdict. The key never goes into the reading call (R38) or a log line (spec 16). The draft analysis sees the key (R39), so every hint passes the leak check before it is shown; one that fails it is replaced by the author's text or a template (spec 6.3, 7.3).
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
    canvas/       the sheet, coordinates, the check images, the frame's conversion
    answer/       normalization, the answer's form, comparison with the key, the verdict; the reading prompt
    tutor/        the route after a verdict, the hint ladder, the question's checks, templates; the analysis prompt
  check/          one check from start to end: lease, limits, calls and events through its own ports
  infra/          adapters to the outside world
    firebase/ openfga/ postgres/ gemini/ centrifugo/
  transport/http/ handlers; middleware in this order: CORS, App Check, ID token, limits
proto/canvasapi/v1/  the API contract (buf)
gen/              generated Go
tasks/            the static tasks: JSON, embedded with go:embed
migrations/       goose SQL migrations
deploy/
  openfga/model.fga        the access model
  centrifugo/config.json   Centrifugo's config
infra/terraform/  the Google Cloud project as code — not to be confused with internal/infra/
infra/bootstrap.sh  the one-time step that makes what the pipeline signs in with: project, state bucket, WIF
web/              the SPA: React and Konva (R13); src/gen/ holds the generated TypeScript
firebase.json     what Firebase Hosting serves, and with which headers (spec 15.5)
docs/             the documents; live/ holds the reports of spikes and live runs
draft/            the screen mockup: a proposal, not a spec
.devcontainer/ justfile .github/workflows/
```

This is the layout of spec 18. The grouping — `domain/`, `infra/`, `transport/` — is mentor-api's and the sample's, without mentor-api's `repository.go` and `handler.go` inside each domain package, which make its domain import gin and pgx. Our domains are computation — the sheet's geometry and the check images, the answer's normalization and comparison, the tutor's rules — and they stay free of I/O. That is what lets them be tested with tables and properties alone, with no mocks at all (spec 17). Use cases that do I/O sit outside `domain/` for that reason, one package each: `check` runs a check, and deleting a profile or an account (R27), the daily cleanup and the flows behind the other endpoints get packages of their own beside it, named by the MVP epic as it builds them (spec 18). The HTTP router is Т-1's to settle before the MVP epic's first code; the sample and mentor-api use gin.

### The dependency rule

Dependencies point inward: `app` → `transport` → the use cases (`check` and its siblings) → `domain`, and `infra` implements the ports the use cases declare. Nothing in `internal/domain/` or in a use case may import `net/http`, the router, the Firebase, OpenFGA or Centrifugo clients, `genai`, pgx or `database/sql`; pgx itself is imported only by `internal/infra/postgres` and `cmd/migrate`.

A use case reaches the outside world only through the ports it declares (spec 1, 18). A check needs `Authorizer`, `Recognizer`, `Tutor`, `Notifier` and `Store` — the last one, in practice, several narrow storage interfaces, one per consumer, rather than one for everything — and every other use case that checks or changes access declares an `Authorizer` of its own. Their adapters live in `internal/infra/`: Gemini — two types, behind `Recognizer` and `Tutor` — OpenFGA, Centrifugo and Postgres, plus Firebase for sign-in, App Check and deleting an account. The ports are also what keeps a move to the MathTrail platform (k8s + EDA) a change of adapters rather than of the domain (spec 19).

A domain package reaching for infrastructure is not a style problem — it means the code is in the wrong package. Move the I/O out to a use case, `infra/` or `transport/` and leave the decision-making behind. Comparing with the key does not call Gemini:

```go
// wrong — the comparison now needs Gemini, a network and credentials to be tested
func Judge(ctx context.Context, img canvas.AnswerImage, task Task) (Verdict, error) {
    reading, err := gemini.Read(ctx, img) // the domain doing I/O
    ...
}

// right — a pure function over what was read; check gets the reading
// through the Recognizer port and hands it in
func Judge(key Key, reading Reading) Verdict
```

### Interfaces

As in `mentor-api`: `NewX(...)` returns the interface, and the implementation stays unexported. An interface lives in the package that provides it — except a port, which the package that needs it declares; the adapter's constructor then returns the port, the way the sample's `starlark.New` returns `solver.Runner`.

```go
// internal/check — the port, declared by the side that needs it
type Recognizer interface {
    Read(ctx context.Context, img canvas.AnswerImage, form answer.Form) (answer.Reading, error)
}

// internal/infra/gemini — the adapter
type reader struct{ client *genai.Client }                      // unexported
func NewReader(c *genai.Client, model ModelID) check.Recognizer // the constructor returns the port
```

This is deliberately *not* the usual Go advice of "accept interfaces, return structs": we keep the platform's convention so that every service in MathTrail reads the same way and mockery always has an interface to generate from. Do not "fix" it back.

Keep them narrow — every interface in `mentor-api` has one to three methods. Past five, split it: an interface that large usually means the type does several jobs.

Extend by adding an implementation, not another branch. A second `switch` over kinds of recognizer or kinds of notifier is the signal to reach for an interface instead.

Implementations of the same interface must be interchangeable, and that is worth proving in code: when a port gets a second implementation — another model provider behind `Recognizer`, say — both run one shared contract test set.

### Simplicity beats purity

- Introduce an abstraction when there is a second consumer, not in anticipation of one. Three similar lines beat a premature interface.
- No layer that only forwards calls. If a type's whole body is `return s.next.Do(x)`, delete it.
- If explaining the design takes longer than reading the code, the design is wrong.
- When simplicity and a principle collide, simplicity wins — and the trade-off goes into `docs/decisions.md` so nobody relitigates it later.

### Naming and size

- Packages are named after the domain (`canvas`, `answer`, `tutor`, `check`). Never `utils`, `helpers`, `common`, `misc` — those names attract anything and explain nothing.
- A function does one thing. If you have to scroll back to remember what it was doing, split it.
- A file that cannot be taken in at a glance usually means the package mixes topics.
- Errors read `fmt.Errorf("gemini: read answer: %w", err)`: package, action, wrapped cause.

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
- The log is held to a table of its events and the fields each one may carry, not to a list of what it must not hold: a field nobody decided on fails the test the day it is added, before any value it carries can leak (spec 16; standalone: R141, R169). The table and its test come with the MVP epic.

### Errors

- Wrap with short context; use sentinel errors where callers branch, checked with `errors.Is`/`errors.As`.
- Log an error once, at the boundary.
- An HTTP error is the flat `apierror` body — `{code, message}` with an optional `details` — carrying a code from spec 8.3; the request's ID travels in the `X-Request-Id` header, not in the body (R11). Never send internal error text to the client.
- A failure of Gemini or of the check's own code is not an HTTP error: it becomes an `UNCHECKED` verdict or a fallback hint (spec 5.4, 5.5, 9.9). The store is the exception: when Neon or OpenFGA fails, or no slot frees up to decode the images, the answer is `UNAVAILABLE`, and the client's retry with the same `client_check_id` finishes the check (spec 8.3, 8.4, Т-14). A panic in `answer` is recovered at its boundary into `UNCHECKED` with the reason `internal`; one in `tutor`, after the verdict is out, into a fallback hint. A failed publish to Centrifugo is logged, and the HTTP response still carries the events.

### Context and docs

- `ctx context.Context` is the first parameter of anything that blocks or does I/O; slow calls get their own child timeout — Gemini 4 s for reading the answer and 8 s for the draft analysis, both trimmed to what is left of the request's 25 s (spec 7.1, 8.4).
- A request finishes its work before it answers: with request-based billing, Cloud Run gives almost no CPU to a goroutine left running after the response (R23).
- Every package has a package comment; every exported identifier has a doc comment starting with its name. Comments explain *why*.
- `internal/version` variables are set through `-ldflags -X` in both the Dockerfile and CI builds.

### Tests

- Standard `testing` with `t.Fatalf`/`t.Errorf` in "got …, want …" form — the house style in `mentor-api`, where testify appears only as `mock.Anything` in generated mocks. That is the sense in which spec 3.2 lists testify "as in the ecosystem".
- Table-driven subtests (`t.Run`) for similar cases, with `t.Parallel()` where a case shares no state; external `_test` packages for public behaviour, internal ones for unexported helpers.
- Test helpers take `*testing.T` and start with `t.Helper()`, so a failure points at the test that broke, not at the helper.
- Anything that parses what came from outside gets a fuzz test (`FuzzXxx(f *testing.F)`): the model's JSON of both calls, the task JSON, the check request and its PNGs — and the responses and request bodies that carry them. That input is untrusted, a panic there takes a request handler down with it, and fuzzing is how those inputs get found before a child's handwriting finds them.
- Computation that holds an invariant gets a property test with `gopter`, the answer check first of all (spec 17): normalizing twice changes nothing, any accepted spelling of a number equals a key of the same value and no other, the form's whitelist takes no base prefix, exponent, underscore or redundant leading zero (`07`, `010/2`), and no input panics. The test names the invariant and the library looks for the input that breaks it. A property replaces neither the worked examples of spec 5.6 nor a fuzz test; it is what catches the case nobody thought of.
- A new check counts as covered only when its test fails with the check switched off. Break it, watch the test go red, put it back — that is how a test that passes by accident is found, and it is cheaper than any tool that does the same thing by rewriting the code.
- `go test -race` always; fixtures under `testdata/`.
- Under the race detector a clock measures the machine rather than the code. So a test that holds code to a clock gives it ten times the time there and leaves every limit that is not time as it is; the factor comes from one helper built with and without the `race` tag, not from a constant in each test. A test about the clock itself — that a call is cut off at its deadline, say — sets its own. A run that outlasts its clock is not retried: that would hide code that really is slow (R207 standalone).
- Recorded responses of Gemini in `testdata/` are synthetic — never a child's sample (spec 17). They are the reference a test is checked against, so no test rewrites them. A snapshot the code itself produces, such as a re-encoded check image, may have `-update`, and then the regenerated diff is part of the review.
- No network in unit tests: Firebase, Gemini and Centrifugo are faked with `httptest`.
- Integration tests run Postgres and OpenFGA in containers, under a build tag of their own (`just ci-integration`, from the MVP epic's task that brings storage).
- Behind an interface stands the real implementation, whenever that implementation does no I/O, and a small hand-written fake where a test has to force it to behave a particular way. That is what proves a property rather than the fact of a call, and it keeps what a test assumes in one readable place.
- A mock earns its place when the real implementation needs the network, a clock, or is simply slow — and when what is being checked is that a call happened, with these arguments. It is then generated by mockery v3 into a `mocks/` folder next to the interface and committed; CI checks it is up to date. Config lives in `.mockery.yaml`, with expecters on, mirroring `mentor-api`, and its list of packages is empty until an interface of that kind exists.

### Tooling and images

- `gofmt -s`, plus `golangci-lint` with a committed `.golangci.yml` and a pinned version. (`mentor-api` has no lint config checked in — this is one of the gaps we close.)
- The SPA in `web/`: Biome formats and lints it (`biome.json`), TypeScript checks its types and compiles nothing, Vitest runs its tests in happy-dom, and Vite builds it, with `@vitejs/plugin-react` refreshing an edited component in place in the dev server (R84 standalone, R47). Firebase Hosting serves the build, so the service's binary and image carry none of it (R10).
- buf lints `proto/`, holds it to backward compatibility against `main`, and generates Go into `gen/` and TypeScript into `web/src/gen/`; the generated code is committed, and CI checks it is current (R12).
- A pre-commit hook in `.githooks/`, enabled by `post-start`; `govulncheck`, `npm audit` and `gitleaks` in CI.
- The justfile is the entry point for everything; `ci-*` recipes are what CI calls — same recipe names as `mentor-api` (`fmt`, `fmt-check`, `build`, `test`, `mocks`, `ci-lint`, `ci-test`, `ci-mocks-check`).
- Multi-stage Dockerfile, minimal non-root runtime image pinned by digest, `CGO_ENABLED=0`, `-trimpath`.
- Services and tools that are not part of the build run from images pinned by digest instead of being installed into the devcontainer: Postgres, OpenFGA, Centrifugo and Redis for local runs and integration tests, the OpenFGA CLI for the model's tests, Playwright for the end-to-end tests, Trivy for the scan of the runtime image, actionlint for the workflows. The SPA's own packages are the exception, because they are part of the build: they install into `web/node_modules` from its lockfile.

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
  - The Dockerfile has two stages. `toolchain` installs Go, Node.js, just, golangci-lint and mockery — what it takes to build and check the code; `devcontainer` adds gopls, dlv, cloudflared, gh, the Docker Compose plugin, Terraform, gcloud, the Firebase CLI and Claude Code on top, and is the stage the container is built from. cloudflared is the tunnel that opens the SPA served from here on a real tablet. gcloud is installed on x86_64 only, because its archive for arm carries no Python interpreter; there the same commands come from Google's own container image (R28 standalone). The Firebase CLI is installed on x86_64 only as well, as the standalone build that carries its own Node.js, which is made for x86_64 alone; on arm its commands run in Cloud Shell (R58).
  - CI runs the full checks in the `toolchain` stage, which it publishes to GHCR under a hash of that stage and runs by digest, so a verdict there and one in here come from the same tools at the same versions (`just ci-toolchain-image`; R20 standalone, R49). On `main` the checks run only in an image that `main` itself signed with cosign; any other image under the tag is built again and signed (`just ci-toolchain-sign`; R53).
  - Everything is installed straight from the Dockerfile, and downloads refuse a protocol downgrade (R35 standalone). What is verified is verified by the source it comes from: the Go checksum database for `go install`, the registry hash for the npm package; the tarballs fetched with `curl` carry no checksum of their own. Of the npm packages only the Claude Code CLI may run its own install step, named explicitly, because that step is what puts its native binary in place.
  - Docker (docker-in-docker, Moby engine and buildx) comes from a devcontainer feature pinned in `devcontainer.json` and `devcontainer-lock.json`.
  - To change a version: edit the literal everywhere it appears (Dockerfile, `devcontainer.json` for Docker or the Claude Code extension), rebuild the container. The Claude Code CLI and its editor extension are one version in two files, so `just claude-update` raises both at once — with a version, or to the newest published one — and prints what it changed. Terraform is pinned twice as well: `ARG TERRAFORM_VERSION` and `required_version` in `infra/terraform/versions.tf`, which refuses any other. Its providers and `.terraform.lock.hcl` are raised by hand too, `terraform providers lock -platform=linux_amd64 -platform=linux_arm64`, since Dependabot would lock them with a Terraform of its own (R56).
- **Container marker.** `MATHTRAIL_DEVCONTAINER=1` is set only inside the container. A session that does not see it is on the host and must stop (see "Devcontainer only").
- **Claude Code.** Its config lives in the `canvas-api-claude` volume, so the login survives rebuilds. Auto-update is off; the version is pinned.
- **`gh`.** What a change looks like on github.com is otherwise invisible from in here, and two parts of it are asked for by name: the alerts of the code scan, which a task closes before the next one starts, and whether the delivery that carried the change ran. `gh auth login` is answered once: the credentials live in the `canvas-api-gh` volume, as gcloud's live in `canvas-api-gcloud`, where Terraform reads them.
- **Ports.** 8080 (the API) and 5173 (Vite's dev server for the SPA) are forwarded to the host.
- **Just recipes.** `just --list` shows all of them.
  - The everyday ones are `just fmt`, `just test`, `just lint`, `just build` and `just run`, and for the SPA `just web-dev`, which serves it on port 5173, `just web-fmt` and `just web-test`, and for the cloud `just tf-fmt` and `just usage`, which sets what the deployed service used of Cloud Run against a month's free tier. `just web-build` builds the SPA into `web/dist`; `just build` and the image leave it out (R10).
  - The ones that must be green before a task is done are **`just ci-lint`** (formatting and lint), **`just ci-test`** (the race detector and coverage), **`just ci-mocks-check`**, **`just ci-licenses`**, **`just ci-vuln`**, **`just ci-secrets`** (gitleaks over the whole history and over what is not committed yet), **`just ci-image-smoke`** (the runtime image starts, and its `/health` names this build), **`just ci-image-scan`** (Trivy over that image, HIGH and CRITICAL with a fix out), **`just release-version`** (the version the next release takes, from `RELEASE_LINE` in `release.yml` and the tags), **`just release-artifacts`** (what a release holds: the server for amd64 and arm64, the SPA's archive and `SHA256SUMS`, in `dist/`), **`just tf-check`** (the Terraform files formatted, the configuration valid against the providers its lock file names, and its tests on mock providers, with no credential) and **`just ci-workflows`** (actionlint, with shellcheck over the scripts of the workflows' steps but not of the composite actions': what GitHub would refuse or misread in the workflows, the call from `release.yml` to `deploy.yml` among it), and they are what CI runs (`.github/workflows/ci.yml`). `ci-licenses` holds every Go module and npm package to the license list and `THIRD_PARTY_LICENSES` to the dependency graph — `just licenses` rewrites it. `ci-vuln` runs govulncheck and `npm audit` (R48). A false alarm of the secret scan is waived in `.gitleaks.toml`, by a rule, a path and the shape of a value together, a vulnerability only in `.trivyignore`, and a finding of actionlint only in `.github/actionlint.yaml`; each with its reason.
  - A change to `web/` must also pass **`just ci-web`**: Biome, the types, the tests with their coverage, and a build.
  - gofmt never walks into `reference/`: the copy is the sample's, read-only.
- **Git hooks.** `.githooks/pre-commit` is enabled by `post-start` through `core.hooksPath`. It runs formatting, a build and the tests — enough to catch what is embarrassing, while the linter, the race detector, `govulncheck` and `gitleaks` wait for CI. Where no Go toolchain is reachable — a Git client outside the container, for instance — it says so and skips those checks instead of blocking the commit: CI runs them again and is what gates a merge.
- **The runtime image.** `just docker-build` builds it and `just docker-run` starts it on port 8080. Both images in the root `Dockerfile` are pinned: a Go builder by tag and digest, at the version `go.mod` names, and a distroless static runtime by digest, since distroless names no versions in its tags; the runtime has no shell and runs as a non-root user. The SPA is no part of it: Firebase Hosting serves the build (R10).
- **Build context.** Both images are built with the repository root as context: the devcontainer copies nothing from it, the runtime image the Go module. `.dockerignore` keeps out of it the history, every `.env` and its variants such as `.env.local`, `.github/`, `.vscode/`, `.sonarlint/`, `.claude/`, `.githooks/`, `.devcontainer/`, `reference/`, `docs/`, `draft/`, `web/`, `infra/`, every `testdata/`, `tools/`, `bin/`, `dist/`, test and profile output, `.scannerwork/`, a local `go.work`, `node_modules`, the Markdown files at the root, `firebase.json` and what the Firebase CLI leaves behind, and the credentials file Google's sign-in action writes into a workflow's checkout.
- **The release image.** A delivery does not build the server again: the `release` stage of the same `Dockerfile` puts the server a release signed into the same runtime, from a build context of its own (`just ci-image-push`, R58).
