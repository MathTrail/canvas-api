# canvas-api: task runner.
# The ci-* recipes are the full checks, in the names the platform's other services
# use, so that the same command means the same thing in every repository.

set shell := ["bash", "-euo", "pipefail", "-c"]

BINARY := "bin/server"
MODULE := "github.com/MathTrail/canvas-api"

# The image the full checks run inside, published by CI. Lowercase throughout:
# a registry rejects a repository name that is not.
TOOLCHAIN_IMAGE := "ghcr.io/mathtrail/canvas-api/toolchain"
# Who signs that image: the workflow of the checks, run for a commit of main. A
# run for a pull request carries the pull request's ref in its certificate, and
# no run of a fork is this repository's at all, so neither signs as main. The
# issuer is GitHub's, whose short-lived tokens the certificate is given for.
TOOLCHAIN_SIGNER := "https://" + MODULE + "/.github/workflows/ci.yml@refs/heads/main"
GITHUB_ISSUER := "https://token.actions.githubusercontent.com"
# Who signs a release: the release workflow, run for a commit of main.
# A delivery rolls out nothing it did not sign.
RELEASE_SIGNER := "https://" + MODULE + "/.github/workflows/release.yml@refs/heads/main"
# The repository on GitHub, as gh names it.
REPOSITORY := trim_start_match(MODULE, "github.com/")
# The definition of the checks' environment: a hash of the toolchain stage of
# the development image — every line above the stage built on top of it. It is
# the image's tag, and main's signature names it too, so that a signature of an
# image built from another definition does not vouch for this one.
TOOLCHAIN_HASH := `sed '/^FROM toolchain AS devcontainer$/,$d' .devcontainer/Dockerfile 2>/dev/null | sha256sum | cut -c1-16`

# Exact versions of the tools that only the full checks need. Each is a Go
# program run straight from its module, so none of them is installed anywhere.
GOVULNCHECK := "golang.org/x/vuln/cmd/govulncheck@v1.8.0"
GITLEAKS := "github.com/zricethezav/gitleaks/v8@v8.30.1"
GO_LICENSES := "github.com/google/go-licenses/v2@v2.0.1"

# The image the runtime image is scanned with, pinned by tag and digest like
# every other image this repository runs.
TRIVY_IMAGE := "aquasec/trivy:0.75.0@sha256:af6acf9a6b85dfe389a1941505c0ce9efef52a4719635e1a962f022a3d855daa"
# The image the workflows are checked with. It carries the shellcheck that
# actionlint runs on the script of every step of a workflow.
ACTIONLINT_IMAGE := "rhysd/actionlint:1.7.12@sha256:b1934ee5f1c509618f2508e6eb47ee0d3520686341fec936f3b79331f9315667"
# The image a release's signature is checked with: the cosign the release signs
# with, run as SECURITY.md runs it.
COSIGN_IMAGE := "ghcr.io/sigstore/cosign/cosign:v3.1.3@sha256:9e5c2f2edc34351160407ca3416c61855bdf9403c3c5936e0f0be7fc261611b8"

# What a release's tag looks like: v and three numbers as SemVer writes them,
# digits alone with no leading zero. A release is numbered so, and a delivery
# takes nothing else for one.
RELEASE_TAG := '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'

# What the health of a build answers when it is the build named: ok, the version
# exactly, and a commit that begins the full one given — a build stamps git's
# short form of it, as long as it has to be to name one commit.
HEALTH_MATCHES := '.status == "ok" and .version == $version and (.commit | type == "string" and test("^[0-9a-f]{7,40}$")) and (.commit as $short | $commit | startswith($short))'

# The deployment, as Terraform describes it, and the filter what Terraform
# prints about it passes through (_tf-redact, below).
TF_DIR := "infra/terraform"
TF_REDACT := quote(just_executable()) + " --justfile " + quote(justfile()) + " _tf-redact"

# What a dependency's license may be: permissive, and compatible with releasing
# the result under MIT.
ALLOWED_LICENSES := "MIT,BSD-2-Clause,BSD-3-Clause,Apache-2.0,ISC"
# The npm packages allowed a license outside that list, as name=license. Each is
# a tool that builds the SPA and ends up in nothing that is shipped, and it is
# allowed only while it stays a development dependency. lightningcss, under
# MPL-2.0, is the CSS minifier the SPA's bundler cannot be installed without.
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

# A version asked for is the one published. Otherwise it is the next patch of
# the line the releases are numbered in, MAJOR.MINOR, or the line's first when
# none of it is out yet; the line is the one release.yml names, unless another
# is given. Refused: a version that is not vMAJOR.MINOR.PATCH, one already
# released, and one that would come out below the newest release — from a line
# older than it, or asked for so — since each would number the releases out of
# order or twice.
#
# A commit is released once. When it carries a tag of the form a release takes,
# v*, nothing is printed and nothing is published, so a button pressed again,
# or the checks of a released commit run again, hang no second tag on it. The
# arguments reach the script through the environment, never as its text, since
# a version asked for is typed by a person.
# Print the version the next release is published as, or nothing when this commit is released already
release-version $line="" $requested="":
    #!/usr/bin/env bash
    set -euo pipefail
    if [ -z "$line" ]; then
        line=$(sed -n 's/^  RELEASE_LINE: "\(.*\)"$/\1/p' .github/workflows/release.yml)
    fi
    # A number as SemVer writes it: digits alone, with no leading zero.
    number='(0|[1-9][0-9]*)'
    if [[ ! "$line" =~ ^${number}\.${number}$ ]]; then
        echo "release-version: the line '${line}' is not MAJOR.MINOR" >&2
        exit 1
    fi
    released=$(git tag --points-at HEAD --list 'v*' | head -1)
    if [ -n "$released" ]; then
        echo "release-version: this commit is released already, as ${released}" >&2
        exit 0
    fi
    # The tags a release makes, oldest first. Any other tag that starts with a
    # v is no release, and counts for neither the newest nor the line's last.
    release={{ quote(RELEASE_TAG) }}
    releases() { grep -E "$release" | sort -V || true; }
    newest=$(git tag --list 'v*' | releases | tail -1)
    if [ -n "$requested" ]; then
        version="$requested"
    else
        last=$(git tag --list "v${line}.*" | releases | tail -1)
        if [ -z "$last" ]; then
            version="v${line}.0"
        else
            version="v${line}.$((${last##*.} + 1))"
        fi
    fi
    if [[ ! "$version" =~ $release ]]; then
        echo "release-version: '${version}' is not vMAJOR.MINOR.PATCH" >&2
        exit 1
    fi
    if git rev-parse -q --verify "refs/tags/${version}" > /dev/null; then
        echo "release-version: ${version} is released already" >&2
        exit 1
    fi
    if [ -n "$newest" ] && [ "$(printf '%s\n' "${newest#v}" "${version#v}" | sort -V | tail -1)" != "${version#v}" ]; then
        if [ -n "$requested" ]; then
            echo "release-version: ${version} would come out below the newest release, ${newest}" >&2
        else
            echo "release-version: the line ${line} is older than the newest release, ${newest}; raise RELEASE_LINE in release.yml" >&2
        fi
        exit 1
    fi
    echo "$version"

# The version is a parameter, because a release knows the number it is about to
# publish before any tag carries it; without one, the same version the rest of
# the build uses. It reaches the script through the environment, never as its
# text: a version asked for is typed by a person, and the name of a tag may hold
# $ and parentheses. The SPA is built first, naming the same version: Firebase
# Hosting serves it, and no binary carries it.
# Build the release artifacts into dist/: the server for every architecture, the SPA as Hosting serves it, and the sums of what was built
release-artifacts $version=VERSION: (web-build version)
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
        name="canvas-api-server_${version}_linux_${arch}"
        CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build \
            -trimpath -ldflags "$ldflags" -o "dist/$name" ./cmd/server
    done
    # The same files always make the same archive, byte for byte: in the order
    # of their names, with no owner, no time, no name of its own and modes
    # that do not depend on the umask of whoever built them, so that anybody
    # who builds the SPA again can compare the sum.
    tar --sort=name --mtime=@0 --owner=0 --group=0 --numeric-owner \
        --mode='u+rwX,go+rX,go-w' --format=gnu \
        -C web/dist -cf - . | gzip -n > "dist/canvas-web_${version}.tar.gz"
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
    neither the server's binary nor its image carries any of them. Each
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
# edited component is refreshed in place, without reloading the page.
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

# The scan is over the whole history: a key that was committed and then removed
# is still published. A history git declines to read would be scanned as no
# commits at all and reported clean, the one outcome worse than a failure, and a
# shallow clone holds only its last commits; either one refuses to start. Each
# finding is shown where it is — rule, file, line, and the commit with its
# author, as git log shows them — and with its value hidden: the log of a check
# is public, and must not spread a secret any further.
#
# What is not committed yet is no history, and a change checked before it is
# committed would otherwise be scanned only once it is published. So it is
# scanned too, as one commit more on top of HEAD, made in an index and an
# object store of its own that read the repository's and never write to it:
# the repository stays exactly as it was, and what git ignores, such as a local
# .env, stays out, as it would stay out of a commit.
# Fail if anything in the history, or in what is not committed yet, looks like a secret
ci-secrets:
    #!/usr/bin/env bash
    set -euo pipefail
    git rev-parse --verify HEAD > /dev/null
    if [ "$(git rev-parse --is-shallow-repository)" != false ]; then
        echo "ci-secrets: this clone is shallow, and the scan needs the whole history: git fetch --unshallow" >&2
        exit 1
    fi
    go run {{ GITLEAKS }} git --no-banner --redact --verbose .
    if [ -z "$(git status --porcelain)" ]; then
        exit 0
    fi
    scratch=$(mktemp -d)
    trap 'rm -rf "$scratch"' EXIT
    mkdir "$scratch/objects"
    cp "$(git rev-parse --git-path index)" "$scratch/index"
    GIT_ALTERNATE_OBJECT_DIRECTORIES=$(git rev-parse --path-format=absolute --git-path objects)
    export GIT_ALTERNATE_OBJECT_DIRECTORIES
    export GIT_INDEX_FILE="$scratch/index" GIT_OBJECT_DIRECTORY="$scratch/objects"
    git add --all
    tree=$(git write-tree)
    pending=$(GIT_AUTHOR_NAME=ci-secrets GIT_AUTHOR_EMAIL=ci-secrets@invalid \
        GIT_COMMITTER_NAME=ci-secrets GIT_COMMITTER_EMAIL=ci-secrets@invalid \
        git commit-tree "$tree" -p HEAD -m "What is not committed yet")
    echo "ci-secrets: and what is not committed yet, as $pending" >&2
    go run {{ GITLEAKS }} git --no-banner --redact --verbose --log-opts="-1 $pending" .

# An image that lost its version on the way would roll out as a revision nobody
# can tell from the one before, so the image built here has to name the build
# it was made from.
# Fail if the image does not start, or its health does not name this build
ci-image-smoke tag="canvas-api:dev": (docker-build tag)
    #!/usr/bin/env bash
    set -euo pipefail
    # The build identity reaches the script as quoted text, as in docker-build,
    # and it reads it only from these variables.
    version={{ quote(VERSION) }}
    commit={{ quote(COMMIT) }}
    # Where git cannot tell the build, it is named dev at unknown — which is
    # also what a binary that lost its version says, so the answer would prove
    # nothing.
    if [ "$version" = dev ] || [ "$commit" = unknown ]; then
        echo "ci-image-smoke: the build is named ${version} at ${commit}, as a binary that lost its version names itself, so its answer would prove nothing; run the check in a clone, with its history" >&2
        exit 1
    fi
    {{ quote(just_executable()) }} --justfile {{ quote(justfile()) }} _image-health {{ quote(tag) }} "$version" "$(git rev-parse HEAD)"

# The image is started as Cloud Run starts it, with PORT and nothing else, and
# asked for its health as the platform's probe asks. It has to answer, and to
# name the build given: the checks hold the image they build to it, and a
# delivery the image of a release before it pushes it.
_image-health tag version commit:
    #!/usr/bin/env bash
    set -euo pipefail
    tag={{ quote(tag) }}
    version={{ quote(version) }}
    commit={{ quote(commit) }}
    # Created first and started second, so that the container is removed when
    # the check ends, whether it never started, stopped at once — its logs are
    # still there to show — or is serving still.
    container=$(docker create --publish 127.0.0.1::8080 --env PORT=8080 "$tag")
    trap 'docker rm --force "$container" > /dev/null' EXIT
    if ! docker start "$container" > /dev/null; then
        echo "image health: ${tag} did not start" >&2
        exit 1
    fi
    answered=false
    for attempt in $(seq 1 20); do
        # A container that has stopped publishes no port.
        address=$(docker port "$container" 8080/tcp 2> /dev/null | head -1) || true
        if [ -n "$address" ] && body=$(curl -fsS --max-time 2 "http://${address}/health" 2> /dev/null); then
            answered=true
            break
        fi
        echo "health: no answer yet (attempt ${attempt})" >&2
        sleep 0.5
    done
    if [ "$answered" != true ]; then
        echo "image health: ${tag} never answered its health check" >&2
        docker logs "$container" >&2
        exit 1
    fi
    if ! jq --exit-status --arg version "$version" --arg commit "$commit" \
        {{ quote(HEALTH_MATCHES) }} <<< "$body" > /dev/null 2>&1; then
        echo "image health: ${tag} answers ${body}; want status ok, version ${version}, and a commit that begins ${commit}" >&2
        exit 1
    fi
    echo "image health: ${tag} answers as ${version} at $(jq -r .commit <<< "$body")"

# Trivy runs from its own image, so nothing about it is installed here and the
# verdict is the same on a laptop and in CI. It reads a saved copy of the image
# rather than Docker itself: handed Docker's socket, the scanner would hold the
# whole machine. A finding is waived only in .trivyignore, with its reason, and
# the file goes in beside the copy; one in Go's standard library is not waived
# but fixed by a newer Go.
# Fail on a known HIGH or CRITICAL vulnerability, with a fix out, in the runtime image
ci-image-scan tag="canvas-api:dev": (docker-build tag)
    #!/usr/bin/env bash
    set -euo pipefail
    scan=$(mktemp -d)
    trap 'rm -rf "$scan"' EXIT
    docker save --output "$scan/image.tar" {{ quote(tag) }}
    if [ -f .trivyignore ]; then
        cp .trivyignore "$scan/"
    fi
    docker run --rm --volume "$scan:/scan:ro" --workdir /scan \
        {{ TRIVY_IMAGE }} image \
        --input image.tar \
        --severity HIGH,CRITICAL \
        --ignore-unfixed \
        --exit-code 1 \
        --no-progress

# actionlint runs from its own image as well. It reads the workflows the way
# GitHub will: a key GitHub does not know, an expression of the wrong type, a
# job that needs one that does not exist, an input or a secret that a calling
# workflow passes and the called one does not declare. GitHub itself checks a
# call only when it is made, and release.yml calls deploy.yml on main alone, so
# without this a broken call would surface after the merge, as a commit left
# without its release. The scripts of the composite actions under
# .github/actions are not read: actionlint takes from an action only the inputs
# it declares. A finding is waived only in .github/actionlint.yaml, with its
# reason.
#
# The image is handed a copy of .github alone, with an empty .git beside it,
# which is how actionlint knows the root of a repository, and no network: a tool
# from somebody else's image has no business with the rest of the tree, a local
# .env among it. The image runs as a user of its own, so the copy is readable
# by anyone.
# Fail on anything GitHub would refuse or misread in the workflows
ci-workflows:
    #!/usr/bin/env bash
    set -euo pipefail
    lint=$(mktemp -d)
    trap 'rm -rf "$lint"' EXIT
    cp -R .github "$lint/"
    mkdir "$lint/.git"
    chmod -R a+rX "$lint"
    docker run --rm --network none --volume "$lint:/repo:ro" --workdir /repo \
        {{ ACTIONLINT_IMAGE }} -no-color

# -- Container --------------------------------------------------------------

# The tag is a hash of the toolchain stage of the development image — every
# line above the stage built on top of it — so one definition of the checks'
# environment is one image, and any edit to it a new one, while an edit to the
# stage on top, a new Claude Code say, leaves it as it was. Nothing is built
# when that tag is published already. What is printed is the digest, not the
# tag: a tag can be moved, and a version here is exact.
#
# Whoever may publish the package may move the tag, and every run of a pull
# request of this repository may. So a pull request takes the image the
# tag names — accept=published — while main takes only one it signed itself —
# accept=signed — and builds any other again. An image built here is named by
# the digest its own push reports, never by what the tag names afterwards: a run
# of a pull request may move the tag in between.
# Publish the image the full checks run inside, and print its exact reference
ci-toolchain-image $accept="published":
    #!/usr/bin/env bash
    set -euo pipefail
    case "$accept" in
        published) ;;
        signed)
            # Without cosign every image would read as unsigned, and each run
            # would build one again for nothing.
            if ! command -v cosign > /dev/null; then
                echo "ci-toolchain-image: accept=signed checks the signature with cosign, and there is no cosign here" >&2
                exit 1
            fi
            ;;
        *)
            echo "ci-toolchain-image: accept is published or signed, not '${accept}'" >&2
            exit 1
            ;;
    esac
    reference="{{ TOOLCHAIN_IMAGE }}:{{ TOOLCHAIN_HASH }}"
    # Read with awk rather than a Go template, whose braces would collide with
    # the interpolation syntax of this file. A tag that names nothing yet reads
    # as no digest at all.
    digest=$(docker buildx imagetools inspect "$reference" 2> /dev/null | awk '/^Digest:/ { print $2 }' || true)
    if [ -z "$digest" ]; then
        echo "This environment is not published yet; building it." >&2
    elif [ "$accept" = signed ] && ! cosign verify \
        --certificate-identity {{ quote(TOOLCHAIN_SIGNER) }} \
        --certificate-oidc-issuer {{ quote(GITHUB_ISSUER) }} \
        --annotations toolchain={{ TOOLCHAIN_HASH }} \
        "{{ TOOLCHAIN_IMAGE }}@${digest}" > /dev/null; then
        echo "ci-toolchain-image: main did not sign the image ${reference} names as this definition's, as cosign says above; building it again." >&2
        digest=""
    fi
    if [ -z "$digest" ]; then
        # Built with no context at all: the stage copies nothing in, and its
        # tag hashes its text alone, so a COPY added to it fails here instead
        # of building what the tag does not name. The label ties the package
        # to the repository it is built from.
        docker build --target toolchain \
            --label "org.opencontainers.image.source=https://{{ MODULE }}" \
            --tag "$reference" - < .devcontainer/Dockerfile >&2
        if ! pushed=$(docker push "$reference" | tee /dev/stderr); then
            echo "ci-toolchain-image: could not publish $reference; docker push says why above. A pull request from a fork cannot publish at all, so there a change to the toolchain stage reaches main from a branch of this repository first." >&2
            exit 1
        fi
        digest=$(sed -n 's/^.*: digest: \(sha256:[0-9a-f]\{64\}\) size: [0-9]*$/\1/p' <<< "$pushed" | tail -1)
        if [ -z "$digest" ]; then
            echo "ci-toolchain-image: docker push named no digest for $reference" >&2
            exit 1
        fi
    fi
    # The one thing on stdout, so that a caller can read it with a substitution.
    echo "{{ TOOLCHAIN_IMAGE }}@${digest}"

# Signed by its digest, since a tag can be moved between a check and a
# signature, with the definition it was built from, and only once: a second signature of the same image would only
# grow the list a check reads. The signature goes into the package beside the
# image, and its certificate into Sigstore's public log.
# Sign the toolchain image, named by its digest, as main's, unless main signed it already
ci-toolchain-sign image:
    #!/usr/bin/env bash
    set -euo pipefail
    image={{ quote(image) }}
    digest="${image#"{{ TOOLCHAIN_IMAGE }}@"}"
    if [ "$digest" = "$image" ] || [[ ! "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
        echo "ci-toolchain-sign: '${image}' is not {{ TOOLCHAIN_IMAGE }} named by its digest" >&2
        exit 1
    fi
    if cosign verify \
        --certificate-identity {{ quote(TOOLCHAIN_SIGNER) }} \
        --certificate-oidc-issuer {{ quote(GITHUB_ISSUER) }} \
        --annotations toolchain={{ TOOLCHAIN_HASH }} \
        "$image" > /dev/null 2>&1; then
        echo "ci-toolchain-sign: main has signed ${image} already"
        exit 0
    fi
    cosign sign --yes --annotations toolchain={{ TOOLCHAIN_HASH }} "$image"

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

# -- Infrastructure ---------------------------------------------------------

# Rewrite every Terraform file in the canonical format
tf-fmt:
    terraform -chdir={{ TF_DIR }} fmt -recursive

# The check goes near no deployment: there is no backend and no credential, and
# the tests run against mock providers, so nothing is read or made in a cloud.
# The providers come exactly as the lock file names them, so one missing from
# it fails here instead of being added on the way.
# Fail if a Terraform file is not formatted, the configuration does not validate, or its tests fail
tf-check:
    #!/usr/bin/env bash
    set -euo pipefail
    terraform -chdir={{ TF_DIR }} fmt -check -recursive -diff
    terraform -chdir={{ TF_DIR }} init -backend=false -input=false -lockfile=readonly
    terraform -chdir={{ TF_DIR }} validate
    terraform -chdir={{ TF_DIR }} test

# Run once, with TF_VAR_billing_account set, by someone who may create projects;
# again only when infra/bootstrap.sh changes (docs/self-hosting.md).
# Create the project, its state bucket and the identity the pipeline applies as
bootstrap:
    bash infra/bootstrap.sh

# A plan only reads, so it takes no lock: a pull request's plan never stands in
# the way of a delivery's apply. What it prints goes through _tf-redact.
# Show what an apply would change in the deployment (needs credentials and TF_VAR_billing_account)
ci-tf-plan: _tf-init
    terraform -chdir={{ TF_DIR }} plan -input=false -no-color -lock=false 2>&1 | {{ TF_REDACT }}

# An apply waits for a lock that another apply holds rather than failing at
# once. A secret that a revision reads gets its pass before this one, so that
# the revision finds a version of it: there is no secret yet.
# Apply the deployment's configuration (needs credentials and TF_VAR_billing_account)
ci-tf-apply: _tf-init
    terraform -chdir={{ TF_DIR }} apply -auto-approve -input=false -no-color -lock-timeout=5m 2>&1 | {{ TF_REDACT }}

# Read after ci-tf-apply, in the same place, which has initialised the
# configuration already. Each value is read into a variable first: inside the
# echo, an output that failed would print an empty line and pass.
# Print what a delivery reads from the deployment, as name=value lines
ci-tf-outputs:
    #!/usr/bin/env bash
    set -euo pipefail
    for name in project_id deploy_service_account hosting_service_account image_repository service_name region service_url hosting_url; do
        value="$(terraform -chdir={{ TF_DIR }} output -raw "$name")"
        echo "${name}=${value}"
    done

# The plan and the apply initialise the configuration the same way: against
# the deployment's state, with the providers exactly as the lock file names
# them.
_tf-init:
    terraform -chdir={{ TF_DIR }} init -backend-config=backend.hcl -input=false -lockfile=readonly 2>&1 | {{ TF_REDACT }}

# A plan and an apply print what the state holds, and the budget's id holds the
# billing account. GitHub masks a repository's secrets in what a workflow
# prints, but nothing masks a terminal or a log copied out of one, so the
# billing account and the operator's address are replaced by their names
# before any line is shown. Both are replaced as text, never read as patterns.
_tf-redact:
    #!/usr/bin/env bash
    set -euo pipefail
    billing="${TF_VAR_billing_account:-}"
    email="${TF_VAR_operator_email:-}"
    while IFS= read -r line || [ -n "$line" ]; do
        if [ -n "$billing" ]; then line="${line//"$billing"/"<billing account>"}"; fi
        if [ -n "$email" ]; then line="${line//"$email"/"<operator email>"}"; fi
        printf '%s\n' "$line"
    done

# -- Delivery ---------------------------------------------------------------

# A delivery rolls out a release, named by its tag: the server and the SPA that
# release built and signed, with the configuration — Terraform, firebase.json,
# the Dockerfile — of the commit the tag names.
#
# The button delivers the release at the tip of main, and nothing when there is
# none there: it never builds a commit nobody released, and it never reaches
# back for an older release, which would roll prod back. Should the tip carry
# more than one, the highest version is taken. A version handed over — by the
# release just published — is delivered only while main still ends at its
# commit: an older commit released late, or released again because its checks
# were run again, would roll prod back too.
#
# What is printed is for $GITHUB_OUTPUT: the version, the full commit its tag
# names, and whether to deliver it. Why not goes to stderr.
# Say which release a delivery rolls out, the commit it is of, and whether main still ends there
ci-release-target version="":
    #!/usr/bin/env bash
    set -euo pipefail
    version={{ quote(version) }}
    release={{ quote(RELEASE_TAG) }}
    if [ -n "$version" ] && [[ ! "$version" =~ $release ]]; then
        echo "ci-release-target: '${version}' is not vMAJOR.MINOR.PATCH" >&2
        exit 1
    fi
    # Read from GitHub rather than from this clone, which may be behind it.
    tip=$(git ls-remote origin refs/heads/main | cut -f1)
    if [[ ! "$tip" =~ ^[0-9a-f]{40}$ ]]; then
        echo "ci-release-target: could not read where main ends" >&2
        exit 1
    fi
    # Every tag, with the commit it names. An annotated tag's own line names
    # the tag object, and a second line, marked ^{}, the commit.
    tags=$(git ls-remote --tags origin | awk '
        {
            name = $2
            sub(/^refs\/tags\//, "", name)
            if (substr(name, length(name) - 2) == "^{}") {
                peeled[substr(name, 1, length(name) - 3)] = $1
            } else {
                named[name] = $1
            }
        }
        END { for (name in named) print name, ((name in peeled) ? peeled[name] : named[name]) }')
    if [ -z "$version" ]; then
        version=$(awk -v tip="$tip" '$2 == tip { print $1 }' <<< "$tags" | { grep -E "$release" || true; } | sort -V | tail -1)
        if [ -z "$version" ]; then
            echo "ci-release-target: no release is at the tip of main, ${tip}, so nothing is delivered" >&2
            printf 'version=\ncommit=\ndeliver=false\n'
            exit 0
        fi
    fi
    commit=$(awk -v version="$version" '$1 == version { print $2 }' <<< "$tags")
    if [ -z "$commit" ]; then
        echo "ci-release-target: no tag is called ${version}" >&2
        exit 1
    fi
    deliver=true
    if [ "$commit" != "$tip" ]; then
        deliver=false
        echo "ci-release-target: main ends at ${tip} now, past ${version} at ${commit}; the release of the tip delivers itself, so nothing is delivered here" >&2
    fi
    printf 'version=%s\ncommit=%s\ndeliver=%s\n' "$version" "$commit" "$deliver"

# "Re-run failed jobs" keeps what the first attempt of a delivery decided, the
# release to deliver among it, however long ago that was. So every job of a
# later attempt asks again, and goes no further once main has moved on: the
# newer commit's release delivers it, and this one would roll it back.
# Fail unless main still ends at the commit given
ci-release-current commit:
    #!/usr/bin/env bash
    set -euo pipefail
    commit={{ quote(commit) }}
    tip=$(git ls-remote origin refs/heads/main | cut -f1)
    if [ "$tip" != "$commit" ]; then
        echo "ci-release-current: main ends at ${tip:-a commit GitHub did not name} now, not at ${commit}: its own release delivers it, and this one would roll it back" >&2
        exit 1
    fi
    echo "ci-release-current: main still ends at ${commit}"

# Nothing reaches prod that the release did not sign. The sums are checked
# first: their signature has to be the release workflow's, run on main for this
# very commit — so the files of another release, signed as well, do not pass
# for this one's. cosign runs from its image, as SECURITY.md runs it. Only then
# is the file held to its own line among the sums, and only the file that
# passed lands in the folder given.
# Download one file of a release, and keep it only if the release workflow signed it at the commit given
ci-release-fetch version commit file dir:
    #!/usr/bin/env bash
    set -euo pipefail
    version={{ quote(version) }}
    commit={{ quote(commit) }}
    file={{ quote(file) }}
    dir={{ quote(dir) }}
    release={{ quote(RELEASE_TAG) }}
    if [[ ! "$version" =~ $release ]]; then
        echo "ci-release-fetch: '${version}' is not vMAJOR.MINOR.PATCH" >&2
        exit 1
    fi
    if [[ ! "$commit" =~ ^[0-9a-f]{40}$ ]]; then
        echo "ci-release-fetch: '${commit}' is not a commit's full hash" >&2
        exit 1
    fi
    # The name of a file of the release: never a path, and never a pattern,
    # which gh would match against every file there.
    if [[ ! "$file" =~ ^[0-9A-Za-z][0-9A-Za-z._-]*$ ]]; then
        echo "ci-release-fetch: '${file}' is not the name of a file of a release" >&2
        exit 1
    fi
    scratch=$(mktemp -d)
    trap 'rm -rf "$scratch"' EXIT
    # cosign runs as a user of its own, who has to be able to read the folder.
    chmod 0755 "$scratch"
    gh release download "$version" --repo {{ quote(REPOSITORY) }} --dir "$scratch" \
        --pattern SHA256SUMS --pattern SHA256SUMS.sigstore.json --pattern "$file"
    docker run --rm --volume "$scratch:/release:ro" --workdir /release {{ COSIGN_IMAGE }} \
        verify-blob --bundle SHA256SUMS.sigstore.json \
        --certificate-identity {{ quote(RELEASE_SIGNER) }} \
        --certificate-oidc-issuer {{ quote(GITHUB_ISSUER) }} \
        --certificate-github-workflow-sha "$commit" \
        SHA256SUMS
    if ! line=$(awk -v name="$file" 'NF == 2 && $2 == name { print; found++ } END { exit found != 1 }' "$scratch/SHA256SUMS"); then
        echo "ci-release-fetch: the sums of ${version} do not name ${file} once" >&2
        exit 1
    fi
    (cd "$scratch" && sha256sum --check --strict <<< "$line")
    mkdir -p "$dir"
    mv "$scratch/$file" "$dir/$file"

# The image of a release is the runtime around the server that release built
# and signed, never one built again: what is rolled out is what was released,
# byte for byte. It has to name the release before it is pushed, as the
# image the checks build has to name its build, and it is named afterwards by
# the digest its own push reported: a tag can be moved, and a digest cannot.
# The tag is the release's number, for a person reading the registry.
# Build the image of a release around the server it signed, check it, push it, and print its exact reference
ci-image-push binary version commit image:
    #!/usr/bin/env bash
    set -euo pipefail
    binary={{ quote(binary) }}
    version={{ quote(version) }}
    commit={{ quote(commit) }}
    image={{ quote(image) }}
    release={{ quote(RELEASE_TAG) }}
    if [[ ! "$version" =~ $release ]]; then
        echo "ci-image-push: '${version}' is not vMAJOR.MINOR.PATCH" >&2
        exit 1
    fi
    if [[ ! "$commit" =~ ^[0-9a-f]{40}$ ]]; then
        echo "ci-image-push: '${commit}' is not a commit's full hash" >&2
        exit 1
    fi
    # A repository of a registry, which the tag and the digest are added to.
    if [[ ! "$image" =~ ^[a-z0-9.-]+(:[0-9]+)?/[a-z0-9._/-]+$ ]]; then
        echo "ci-image-push: '${image}' is not an image repository" >&2
        exit 1
    fi
    scratch=$(mktemp -d)
    trap 'rm -rf "$scratch"' EXIT
    # The context the Dockerfile's release stage takes the server from, holding
    # it alone. It is the stage's only input from outside, so nothing else of
    # the checkout is sent to the builder.
    mkdir "$scratch/context"
    cp "$binary" "$scratch/context/server"
    tag="${image}:${version}"
    # No provenance: the build would hang it on the image as a second
    # manifest, and what is pushed would be an index rather than the image.
    docker build --target release --build-context "binary=${scratch}/context" \
        --provenance=false \
        --label "org.opencontainers.image.source=https://{{ MODULE }}" \
        --label "org.opencontainers.image.version=${version}" \
        --label "org.opencontainers.image.revision=${commit}" \
        --tag "$tag" . >&2
    {{ quote(just_executable()) }} --justfile {{ quote(justfile()) }} _image-health "$tag" "$version" "$commit" >&2
    # A push needs a credential helper for that registry, the first component
    # of the image's path.
    gcloud auth configure-docker "${image%%/*}" --quiet >&2
    if ! docker push "$tag" | tee "$scratch/push" >&2; then
        echo "ci-image-push: could not push ${tag}; docker push says why above" >&2
        exit 1
    fi
    digest=$(sed -n 's/^.*: digest: \(sha256:[0-9a-f]\{64\}\) size: [0-9]*$/\1/p' "$scratch/push" | tail -1)
    if [ -z "$digest" ]; then
        echo "ci-image-push: docker push named no digest for ${tag}" >&2
        exit 1
    fi
    # The one thing on stdout, so that a caller can read it with a substitution.
    echo "${image}@${digest}"

# The service is created once, by the configuration, with everything around
# it; a delivery only ever changes the image it serves, and only by digest. The
# new revision then takes all the traffic, wherever a rollback left it: once
# the traffic is pinned to a revision, a new one gets none until it is sent
# back to the latest (Cloud Run's documentation, read 2026-10-14).
# Roll a revision of the service onto the image given, and send it all the traffic
ci-deploy service region image:
    #!/usr/bin/env bash
    set -euo pipefail
    service={{ quote(service) }}
    region={{ quote(region) }}
    image={{ quote(image) }}
    if [[ ! "$image" =~ @sha256:[0-9a-f]{64}$ ]]; then
        echo "ci-deploy: an image is deployed by its digest, never by a tag: ${image}" >&2
        exit 1
    fi
    gcloud run services update "$service" --region="$region" --image="$image" --quiet
    gcloud run services update-traffic "$service" --region="$region" --to-latest --quiet

# Answering is half of it: what answers has to be the release just rolled out,
# by its version and its commit, rather than the revision before it. No
# redirect is followed: the answer has to come from the address itself, which
# is the caller's to choose, so this also probes a container on this machine.
# Fail unless the service answers its health as the release given, at the commit given
ci-smoke url version commit:
    #!/usr/bin/env bash
    set -euo pipefail
    url={{ quote(url) }}
    version={{ quote(version) }}
    commit={{ quote(commit) }}
    body=""
    for attempt in $(seq 1 10); do
        if body=$(curl -fsS --max-time 10 "${url}/health") \
            && jq --exit-status --arg version "$version" --arg commit "$commit" \
                {{ quote(HEALTH_MATCHES) }} <<< "$body" > /dev/null 2>&1; then
            echo "ci-smoke: ${url} answers as ${version} at $(jq -r .commit <<< "$body")"
            exit 0
        fi
        echo "health: not this release yet (attempt ${attempt})" >&2
        sleep 3
    done
    echo "ci-smoke: ${url}/health answers ${body:-nothing}; want status ok, version ${version}, and a commit that begins ${commit}" >&2
    exit 1

# Hosting serves exactly the SPA the release built and signed: the archive is
# unpacked where firebase.json says the site's files are, over nothing an
# earlier build left there, and published to the project's default site under
# the release's number, which Hosting keeps beside the version in its history.
# Publish the SPA of a release, from its archive, to Firebase Hosting
ci-hosting-deploy archive project version:
    #!/usr/bin/env bash
    set -euo pipefail
    archive={{ quote(archive) }}
    project={{ quote(project) }}
    version={{ quote(version) }}
    rm -rf web/dist
    mkdir -p web/dist
    tar -C web/dist -xzf "$archive"
    if [ ! -f web/dist/index.html ]; then
        echo "ci-hosting-deploy: ${archive} holds no index.html" >&2
        exit 1
    fi
    firebase deploy --only hosting --project "$project" --non-interactive --message "$version"

# The page has to be the one just published, byte for byte, and served
# uncached: a browser that kept it would ask for the files of a release that
# the next one removed. No two releases share a page: it names the script by a
# hash of the script, and the script carries the release's number. So the page
# of the release before never passes for this one's, and the check waits the
# few seconds Hosting takes to put a new version in front of everybody. As
# with the service, no redirect is followed.
# Fail unless the site serves the page in web/dist, uncached
ci-hosting-check url:
    #!/usr/bin/env bash
    set -euo pipefail
    url={{ quote(url) }}
    scratch=$(mktemp -d)
    trap 'rm -rf "$scratch"' EXIT
    for attempt in $(seq 1 10); do
        if curl -fsS --max-time 10 \
            --dump-header "$scratch/headers" --output "$scratch/page" "${url}/" \
            && cmp -s "$scratch/page" web/dist/index.html; then
            cache=$(tr -d '\r' < "$scratch/headers" | sed -n 's/^[Cc][Aa][Cc][Hh][Ee]-[Cc][Oo][Nn][Tt][Rr][Oo][Ll]: *//p' | tail -1)
            if [ "$cache" != no-cache ]; then
                echo "ci-hosting-check: ${url}/ is the page in web/dist, served with Cache-Control '${cache}'; want no-cache" >&2
                exit 1
            fi
            echo "ci-hosting-check: ${url}/ is the page in web/dist, uncached"
            exit 0
        fi
        echo "hosting: not this page yet (attempt ${attempt})" >&2
        sleep 3
    done
    echo "ci-hosting-check: ${url}/ is not the page in web/dist" >&2
    exit 1
