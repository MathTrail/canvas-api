# MathTrail Canvas API

## Overview

Interactive math canvas service for the MathTrail platform. Students solve problems (e.g. "1 + 1 =") by drawing with a stylus on a Konva canvas. Strokes are published to AutoMQ; hints are returned in real-time via Centrifugo WebSocket.

**Language:** Go 1.23
**Port:** 8080
**Cluster:** k3d `mathtrail-dev`, namespace `mathtrail`
**KUBECONFIG:** `/home/vscode/.kube/k3d-mathtrail-dev.yaml`

## Tech Stack

| Layer | Library |
|-------|---------|
| HTTP | `github.com/go-chi/chi/v5` |
| Kafka | `github.com/twmb/franz-go` — SASL/SCRAM-SHA-512, AutoMQ |
| Protobuf | `google.golang.org/protobuf` — contracts from `../contracts` |
| Auth | Ory Kratos session validation (`GET /sessions/whoami`) |
| JWT | `github.com/golang-jwt/jwt/v5` — Centrifugo connection + channel tokens |
| Config | `github.com/spf13/viper` |
| Logging | `go.uber.org/zap` |
| Real-time | Centrifugo HTTP API client → publishes `HintEvent` to WebSocket subscribers |

## Key Files

| File | Purpose |
|------|---------|
| `cmd/canvas-api/main.go` | Entry point — chi router, errgroup (HTTP + hint consumer + shutdown) |
| `internal/config/config.go` | Config (Viper) — all settings loaded from env |
| `internal/handlers/token.go` | `GET /api/canvas/token` — Centrifugo connection + channel JWT |
| `internal/handlers/stroke.go` | `POST /api/canvas/strokes` — validate session, publish to Kafka |
| `internal/handlers/health.go` | `GET /health` — liveness probe |
| `internal/kafka/producer.go` | franz-go producer — partition key `[]byte(sessionID)` for ordering |
| `internal/kafka/consumer.go` | Consumes `canvas.hints` → decodes `HintEvent` → Centrifugo publish |
| `internal/infra/centrifugo/client.go` | HTTP API client — POST `/api/publish` with `b64data` binary payload |
| `internal/ory/session.go` | Kratos `WhoAmI()` — forwards cookies, returns `Session{Identity.ID}` |
| `internal/middleware/auth.go` | Ory session validation middleware → stores user ID in context |
| `internal/middleware/cors.go` | CORS — `AllowCredentials: true`, configurable `AllowedOrigins` |
| `infra/helm/canvas-api/` | Helm chart (uses `mathtrail-service-lib`) |
| `infra/helm/values-dev.yaml` | Dev environment values (env vars, VSO secret refs) |
| `skaffold.yaml` | Skaffold pipeline config |
| `justfile` | Build, deploy, Telepresence automation |
| `.devcontainer/devcontainer.json` | VS Code devcontainer config |

## API

```
GET  /health                            — liveness probe
GET  /api/canvas/token?session_id={id}  — Centrifugo connection JWT + channel token
POST /api/canvas/strokes                — submit strokes (Protobuf octet-stream) → 202
```

### Token endpoint flow
1. Validate Ory session cookie → `GET kratos /sessions/whoami`
2. Generate Centrifugo connection JWT: `{ sub: userId, exp: +1h }` signed with `CENTRIFUGO_HMAC_KEY`
3. Generate channel subscription token: `{ sub: userId, channel: "canvas:{sessionId}", exp: +1h }`
4. Return `{ token, channel, channel_token }`

### Strokes endpoint flow
1. Auth middleware validates session, injects `userId` into context
2. Unmarshal Protobuf `CanvasStrokeEvent` from request body
3. Stamp `user_id` from session (prevents spoofing)
4. Publish to AutoMQ `canvas.strokes`, **partition key = `[]byte(sessionID)`** — ensures stroke order
5. Return `202 Accepted`

### Hint consumer goroutine
Consumes `canvas.hints` → deserializes `HintEvent` → POST Centrifugo HTTP API → pushes to `canvas:{sessionId}`.

## Architecture

- **Secrets:** VSO (Vault Secrets Operator) — `VaultAuth` + `VaultStaticSecret` → K8s Secret → env vars.
  - `canvas-api-secrets` K8s Secret: `centrifugo_api_key`, `centrifugo_hmac_key`
  - `automq-canvas-api-scram` K8s Secret: `username`, `password`
  - See `infra/helm/canvas-api/templates/vault-secrets.yaml`
- **Protobuf contracts:** `github.com/mathtrail/contracts` — local replace directive `../contracts`.
  Run `just generate` in contracts devcontainer to regenerate Go bindings.
- **Kafka:** franz-go with SASL/SCRAM-SHA-512. Partition key `[]byte(sessionID)` guarantees per-student stroke ordering — franz-go requires `[]byte`, not `string`.
- **Centrifugo channel namespace:** `canvas` — protected (subscription tokens required).
  Channel format: `canvas:{sessionId}`.
- **CORS:** `AllowCredentials: true` required — Ory session cookie travels cross-origin from shell at `localhost:3000`.
- Helm chart uses `mathtrail-service-lib` library chart from `https://MathTrail.github.io/charts/charts`

## Service-Lib Contract (MUST follow)

- **Health probe:** `GET /health` — must return 200
- **Security:** Container runs as non-root UID 10001, `readOnlyRootFilesystem: true`
- **Validation:** `image.repository`, `image.tag`, `resources.requests`, `resources.limits` must be defined in values.yaml

## React MFE (`ui/`)

Vite 6 + React 19 + TypeScript + Tailwind 4, exposed as Module Federation **Remote** on port 3001.

| File | Purpose |
|------|---------|
| `ui/vite.config.ts` | MFE Remote — exposes `./OlympiadCanvas`, port 3001 |
| `ui/src/components/OlympiadCanvas.tsx` | Konva Stage with 3 layers: TaskLayer, DrawingLayer, HintLayer |
| `ui/src/components/Toolbar.tsx` | Pen / Eraser / Clear (Lucide icons) |
| `ui/src/store/useCanvasStore.ts` | Zustand — strokes, active stroke, tool, server hints |
| `ui/src/hooks/useCentrifuge.ts` | centrifuge-js Protobuf build — subscribes `canvas:{sessionId}`, decodes `HintEvent` |
| `ui/src/hooks/useCanvasToken.ts` | `GET /api/canvas/token` → `{token, channel, channelToken}` |
| `ui/src/transport/strokeApi.ts` | `CanvasStrokeEvent.toBinary()` → POST octet-stream |
| `ui/src/gen/canvas/v1/` | **Placeholders** — replace with `buf generate` output from contracts repo |

### Konva layer architecture

| # | Layer | `listening` | Content |
|---|-------|-------------|---------|
| 1 | TaskLayer | `false` | Task text — never erased |
| 2 | DrawingLayer | `true` | Pen strokes (`Konva.Shape` + `sceneFunc`) + eraser (`destination-out`) |
| 3 | HintLayer | `false` | Server hint overlays (`Rect` + `Text`) |

**Pen:** `perfect-freehand` `getStroke()` returns an outline contour → `Konva.Shape` with `sceneFunc` + `fill()`.
**Eraser:** plain `Konva.Line` with `globalCompositeOperation="destination-out"`. Acts only within DrawingLayer canvas — TaskLayer (separate DOM canvas) is safe.

## Development Workflow

```bash
just build           # go build → bin/canvas-api
just test            # go test ./...
just first-build     # seed image in k3d registry (required before first skaffold dev)
just dev             # skaffold dev -m canvas-api --port-forward

just tp-intercept    # Telepresence intercept (--mapped-namespaces all)
                     # then: go run ./cmd/canvas-api
just tp-stop         # leave intercept + quit telepresence
just tp-status       # show current intercept state

# UI dev server (port 3001)
cd ui && npm run dev
```

## Development Standards

- Handle errors explicitly — never ignore error returns
- All comments in English
- Middleware order: `RequestID` → `Recoverer` → `CORS` → route-level `Auth`
- Commit convention: `feat(canvas):`, `fix(canvas):`, `test(canvas):`, `docs(canvas):`

## External Dependencies

| Repo | Purpose |
|------|---------|
| `mathtrail-contracts` | Protobuf schemas (`canvas/v1/stroke.proto`, `hint.proto`) |
| `mathtrail-charts` | Hosts `mathtrail-service-lib` library chart |
| `mathtrail-infra-streaming` | Centrifugo deployment (namespace `streaming`) |
| `mathtrail-infra-local-k3s` | k3d cluster setup |

## Pre-requisites (run on host before opening devcontainer)

```bash
# In mathtrail-infra-local-k3s repo:
just create        # Creates k3d cluster
just kubeconfig    # Saves kubeconfig to ~/.kube/k3d-mathtrail-dev.yaml

# Vault secrets (create manually before first deploy):
# secret/data/centrifugo/secrets  → token_hmac_secret_key, api_key
# secret/data/canvas-api/secrets  → centrifugo_api_key, centrifugo_hmac_key
# secret/data/local/streaming-automq-canvas-api → username, password

# Seed k3d registry before first skaffold dev:
just first-build
```
