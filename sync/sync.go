package sync

import (
	"context"
	"fmt"
	"time"

	genql "github.com/Khan/genqlient/graphql"
	aha "github.com/grokify/aha-go"
	"github.com/grokify/aha-go/graphql"
	"github.com/grokify/aha-go/graphql/generated"
	"github.com/grokify/aha-studio/result"
)

// ProgressFunc is called during sync to report progress.
// current is the current record count, total is the estimated total (0 if unknown),
// and message describes the current operation.
type ProgressFunc func(current, total int, message string)

// Syncer synchronizes Aha data to a local SQLite database.
type Syncer struct {
	db         *DB
	client     *aha.Client
	gqlClient  genql.Client
	useGraphQL bool
	progressFn ProgressFunc
}

// NewSyncer creates a new Syncer with the given database and Aha client.
func NewSyncer(db *DB, client *aha.Client) *Syncer {
	return &Syncer{db: db, client: client}
}

// NewSyncerWithGraphQL creates a new Syncer with GraphQL support for enhanced feature sync.
// The subdomain and apiKey are used to create a GraphQL client for fetching features with release info.
func NewSyncerWithGraphQL(db *DB, client *aha.Client, subdomain, apiKey string) *Syncer {
	return &Syncer{
		db:         db,
		client:     client,
		gqlClient:  graphql.NewGenqlientClient(subdomain, apiKey),
		useGraphQL: true,
	}
}

// WithProgress sets a progress callback for the syncer.
func (s *Syncer) WithProgress(fn ProgressFunc) *Syncer {
	s.progressFn = fn
	return s
}

// resolveProjectID resolves a reference prefix (like "PROJ") to an internal project ID.
// If the input is already an internal ID, it's returned as-is.
func (s *Syncer) resolveProjectID(ctx context.Context, refPrefix string) (string, error) {
	if s.gqlClient == nil {
		return refPrefix, nil // Can't resolve without GraphQL client
	}

	// Try to find the project by listing all projects and matching reference prefix
	page := 1
	perPage := 100
	for {
		resp, err := generated.ListProjects(ctx, s.gqlClient, &page, &perPage, nil, nil)
		if err != nil {
			return "", fmt.Errorf("listing projects: %w", err)
		}

		for _, p := range resp.Projects.Nodes {
			if p.ReferencePrefix == refPrefix {
				return p.Id, nil
			}
		}

		if resp.Projects.IsLastPage || resp.Projects.CurrentPage >= resp.Projects.TotalPages {
			break
		}
		page++
	}

	return "", fmt.Errorf("project with reference prefix %q not found", refPrefix)
}

// reportProgress calls the progress callback if set.
func (s *Syncer) reportProgress(current, total int, message string) {
	if s.progressFn != nil {
		s.progressFn(current, total, message)
	}
}

// SyncOptions configures a sync operation.
type SyncOptions struct {
	Product     string    // Product ID to sync
	Incremental bool      // Only sync changes since last sync
	Since       time.Time // Custom since time (overrides incremental lookup)
	Entities    []string  // Specific entities to sync (nil = all)
}

// SyncResult contains the result of a sync operation.
type SyncResult struct {
	Entity      string
	RecordCount int
	Duration    time.Duration
	Error       error
}

// SyncAll syncs all entities for a product.
func (s *Syncer) SyncAll(ctx context.Context, opts SyncOptions) ([]SyncResult, error) {
	entities := opts.Entities
	if len(entities) == 0 {
		entities = []string{"features", "ideas", "releases", "initiatives", "goals", "epics", "comments", "requirements"}
	}

	var results []SyncResult
	for _, entity := range entities {
		start := time.Now()
		count, err := s.syncEntity(ctx, entity, opts)
		results = append(results, SyncResult{
			Entity:      entity,
			RecordCount: count,
			Duration:    time.Since(start),
			Error:       err,
		})
	}

	return results, nil
}

// syncEntity syncs a single entity type.
func (s *Syncer) syncEntity(ctx context.Context, entity string, opts SyncOptions) (int, error) {
	// Determine since time
	var since time.Time
	if !opts.Since.IsZero() {
		since = opts.Since
	} else if opts.Incremental {
		var err error
		since, err = s.db.GetLastSync(entity, opts.Product)
		if err != nil {
			return 0, fmt.Errorf("getting last sync time: %w", err)
		}
	}

	var count int
	var err error

	switch entity {
	case "features":
		count, err = s.syncFeatures(ctx, opts.Product, since)
	case "ideas":
		count, err = s.syncIdeas(ctx, opts.Product, since)
	case "releases":
		count, err = s.syncReleases(ctx, opts.Product)
	case "initiatives":
		count, err = s.syncInitiatives(ctx, opts.Product, since)
	case "goals":
		count, err = s.syncGoals(ctx, opts.Product, since)
	case "epics":
		count, err = s.syncEpics(ctx, opts.Product, since)
	case "users":
		count, err = s.syncUsers(ctx)
	case "products":
		count, err = s.syncProducts(ctx)
	case "comments":
		count, err = s.syncComments(ctx, opts.Product)
	case "requirements":
		count, err = s.syncRequirements(ctx, opts.Product)
	default:
		return 0, fmt.Errorf("unknown entity: %s", entity)
	}

	if err != nil {
		return 0, err
	}

	// Update sync metadata
	if err := s.db.SetLastSync(entity, opts.Product, time.Now(), count); err != nil {
		return count, fmt.Errorf("updating sync metadata: %w", err)
	}

	return count, nil
}

// syncFeatures syncs features from the API.
// If GraphQL is enabled, it uses the GraphQL API to fetch features with release info.
func (s *Syncer) syncFeatures(ctx context.Context, product string, since time.Time) (int, error) {
	// Use GraphQL if available for richer feature data including release info
	if s.useGraphQL && s.gqlClient != nil {
		return s.syncFeaturesGraphQL(ctx, product)
	}
	return s.syncFeaturesREST(ctx, product, since)
}

// syncFeaturesREST syncs features using the REST API (original implementation).
//
//nolint:dupl // Pagination pattern is similar but type-specific API calls differ
func (s *Syncer) syncFeaturesREST(ctx context.Context, product string, since time.Time) (int, error) {
	var count int
	page := 1

	for {
		var opts []aha.ListFeaturesOption
		opts = append(opts, aha.WithFeaturePage(page), aha.WithFeaturePerPage(100))
		if !since.IsZero() {
			opts = append(opts, aha.WithFeatureUpdatedSince(since))
		}

		list, err := s.client.ListFeatures(ctx, opts...)
		if err != nil {
			return count, fmt.Errorf("listing features: %w", err)
		}

		for _, f := range list.Features {
			data := featureMetaToMap(f)
			if err := s.db.UpsertFeature(product, data); err != nil {
				return count, fmt.Errorf("upserting feature: %w", err)
			}
			count++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return count, nil
}

// syncFeaturesGraphQL syncs features using the GraphQL API to capture release info.
func (s *Syncer) syncFeaturesGraphQL(ctx context.Context, product string) (int, error) {
	// Resolve reference prefix to internal project ID
	s.reportProgress(0, 0, "Resolving project ID...")
	projectID, err := s.resolveProjectID(ctx, product)
	if err != nil {
		return 0, fmt.Errorf("resolving project ID: %w", err)
	}

	var count int
	page := 1
	perPage := 100
	totalPages := 0

	for {
		s.reportProgress(count, 0, fmt.Sprintf("Fetching features page %d...", page))

		resp, err := generated.ListFeatures(ctx, s.gqlClient, projectID, &page, &perPage, nil, nil)
		if err != nil {
			return count, fmt.Errorf("listing features via GraphQL: %w", err)
		}

		features := resp.Features
		if totalPages == 0 {
			totalPages = features.TotalPages
		}

		for _, f := range features.Nodes {
			data := graphqlFeatureToMap(f)
			if err := s.db.UpsertFeature(product, data); err != nil {
				return count, fmt.Errorf("upserting feature: %w", err)
			}

			// Create feature->release relationship if release exists
			releaseID := f.Release.Id
			if releaseID != "" {
				if err := s.db.UpsertRelationship("feature", f.Id, "BELONGS_TO_RELEASE", "release", releaseID, product); err != nil {
					return count, fmt.Errorf("upserting feature-release relationship: %w", err)
				}
			}

			count++
		}

		s.reportProgress(count, features.TotalCount, fmt.Sprintf("Synced page %d/%d (%d features)", page, totalPages, count))

		if features.IsLastPage || features.CurrentPage >= features.TotalPages {
			break
		}
		page++
	}

	return count, nil
}

// syncIdeas syncs ideas from the API.
//
//nolint:dupl // Pagination pattern is similar but type-specific API calls differ
func (s *Syncer) syncIdeas(ctx context.Context, product string, since time.Time) (int, error) {
	var count int
	page := 1

	for {
		var opts []aha.ListIdeasOption
		opts = append(opts, aha.WithIdeaPage(page), aha.WithIdeaPerPage(100))
		if !since.IsZero() {
			opts = append(opts, aha.WithIdeaUpdatedSince(since))
		}

		list, err := s.client.ListIdeas(ctx, opts...)
		if err != nil {
			return count, fmt.Errorf("listing ideas: %w", err)
		}

		for _, i := range list.Ideas {
			data := ideaToMap(i)
			if err := s.db.UpsertIdea(product, data); err != nil {
				return count, fmt.Errorf("upserting idea: %w", err)
			}
			count++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return count, nil
}

// syncReleases syncs releases from the API.
func (s *Syncer) syncReleases(ctx context.Context, product string) (int, error) {
	var count int
	page := 1

	for {
		list, err := s.client.ListProductReleases(ctx, product, aha.WithPage(page), aha.WithPerPage(100))
		if err != nil {
			return count, fmt.Errorf("listing releases: %w", err)
		}

		for _, r := range list.Releases {
			data := releaseToMap(r)
			if err := s.db.UpsertRelease(product, data); err != nil {
				return count, fmt.Errorf("upserting release: %w", err)
			}
			count++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return count, nil
}

// syncInitiatives syncs initiatives from the API.
//
//nolint:dupl // Pagination pattern is similar but type-specific API calls differ
func (s *Syncer) syncInitiatives(ctx context.Context, product string, since time.Time) (int, error) {
	var count int
	page := 1

	for {
		var opts []aha.ListInitiativesOption
		opts = append(opts, aha.WithInitiativePage(page), aha.WithInitiativePerPage(100))
		if !since.IsZero() {
			opts = append(opts, aha.WithInitiativeUpdatedSince(since))
		}

		list, err := s.client.ListInitiatives(ctx, opts...)
		if err != nil {
			return count, fmt.Errorf("listing initiatives: %w", err)
		}

		for _, i := range list.Initiatives {
			data := initiativeMetaToMap(i)
			if err := s.db.UpsertInitiative(product, data); err != nil {
				return count, fmt.Errorf("upserting initiative: %w", err)
			}
			count++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return count, nil
}

// syncGoals syncs goals from the API.
//
//nolint:dupl // Pagination pattern is similar but type-specific API calls differ
func (s *Syncer) syncGoals(ctx context.Context, product string, since time.Time) (int, error) {
	var count int
	page := 1

	for {
		var opts []aha.ListGoalsOption
		opts = append(opts, aha.WithGoalPage(page), aha.WithGoalPerPage(100))
		if !since.IsZero() {
			opts = append(opts, aha.WithGoalUpdatedSince(since))
		}

		list, err := s.client.ListGoals(ctx, opts...)
		if err != nil {
			return count, fmt.Errorf("listing goals: %w", err)
		}

		for _, g := range list.Goals {
			data := goalMetaToMap(g)
			if err := s.db.UpsertGoal(product, data); err != nil {
				return count, fmt.Errorf("upserting goal: %w", err)
			}
			count++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return count, nil
}

// syncEpics syncs epics from the API.
//
//nolint:dupl // Pagination pattern is similar but type-specific API calls differ
func (s *Syncer) syncEpics(ctx context.Context, product string, since time.Time) (int, error) {
	var count int
	page := 1

	for {
		var opts []aha.ListEpicsOption
		opts = append(opts, aha.WithEpicPage(page), aha.WithEpicPerPage(100))
		if !since.IsZero() {
			opts = append(opts, aha.WithEpicUpdatedSince(since))
		}

		list, err := s.client.ListEpics(ctx, opts...)
		if err != nil {
			return count, fmt.Errorf("listing epics: %w", err)
		}

		for _, e := range list.Epics {
			data := epicMetaToMap(e)
			if err := s.db.UpsertEpic(product, data); err != nil {
				return count, fmt.Errorf("upserting epic: %w", err)
			}
			count++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return count, nil
}

// syncUsers syncs users from the API.
func (s *Syncer) syncUsers(ctx context.Context) (int, error) {
	var count int
	page := 1

	for {
		list, err := s.client.ListUsers(ctx, aha.WithPage(page), aha.WithPerPage(100))
		if err != nil {
			return count, fmt.Errorf("listing users: %w", err)
		}

		for _, u := range list.Users {
			data := userToMap(u)
			if err := s.db.UpsertUser(data); err != nil {
				return count, fmt.Errorf("upserting user: %w", err)
			}
			count++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return count, nil
}

// syncProducts syncs products from the API.
func (s *Syncer) syncProducts(ctx context.Context) (int, error) {
	var count int
	page := 1

	for {
		list, err := s.client.ListProducts(ctx, aha.WithProductsPage(page), aha.WithProductsPerPage(100))
		if err != nil {
			return count, fmt.Errorf("listing products: %w", err)
		}

		for _, p := range list.Products {
			data := productMetaToMap(p)
			if err := s.db.UpsertProduct(data); err != nil {
				return count, fmt.Errorf("upserting product: %w", err)
			}
			count++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return count, nil
}

// syncComments syncs comments for a product from the API.
func (s *Syncer) syncComments(ctx context.Context, product string) (int, error) {
	var count int
	page := 1

	for {
		list, err := s.client.ListProductComments(ctx, product, aha.WithPage(page), aha.WithPerPage(100))
		if err != nil {
			return count, fmt.Errorf("listing comments: %w", err)
		}

		for _, c := range list.Comments {
			data := commentMetaToMap(c)
			if err := s.db.UpsertComment(product, data); err != nil {
				return count, fmt.Errorf("upserting comment: %w", err)
			}
			count++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return count, nil
}

// syncRequirements syncs requirements for all features in a product.
// Note: Requirements are fetched per-feature, so this requires features to be synced first.
func (s *Syncer) syncRequirements(ctx context.Context, product string) (int, error) {
	// Get all feature IDs from the database
	featureIDs, err := s.db.GetFeatureIDs(product)
	if err != nil {
		return 0, fmt.Errorf("getting feature IDs: %w", err)
	}

	var count int
	for _, featureID := range featureIDs {
		page := 1
		for {
			list, err := s.client.ListFeatureRequirements(ctx, featureID, aha.WithPage(page), aha.WithPerPage(100))
			if err != nil {
				// Skip features that don't have requirements access
				break
			}

			for _, r := range list.Requirements {
				data := requirementMetaToMap(r, featureID)
				if err := s.db.UpsertRequirement(product, data); err != nil {
					return count, fmt.Errorf("upserting requirement: %w", err)
				}
				count++

				// Create feature->requirement relationship
				if err := s.db.UpsertRelationship("feature", featureID, "HAS_REQUIREMENT", "requirement", r.ID, product); err != nil {
					return count, fmt.Errorf("upserting relationship: %w", err)
				}
			}

			if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
				break
			}
			page++
		}
	}

	return count, nil
}

// Conversion functions

func featureMetaToMap(f aha.FeatureMeta) map[string]any {
	return result.Record{
		"id":            f.ID,
		"reference_num": f.ReferenceNum,
		"name":          f.Name,
		"url":           f.URL,
		"created_at":    f.CreatedAt,
	}
}

func ideaToMap(i aha.Idea) map[string]any {
	return result.Record{
		"id":            i.ID,
		"reference_num": i.ReferenceNum,
		"name":          i.Name,
		"votes":         i.Votes,
		"created_at":    i.CreatedAt,
		"updated_at":    i.UpdatedAt,
	}
}

func releaseToMap(r aha.Release) map[string]any {
	rec := result.Record{
		"id":            r.ID,
		"reference_num": r.ReferenceNum,
		"name":          r.Name,
		"url":           r.URL,
		"released":      r.Released,
		"parking_lot":   r.ParkingLot,
	}
	if r.StartDate != nil {
		rec["start_date"] = *r.StartDate
	}
	if r.ReleaseDate != nil {
		rec["release_date"] = *r.ReleaseDate
	}
	return rec
}

func initiativeMetaToMap(i aha.InitiativeMeta) map[string]any {
	return result.Record{
		"id":            i.ID,
		"reference_num": i.ReferenceNum,
		"name":          i.Name,
		"url":           i.URL,
		"created_at":    i.CreatedAt,
	}
}

func goalMetaToMap(g aha.GoalMeta) map[string]any {
	return result.Record{
		"id":            g.ID,
		"reference_num": g.ReferenceNum,
		"name":          g.Name,
		"url":           g.URL,
		"created_at":    g.CreatedAt,
	}
}

func epicMetaToMap(e aha.EpicMeta) map[string]any {
	return result.Record{
		"id":            e.ID,
		"reference_num": e.ReferenceNum,
		"name":          e.Name,
		"url":           e.URL,
		"created_at":    e.CreatedAt,
	}
}

func userToMap(u aha.User) map[string]any {
	rec := result.Record{
		"id":         u.ID,
		"first_name": u.FirstName,
		"last_name":  u.LastName,
		"email":      u.Email,
		"role":       u.Role,
	}
	if u.CreatedAt != nil {
		rec["created_at"] = *u.CreatedAt
	}
	return rec
}

func productMetaToMap(p aha.ProductMeta) map[string]any {
	return result.Record{
		"id":               p.ID,
		"reference_prefix": p.ReferencePrefix,
		"name":             p.Name,
		"product_line":     p.ProductLine,
		"created_at":       p.CreatedAt,
	}
}

func commentMetaToMap(c aha.CommentMeta) map[string]any {
	rec := result.Record{
		"id":         c.ID,
		"body":       c.Body,
		"url":        c.URL,
		"resource":   c.Resource,
		"created_at": c.CreatedAt,
	}
	if c.User != nil {
		rec["user_id"] = c.User.ID
	}
	return rec
}

func requirementMetaToMap(r aha.RequirementMeta, featureID string) map[string]any {
	return result.Record{
		"id":            r.ID,
		"feature_id":    featureID,
		"reference_num": r.ReferenceNum,
		"name":          r.Name,
		"url":           r.URL,
		"resource":      r.Resource,
		"created_at":    r.CreatedAt,
	}
}

// graphqlFeatureToMap converts a GraphQL feature response to a map for database storage.
// This includes release information that is not available in the REST API FeatureMeta.
func graphqlFeatureToMap(f generated.ListFeaturesFeaturesFeaturePageNodesFeature) map[string]any {
	rec := result.Record{
		"id":            f.Id,
		"reference_num": f.ReferenceNum,
		"name":          f.Name,
		"position":      f.Position,
		"tag_list":      f.TagList,
		"workspace":     f.Project.Name,
		"status":        f.WorkflowStatus.Name,
		"created_at":    f.CreatedAt,
		"updated_at":    f.UpdatedAt,
	}

	// Add optional fields
	if f.DueDate != nil {
		rec["due_date"] = *f.DueDate
	}
	if f.StartDate != nil {
		rec["start_date"] = *f.StartDate
	}

	// Add assigned user
	if f.AssignedToUser != nil {
		rec["assigned_to"] = f.AssignedToUser.Name
	}

	// Add release info - this is the key enhancement
	release := f.Release
	if release.Id != "" {
		rec["release_id"] = release.Id
		rec["release"] = release.Name
	}

	return rec
}
