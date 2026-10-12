# Exact versions only. The builder is pinned by its exact tag and its digest;
# distroless names no versions in its tags, so the runtime is pinned by digest
# alone. The builder's Go version is the one go.mod names and the tests run on,
# so the shipped binary is built by the same toolchain that proved it works.

FROM golang:1.27.2-trixie@sha256:e58d6f83b3416618d8bcac2b3dde1b7f7e3c4a77d25e88637f8bbae81536c48d AS build

WORKDIR /src

# Dependencies first: they change far less often than the code, so this layer
# survives most rebuilds. go.* rather than go.mod and go.sum by name: the module
# has no dependencies yet, so there is no go.sum to copy.
COPY go.* ./
RUN go mod download

COPY . .

# The build identity comes in as arguments rather than being read from git:
# the build context has no .git, and a version the image guesses is worse than
# one it is told (internal/version).
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

# CGO off so the binary is static and can run on a distroless image with no
# libc; -trimpath so the paths of whoever built it are not in the binary.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w \
      -X github.com/MathTrail/canvas-api/internal/version.Version=${VERSION} \
      -X github.com/MathTrail/canvas-api/internal/version.Commit=${COMMIT} \
      -X github.com/MathTrail/canvas-api/internal/version.Date=${DATE}" \
    -o /server ./cmd/server

# The runtime holds the binary and a certificate bundle, and nothing else: no
# shell, no package manager, and a non-root user (uid 65532).
FROM gcr.io/distroless/static-debian13@sha256:58133991db06659feaabe0f4e97a35cebf15ef4ea08f8a4c6d2ee5f75e4aa6a0

COPY --from=build /server /server

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
