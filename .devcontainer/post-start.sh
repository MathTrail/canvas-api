#!/bin/bash
set -e

# Load platform environment
set -a; source /etc/mathtrail/platform.env; set +a

# Set up skaffold.env symlink
ln -sf /etc/mathtrail/platform.env "$PWD/skaffold.env"

# Set up kubeconfig from host bind mount
mkdir -p /home/vscode/.kube
KUBECONFIG_SRC="/home/vscode/.kube-host/${CLUSTER_NAME}.yaml"

if sudo test -f "$KUBECONFIG_SRC"; then
    sudo install -o vscode -g vscode -m 600 "$KUBECONFIG_SRC" /home/vscode/.kube/config
    sed -i 's|https://0.0.0.0:|https://host.docker.internal:|g' /home/vscode/.kube/config
    sed -i 's/certificate-authority-data:.*/insecure-skip-tls-verify: true/' /home/vscode/.kube/config
    echo "Kubeconfig ready"
else
    echo "Warning: kubeconfig not found at $KUBECONFIG_SRC"
    echo "Run 'just kubeconfig' in infra-local-k3s on host first"
fi

# Set up git hooks
git config core.hooksPath .githooks

# Verify cluster connection
kubectl cluster-info 2>/dev/null && echo "Connected to cluster" \
    || echo "Cluster not accessible — check that k3d is running on host"
