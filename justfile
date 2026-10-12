# canvas-api: task runner.
# The ci-* recipes are the full checks, in the names the platform's other services
# use, so that the same command means the same thing in every repository.

set shell := ["bash", "-euo", "pipefail", "-c"]

BINARY := "bin/server"
MODULE := "github.com/MathTrail/canvas-api"

# Exact versions of the tools that only the full checks need. Each is a Go
# program run straight from its module, so none of them is installed anywhere.
GOVULNCHECK := "golang.org/x/vuln/cmd/govulncheck@v1.8.0"
GO_LICENSES := "github.com/google/go-licenses/v2@v2.0.1"

# What a dependency's license may be: permissive, and compatible with releasing
# the result under MIT.
ALLOWED_LICENSES := "MIT,BSD-2-Clause,BSD-3-Clause,Apache-2.0,ISC"
# The npm packages allowed a license outside that list, as name=license. Each is
# a tool that builds the SPA and ends up in nothing that is shipped, and it is
# allowed only while it stays a development dependency. lightningcss, under
# MPL-2.0, is the CSS minifier the SPA's bundler cannot be installed without
# (R48).
NPM_LICENSE_EXCEPTIONS := "lightningcss=MPL-2.0"

# The build identity, stamped into the binary at link time. Computed once per
# run of just, so that a binary and the image built beside it carry the same
# words. Releases are tagged vMAJOR.MINOR.PATCH, and a build is named by them.
VERSION := `git describe --tags --match 'v[0-9]*' --always --dirty 2>/dev/null || echo dev`
COMMIT := `git rev-parse --short HEAD 2>/dev/null || echo unknown`
DATE := `date -u +%Y-%m-%dT%H:%M:%SZ`
SYMBOLS := MODULE + "/internal/version"
LDFLAGS := "-X " + SYMBOLS + ".Version=" + VERSION + " -X " + SYMBOLS + ".Commit=" + COMMIT + " -X " + SYMBOLS + ".Date=" + DATE

# The repository's own Go files, as a NUL-separated list. The copy of the
# sample under reference/ is read-only and not this repository's to format, so
# gofmt never walks into it, nor into installed packages and providers. The
# pre-commit hook keeps the same list in go_files.
GO_FILES := "find . -path ./reference -prune -o -path ./.git -prune -o -name node_modules -prune -o -name .terraform -prune -o -type f -name '*.go' -print0"

# List all recipes
default:
    @just --list

# -- Go ---------------------------------------------------------------------

# Format every Go file in place
fmt:
    {{ GO_FILES }} | xargs -0 -r gofmt -s -w

# Fail if any Go file needs formatting
fmt-check:
    #!/usr/bin/env bash
    set -euo pipefail
    unformatted=$({{ GO_FILES }} | xargs -0 -r gofmt -s -l)
    if [ -n "$unformatted" ]; then
        echo "These files need gofmt -s; run just fmt:" >&2
        echo "$unformatted" >&2
        exit 1
    fi

# Lint
lint:
    golangci-lint run ./...

# Run the tests
test:
    go test ./... -count=1

# The flags carry the version git describes, and the name of a tag may hold $
# and parentheses: quote() hands them to the shell as text it never runs.
# Build the server binary into bin/
build:
    go build -trimpath -ldflags {{ quote(LDFLAGS) }} -o {{ BINARY }} ./cmd/server

# The version is a parameter, because a release knows the number it is about to
# publish before any tag carries it; without one, the same version the rest of
# the build uses. It reaches the script through the environment, never as its
# text: a version asked for is typed by a person, and the name of a tag may hold
# $ and parentheses.
# Build the release artifacts into dist/: the server for every architecture, and the sums of what was built
release-artifacts $version=VERSION:
    #!/usr/bin/env bash
    set -euo pipefail
    # A release's tag, how far past one a build is, or a bare commit: letters,
    # digits, dots and dashes, with nothing that would split the linker's flags
    # or turn the name of a file into a directory.
    if [[ ! "$version" =~ ^[0-9A-Za-z][0-9A-Za-z.-]*$ ]]; then
        echo "release-artifacts: '${version}' is not a version" >&2
        exit 1
    fi
    ldflags="-s -w \
        -X {{ SYMBOLS }}.Version=${version} \
        -X {{ SYMBOLS }}.Commit={{ COMMIT }} \
        -X {{ SYMBOLS }}.Date={{ DATE }}"
    rm -rf dist
    mkdir -p dist
    for arch in amd64 arm64; do
        name="canvas-api_${version}_linux_${arch}"
        CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build \
            -trimpath -ldflags "$ldflags" -o "dist/$name" ./cmd/server
    done
    cd dist
    sha256sum * > SHA256SUMS
    echo "dist: $(ls | wc -l) files"

# Run the server from source on port 8080
run:
    go run ./cmd/server

# Generate the mocks (mockery, config in .mockery.yaml)
mocks:
    #!/usr/bin/env bash
    set -euo pipefail
    if grep -qE '^packages: *\{\}' .mockery.yaml; then
        echo "mocks: no interfaces are configured yet; nothing to generate."
        exit 0
    fi
    mockery

# Rewrite THIRD_PARTY_LICENSES from the dependency graph
licenses:
    #!/usr/bin/env bash
    set -euo pipefail
    list=$({{ quote(just_executable()) }} _license-list)
    # The file is replaced only once the whole list is in hand: a report that
    # died halfway would otherwise leave the repository claiming fewer
    # dependencies than it ships.
    printf '%s\n' "$list" > THIRD_PARTY_LICENSES
    echo "THIRD_PARTY_LICENSES: $(grep -c 'https://' THIRD_PARTY_LICENSES) entries."

# The license list as the dependency graphs report it right now
_license-list:
    #!/usr/bin/env bash
    set -euo pipefail
    cat <<'HEADER'
    Third-party licenses
    ====================

    canvas-api itself is MIT; see LICENSE. Rewrite this file with: just licenses

    The Go modules below are linked into the programs this repository builds,
    and so are the standard library and the runtime of the Go release that
    builds them, listed as std. Each line is a license, what it covers, and the
    text of that license at the exact version that is built.

    HEADER
    # The release go.mod names is the one the releases are built with, and it
    # names its patch too, as every version here does: every Go release since
    # 1.21 is tagged with its patch, so a link without one would lead nowhere.
    release=$(awk '$1 == "go" { print $2; exit }' go.mod)
    if [[ ! "$release" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "go.mod names Go ${release}; the license list needs the exact release" >&2
        exit 1
    fi
    # The report goes to stdout and the progress of the scan to stderr, which is
    # noise here unless the scan fails: then it is the only word on why. The
    # columns are module, license text, license. No module graph lists the
    # standard library and the runtime, which every binary carries, so they are
    # put at the head of the report as one more module.
    scan=$(mktemp)
    trap 'rm -f "$scan"' EXIT
    if ! report=$(go run {{ GO_LICENSES }} report ./... --ignore {{ MODULE }} 2>"$scan"); then
        cat "$scan" >&2
        echo "go-licenses could not report the Go modules" >&2
        exit 1
    fi
    {
        echo "std,https://github.com/golang/go/blob/go${release}/LICENSE,BSD-3-Clause"
        if [ -n "$report" ]; then printf '%s\n' "$report"; fi
    } | awk -F, '{ printf "%-13s %-46s %s\n", $3, $1, $2 }'
    cat <<'WEB'

    The npm packages below are what the SPA is built from: every package its
    lockfile needs outside development, and every package one of those names
    as an optional companion, used or not. Firebase Hosting serves the SPA, so
    neither the server's binary nor its image carries any of them (R10). Each
    line is a license, the package at the exact version in
    web/package-lock.json, and the page of that version in the npm registry.

    WEB
    node web/scripts/licenses.ts list

# -- SPA --------------------------------------------------------------------

# Nothing is installed while the installed packages are newer than both
# package.json and the lockfile, so that a build does not pay for an install it
# does not need. A package.json changed since the install is a reason too:
# `npm ci` then refuses a lockfile that no longer matches it, as it would in CI,
# instead of the checks passing on what was installed before.
# Install the SPA's packages as its lockfile pins them, when they are missing or stale
[working-directory('web')]
web-install:
    #!/usr/bin/env bash
    set -euo pipefail
    installed=node_modules/.package-lock.json
    if [ ! "$installed" -nt package-lock.json ] || [ ! "$installed" -nt package.json ]; then
        npm ci --no-audit --no-fund
    fi

# The page names the version it is built as, which reaches the build through
# the environment rather than as the text of a command, since a version asked
# for is typed by a person. The last build is removed first: Vite empties
# web/dist only once it starts writing, so a build that failed before then
# would leave the last one in place, to be shipped as this one.
# Build the SPA into web/dist, naming the version given or this tree's
[working-directory('web')]
web-build $version=VERSION: web-install
    rm -rf dist
    VITE_VERSION="$version" npm run --silent build

# Format the SPA's sources in place
[working-directory('web')]
web-fmt: web-install
    npm run --silent fmt

# Fail if a source of the SPA needs formatting or breaks a lint rule
[working-directory('web')]
web-lint: web-install
    npm run --silent check

# Run the SPA's tests
[working-directory('web')]
web-test: web-install
    npm run --silent test

# Vite's dev server, on the port the editor forwards from the container. An
# edited component is refreshed in place, without reloading the page (R47).
# The version reaches the page as it does in web-build.
# Serve the SPA on port 5173 while it is worked on, naming the version given or this tree's
[working-directory('web')]
web-dev $version=VERSION: web-install
    VITE_VERSION="$version" npm run --silent dev

# -- The full checks --------------------------------------------------------

# Formatting and lint
ci-lint: fmt-check
    golangci-lint run ./...

# Tests with the race detector and a coverage report
ci-test:
    go test ./... -race -count=1 -coverprofile=coverage.out -covermode=atomic
    go tool cover -func=coverage.out | tail -1

# The SPA's formatting and lint, its types, its tests with their coverage, and a build
[working-directory('web')]
ci-web: web-build
    npm run --silent check
    npm run --silent typecheck
    npm run --silent test -- --coverage

# Fail if the committed mocks are not what mockery generates
ci-mocks-check:
    #!/usr/bin/env bash
    set -euo pipefail
    if grep -qE '^packages: *\{\}' .mockery.yaml; then
        echo "ci-mocks-check: no interfaces are configured yet; nothing to compare."
        exit 0
    fi
    mockery
    # --porcelain rather than diff: a mock generated for the first time is not
    # tracked yet, and a diff cannot see a file git has never heard of.
    changed=$(git status --porcelain -- "*mocks*")
    if [ -n "$changed" ]; then
        echo "Mocks are outdated: run just mocks and commit the result." >&2
        echo "$changed" >&2
        exit 1
    fi

# govulncheck reports only what the code can reach. npm has no such analysis, so
# every package the SPA is built from is held to the advisories, at high and
# above; the tools that only build it are not, since none of them is shipped.
# Neither reads more than the code and the lockfile, so nothing is installed.
# Fail on a known vulnerability in code the service reaches or a package the SPA is built from
ci-vuln:
    go run {{ GOVULNCHECK }} ./...
    npm audit --prefix web --omit=dev --audit-level=high

# Fail if a dependency carries a license we may not ship, or the list is stale
ci-licenses:
    #!/usr/bin/env bash
    set -euo pipefail
    go run {{ GO_LICENSES }} check ./... --allowed_licenses={{ ALLOWED_LICENSES }}
    node web/scripts/licenses.ts check --allowed {{ ALLOWED_LICENSES }} --except {{ NPM_LICENSE_EXCEPTIONS }}
    # The list is built first and compared second: a report that failed to run
    # would otherwise look exactly like a list somebody forgot to update.
    list=$({{ quote(just_executable()) }} _license-list)
    if ! printf '%s\n' "$list" | diff -u THIRD_PARTY_LICENSES - ; then
        echo "THIRD_PARTY_LICENSES is out of date: run just licenses and commit the result." >&2
        exit 1
    fi

# -- Container --------------------------------------------------------------

# The version stays exact — this only removes the part where a person edits the
# same number in two files and misses one. Without an argument, whatever npm
# calls latest today; the change is printed and committed like any other.
# Raise the pinned version of the Claude Code CLI and its editor extension
claude-update $version="":
    #!/usr/bin/env bash
    set -euo pipefail

    dockerfile=".devcontainer/Dockerfile"
    devcontainer=".devcontainer/devcontainer.json"

    # The argument arrives through the environment rather than pasted into this
    # script, so that the check below sees it before the shell does anything with it.
    wanted="$version"
    if [ -z "$wanted" ]; then
        wanted=$(curl -fsSL --proto "=https" --proto-redir "=https" \
            https://registry.npmjs.org/@anthropic-ai/claude-code/latest \
        | sed -n 's/.*"version":"\([^"]*\)".*/\1/p')
    fi
    if [[ ! "$wanted" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "claude-update: $wanted is not a version" >&2
        exit 1
    fi

    current=$(sed -n 's/^ARG CLAUDE_CODE_VERSION=//p' "$dockerfile")
    # Both pins must be where the edits below look for them: otherwise one file
    # would move and the other stay behind, with nothing to say so.
    if [ -z "$current" ] || ! grep -q '"anthropic.claude-code@[^"]*"' "$devcontainer"; then
        echo "claude-update: expected ARG CLAUDE_CODE_VERSION= in $dockerfile and anthropic.claude-code@ in $devcontainer" >&2
        exit 1
    fi
    if [ "$current" = "$wanted" ]; then
        echo "claude code $current: already pinned"
        exit 0
    fi

    # The CLI and the extension are one version in two files, and a rebuild with
    # them apart is the failure this recipe exists to prevent.
    sed -i "s/^ARG CLAUDE_CODE_VERSION=.*/ARG CLAUDE_CODE_VERSION=${wanted}/" "$dockerfile"
    sed -i "s/\"anthropic.claude-code@[^\"]*\"/\"anthropic.claude-code@${wanted}\"/" "$devcontainer"

    echo "claude code $current -> $wanted"
    git --no-pager diff -- "$dockerfile" "$devcontainer"
    echo
    echo "Rebuild the container for this to take effect: Dev Containers: Rebuild Container."

# The build identity and the image's tag are quoted for the shell, as in
# build: the one comes from git, the other from whoever types it.
# Build the runtime image
docker-build tag="canvas-api:dev":
    docker build \
        --build-arg VERSION={{ quote(VERSION) }} \
        --build-arg COMMIT={{ quote(COMMIT) }} \
        --build-arg DATE={{ quote(DATE) }} \
        -t {{ quote(tag) }} .

# Build the image and run it on port 8080
docker-run tag="canvas-api:dev": (docker-build tag)
    docker run --rm -e PORT=8080 -p 8080:8080 {{ quote(tag) }}
