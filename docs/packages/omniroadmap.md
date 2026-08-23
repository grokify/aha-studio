# omniroadmap Provider

The `omniroadmap/` package implements
[omniroadmap-core](https://github.com/grokify/omniroadmap-core)'s
`provider.Provider` interface over aha-studio's local SQLite cache,
registered as **`aha-studio`**. It lets the
[omniroadmap](https://github.com/grokify/omniroadmap) ecosystem — a
tool-agnostic canonical model for roadmap/PM data — read everything
aha-studio has synced **without any Aha API traffic**.

aha-studio remains the Aha-native system of record; this provider
generalizes its data into canonical `Item`/`Release` types for downstream
consumers (the omniroadmap Dolt store, prism-roadmap export, future
visualization).

## Usage

Via the omniroadmap CLI (most common):

```bash
# Sync the local cache into omniroadmap's canonical Dolt store
omniroadmap sync --provider aha-studio

# Scope to one product, from a specific cache file
omniroadmap sync --provider aha-studio --product PROJ --cache-db /path/to/cache.db
```

As a library:

```go
import (
    ahastudio "github.com/grokify/aha-studio/omniroadmap"
    studiosync "github.com/grokify/aha-studio/sync"
)

db, err := studiosync.Open(studiosync.DefaultDBPath()) // ~/.ahastudio/cache.db
p := ahastudio.NewProvider(db, ahastudio.WithProduct("PROJ"))

resp, err := p.ListItems(ctx, &provider.ListItemsRequest{})
```

## Capabilities

| Operation | Support |
|---|---|
| `ListItems` / `GetItem` | features, epics, initiatives (by ID or reference number) |
| `ListReleases` | ✓ |
| `ListStatuses` | Derived from observed item statuses (names only) |
| `ListCustomFieldDefinitions` | ✗ — definitions aren't cached; per-record custom fields **are** available |
| Writes | ✗ (read-only, like all omniroadmap providers) |

## Fidelity notes vs. the live `aha` provider

The cache stores a curated subset of each record, so a few things differ
from [aha-go](https://github.com/grokify/aha-go)'s live-API `aha` provider:

- **Status**: only the workflow status *name* is cached, so the canonical
  `StatusCategory` (todo/in_progress/done/canceled) is a name heuristic.
  The live provider has Aha's `Complete` flag and position for reliable
  normalization.
- **Custom fields**: present only for records fetched via *detailed* sync;
  lightweight-synced records have none. Run a detailed sync first if you
  need custom-field-based prioritization mapping (omniroadmap's fieldmap).
- **IDs**: canonical IDs are prefixed `aha-studio:` — syncing both this
  provider and the live `aha` provider into one store produces parallel
  copies, not merged records.

## Supporting read API

The provider is built on two exported `sync.DB` accessors added for this
purpose, useful in their own right:

```go
// All cached records of an entity type, optionally product-scoped, in the
// flattened shape (columns + data-JSON keys; custom_fields as an array of
// {key,name,value,type} maps):
records, err := db.ListRecords(ctx, "features", "PROJ")

// One record by ID or Aha reference number:
rec, err := db.GetRecord(ctx, "features", "PROJ-123")
```
