# Security policy

MathTrail Canvas is a maths tutor for children in grades 1–6: a child solves a task by hand, an AI reads the written answer, and a program checks it against the task's key. It is built around children, so a fault here is not only a technical one, and a report about it is welcome from anybody who finds one.

## Reporting a vulnerability

Report it privately, through GitHub:

**[Report a vulnerability](https://github.com/MathTrail/canvas-api/security/advisories/new)** — or open the repository's **Security** tab and choose *Report a vulnerability*.

The thread is private until it is resolved, it stays attached to the repository, and an advisory with a CVE can be published from it when the finding warrants one. Please do not open a public issue for something exploitable.

There is no other reporting channel: this project is run by one person, and a private thread here is the only place that will be read.

What helps, in whatever detail you have it: what you did, what happened, what you expected instead, and how an attacker would use it. A proof of concept is welcome and never required.

**What to expect.** This is a free project maintained in spare time, and there is no response-time guarantee. A report that lands will get an acknowledgement, a fix for what is exploitable, and credit in the advisory unless you ask otherwise.

## What is in scope

- The `canvas-api` service and the SPA in `web/`, as they are deployed from this repository: the API, the sign-in it accepts, and the events it publishes to the client.
- Everything in this repository, including the workflows that check, build and deploy it.

Out of scope: the services it is built on — Google Cloud, Firebase, Gemini, Neon and Upstash — the code of Centrifugo and OpenFGA themselves, and anything running on a fork.

## Testing, and what not to do

Research in good faith is welcome, and nobody will be pursued for it. Two limits, and both are about people rather than about us:

- **Only your own accounts and your own data.** No access to anybody else's account, child profile or checks: a profile is a real child's.
- **No load testing, no denial of service, no automated sign-ups.** The service runs inside free tiers and a daily limit on the tutor's help, and flooding it takes the lesson away from a child in the middle of one.

## What the service holds, and what it does not

The service is still being built, in the order [`RUN.md`](RUN.md) sets; this is what it is designed to hold, and every change is held to it. The full register, in Russian, is section 4 of [`docs/privacy.md`](docs/privacy.md).

- **About a child, only a pseudonym, an avatar from a set and the grade.** No name, photo, birth date, school or contacts is asked for or kept. The parent owns the account, and no child's data exists before the parent's consent is verified.
- **The handwriting stays on the device.** The child's strokes live in the browser. On Check the app sends two pictures drawn from them, and the server keeps the pictures only in the memory of that request — never in the database, a log or a trace.
- **The AI receives what is listed, and nothing more.** Gemini gets the answer box's picture to read it and, after a wrong answer, the draft's picture with the task, so that it can frame the mistake. No identifier, no pseudonym and no parent's data goes with them, and Gemini's caching is switched off.
- **No personal data in logs, metrics or traces.** They carry IDs, codes and timings: an ID is a UUID or the parent's Firebase UID, never an email, a name, a pseudonym, a picture, a read answer or the text of a hint.
- **The task's answer stays hidden** until the task is solved: no endpoint returns it before then, and every hint the AI writes is checked for leaking it before the child sees it.
- **No secrets in this repository.** Keys live in Google's Secret Manager and reach the service through its environment. The whole history is scanned for secrets on every pull request, and a key that was ever committed would be treated as leaked and rotated.

## Supported versions

There is one: the newest commit of `main`. There is no maintenance branch — a fix goes to `main`. The delivery is still being set up: once it is, every merge to `main` whose checks are green is published as a release and deployed from it, and the service names the release it runs.

## How this is kept honest

Every change arrives through a pull request that cannot be merged until the following pass:

- formatting and the linter, `gosec` among its checks;
- the tests with the race detector;
- the SPA's formatting, lint, types, tests and build;
- `govulncheck` over the Go code, and `npm audit` over the packages the SPA is built from;
- a licence check of every dependency;
- a secret scan of the whole history;
- the runtime image: that it starts and names its build, and a Trivy scan of it;
- CodeQL with the `security-extended` queries;
- 90% coverage of the lines the change touches.

The reasoning behind each of these is written down in [`docs/decisions.md`](docs/decisions.md), in Russian: R49 and R50.
