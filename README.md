# canvas-api

[![CI](https://github.com/MathTrail/canvas-api/actions/workflows/ci.yml/badge.svg)](https://github.com/MathTrail/canvas-api/actions)
[![Latest Release](https://img.shields.io/github/v/release/MathTrail/canvas-api?style=flat-square)](https://github.com/MathTrail/canvas-api/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/MathTrail/canvas-api)](https://github.com/MathTrail/canvas-api/blob/main/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/MathTrail/canvas-api)](https://goreportcard.com/report/github.com/MathTrail/canvas-api)

Canvas API is the real-time drawing and feedback service of the MathTrail platform. Students solve math problems by writing with a stylus on an interactive canvas — the service streams strokes to the event bus and delivers AI-generated hints back to the canvas in real time via WebSocket.

## Business Capabilities

- **Stroke Ingestion** — Receives stylus input (pen pressure, coordinates) from the browser and publishes strokes to AutoMQ for downstream AI analysis.
- **Real-time Hints** — Consumes AI-generated hint events and pushes them instantly to the student's canvas via Centrifugo WebSocket.
- **Secure Sessions** — Issues scoped Centrifugo JWT tokens bound to the student's Ory session, preventing channel spoofing.
- **Interactive Canvas UI** — Ships a Vite Module Federation Remote (React + Konva) consumed by the `ui-web` shell: task text on a protected read-only layer, drawing on a separate erasable layer, hint overlays on a third layer.

## System Architecture

[![AutoMQ](https://img.shields.io/badge/AutoMQ-FF6A00?style=for-the-badge&logo=apachekafka&logoColor=white)](https://www.automq.com/)
[![Centrifugo](https://img.shields.io/badge/Centrifugo-WebSocket-0070CC?style=for-the-badge&logo=websocket&logoColor=white)](https://centrifugal.dev/)
[![Protobuf](https://img.shields.io/badge/Protobuf-Binary-4285F4?style=for-the-badge&logo=google&logoColor=white)](https://protobuf.dev/)

[![Architecture: EDA](https://img.shields.io/badge/Architecture-Event--Driven-8A2BE2?style=for-the-badge&logo=eventstore)](https://aws.amazon.com/event-driven-architecture/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-326CE5?style=for-the-badge&logo=kubernetes&logoColor=white)](./infra/helm/canvas-api)
[![Vault](https://img.shields.io/badge/Vault-FF3E00?style=for-the-badge&logo=hashicorpvault&logoColor=white)](https://www.vaultproject.io/)

```mermaid
graph LR
    Student([Student UI\nReact MFE]) -- "session cookie" --> OK[Oathkeeper]

    subgraph CanvasService [Canvas API]
        direction TB
        App["canvas-api\n(Go / chi)"]
        UI["canvas-api/ui\n(React / Konva\nMFE Remote)"]
    end

    OK -- "forwards session" --> App
    UI -- "POST strokes\noctet-stream" --> App
    UI -- "GET token" --> App

    subgraph Bus [Event Bus]
        direction TB
        AMQ{AutoMQ}
    end

    App -- "canvas.strokes\nCanvasStrokeEvent\nProtobuf" --> AMQ
    AMQ -- "canvas.hints\nHintEvent\nProtobuf" --> App

    subgraph Realtime [Real-time Layer]
        CFG["Centrifugo\n(WebSocket + Protobuf)"]
    end

    App -- "HTTP Publish API\nb64data" --> CFG
    CFG -- "WS push\nHintEvent" --> UI

    subgraph Identity [Identity]
        Kratos["Ory Kratos"]
    end

    App -- "GET /sessions/whoami" --> Kratos

    subgraph Support [Infra Support]
        direction TB
        VSO["VSO"] --> KSec["K8s Secret"]
        Vault["Vault"] --> VSO
    end

    KSec -- "env vars\n(HMAC key, API key,\nSCRAM creds)" --> App

    %% Styling
    classDef svc fill:#5b21b6,stroke:#7c3aed,color:#fff
    classDef ui fill:#1e3a5f,stroke:#3b82f6,color:#fff
    classDef authCls fill:#b45309,stroke:#f59e0b,color:#fff
    classDef eventCls fill:#1c1917,stroke:#78716c,color:#fff
    classDef rtCls fill:#064e3b,stroke:#10b981,color:#fff
    classDef idCls fill:#374151,stroke:#9ca3af,color:#fff
    classDef secretCls fill:#7f1d1d,stroke:#ef4444,color:#fff
    classDef actorCls fill:#1e1b4b,stroke:#818cf8,color:#fff

    class App svc; class UI ui;
    class OK authCls;
    class AMQ eventCls;
    class CFG rtCls;
    class Kratos idCls;
    class Vault,VSO,KSec secretCls;
    class Student actorCls;
```

### Canvas Layers

The React UI uses a three-layer Konva Stage so that erasing never touches the task text:

```
┌─────────────────────────────────┐
│  Layer 3 — HintLayer            │  listening=false  server hint overlays
│  Layer 2 — DrawingLayer         │  listening=true   pen strokes + eraser
│  Layer 1 — TaskLayer            │  listening=false  task text (read-only)
└─────────────────────────────────┘
```

Each Konva layer is a separate DOM `<canvas>` element. The eraser uses `globalCompositeOperation: "destination-out"` which acts only within the DrawingLayer canvas, leaving the TaskLayer untouched.

## Development

All commands are run via `just`.

```bash
just first-build     # seed k3d registry before first skaffold run
just dev             # skaffold dev — hot-reload + port-forward :8080

cd ui && npm install
cd ui && npm run dev  # MFE dev server on :3001
```

## Debug

[Telepresence](https://www.telepresence.io/) intercepts live cluster traffic and routes it to your local process. `--mapped-namespaces all` ensures that `kratos.identity.svc` and `automq.streaming.svc` DNS names resolve correctly in the local environment.

```bash
just tp-intercept
go run ./cmd/server

just tp-stop
```

## Releases

```bash
git tag -a v0.1.0 -m "Release description"
git push origin v0.1.0
```

GitHub Actions will build binaries, generate a Changelog, and publish a GitHub Release.
