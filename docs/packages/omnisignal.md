# omnisignal Package

The `omnisignal` package provides an [OmniSignal](https://github.com/plexusone/omnisignal) provider that normalizes Aha Ideas into `enhancement_request` signals for the ProductContext signal pipeline.

```go
import (
    "github.com/plexusone/omnisignal"
    _ "github.com/grokify/aha-studio/omnisignal" // Register the "aha" provider
)
```

## Overview

OmniSignal is a unified interface for ingesting operational signals (tickets, alerts, incidents, enhancement requests) from external systems into a canonical format for correlation and root-cause mapping. This package implements that interface for Aha!, so customer feedback captured as Aha Ideas can flow into the same pipeline as PagerDuty alerts, Jira tickets, and other signal sources.

The provider registers itself as `"aha"` via `init()` — importing the package for its side effect is enough to make it available through `omnisignal.New`/`omnisignal.MustNew`.

## Quick Start

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/plexusone/omnisignal"
    _ "github.com/grokify/aha-studio/omnisignal" // Register the "aha" provider
)

func main() {
    provider, err := omnisignal.New("aha", omnisignal.Config{
        APIKey: os.Getenv("AHA_API_KEY"),
        Options: map[string]any{
            "subdomain": "mycompany",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Close()

    signals, err := provider.Fetch(context.Background(), omnisignal.FetchOptions{
        Since: time.Now().AddDate(0, 0, -30),
        Limit: 100,
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, sig := range signals {
        log.Printf("%s: %s (%d votes)", sig.ID, sig.Summary, sig.Metadata["votes"])
    }
}
```

## Configuration

`omnisignal.Config` fields the Aha provider reads:

| Field | Required | Description |
|-------|----------|--------------|
| `APIKey` | Yes | Aha API key |
| `BaseURL` | No | Full API base URL override |
| `Options["subdomain"]` | No | Aha subdomain, used to build the API URL when `BaseURL` isn't set |
| `Options["capability_mappings"]` | No | `map[string]string` mapping Aha category names to capability refs (e.g. `"Integrations": "capability:integrations"`), attached to signals as `metadata["capability_ref"]` |
| `Options["customer_mappings"]` | No | Accepted for cross-provider consistency but not yet applied by this provider |

Either `BaseURL` or `Options["subdomain"]` must resolve to a valid Aha account; `aha.NewClient` also falls back to the `AHA_SUBDOMAIN` environment variable if neither is set.

## Fetching Signals

`Fetch` lists Aha Ideas and converts each to a `signal.Signal`, paginating internally until `FetchOptions.Limit` is reached or all pages are consumed:

| `FetchOptions` field | Maps to |
|-----------------------|---------|
| `Since` | `aha.WithIdeaCreatedSince` |
| `Until` | `aha.WithIdeaCreatedBefore` |
| `Limit` | Stops fetching once this many signals are collected |
| `Filters["workflow_status"]` | `aha.WithIdeaStatus` |
| `Filters["query"]` | `aha.WithIdeaQuery` |

`Subscribe` is not supported (Aha has no real-time push API for ideas) and returns `omnisignal.ErrNotSupported`.

## Signal Mapping

Each Aha Idea becomes one `signal.Signal` of type `signal.TypeEnhancementRequest`:

| Signal field | Source |
|--------------|--------|
| `ID` | `"aha-" + idea.ReferenceNum` |
| `Status` | Derived from `idea.WorkflowStatus.Name` — see status mapping below |
| `Source` | `{Type: "product_management", Name: "aha", ExternalID: idea.ReferenceNum}` |
| `Domain` | `{Name: "product", Subdomain: <normalized first category>}` |
| `Severity` | Derived from `idea.Votes` — see severity mapping below |
| `Summary` / `Description` | `idea.Name` / `idea.Description` |
| `Entities` | One `common.Entity{Type: "category", ...}` per Aha category, with `Ref` set when a `capability_mappings` entry matches |
| `ObservedAt` | `idea.CreatedAt` |
| `Metadata` | `votes`, `aha_reference_num`, `aha_score`, `curated: true`, `capability_ref` (if mapped), and `aha_feature_id`/`aha_feature_ref`/`aha_feature_name` when the idea is linked to a feature |
| `Fingerprint` | Computed via `signal.ComputeFingerprint`; silently left empty if computation fails |

### Workflow status → signal status

Substring match (case-insensitive) against the Aha workflow status name:

| Contains | Signal status |
|----------|----------------|
| `shipped`, `complete`, `done` | `signal.StatusArchived` |
| `progress`, `building`, `planned` | `signal.StatusProcessing` |
| `won't`, `archived`, `declined` | `signal.StatusIgnored` |
| (anything else, or no status) | `signal.StatusNew` |

### Votes → severity

| Votes | Severity |
|-------|----------|
| ≥ 50 | `common.SeverityCritical` |
| ≥ 20 | `common.SeverityHigh` |
| ≥ 5 | `common.SeverityMedium` |
| ≥ 1 | `common.SeverityLow` |
| 0 | `common.SeverityInfo` |

This is a rough, tunable heuristic — it isn't Aha's own prioritization data, just a way to give freshly-ingested signals a reasonable starting severity.

## Capabilities

```go
Capabilities{
    SupportsStreaming:   false,
    SupportsBatchFetch:  true,
    SupportsFiltering:   true,
    SupportsAcknowledge: false,
    MaxBatchSize:        200,
    RateLimitPerMinute:  300,
    SignalTypes:         []signal.Type{signal.TypeEnhancementRequest},
}
```

## API Reference

See [pkg.go.dev/github.com/grokify/aha-studio/omnisignal](https://pkg.go.dev/github.com/grokify/aha-studio/omnisignal) and the [omnisignal](https://pkg.go.dev/github.com/plexusone/omnisignal) / [signal-spec](https://pkg.go.dev/github.com/plexusone/signal-spec) packages it implements.
