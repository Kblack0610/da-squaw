---
description: Makefile docs target
---

If you want to regenerate diagrams or markdown docs, run:

```bash
make docs
```

This simply calls `go run internal/cmd/godocgen` (to be implemented) and stores outputs in `docs/`.
