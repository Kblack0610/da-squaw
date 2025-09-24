---
description: Developer Handbook – Contribution & Testing
---

## Directory Map (TL;DR)
| Folder            | Purpose                                    | Can import                   |
| ----------------- | ------------------------------------------ | ---------------------------- |
| `core/`           | Pure business logic, orchestrator, types   | stdlib, **no delivery code** |
| `services/`       | Git, tmux, storage adapters                | stdlib + core types          |
| `interface/`      | Façade interfaces & concrete adapters      | core                         |
| `delivery/cmd/`   | Cobra CLI                                  | interface                    |
| `delivery/ui/`    | Bubbletea TUI                              | interface                    |
| `docs/`           | Project documentation                      | –                            |

## Running the App
```bash
# development
make dev
# release build
make build
```

## Testing Strategy
* **Unit tests** in `core/` and `services/` use mocks.
* **Integration tests** spin up real git / tmux when possible.
* **E2E** – `cmd_test/` runs cobra commands with a tmp workdir.

Run all tests:
```bash
make test
```

## Linting & Formatting
We use `golangci-lint`.  Run locally with:
```bash
make lint
```

CI blocks merges if lint fails.

## Adding a Feature – Checklist
1. Write/adjust a façade if new capabilities are required.
2. Add tests in `core/` or `services/`.
3. Update documentation under `docs/`.
4. Open PR and fill out the template.

## Release Flow
1. Merge to `main` triggers `build.yml`.  Artifacts are uploaded.
2. Tag with `vX.Y.Z` to trigger `goreleaser`.
