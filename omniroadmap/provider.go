// Package omniroadmap implements omniroadmap-core's provider.Provider over
// aha-studio's local SQLite cache (*sync.DB), registered as "aha-studio" —
// distinct from aha-go/omniroadmap's live-API "aha" provider. Syncing from
// the cache needs no Aha API traffic: aha-studio keeps owning Aha-native
// entities while downstream consumers work with generalized ones.
//
// Fidelity caveats vs. the live "aha" provider:
//   - Only the workflow status *name* is cached, so StatusCategory
//     normalization is a name heuristic (no Complete flag or Position).
//   - Custom fields are present only for records fetched via detailed sync.
//   - Custom field *definitions* aren't cached — ListCustomFieldDefinitions
//     returns ErrUnsupportedOperation (per-record CustomFields still work,
//     so fieldmap-based MoSCoW/RICE mapping is unaffected).
//
// Note the canonical IDs are prefixed "aha-studio:", so syncing both this
// provider and the live "aha" provider into the same store produces
// parallel copies, not merged records.
package omniroadmap

import (
	studiosync "github.com/grokify/aha-studio/sync"
	omniroadmap "github.com/grokify/omniroadmap-core"
	"github.com/grokify/omniroadmap-core/provider"
)

const providerName = "aha-studio"

// Provider implements provider.Provider over aha-studio's cache DB.
type Provider struct {
	db      *studiosync.DB
	product string
}

var _ provider.Provider = (*Provider)(nil)

// Option configures a Provider.
type Option func(*Provider)

// WithProduct scopes list operations to a single Aha product (workspace)
// reference. Empty means all cached products.
func WithProduct(product string) Option {
	return func(p *Provider) { p.product = product }
}

// NewProvider wraps an opened cache DB as a provider.Provider. Callers
// typically open the DB via sync.Open(sync.DefaultDBPath()).
func NewProvider(db *studiosync.DB, opts ...Option) *Provider {
	p := &Provider{db: db}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Provider) Name() string { return providerName }

// Close is a no-op — the caller owns the DB handle's lifecycle (it may be
// shared with other aha-studio components).
func (p *Provider) Close() error { return nil }

func (p *Provider) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		Kinds: []provider.ItemKind{
			provider.ItemKindFeature,
			provider.ItemKindEpic,
			provider.ItemKindInitiative,
		},
		SupportsReleases: true,
		// Per-record custom fields are cached (for detail-synced records);
		// definitions are not — see ListCustomFieldDefinitions.
		SupportsCustomFields: true,
		SupportsWrite:        false,
	}
}

func init() {
	_ = omniroadmap.RegisterProvider(providerName, func(config any) (provider.Provider, error) {
		db, ok := config.(*studiosync.DB)
		if !ok {
			return nil, omniroadmap.NewAPIError(providerName, 0, "invalid_config",
				"omniroadmap/aha-studio: expected *sync.DB config")
		}
		return NewProvider(db), nil
	})
}
