# canvas-api: task runner.
# The ci-* recipes are the full checks, in the names the platform's other services
# use, so that the same command means the same thing in every repository.

set shell := ["bash", "-euo", "pipefail", "-c"]

BINARY := "bin/server"
MODULE := "github.com/MathTrail/canvas-api"

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

# Build the server binary into bin/
build:
    go build -trimpath -ldflags "{{ LDFLAGS }}" -o {{ BINARY }} ./cmd/server

# The version is a parameter, because a release knows the number it is about to
# publish before any tag carries it; without one, the same version the rest of
# the build uses.
# Build the release artifacts into dist/: the server for every architecture, and the sums of what was built
release-artifacts version=VERSION:
    #!/usr/bin/env bash
    set -euo pipefail
    ldflags="-s -w \
        -X {{ SYMBOLS }}.Version={{ version }} \
        -X {{ SYMBOLS }}.Commit={{ COMMIT }} \
        -X {{ SYMBOLS }}.Date={{ DATE }}"
    rm -rf dist
    mkdir -p dist
    for arch in amd64 arm64; do
        name="canvas-api_{{ version }}_linux_${arch}"
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

# -- The full checks --------------------------------------------------------

# Formatting and lint
ci-lint: fmt-check
    golangci-lint run ./...

# Tests with the race detector and a coverage report
ci-test:
    go test ./... -race -count=1 -coverprofile=coverage.out -covermode=atomic
    go tool cover -func=coverage.out | tail -1

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

# Build the runtime image
docker-build tag="canvas-api:dev":
    docker build \
        --build-arg VERSION="{{ VERSION }}" \
        --build-arg COMMIT="{{ COMMIT }}" \
        --build-arg DATE="{{ DATE }}" \
        -t {{ tag }} .

# Build the image and run it on port 8080
docker-run tag="canvas-api:dev": (docker-build tag)
    docker run --rm -e PORT=8080 -p 8080:8080 {{ tag }}
