// Package omnisignal provides an OmniSignal provider for Aha Ideas.
//
// This provider normalizes Aha Ideas to enhancement_request signals,
// preserving vote counts, categories, and other metadata.
//
// Usage:
//
//	import (
//	    "github.com/plexusone/omnisignal"
//	    _ "github.com/grokify/aha-studio/omnisignal" // Register Aha provider
//	)
//
//	provider, err := omnisignal.New("aha", omnisignal.Config{
//	    BaseURL: "https://company.aha.io",
//	    APIKey:  os.Getenv("AHA_API_KEY"),
//	})
package omnisignal

import (
	"context"
	"fmt"
	"strings"
	"time"

	aha "github.com/grokify/aha-go"
	"github.com/plexusone/omnisignal"
	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
)

const (
	ProviderName   = "aha"
	DefaultPerPage = 200
)

func init() {
	omnisignal.Register(ProviderName, NewProvider, omnisignal.PriorityThick)
}

// Provider implements omnisignal.Provider for Aha Ideas.
type Provider struct {
	client             *aha.Client
	config             omnisignal.Config
	customerMappings   map[string]string
	capabilityMappings map[string]string
}

// Config option keys for Aha provider.
const (
	OptSubdomain = "subdomain"
)

// NewProvider creates a new Aha provider.
func NewProvider(cfg omnisignal.Config) (omnisignal.Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("%w: APIKey is required", omnisignal.ErrInvalidConfig)
	}

	opts := []aha.Option{
		aha.WithAPIKey(cfg.APIKey),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, aha.WithBaseURL(cfg.BaseURL))
	}
	if subdomain := cfg.GetStringOption(OptSubdomain, ""); subdomain != "" {
		opts = append(opts, aha.WithSubdomain(subdomain))
	}

	client, err := aha.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating Aha client: %w", err)
	}

	return &Provider{
		client:             client,
		config:             cfg,
		customerMappings:   cfg.GetStringMap(omnisignal.OptCustomerMappings),
		capabilityMappings: cfg.GetStringMap(omnisignal.OptCapabilityMappings),
	}, nil
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return ProviderName
}

// Fetch retrieves ideas from Aha as signals.
func (p *Provider) Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]signal.Signal, error) {
	var signals []signal.Signal

	listOpts := []aha.ListIdeasOption{
		aha.WithIdeaPerPage(DefaultPerPage),
	}

	if !opts.Since.IsZero() {
		listOpts = append(listOpts, aha.WithIdeaCreatedSince(opts.Since))
	}
	if !opts.Until.IsZero() {
		listOpts = append(listOpts, aha.WithIdeaCreatedBefore(opts.Until))
	}
	if status, ok := opts.Filters["workflow_status"]; ok && status != "" {
		listOpts = append(listOpts, aha.WithIdeaStatus(status))
	}
	if query, ok := opts.Filters["query"]; ok && query != "" {
		listOpts = append(listOpts, aha.WithIdeaQuery(query))
	}

	page := 1
	for {
		pageOpts := append(listOpts, aha.WithIdeaPage(page))

		ideaList, err := p.client.ListIdeas(ctx, pageOpts...)
		if err != nil {
			return nil, omnisignal.WrapErrorByMessage(err, "listing ideas")
		}

		for _, idea := range ideaList.Ideas {
			sig := p.normalizeIdea(idea)
			signals = append(signals, sig)

			if opts.Limit > 0 && len(signals) >= opts.Limit {
				return signals, nil
			}
		}

		if ideaList.Pagination.CurrentPage >= ideaList.Pagination.TotalPages {
			break
		}
		page++
	}

	return signals, nil
}

// Subscribe is not supported for Aha.
func (p *Provider) Subscribe(ctx context.Context, opts omnisignal.SubscribeOptions) (<-chan signal.Signal, error) {
	return nil, omnisignal.ErrNotSupported
}

// Capabilities returns what this provider supports.
func (p *Provider) Capabilities() omnisignal.Capabilities {
	return omnisignal.Capabilities{
		SupportsStreaming:   false,
		SupportsBatchFetch:  true,
		SupportsFiltering:   true,
		SupportsAcknowledge: false,
		MaxBatchSize:        DefaultPerPage,
		RateLimitPerMinute:  300,
		SignalTypes: []signal.Type{
			signal.TypeEnhancementRequest,
		},
	}
}

// Close releases resources.
func (p *Provider) Close() error {
	return nil
}

// normalizeIdea converts an Aha Idea to a signal-spec Signal.
func (p *Provider) normalizeIdea(idea aha.Idea) signal.Signal {
	// Map workflow status to signal status
	status := signal.StatusNew
	if idea.WorkflowStatus != nil {
		status = mapWorkflowStatus(idea.WorkflowStatus.Name)
	}

	// Map votes to severity (rough heuristic)
	severity := mapVotesToSeverity(idea.Votes)

	// Build domain from categories
	domain := common.Domain{
		Name: "product",
	}
	if len(idea.Categories) > 0 {
		domain.Subdomain = normalizeCategory(idea.Categories[0].Name)
	}

	// Build entities from categories, applying capability mappings
	var entities []common.Entity
	var capabilityRefs []string
	for _, cat := range idea.Categories {
		entity := common.Entity{
			Type: "category",
			Name: cat.Name,
			Attributes: map[string]string{
				"aha_id": cat.ID,
			},
		}
		if ref, ok := p.capabilityMappings[cat.Name]; ok {
			entity.Ref = ref
			capabilityRefs = append(capabilityRefs, ref)
		}
		entities = append(entities, entity)
	}

	// Build metadata with enhancement signal conventions
	metadata := map[string]any{
		signal.MetaVotes:       idea.Votes,
		"aha_reference_num":    idea.ReferenceNum,
		"aha_score":            idea.Score,
		omnisignal.MetaCurated: true,
	}

	// Add capability refs
	if len(capabilityRefs) == 1 {
		metadata[signal.MetaCapabilityRef] = capabilityRefs[0]
	} else if len(capabilityRefs) > 1 {
		metadata[signal.MetaCapabilityRef] = capabilityRefs
	}

	// Add linked feature info
	if idea.Feature != nil {
		metadata["aha_feature_id"] = idea.Feature.ID
		metadata["aha_feature_ref"] = idea.Feature.ReferenceNum
		metadata["aha_feature_name"] = idea.Feature.Name
	}

	sig := signal.Signal{
		ID:     fmt.Sprintf("aha-%s", idea.ReferenceNum),
		Type:   signal.TypeEnhancementRequest,
		Status: status,
		Source: common.SourceSystem{
			Type:       "product_management",
			Name:       "aha",
			ExternalID: idea.ReferenceNum,
		},
		Domain:      domain,
		Severity:    severity,
		Summary:     idea.Name,
		Description: idea.Description,
		Entities:    entities,
		ObservedAt:  idea.CreatedAt,
		ReceivedAt:  time.Now(),
		Metadata:    metadata,
	}

	if fp, err := signal.ComputeFingerprint(sig); err == nil {
		sig.Fingerprint = fp
	}

	return sig
}

// mapWorkflowStatus maps Aha workflow status to signal status.
func mapWorkflowStatus(name string) signal.Status {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "shipped") || strings.Contains(lower, "complete") || strings.Contains(lower, "done"):
		return signal.StatusArchived
	case strings.Contains(lower, "progress") || strings.Contains(lower, "building") || strings.Contains(lower, "planned"):
		return signal.StatusProcessing
	case strings.Contains(lower, "won't") || strings.Contains(lower, "archived") || strings.Contains(lower, "declined"):
		return signal.StatusIgnored
	default:
		return signal.StatusNew
	}
}

// mapVotesToSeverity provides a rough severity based on vote count.
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

// normalizeCategory converts a category name to a valid subdomain.
func normalizeCategory(name string) string {
	result := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result += string(c)
		} else if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		} else if c == ' ' || c == '_' {
			result += "-"
		}
	}
	return result
}
