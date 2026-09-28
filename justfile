# canvas-api: task runner.

set shell := ["bash", "-euo", "pipefail", "-c"]

# List all recipes
default:
    @just --list

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
