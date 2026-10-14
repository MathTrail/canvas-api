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
# issuer is GitHub's, whose short-lived tokens the certificate is given for
# (R53).
TOOLCHAIN_SIGNER := "https://" + MODULE + "/.github/workflows/ci.yml@refs/heads/main"
GITHUB_ISSUER := "https://token.actions.githubusercontent.com"
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

# The deployment, as Terraform describes it (spec 15.2), and the filter what
# Terraform prints about it passes through (_tf-redact, below).
TF_DIR := "infra/terraform"
TF_REDACT := quote(just_executable()) + " --justfile " + quote(justfile()) + " _tf-redact"

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

# A version asked for is the one published. Otherwise it is the next patch of
# the line the releases are numbered in, MAJOR.MINOR, or the line's first when
# none of it is out yet; the line is the one release.yml names, unless another
# is given (R149 standalone). Refused: a version that is not vMAJOR.MINOR.PATCH,
# one already released, and one that would come out below the newest release —
# from a line older than it, or asked for so — since each would number the
# releases out of order or twice.
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
    releases() { grep -E "^v${number}\.${number}\.${number}$" | sort -V || true; }
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
    if [[ ! "$version" =~ ^v${number}\.${number}\.${number}$ ]]; then
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
# Hosting serves it, and no binary carries it (R10).
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

# The image is started as Cloud Run starts it, with PORT and nothing else, and
# asked for its health as the platform's probe asks. It has to answer, and to
# name the build it was made from: an image that lost its version on the way
# would roll out as a revision nobody can tell from the one before.
# Fail if the image does not start, or its health does not name this build
ci-image-smoke tag="canvas-api:dev": (docker-build tag)
    #!/usr/bin/env bash
    set -euo pipefail
    # The tag and the build identity reach the script as quoted text, as in
    # docker-build, and it reads them only from these variables.
    tag={{ quote(tag) }}
    version={{ quote(VERSION) }}
    commit={{ quote(COMMIT) }}
    # Where git cannot tell the build, it is named dev at unknown — which is
    # also what a binary that lost its version says, so the answer would prove
    # nothing.
    if [ "$version" = dev ] || [ "$commit" = unknown ]; then
        echo "ci-image-smoke: the build is named ${version} at ${commit}, as a binary that lost its version names itself, so its answer would prove nothing; run the check in a clone, with its history" >&2
        exit 1
    fi
    # Created first and started second, so that the container is removed when
    # the check ends, whether it never started, stopped at once — its logs are
    # still there to show — or is serving still.
    container=$(docker create --publish 127.0.0.1::8080 --env PORT=8080 "$tag")
    trap 'docker rm --force "$container" > /dev/null' EXIT
    if ! docker start "$container" > /dev/null; then
        echo "ci-image-smoke: ${tag} did not start" >&2
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
        echo "ci-image-smoke: ${tag} never answered its health check" >&2
        docker logs "$container" >&2
        exit 1
    fi
    if ! jq --exit-status --arg version "$version" --arg commit "$commit" \
        '.status == "ok" and .version == $version and .commit == $commit' <<< "$body" > /dev/null 2>&1; then
        echo "ci-image-smoke: ${tag} answers ${body}; want status ok, version ${version}, commit ${commit}" >&2
        exit 1
    fi
    echo "ci-image-smoke: ${tag} answers as ${version} at ${commit}"

# Trivy runs from its own image, so nothing about it is installed here and the
# verdict is the same on a laptop and in CI. It reads a saved copy of the image
# rather than Docker itself: handed Docker's socket, the scanner would hold the
# whole machine. A finding is waived only in .trivyignore, with its reason, and
# the file goes in beside the copy; one in Go's standard library is not waived
# but fixed by a newer Go (R38 standalone).
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

# -- Container --------------------------------------------------------------

# The tag is a hash of the toolchain stage of the development image — every
# line above the stage built on top of it — so one definition of the checks'
# environment is one image, and any edit to it a new one, while an edit to the
# stage on top, a new Claude Code say, leaves it as it was. Nothing is built
# when that tag is published already. What is printed is the digest, not the
# tag: a tag can be moved, and a version here is exact (R20 standalone).
#
# Whoever may publish the package may move the tag, and every run of a pull
# request of this repository may (Т-17). So a pull request takes the image the
# tag names — accept=published — while main takes only one it signed itself —
# accept=signed — and builds any other again. An image built here is named by
# the digest its own push reports, never by what the tag names afterwards: a run
# of a pull request may move the tag in between (R53).
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
# image, and its certificate into Sigstore's public log (R53).
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
# the revision finds a version of it (RUN.md, rule 13): there is no secret yet.
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
    for name in deploy_service_account image_repository service_name region service_url hosting_url; do
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
