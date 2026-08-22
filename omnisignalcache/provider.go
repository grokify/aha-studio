// Package omnisignalcache implements omnisignal.Provider over aha-studio's
// local SQLite cache (*sync.DB), registered as "aha-studio" -- distinct from
// aha-studio/omnisignal's live-API "aha" provider. Mirrors the same
// live/cached split already established by aha-go/omniroadmap ("aha") vs.
// aha-studio/omniroadmap ("aha-studio"), for the same reason: reading from
// the cache needs no Aha API traffic, so it can afford per-idea enrichment
// (voter/customer-org resolution) that would be an unacceptable N+1 cost
// against the live API.
//
// Fidelity caveats vs. the live "aha" provider:
//   - Only id/reference_num/name/status/description/votes/created_at/
//     updated_at are cached per idea today (see sync/sync.go's ideaToMap) --
//     no categories, workflow status detail, or linked feature, so Status
//     is always signal.StatusNew, Domain is always the flat "product"
//     default, and Entities is always empty. Closing this gap is tracked
//     separately, not part of this provider.
//   - Voter/customer-org tiering requires idea_organizations to have been
//     synced with SyncOptions.Detailed=true (only the single-GET response
//     carries email_domains) -- see sync/sync.go's syncIdeaOrganizations.
//     Without it, signal.MetaCustomers is empty and aha_voter_org_count is 0.
//   - No raw voter names/emails ever land in Signal.Metadata -- only
//     resolved typed refs and aggregates (aha_voter_domain_histogram).
package omnisignalcache

import (
	"context"
	"fmt"
	"strings"
	"time"

	studiosync "github.com/grokify/aha-studio/sync"
	"github.com/plexusone/omnisignal"
	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
)

const providerName = "aha-studio"

// Config option keys.
const (
	// OptDBPath is the path to the aha-studio SQLite cache, used by the
	// omnisignal.Register factory (NewProviderFromConfig). Not needed when
	// constructing directly via NewProvider with an already-open *sync.DB.
	OptDBPath = "db_path"
)

// Provider implements omnisignal.Provider over aha-studio's cache DB.
type Provider struct {
	db               *studiosync.DB
	product          string
	customerMappings map[string]string
}

// Option configures a Provider.
type Option func(*Provider)

// WithProduct scopes Fetch to a single Aha product (workspace) reference.
// Required -- unlike omniroadmap's cache-backed provider, GetIdeas needs a
// product to scope its query.
func WithProduct(product string) Option {
	return func(p *Provider) { p.product = product }
}

// WithCustomerMappings sets an override/fallback org-name -> customer-ref
// mapping, consulted when a voter's email domain doesn't resolve against
// any synced IdeaOrganization.email_domains. Same resolution shape the
// competitive provider already uses (provider/competitive/competitive.go),
// keyed by organization name here instead of account name.
func WithCustomerMappings(m map[string]string) Option {
	return func(p *Provider) { p.customerMappings = m }
}

// NewProvider wraps an opened cache DB as an omnisignal.Provider. Callers
// typically open the DB via sync.Open(sync.DefaultDBPath()).
func NewProvider(db *studiosync.DB, opts ...Option) *Provider {
	p := &Provider{db: db}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// NewProviderFromConfig adapts NewProvider to omnisignal's
// func(Config) (Provider, error) registration signature (unlike
// omniroadmap-core's registry, which accepts an opaque `config any` and so
// can take a *sync.DB directly, omnisignal.Register requires a
// Config-shaped factory). Opens its own *sync.DB from cfg.Options[OptDBPath].
// The caller owns the returned Provider's Close(), which closes the DB this
// function opened.
func NewProviderFromConfig(cfg omnisignal.Config) (omnisignal.Provider, error) {
	dbPath := cfg.GetStringOption(OptDBPath, "")
	if dbPath == "" {
		return nil, fmt.Errorf("%w: Options[%q] (db_path) is required", omnisignal.ErrInvalidConfig, OptDBPath)
	}
	db, err := studiosync.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening aha-studio cache at %s: %w", dbPath, err)
	}

	product := cfg.GetStringOption("product", "")
	if product == "" {
		return nil, fmt.Errorf("%w: Options[\"product\"] is required", omnisignal.ErrInvalidConfig)
	}

	return NewProvider(db,
		WithProduct(product),
		WithCustomerMappings(cfg.GetStringMap(omnisignal.OptCustomerMappings)),
	), nil
}

func init() {
	omnisignal.Register(providerName, NewProviderFromConfig, omnisignal.PriorityThick)
}

// Name returns the provider identifier.
func (p *Provider) Name() string { return providerName }

// Close closes the underlying cache DB. If the DB was passed in via
// NewProvider (not opened by NewProviderFromConfig), the caller may still
// own its lifecycle -- Close is safe to call once either way.
func (p *Provider) Close() error {
	return p.db.Close()
}

// Capabilities returns what this provider supports. Unlike the live "aha"
// provider, RateLimitPerMinute doesn't apply -- there are no live API calls;
// the cost was already paid by whatever sync populated the cache.
func (p *Provider) Capabilities() omnisignal.Capabilities {
	return omnisignal.Capabilities{
		SupportsStreaming:   false,
		SupportsBatchFetch:  true,
		SupportsFiltering:   false,
		SupportsAcknowledge: false,
		SignalTypes: []signal.Type{
			signal.TypeEnhancementRequest,
		},
	}
}

// Subscribe is not supported.
func (p *Provider) Subscribe(ctx context.Context, opts omnisignal.SubscribeOptions) (<-chan signal.Signal, error) {
	return nil, omnisignal.ErrNotSupported
}

// Fetch retrieves ideas from the cache as signals, enriched with
// voter/customer-org tiering resolved from cached idea_endorsements and
// idea_organizations.
func (p *Provider) Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]signal.Signal, error) {
	if p.product == "" {
		return nil, fmt.Errorf("%w: WithProduct is required", omnisignal.ErrInvalidConfig)
	}

	ideas, err := p.db.GetIdeas(ctx, p.product, opts.Since)
	if err != nil {
		return nil, fmt.Errorf("getting cached ideas: %w", err)
	}

	orgsByDomain, err := p.db.GetIdeaOrganizationsByDomain(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting cached idea organizations: %w", err)
	}

	var signals []signal.Signal
	for _, idea := range ideas {
		sig, err := p.normalizeIdea(ctx, idea, orgsByDomain)
		if err != nil {
			return nil, err
		}
		signals = append(signals, sig)

		if opts.Limit > 0 && len(signals) >= opts.Limit {
			break
		}
	}

	return signals, nil
}

func (p *Provider) normalizeIdea(ctx context.Context, idea map[string]any, orgsByDomain map[string]studiosync.IdeaOrganizationSummary) (signal.Signal, error) {
	id, _ := idea["id"].(string)
	refNum, _ := idea["reference_num"].(string)
	name, _ := idea["name"].(string)
	description, _ := idea["description"].(string)
	votes, _ := idea["votes"].(int)
	createdAt, _ := idea["created_at"].(time.Time)

	endorsements, err := p.db.GetIdeaEndorsementsByIdea(ctx, id)
	if err != nil {
		return signal.Signal{}, fmt.Errorf("getting endorsements for idea %s: %w", id, err)
	}

	customerRefs := make(map[string]struct{})
	domainHistogram := make(map[string]int)
	resolvedOrgs := make(map[string]struct{})
	for _, e := range endorsements {
		email, _ := e["portal_user_email"].(string)
		domain := emailDomain(email)
		if domain == "" {
			continue
		}
		domainHistogram[domain]++

		if org, ok := orgsByDomain[domain]; ok {
			resolvedOrgs[org.ID] = struct{}{}
			ref := resolveCustomerRef(org.Name, p.customerMappings)
			if ref != "" {
				customerRefs[ref] = struct{}{}
			}
		}
	}

	metadata := map[string]any{
		signal.MetaVotes:       votes,
		"aha_reference_num":    refNum,
		"aha_voter_count":      len(endorsements),
		"aha_voter_org_count":  len(resolvedOrgs),
		omnisignal.MetaCurated: true,
	}
	if len(domainHistogram) > 0 {
		metadata["aha_voter_domain_histogram"] = domainHistogram
	}
	if len(customerRefs) > 0 {
		refs := make([]string, 0, len(customerRefs))
		for ref := range customerRefs {
			refs = append(refs, ref)
		}
		metadata[signal.MetaCustomers] = refs
	}

	sig := signal.Signal{
		ID:     fmt.Sprintf("aha-%s", refNum),
		Type:   signal.TypeEnhancementRequest,
		Status: signal.StatusNew, // no cached workflow status -- see package doc fidelity caveats
		Source: common.SourceSystem{
			Type:       "product_management",
			Name:       "aha",
			ExternalID: refNum,
		},
		Domain:      common.Domain{Name: "product"},
		Severity:    mapVotesToSeverity(votes),
		Summary:     name,
		Description: description,
		ObservedAt:  createdAt,
		ReceivedAt:  time.Now(),
		Metadata:    metadata,
	}

	if fp, err := signal.ComputeFingerprint(sig); err == nil {
		sig.Fingerprint = fp
	}

	return sig, nil
}

// emailDomain returns the lowercased part of an email address after "@",
// or "" if email isn't a plausible address.
func emailDomain(email string) string {
	i := strings.LastIndex(email, "@")
	if i < 0 || i == len(email)-1 {
		return ""
	}
	return strings.ToLower(email[i+1:])
}

// resolveCustomerRef turns an Aha idea organization name into a customer
// ref. Prefers an explicit override in customerMappings (matching
// provider/competitive's org-name -> ref convention); falls back to a
// synthetic ref derived from the org name so ideas with a resolved
// organization always get *some* ref, even without a curated mapping.
func resolveCustomerRef(orgName string, customerMappings map[string]string) string {
	if orgName == "" {
		return ""
	}
	if ref, ok := customerMappings[orgName]; ok {
		return ref
	}
	return "customer:" + slugify(orgName)
}

func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case !prevDash:
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// mapVotesToSeverity provides a rough severity based on vote count, mirroring
// aha-studio/omnisignal's mapVotesToSeverity (duplicated rather than shared
// -- see package doc comment on the omniroadmap-precedented decision to
// tolerate small fidelity divergence between live and cached providers
// instead of an export/shared-internal-package dependency between them).
func mapVotesToSeverity(votes int) common.Severity {
	switch {
	case votes >= 50:
		return common.SeverityCritical
	case votes >= 20:
		return common.SeverityHigh
	case votes >= 5:
		return common.SeverityMedium
	case votes >= 1:
		return common.SeverityLow
	default:
		return common.SeverityInfo
	}
}
