package omniroadmap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	omniroadmap "github.com/grokify/omniroadmap-core"
	"github.com/grokify/omniroadmap-core/provider"
)

// ListItems reads cached records for the requested kinds (or all supported
// kinds if req.Kinds is empty). The cache is local, so pagination is
// unnecessary — everything is returned in one response regardless of
// req.Page/PerPage.
func (p *Provider) ListItems(ctx context.Context, req *provider.ListItemsRequest) (*provider.ListItemsResponse, error) {
	kinds := req.Kinds
	if len(kinds) == 0 {
		kinds = p.Capabilities().Kinds
	}

	resp := &provider.ListItemsResponse{}
	for _, kind := range kinds {
		table, ok := entityTable(kind)
		if !ok {
			if !slices.Contains(p.Capabilities().Kinds, kind) {
				return nil, omniroadmap.ErrUnsupportedOperation
			}
			continue
		}
		records, err := p.db.ListRecords(ctx, table, p.product)
		if err != nil {
			return nil, wrapErr("ListRecords("+table+")", err)
		}
		for _, rec := range records {
			resp.Items = append(resp.Items, itemFromRecord(rec, kind))
		}
	}
	resp.TotalCount = len(resp.Items)
	return resp, nil
}

// GetItem fetches one cached record by kind and source ID (or Aha
// reference number).
func (p *Provider) GetItem(ctx context.Context, req *provider.GetItemRequest) (*provider.Item, error) {
	table, ok := entityTable(req.Kind)
	if !ok {
		return nil, omniroadmap.ErrUnsupportedOperation
	}
	rec, err := p.db.GetRecord(ctx, table, req.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s %q", omniroadmap.ErrNotFound, req.Kind, req.ID)
	}
	if err != nil {
		return nil, wrapErr("GetRecord("+table+")", err)
	}
	item := itemFromRecord(rec, req.Kind)
	return &item, nil
}

// ListReleases reads cached releases. Like ListItems, pagination is
// unnecessary for the local cache.
func (p *Provider) ListReleases(ctx context.Context, req *provider.ListReleasesRequest) (*provider.ListReleasesResponse, error) {
	records, err := p.db.ListRecords(ctx, "releases", p.product)
	if err != nil {
		return nil, wrapErr("ListRecords(releases)", err)
	}
	resp := &provider.ListReleasesResponse{}
	for _, rec := range records {
		resp.Releases = append(resp.Releases, releaseFromRecord(rec))
	}
	resp.TotalCount = len(resp.Releases)
	return resp, nil
}

// ListStatuses derives the distinct statuses present across cached items.
// Only status names are cached (no workflow definitions), so this reflects
// observed data, not the product's full workflow configuration.
func (p *Provider) ListStatuses(ctx context.Context, req *provider.ListStatusesRequest) (*provider.ListStatusesResponse, error) {
	kinds := p.Capabilities().Kinds
	if req.Kind != "" {
		kinds = []provider.ItemKind{req.Kind}
	}

	seen := map[string]bool{}
	resp := &provider.ListStatusesResponse{}
	for _, kind := range kinds {
		table, ok := entityTable(kind)
		if !ok {
			continue
		}
		records, err := p.db.ListRecords(ctx, table, p.product)
		if err != nil {
			return nil, wrapErr("ListRecords("+table+")", err)
		}
		for _, rec := range records {
			s := statusFromName(recString(rec, "status"))
			if s == nil || seen[s.Name] {
				continue
			}
			seen[s.Name] = true
			resp.Statuses = append(resp.Statuses, *s)
		}
	}
	return resp, nil
}

// ListCustomFieldDefinitions is unsupported: the cache stores per-record
// custom field values but not field definitions/schema. Per-record
// CustomFields on Items still work, so fieldmap-based MoSCoW/RICE mapping
// is unaffected.
func (p *Provider) ListCustomFieldDefinitions(ctx context.Context, req *provider.ListCustomFieldDefinitionsRequest) (*provider.ListCustomFieldDefinitionsResponse, error) {
	return nil, omniroadmap.ErrUnsupportedOperation
}

func wrapErr(op string, err error) error {
	return &opError{op: op, err: err}
}

type opError struct {
	op  string
	err error
}

func (e *opError) Error() string { return "omniroadmap/aha-studio: " + e.op + ": " + e.err.Error() }
func (e *opError) Unwrap() error { return e.err }
