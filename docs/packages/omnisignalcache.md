# omnisignalcache Package

The `omnisignalcache` package implements
[omnisignal](https://github.com/plexusone/omnisignal)'s `Provider` interface
over aha-studio's local SQLite cache, registered as **`aha-studio`** —
distinct from [`omnisignal`](omnisignal.md)'s live-API `"aha"` provider.
Reading from the cache needs no Aha API traffic, so it can afford per-idea
enrichment (voter/customer-org resolution) that would be an unacceptable N+1
cost against the live API. This mirrors the same live/cached split already
established by [`aha-go/omniroadmap`](https://pkg.go.dev/github.com/grokify/aha-go/omniroadmap)
(`"aha"`) vs. [`aha-studio/omniroadmap`](omniroadmap.md) (`"aha-studio"`).

```go
import (
    "github.com/plexusone/omnisignal"
    _ "github.com/grokify/aha-studio/omnisignalcache" // Register the "aha-studio" provider
)
```

## Quick Start

```go
import (
    studiosync "github.com/grokify/aha-studio/sync"
    "github.com/grokify/aha-studio/omnisignalcache"
)

db, err := studiosync.Open(studiosync.DefaultDBPath())
if err != nil {
    log.Fatal(err)
}
defer db.Close()

provider := omnisignalcache.NewProvider(db, omnisignalcache.WithProduct("PROD"))

signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{Limit: 100})
```

Or via `omnisignal.New("aha-studio", ...)`, which resolves a DB path itself
from `Config.Options["db_path"]`:

```go
provider, err := omnisignal.New("aha-studio", omnisignal.Config{
    Options: map[string]any{
        "db_path": studiosync.DefaultDBPath(),
        "product": "PROD",
    },
})
```

## Voter and Customer-Org Tiering

`Fetch` resolves each idea's endorsements (voters) against synced
`idea_organizations` by email domain, populating:

- `aha_voter_count`, `aha_voter_org_count` — raw counts
- `aha_voter_domain_histogram` — votes per email domain
- `signal.MetaCustomers` — resolved customer refs, one per matched
  organization (via `WithCustomerMappings` override, or a synthetic
  `customer:<slugified-org-name>` ref as a fallback)

This requires `idea_endorsements` and `idea_organizations` (with
`SyncOptions.Detailed=true`, since only the single-GET response carries
`email_domains`) to have been synced first — see the `sync_data` MCP tool or
`aha-studio sync --entities idea_endorsements,idea_organizations --detailed`.
Without it, voter/org fields are empty or zero, not an error.

## Fidelity notes vs. the live `aha` provider

- Only `id`/`reference_num`/`name`/`status`/`description`/`votes`/
  `created_at`/`updated_at` are cached per idea today — no categories,
  workflow status detail, or linked feature, so `Status` is always
  `signal.StatusNew` and `Domain` is always the flat `"product"` default.
- No raw voter names or emails ever land in `Signal.Metadata` — only
  resolved typed refs and aggregates.

## Capabilities

```go
Capabilities{
    SupportsStreaming:   false,
    SupportsBatchFetch:  true,
    SupportsFiltering:   false,
    SupportsAcknowledge: false,
    SignalTypes:         []signal.Type{signal.TypeEnhancementRequest},
}
```

`Subscribe` is not supported and returns `omnisignal.ErrNotSupported`.

## API Reference

See [pkg.go.dev/github.com/grokify/aha-studio/omnisignalcache](https://pkg.go.dev/github.com/grokify/aha-studio/omnisignalcache).
