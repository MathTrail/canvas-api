set shell := ["bash", "-c"]
set dotenv-load
set dotenv-path := "/etc/mathtrail/platform.env"

SERVICE   := "canvas-api"
NAMESPACE := env_var_or_default("NAMESPACE", "mathtrail")

# Build the binary locally
build:
    go build -o bin/canvas-api ./cmd/server

# Build and push image to k3d registry via buildah
build-push-image tag=env("IMAGE", ""):
    #!/bin/bash
    set -euo pipefail
    TAG="{{ tag }}"
    if [ -z "$TAG" ]; then
        echo "Error: no image tag provided (set IMAGE env var or pass as argument)" >&2
        exit 1
    fi
    buildah --storage-driver=vfs bud --log-level=error --tag "$TAG" .
    buildah --storage-driver=vfs push --log-level=error --tls-verify=false "$TAG"

# Run go tests
test:
    go test ./...

# Deploy to k3d via Skaffold (includes build on change)
dev:
    skaffold dev -m canvas-api --port-forward

# IMPORTANT: Run this before the first `just dev` to seed the image in the registry.
# inputDigest tagPolicy requires an existing image digest to compute tags.
first-build:
    IMAGE="$REGISTRY_CLUSTER/canvas-api:local" just build-push-image

# Intercept canvas-api traffic from k3d to local process via Telepresence.
# --mapped-namespaces all ensures DNS for kratos.identity.svc and
# automq.streaming.svc resolves correctly in the local environment.
tp-intercept:
    telepresence connect -n {{ NAMESPACE }} --mapped-namespaces all
    telepresence intercept {{ SERVICE }} --port 8080:8080
    @echo "Intercepting {{ SERVICE }}. Start the service with: go run ./cmd/server"

# Stop Telepresence intercept
tp-stop:
    telepresence leave {{ SERVICE }} 2>/dev/null || true
    telepresence quit

# Show Telepresence status
tp-status:
    telepresence status
    telepresence list
