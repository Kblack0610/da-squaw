---
description: Migration Guide – From Monolith to 3-Layer
---

This guide shows how to migrate code that currently depends on the legacy `session.Instance` into the new **façade-based architecture** without a big-bang rewrite.

## 0. Checklist Before You Start
1. `core/` and `services/` compile without delivery code.
2. Circular imports are fixed (already done via `services/types`).
3. All new code obeys the import rules laid out in `docs/architecture.md`.

## 1. Identify Calls
`grep -R "instance\." ui/ cmd/ | sort -u` lists every method the UI/CLI currently calls.  Create a spreadsheet and group them into logical façades (Session, DiffViewer, GitOps …).

## 2. Create Façade Interfaces
Place them in `interface/facade/`.  Keep each interface small (Single Responsibility Principle).

```go
// interface/facade/diff.go
package facade

type DiffViewer interface {
    Diff(ctx context.Context) (string, error)
    Stats(ctx context.Context) (stats.Stats, error)
}
```

## 3. Implement in `coreadapter/`
For every façade, create a file in `interface/coreadapter/` that depends on orchestrator & services.

```go
func NewDiffViewer(orch *orchestrator.Orchestrator) facade.DiffViewer { … }
```

## 4. Wire in `main.go`
```go
viewer := coreadapter.NewDiffViewer(orch)
rootCmd.AddCommand(cmd.NewDiffCmd(viewer))
```

## 5. Refactor One Command/Widget at a Time
Replace the monolithic dependency with the façade. Compile & run tests after each command.

## 6. Delete Legacy Code
When no package imports `session.Instance`, remove the old code. Run `go test ./...` and smoke-test the app.

## 7. Celebrate 🎉

## FAQ
**Q: Can I keep adding methods to a façade?**  Prefer creating a new façade.  Large façades become de-facto God objects.

**Q: Should façades return domain structs or DTOs?**  Return domain structs defined in `core/` whenever possible to avoid duplication.
