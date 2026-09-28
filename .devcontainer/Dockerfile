# Development container for canvas-api.
# Exact versions only: the base image is pinned by digest, every tool below by an ARG default.
# Anything a builder understands: no heredocs, no RUN --mount, no TARGETARCH — architecture is
# detected at RUN time via dpkg instead. A host that has Docker can build this, and that is the
# whole requirement.
#
# Two stages. `toolchain` holds what it takes to build and check the code, and is the stage
# the automated checks run inside. `devcontainer` adds what a person needs on top of that and
# is the default stage, so building this file with no target gives the full environment.

# -- toolchain: builds and checks the code -----------------------------------

FROM mcr.microsoft.com/devcontainers/base:3.0.8-noble@sha256:d7c468679f45a52ad3673d06656b5bf16e488990b17216a1fa295d9e1f89d724 AS toolchain

SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

ARG GO_VERSION=1.27.1
# Node.js LTS for the SPA in web/: its checks and its build.
ARG NODE_VERSION=24.21.0
ARG JUST_VERSION=1.58.0
ARG GOLANGCI_LINT_VERSION=2.13.2
ARG MOCKERY_VERSION=3.8.0

# Every archive comes through one helper, so that every download is retried the same way.
# --retry-all-errors is the point of it: a transfer that dies partway is not something a plain
# --retry repeats, and that is exactly the failure a small tunnel MTU produces. The file is
# renamed only once curl is happy, so a half-written archive is never unpacked as a whole one.
RUN arch="$(dpkg --print-architecture)"; \
    case "$arch" in \
        amd64) go_arch=amd64; node_arch=x64; gnu_arch=x86_64; mockery_arch=x86_64 ;; \
        arm64) go_arch=arm64; node_arch=arm64; gnu_arch=aarch64; mockery_arch=arm64 ;; \
        *) echo "unsupported architecture: $arch" >&2; exit 1 ;; \
    esac; \
    workdir="$(mktemp -d)"; \
    fetch() { \
        local url="$1" out="$2"; \
        if [[ ! -s "$out" ]]; then \
            curl -fsSL --proto "=https" --proto-redir "=https" --retry 3 --retry-all-errors \
                -o "$out.part" "$url"; \
            mv "$out.part" "$out"; \
        fi; \
    }; \
    \
    echo "go ${GO_VERSION}"; \
    fetch "https://go.dev/dl/go${GO_VERSION}.linux-${go_arch}.tar.gz" \
        "$workdir/go-${GO_VERSION}-${go_arch}.tar.gz"; \
    tar -C /usr/local -xzf "$workdir/go-${GO_VERSION}-${go_arch}.tar.gz"; \
    \
    echo "node ${NODE_VERSION}"; \
    mkdir -p /usr/local/node; \
    fetch "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${node_arch}.tar.gz" \
        "$workdir/node-${NODE_VERSION}-${node_arch}.tar.gz"; \
    tar -C /usr/local/node --strip-components=1 -xzf "$workdir/node-${NODE_VERSION}-${node_arch}.tar.gz"; \
    \
    echo "just ${JUST_VERSION}"; \
    fetch "https://github.com/casey/just/releases/download/${JUST_VERSION}/just-${JUST_VERSION}-${gnu_arch}-unknown-linux-musl.tar.gz" \
        "$workdir/just-${JUST_VERSION}-${gnu_arch}.tar.gz"; \
    tar -C "$workdir" -xzf "$workdir/just-${JUST_VERSION}-${gnu_arch}.tar.gz" just; \
    install -m 0755 "$workdir/just" /usr/local/bin/just; \
    \
    echo "golangci-lint ${GOLANGCI_LINT_VERSION}"; \
    fetch "https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}/golangci-lint-${GOLANGCI_LINT_VERSION}-linux-${go_arch}.tar.gz" \
        "$workdir/golangci-lint-${GOLANGCI_LINT_VERSION}-${go_arch}.tar.gz"; \
    tar -C "$workdir" --strip-components=1 \
        -xzf "$workdir/golangci-lint-${GOLANGCI_LINT_VERSION}-${go_arch}.tar.gz" \
        "golangci-lint-${GOLANGCI_LINT_VERSION}-linux-${go_arch}/golangci-lint"; \
    install -m 0755 "$workdir/golangci-lint" /usr/local/bin/golangci-lint; \
    \
    echo "mockery ${MOCKERY_VERSION}"; \
    fetch "https://github.com/vektra/mockery/releases/download/v${MOCKERY_VERSION}/mockery_${MOCKERY_VERSION}_Linux_${mockery_arch}.tar.gz" \
        "$workdir/mockery-${MOCKERY_VERSION}-${mockery_arch}.tar.gz"; \
    tar -C "$workdir" -xzf "$workdir/mockery-${MOCKERY_VERSION}-${mockery_arch}.tar.gz" mockery; \
    install -m 0755 "$workdir/mockery" /usr/local/bin/mockery; \
    \
    rm -rf "$workdir"

ENV PATH="/usr/local/go/bin:/usr/local/node/bin:/home/vscode/go/bin:/home/vscode/.local/bin:${PATH}" \
    GOTOOLCHAIN=local

# -- devcontainer: the toolchain, plus what a person works with --------------

FROM toolchain AS devcontainer

SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

# Go editor tooling, installed with `go install`; the Go extension must not fetch its own
# copies (see devcontainer.json settings).
ARG GOPLS_VERSION=0.23.0
ARG DELVE_VERSION=1.27.2
# Tunnel that opens the SPA served from here on a real tablet, to try the pen on it.
ARG CLOUDFLARED_VERSION=2026.9.1
# The GitHub CLI. Two things about a change live only on github.com and are
# invisible from inside a container otherwise: the alerts of the code scan,
# which every task is supposed to close before starting the next one, and
# whether the delivery that carried the change actually ran.
ARG GH_VERSION=2.101.0
# Docker Compose plugin. Installed by us because the docker-in-docker feature cannot pin it;
# ours in /usr/local/lib/docker/cli-plugins takes precedence over the feature's copy.
ARG DOCKER_COMPOSE_VERSION=5.5.1
# Claude Code CLI. The VS Code extension in devcontainer.json must have the same version.
ARG CLAUDE_CODE_VERSION=2.1.280
# The cloud is described in Terraform and signed into with gcloud, both from here.
ARG TERRAFORM_VERSION=1.16.3
ARG GCLOUD_VERSION=585.0.0

# The tunnel, the GitHub CLI, the compose plugin and the infrastructure tools, in one
# layer and behind one architecture check. Cloudflared, gh and Terraform name
# architectures the way dpkg does, so they take $arch directly; only compose needs the
# GNU spelling.
#
# The x86_64 archive of gcloud carries a Python interpreter of its own and runs
# straight out of the tarball; the arm one expects to find a system Python, which
# this image deliberately does not have, so there gcloud is left out and the same
# commands are run from Google's own container image instead.
RUN arch="$(dpkg --print-architecture)"; \
    case "$arch" in \
        amd64) gnu_arch=x86_64 ;; \
        arm64) gnu_arch=aarch64 ;; \
        *) echo "unsupported architecture: $arch" >&2; exit 1 ;; \
    esac; \
    workdir="$(mktemp -d)"; \
    fetch() { \
        local url="$1" out="$2"; \
        if [[ ! -s "$out" ]]; then \
            curl -fsSL --proto "=https" --proto-redir "=https" --retry 3 --retry-all-errors \
                -o "$out.part" "$url"; \
            mv "$out.part" "$out"; \
        fi; \
    }; \
    \
    echo "cloudflared ${CLOUDFLARED_VERSION}"; \
    fetch "https://github.com/cloudflare/cloudflared/releases/download/${CLOUDFLARED_VERSION}/cloudflared-linux-${arch}" \
        "$workdir/cloudflared-${CLOUDFLARED_VERSION}-${arch}"; \
    install -m 0755 "$workdir/cloudflared-${CLOUDFLARED_VERSION}-${arch}" /usr/local/bin/cloudflared; \
    \
    echo "gh ${GH_VERSION}"; \
    fetch "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_${arch}.tar.gz" \
        "$workdir/gh-${GH_VERSION}-${arch}.tar.gz"; \
    tar -C "$workdir" --strip-components=2 \
        -xzf "$workdir/gh-${GH_VERSION}-${arch}.tar.gz" \
        "gh_${GH_VERSION}_linux_${arch}/bin/gh"; \
    install -m 0755 "$workdir/gh" /usr/local/bin/gh; \
    \
    echo "docker compose ${DOCKER_COMPOSE_VERSION}"; \
    install -d -m 0755 /usr/local/lib/docker/cli-plugins; \
    fetch "https://github.com/docker/compose/releases/download/v${DOCKER_COMPOSE_VERSION}/docker-compose-linux-${gnu_arch}" \
        "$workdir/docker-compose-${DOCKER_COMPOSE_VERSION}-${gnu_arch}"; \
    install -m 0755 "$workdir/docker-compose-${DOCKER_COMPOSE_VERSION}-${gnu_arch}" \
        /usr/local/lib/docker/cli-plugins/docker-compose; \
    \
    echo "terraform ${TERRAFORM_VERSION}"; \
    fetch "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_${arch}.zip" \
        "$workdir/terraform-${TERRAFORM_VERSION}-${arch}.zip"; \
    unzip -q -d "$workdir" "$workdir/terraform-${TERRAFORM_VERSION}-${arch}.zip"; \
    install -m 0755 "$workdir/terraform" /usr/local/bin/terraform; \
    \
    if [[ "$arch" == "amd64" ]]; then \
        echo "gcloud ${GCLOUD_VERSION}"; \
        fetch "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${GCLOUD_VERSION}-linux-x86_64.tar.gz" \
            "$workdir/gcloud-${GCLOUD_VERSION}-x86_64.tar.gz"; \
        tar -C /usr/local -xzf "$workdir/gcloud-${GCLOUD_VERSION}-x86_64.tar.gz"; \
    else \
        echo "gcloud ${GCLOUD_VERSION}: skipped, no interpreter in the ${arch} archive"; \
    fi; \
    \
    rm -rf "$workdir"

ENV PATH="/usr/local/google-cloud-sdk/bin:${PATH}"

# What a person edits and debugs with, in one layer.
#
# gopls and dlv are built from source at an exact version, and the Go checksum
# database verifies what that version pulls in; read-only says that an install
# may not rewrite what it was pinned to.
#
# Claude Code comes from the registry at an exact version, rather than from an
# installer script run off the network: what arrives is the package that version
# names, and npm checks the archive against the hash the registry publishes for
# it. That package's own install step is allowed by name — it is what puts the
# native binary in place of the stub the package ships with, and without it the
# CLI starts through a Node wrapper on every invocation.
#
# The caches those installs fill under root's home — Go's modules and builds,
# npm's downloads — are cleared in the same layer: some 800 MB that nothing reads
# again, since the container runs as vscode.
RUN echo "gopls ${GOPLS_VERSION}, dlv ${DELVE_VERSION}, claude code ${CLAUDE_CODE_VERSION}" \
    && GOBIN=/usr/local/bin go install -mod=readonly "golang.org/x/tools/gopls@v${GOPLS_VERSION}" \
    && GOBIN=/usr/local/bin go install -mod=readonly "github.com/go-delve/delve/cmd/dlv@v${DELVE_VERSION}" \
    && npm install -g --allow-scripts=@anthropic-ai/claude-code \
        "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}" \
    && go clean -cache -modcache \
    && npm cache clean --force

# The config dirs are created here so the named volumes mounted over them inherit
# vscode ownership, and the image is left as that user: nothing in it needs root
# once it is built, and the one thing that does — the docker daemon started by
# the docker-in-docker feature — asks for root through sudo itself.
USER vscode
RUN mkdir -p /home/vscode/.claude /home/vscode/.config/gcloud /home/vscode/.config/gh
