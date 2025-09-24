---
description: Project Architecture Overview
---

# Architecture Overview

This document explains the high-level structure of the code-base **after** the refactor that introduces a clean 3-layer separation.  It is intended for new contributors as well as for long-time maintainers that want to understand where code should live.

```
core/          – domain logic, pure & testable
interface/     – tiny façades that present *use-cases* to callers
  coreadapter/ – concrete façade implementations that delegate to core

# imports allowed  (← means “may import”)
core            ← 0 external project packages
interface/facade← core
coreadapter     ← core + services (tmux, git…)

delivery/      – user-facing entrypoints
  cmd/          – Cobra CLI
  ui/           – Bubbletea TUI (or future Web / Desktop front-ends)

delivery/* MAY import only packages under `interface/`.
```

## 1. `core/`
Pure Go packages that hold the **business rules**.  There must be **no** direct use of fmt, log, cobra, or UI libraries here.  Side-effects (git, tmux, fs) are hidden behind **service interfaces** so they can be mocked in unit tests.

## 2. `interface/`
Defines role-focused, minimal **façade interfaces** consumed by delivery code.  Example:

```go
// interface/facade/session.go
package facade

type Session interface {
    Start(ctx context.Context) error
    Kill(ctx context.Context) error
    Diff(ctx context.Context) (string, error)
    SendInput(ctx context.Context, input string) error
}
```

### `coreadapter/`
Concrete façade implementations that translate the above calls into orchestrator / service calls.  These are thin – usually <50 LOC – and have no knowledge of UI or CLI conventions.

## 3. `delivery/`
Everything the end-user runs or sees.  At the moment:

* `cmd/` – Cobra commands (e.g. `cs diff`) wire façades and parse flags.
* `ui/`  – Bubbletea widgets and overlays.

Future web or desktop front-ends also live here, consuming the same façades.

## Dependency Diagram
![architecture diagram](../assets/architecture.svg)

(If the image is missing, run `make docs` to regenerate diagrams.)

## Benefits
* **Clear dependency flow** – one-way arrows, no cycles.
* **Testability** – mock façades in UI tests, mock services in core tests.
* **Extensibility** – add a gRPC façade without touching core or UI.
* **Incremental migration** – old monolithic `session.Instance` can be removed piece-by-piece.
