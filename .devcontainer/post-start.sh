#!/usr/bin/env bash
# Runs on every devcontainer start, from the repository root.
set -euo pipefail

# Git hooks live in .githooks/; enable them once the folder exists.
if [[ -d .githooks ]]; then
    git config core.hooksPath .githooks
fi

# The docker inside this container hands its networks an MTU of 1500. When this container's
# own interface is smaller — a tunnel on the host is the usual reason — everything that
# crosses both stalls: a packet that fits here is too large for the tunnel, and nothing
# reports it. Large downloads die a few kilobytes in; small ones are fine, which is what
# makes it look like anything but the network.
interface="$(ip -o route show default | awk '{ print $5; exit }')"
mtu="$(cat "/sys/class/net/${interface}/mtu" 2> /dev/null || echo 1500)"

if [[ "$mtu" -lt 1500 ]]; then
    # Both keys, for the same reason they are both needed on the host: the first one governs
    # the default bridge, the second one every network created later.
    config="$(cat <<JSON
{
    "mtu": ${mtu},
    "default-network-opts": {
        "bridge": {
            "com.docker.network.driver.mtu": "${mtu}"
        }
    }
}
JSON
)"

    # The daemon reads that file only when it starts. Found in place, it was read at this
    # start. Written only now — the first start of a rebuilt container — it usually comes
    # after the daemon, whose containers and image builds then keep 1500 until the next
    # start. Lowering its bridge would not help: each container is given the MTU the daemon
    # started with, whatever the bridge says.
    if [[ "$(cat /etc/docker/daemon.json 2> /dev/null)" != "$config" ]]; then
        printf '%s\n' "$config" | sudo tee /etc/docker/daemon.json > /dev/null
        echo "docker: mtu ${mtu} from the next start; restart this container once to pick it up"
    fi
fi
