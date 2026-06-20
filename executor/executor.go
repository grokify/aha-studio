// Package executor executes AQL query plans against the Aha API.
package executor

import (
	"context"
	"fmt"
	"sort"

	aha "github.com/grokify/aha-go"
	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/result"
)

// ProgressFunc is called during query execution to report progress.
// current is the current item count, total is the estimated total (0 if unknown),
// and message describes the current operation.
type ProgressFunc func(current, total int, message string)

// Executor executes query plans against the Aha API.
type Executor struct {
	client     *aha.Client
	progressFn ProgressFunc
}

// New creates a new executor with the given Aha client.
func New(client *aha.Client) *Executor {
	return &Executor{client: client}
}

// WithProgress sets a progress callback for the executor.
func (e *Executor) WithProgress(fn ProgressFunc) *Executor {
	e.progressFn = fn
	return e
}

// reportProgress calls the progress callback if set.
func (e *Executor) reportProgress(current, total int, message string) {
	if e.progressFn != nil {
		e.progressFn(current, total, message)
	}
}

// Execute executes a query plan and returns the results.
func (e *Executor) Execute(ctx context.Context, plan *planner.Plan) (*result.Result, error) {
	// Execute subqueries first and resolve their results
	if len(plan.Subqueries) > 0 {
		if err := e.executeSubqueries(ctx, plan); err != nil {
			return nil, fmt.Errorf("executing subqueries: %w", err)
		}
	}

	var records []result.Record
	var err error

	switch plan.Entity {
	case ast.EntityComments:
		records, err = e.executeComments(ctx, plan)
	case ast.EntityEpics:
		records, err = e.executeEpics(ctx, plan)
	case ast.EntityFeatures:
		records, err = e.executeFeatures(ctx, plan)
	case ast.EntityGoals:
		records, err = e.executeGoals(ctx, plan)
	case ast.EntityIdeas:
		records, err = e.executeIdeas(ctx, plan)
	case ast.EntityReleases:
		records, err = e.executeReleases(ctx, plan)
	case ast.EntityRequirements:
		records, err = e.executeRequirements(ctx, plan)
	case ast.EntityInitiatives:
		records, err = e.executeInitiatives(ctx, plan)
	case ast.EntityProducts:
		records, err = e.executeProducts(ctx, plan)
	case ast.EntityTags:
		records, err = e.executeTags(ctx, plan)
	case ast.EntityUsers:
		records, err = e.executeUsers(ctx, plan)
	default:
		return nil, fmt.Errorf("unsupported entity type: %s", plan.Entity)
	}

	if err != nil {
		return nil, err
	}

	// Apply JOINs
	if len(plan.Joins) > 0 {
		records, err = e.executeWithJoins(ctx, plan, records)
		if err != nil {
			return nil, err
		}
	}

	// Apply client-side filters
	if len(plan.ClientFilters) > 0 {
		records = e.applyFilters(records, plan.ClientFilters)
	}

	// Apply aggregations (GROUP BY and aggregate functions)
	if plan.HasAggregates {
		records = e.applyAggregations(records, plan)
	}

	// Apply HAVING clause (filter after aggregation)
	if plan.Having != nil {
		records = e.applyHaving(records, plan.Having)
	}

	// Apply sorting
	if plan.OrderBy != nil {
		records = e.sortRecords(records, plan.OrderBy)
	}

	// Apply limit
	if plan.Limit != nil && len(records) > *plan.Limit {
		records = records[:*plan.Limit]
	}

	// Apply field selection (only for non-aggregate queries)
	if !plan.HasAggregates && plan.SelectFields != nil {
		records = e.selectFields(records, plan.SelectFields)
	}

	return &result.Result{
		Entity:  plan.Entity,
		Records: records,
	}, nil
}

// executeFeatures fetches features from the Aha API.
func (e *Executor) executeFeatures(ctx context.Context, plan *planner.Plan) ([]result.Record, error) {
	// If custom fields are needed, use the detailed fetch mode
	if plan.NeedsCustomFields {
		return e.executeFeaturesWithCustomFields(ctx, plan)
	}

	buildOpts := func(page int) []aha.ListFeaturesOption {
		var opts []aha.ListFeaturesOption

		opts = append(opts, aha.WithFeaturePage(page), aha.WithFeaturePerPage(100))

		if plan.APIParams.Query != "" {
			opts = append(opts, aha.WithFeatureQuery(plan.APIParams.Query))
		}
		if plan.APIParams.Tag != "" {
			opts = append(opts, aha.WithFeatureTag(plan.APIParams.Tag))
		}
		if plan.APIParams.AssignedToUser != "" {
			opts = append(opts, aha.WithFeatureAssignee(plan.APIParams.AssignedToUser))
		}
		if plan.APIParams.UpdatedSince != nil {
			opts = append(opts, aha.WithFeatureUpdatedSince(*plan.APIParams.UpdatedSince))
		}

		return opts
	}

	var records []result.Record

	if plan.RequiresPagination {
		// Fetch all pages
		page := 1
		var totalPages int64
		for {
			e.reportProgress(len(records), 0, fmt.Sprintf("Fetching features page %d...", page))

			opts := buildOpts(page)
			list, err := e.client.ListFeatures(ctx, opts...)
			if err != nil {
				return nil, fmt.Errorf("listing features: %w", err)
			}

			if totalPages == 0 {
				totalPages = list.Pagination.TotalPages
			}

			for _, f := range list.Features {
				records = append(records, featureMetaToRecord(f))
			}

			e.reportProgress(len(records), int(totalPages)*100, fmt.Sprintf("Fetched page %d/%d", page, totalPages))

			if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
				break
			}
			page++
		}
	} else {
		// Single page fetch
		e.reportProgress(0, 0, "Fetching features...")
		opts := buildOpts(1)
		list, err := e.client.ListFeatures(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("listing features: %w", err)
		}

		for _, f := range list.Features {
			records = append(records, featureMetaToRecord(f))
		}
		e.reportProgress(len(records), len(records), "Features fetched")
	}

	return records, nil
}

// executeFeaturesWithCustomFields fetches features with full details including custom fields.
// This requires fetching each feature individually to get custom field values.
func (e *Executor) executeFeaturesWithCustomFields(ctx context.Context, plan *planner.Plan) ([]result.Record, error) {
	buildOpts := func() []aha.ListFeaturesOption {
		var opts []aha.ListFeaturesOption

		if plan.APIParams.Query != "" {
			opts = append(opts, aha.WithFeatureQuery(plan.APIParams.Query))
		}
		if plan.APIParams.Tag != "" {
			opts = append(opts, aha.WithFeatureTag(plan.APIParams.Tag))
		}
		if plan.APIParams.AssignedToUser != "" {
			opts = append(opts, aha.WithFeatureAssignee(plan.APIParams.AssignedToUser))
		}
		if plan.APIParams.UpdatedSince != nil {
			opts = append(opts, aha.WithFeatureUpdatedSince(*plan.APIParams.UpdatedSince))
		}

		return opts
	}

	// First, list all feature IDs
	var featureIDs []string
	page := 1
	for {
		opts := buildOpts()
		list, err := e.client.ListFeatures(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("listing features: %w", err)
		}

		for _, f := range list.Features {
			featureIDs = append(featureIDs, f.ID)
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	// Now fetch each feature individually to get custom fields
	var records []result.Record
	for _, id := range featureIDs {
		feature, err := e.client.GetFeature(ctx, id)
		if err != nil {
			// Log and continue on individual fetch failures
			continue
		}
		records = append(records, featureToRecord(feature))
	}

	return records, nil
}

// executeIdeas fetches ideas from the Aha API.
func (e *Executor) executeIdeas(ctx context.Context, plan *planner.Plan) ([]result.Record, error) {
	buildOpts := func(page int) []aha.ListIdeasOption {
		var opts []aha.ListIdeasOption

		opts = append(opts, aha.WithIdeaPage(page), aha.WithIdeaPerPage(100))

		if plan.APIParams.Query != "" {
			opts = append(opts, aha.WithIdeaQuery(plan.APIParams.Query))
		}
		if plan.APIParams.Tag != "" {
			opts = append(opts, aha.WithIdeaTag(plan.APIParams.Tag))
		}
		if plan.APIParams.WorkflowStatus != "" {
			opts = append(opts, aha.WithIdeaStatus(plan.APIParams.WorkflowStatus))
		}
		if plan.APIParams.UpdatedSince != nil {
			opts = append(opts, aha.WithIdeaUpdatedSince(*plan.APIParams.UpdatedSince))
		}
		if plan.APIParams.CreatedSince != nil {
			opts = append(opts, aha.WithIdeaCreatedSince(*plan.APIParams.CreatedSince))
		}

		return opts
	}

	var records []result.Record

	if plan.RequiresPagination {
		page := 1
		var totalPages int64
		for {
			e.reportProgress(len(records), 0, fmt.Sprintf("Fetching ideas page %d...", page))

			opts := buildOpts(page)
			list, err := e.client.ListIdeas(ctx, opts...)
			if err != nil {
				return nil, fmt.Errorf("listing ideas: %w", err)
			}

			if totalPages == 0 {
				totalPages = list.Pagination.TotalPages
			}

			for _, idea := range list.Ideas {
				records = append(records, ideaToRecord(idea))
			}

			e.reportProgress(len(records), int(totalPages)*100, fmt.Sprintf("Fetched page %d/%d", page, totalPages))

			if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
				break
			}
			page++
		}
	} else {
		e.reportProgress(0, 0, "Fetching ideas...")
		opts := buildOpts(1)
		list, err := e.client.ListIdeas(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("listing ideas: %w", err)
		}

		for _, idea := range list.Ideas {
			records = append(records, ideaToRecord(idea))
		}
		e.reportProgress(len(records), len(records), "Ideas fetched")
	}

	return records, nil
}

// executeReleases fetches releases from the Aha API.
func (e *Executor) executeReleases(ctx context.Context, plan *planner.Plan) ([]result.Record, error) {
	// Releases require a product ID
	if plan.APIParams.ProductID == "" {
		return nil, fmt.Errorf("releases query requires a product context (use --product flag)")
	}

	var records []result.Record

	if plan.RequiresPagination {
		page := 1
		var totalPages int64
		for {
			e.reportProgress(len(records), 0, fmt.Sprintf("Fetching releases page %d...", page))

			list, err := e.client.ListProductReleases(ctx, plan.APIParams.ProductID,
				aha.WithPage(page), aha.WithPerPage(100))
			if err != nil {
				return nil, fmt.Errorf("listing releases: %w", err)
			}

			if totalPages == 0 {
				totalPages = list.Pagination.TotalPages
			}

			for _, r := range list.Releases {
				records = append(records, releaseToRecord(r))
			}

			e.reportProgress(len(records), int(totalPages)*100, fmt.Sprintf("Fetched page %d/%d", page, totalPages))

			if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
				break
			}
			page++
		}
	} else {
		e.reportProgress(0, 0, "Fetching releases...")
		list, err := e.client.ListProductReleases(ctx, plan.APIParams.ProductID)
		if err != nil {
			return nil, fmt.Errorf("listing releases: %w", err)
		}

		for _, r := range list.Releases {
			records = append(records, releaseToRecord(r))
		}
		e.reportProgress(len(records), len(records), "Releases fetched")
	}

	return records, nil
}

// executeInitiatives fetches initiatives from the Aha API.
func (e *Executor) executeInitiatives(ctx context.Context, plan *planner.Plan) ([]result.Record, error) {
	buildOpts := func(page int) []aha.ListInitiativesOption {
		var opts []aha.ListInitiativesOption

		opts = append(opts, aha.WithInitiativePage(page), aha.WithInitiativePerPage(100))

		if plan.APIParams.Query != "" {
			opts = append(opts, aha.WithInitiativeQuery(plan.APIParams.Query))
		}
		if plan.APIParams.UpdatedSince != nil {
			opts = append(opts, aha.WithInitiativeUpdatedSince(*plan.APIParams.UpdatedSince))
		}

		return opts
	}

	var records []result.Record

	if plan.RequiresPagination {
		page := 1
		var totalPages int64
		for {
			e.reportProgress(len(records), 0, fmt.Sprintf("Fetching initiatives page %d...", page))

			opts := buildOpts(page)
			list, err := e.client.ListInitiatives(ctx, opts...)
			if err != nil {
				return nil, fmt.Errorf("listing initiatives: %w", err)
			}

			if totalPages == 0 {
				totalPages = list.Pagination.TotalPages
			}

			for _, i := range list.Initiatives {
				records = append(records, initiativeMetaToRecord(i))
			}

			e.reportProgress(len(records), int(totalPages)*100, fmt.Sprintf("Fetched page %d/%d", page, totalPages))

			if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
				break
			}
			page++
		}
	} else {
		e.reportProgress(0, 0, "Fetching initiatives...")
		opts := buildOpts(1)
		list, err := e.client.ListInitiatives(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("listing initiatives: %w", err)
		}

		for _, i := range list.Initiatives {
			records = append(records, initiativeMetaToRecord(i))
		}
		e.reportProgress(len(records), len(records), "Initiatives fetched")
	}

	return records, nil
}

// applyFilters applies client-side filters to records.
func (e *Executor) applyFilters(records []result.Record, filters []planner.Filter) []result.Record {
	var filtered []result.Record
	for _, r := range records {
		if e.matchesFilters(r, filters) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// matchesFilters checks if a record matches all filters.
func (e *Executor) matchesFilters(r result.Record, filters []planner.Filter) bool {
	for _, f := range filters {
		if !matchFilter(r, f) {
			return false
		}
	}
	return true
}

// sortRecords sorts records by the specified field and direction.
func (e *Executor) sortRecords(records []result.Record, orderBy *planner.OrderBy) []result.Record {
	sort.Slice(records, func(i, j int) bool {
		vi := records[i].Get(orderBy.Field)
		vj := records[j].Get(orderBy.Field)

		cmp := compareValues(vi, vj)
		if orderBy.Direction == ast.SortDesc {
			cmp = -cmp
		}
		return cmp < 0
	})
	return records
}

// selectFields filters records to only include the specified fields.
func (e *Executor) selectFields(records []result.Record, fields []string) []result.Record {
	output := make([]result.Record, len(records))
	fieldSet := make(map[string]bool)
	for _, f := range fields {
		fieldSet[f] = true
	}

	for i, r := range records {
		filtered := make(result.Record)
		for k, v := range r {
			if fieldSet[k] {
				filtered[k] = v
			}
		}
		output[i] = filtered
	}
	return output
}

// Record conversion helpers

func featureMetaToRecord(f aha.FeatureMeta) result.Record {
	return result.Record{
		"id":            f.ID,
		"reference_num": f.ReferenceNum,
		"name":          f.Name,
		"url":           f.URL,
		"created_at":    f.CreatedAt,
	}
}

// featureToRecord converts a full Feature (with custom fields) to a Record.
func featureToRecord(f *aha.Feature) result.Record {
	rec := result.Record{
		"id":             f.ID,
		"reference_num":  f.ReferenceNum,
		"name":           f.Name,
		"description":    f.Description,
		"product_id":     f.ProductID,
		"url":            f.URL,
		"comments_count": f.CommentsCount,
		"work_units":     f.WorkUnits,
		"created_at":     f.CreatedAt,
		"tags":           f.Tags,
	}

	if f.StartDate != nil {
		rec["start_date"] = *f.StartDate
	}
	if f.DueDate != nil {
		rec["due_date"] = *f.DueDate
	}
	if f.UpdatedAt != nil {
		rec["updated_at"] = *f.UpdatedAt
	}
	if f.WorkflowStatus != nil {
		rec["status"] = f.WorkflowStatus.Name
		rec["workflow_status_id"] = f.WorkflowStatus.ID
	}
	if f.Release != nil {
		rec["release"] = f.Release.Name
		rec["release_id"] = f.Release.ID
		if f.Release.ReleaseDate != nil {
			rec["release_date"] = *f.Release.ReleaseDate
		}
	}
	if f.AssignedTo != nil {
		rec["assigned_to"] = f.AssignedTo.Name()
		rec["assigned_to_id"] = f.AssignedTo.ID
	}

	// Add custom fields with "custom." prefix
	for _, cf := range f.CustomFields {
		// Use the custom field key as the field name
		key := cf.Key
		if key == "" {
			key = cf.Name
		}
		rec.SetCustomField(key, cf.Value)
	}

	return rec
}

func ideaToRecord(i aha.Idea) result.Record {
	r := result.Record{
		"id":            i.ID,
		"reference_num": i.ReferenceNum,
		"name":          i.Name,
		"votes":         i.Votes,
		"created_at":    i.CreatedAt,
		"updated_at":    i.UpdatedAt,
	}
	if i.WorkflowStatus != nil {
		r["status"] = i.WorkflowStatus.Name
	}
	return r
}

func releaseToRecord(r aha.Release) result.Record {
	rec := result.Record{
		"id":            r.ID,
		"reference_num": r.ReferenceNum,
		"name":          r.Name,
		"released":      r.Released,
		"parking_lot":   r.ParkingLot,
		"url":           r.URL,
	}
	if r.StartDate != nil {
		rec["start_date"] = *r.StartDate
	}
	if r.ReleaseDate != nil {
		rec["release_date"] = *r.ReleaseDate
	}
	return rec
}

func initiativeMetaToRecord(i aha.InitiativeMeta) result.Record {
	return result.Record{
		"id":            i.ID,
		"reference_num": i.ReferenceNum,
		"name":          i.Name,
		"url":           i.URL,
		"created_at":    i.CreatedAt,
	}
}

// executeProducts fetches products from the Aha API.
//
//nolint:dupl // Pagination pattern is similar but type-specific API calls differ
func (e *Executor) executeProducts(ctx context.Context, plan *planner.Plan) ([]result.Record, error) {
	var records []result.Record

	if plan.RequiresPagination {
		page := 1
		for {
			list, err := e.client.ListProducts(ctx, aha.WithProductsPage(page), aha.WithProductsPerPage(100))
			if err != nil {
				return nil, fmt.Errorf("listing products: %w", err)
			}

			for _, p := range list.Products {
				records = append(records, productMetaToRecord(p))
			}

			if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
				break
			}
			page++
		}
	} else {
		list, err := e.client.ListProducts(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing products: %w", err)
		}

		for _, p := range list.Products {
			records = append(records, productMetaToRecord(p))
		}
	}

	return records, nil
}

// executeUsers fetches users from the Aha API.
//
//nolint:dupl // Pagination pattern is similar but type-specific API calls differ
func (e *Executor) executeUsers(ctx context.Context, plan *planner.Plan) ([]result.Record, error) {
	var records []result.Record

	if plan.RequiresPagination {
		page := 1
		for {
			list, err := e.client.ListUsers(ctx, aha.WithPage(page), aha.WithPerPage(100))
			if err != nil {
				return nil, fmt.Errorf("listing users: %w", err)
			}

			for _, u := range list.Users {
				records = append(records, userToRecord(u))
			}

			if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
				break
			}
			page++
		}
	} else {
		list, err := e.client.ListUsers(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing users: %w", err)
		}

		for _, u := range list.Users {
			records = append(records, userToRecord(u))
		}
	}

	return records, nil
}

func productMetaToRecord(p aha.ProductMeta) result.Record {
	return result.Record{
		"id":               p.ID,
		"reference_prefix": p.ReferencePrefix,
		"name":             p.Name,
		"product_line":     p.ProductLine,
		"created_at":       p.CreatedAt,
	}
}

func userToRecord(u aha.User) result.Record {
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

// executeTags aggregates tags from features and epics.
// Tags are a derived entity since the Aha API doesn't have a dedicated tags endpoint.
func (e *Executor) executeTags(ctx context.Context, _ *planner.Plan) ([]result.Record, error) {
	// Map to track tag usage: name -> {feature_count, epic_count}
	type tagStats struct {
		featureCount int
		epicCount    int
	}
	tagMap := make(map[string]*tagStats)

	// Fetch features and extract tags
	featurePage := 1
	for {
		list, err := e.client.ListFeatures(ctx, aha.WithFeaturePage(featurePage), aha.WithFeaturePerPage(100))
		if err != nil {
			return nil, fmt.Errorf("listing features for tags: %w", err)
		}

		// Need to fetch full feature details to get tags
		for _, f := range list.Features {
			feature, err := e.client.GetFeature(ctx, f.ID)
			if err != nil {
				continue // Skip on error
			}
			for _, tag := range feature.Tags {
				if tag == "" {
					continue
				}
				if _, ok := tagMap[tag]; !ok {
					tagMap[tag] = &tagStats{}
				}
				tagMap[tag].featureCount++
			}
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		featurePage++
	}

	// Fetch epics and extract tags
	epicPage := 1
	for {
		list, err := e.client.ListEpics(ctx, aha.WithEpicPage(epicPage), aha.WithEpicPerPage(100))
		if err != nil {
			return nil, fmt.Errorf("listing epics for tags: %w", err)
		}

		// Need to fetch full epic details to get tags
		for _, ep := range list.Epics {
			epic, err := e.client.GetEpic(ctx, ep.ID)
			if err != nil {
				continue // Skip on error
			}
			for _, tag := range epic.Tags {
				if tag == "" {
					continue
				}
				if _, ok := tagMap[tag]; !ok {
					tagMap[tag] = &tagStats{}
				}
				tagMap[tag].epicCount++
			}
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		epicPage++
	}

	// Convert to records
	var records []result.Record
	for name, stats := range tagMap {
		records = append(records, result.Record{
			"name":          name,
			"feature_count": stats.featureCount,
			"epic_count":    stats.epicCount,
			"total_count":   stats.featureCount + stats.epicCount,
		})
	}

	return records, nil
}

// executeGoals fetches goals from the Aha API.
func (e *Executor) executeGoals(ctx context.Context, plan *planner.Plan) ([]result.Record, error) {
	// If custom fields are needed, use the detailed fetch mode
	if plan.NeedsCustomFields {
		return e.executeGoalsWithCustomFields(ctx, plan)
	}

	var records []result.Record

	if plan.RequiresPagination {
		page := 1
		for {
			list, err := e.client.ListGoals(ctx, aha.WithGoalPage(page), aha.WithGoalPerPage(100))
			if err != nil {
				return nil, fmt.Errorf("listing goals: %w", err)
			}

			for _, g := range list.Goals {
				records = append(records, goalMetaToRecord(g))
			}

			if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
				break
			}
			page++
		}
	} else {
		list, err := e.client.ListGoals(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing goals: %w", err)
		}

		for _, g := range list.Goals {
			records = append(records, goalMetaToRecord(g))
		}
	}

	return records, nil
}

// executeGoalsWithCustomFields fetches goals with full details including custom fields.
func (e *Executor) executeGoalsWithCustomFields(ctx context.Context, _ *planner.Plan) ([]result.Record, error) {
	// First, list all goal IDs
	var goalIDs []string
	page := 1
	for {
		list, err := e.client.ListGoals(ctx, aha.WithGoalPage(page), aha.WithGoalPerPage(100))
		if err != nil {
			return nil, fmt.Errorf("listing goals: %w", err)
		}

		for _, g := range list.Goals {
			goalIDs = append(goalIDs, g.ID)
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	// Now fetch each goal individually to get custom fields
	var records []result.Record
	for _, id := range goalIDs {
		goal, err := e.client.GetGoal(ctx, id)
		if err != nil {
			// Log and continue on individual fetch failures
			continue
		}
		records = append(records, goalToRecord(goal))
	}

	return records, nil
}

func goalMetaToRecord(g aha.GoalMeta) result.Record {
	return result.Record{
		"id":            g.ID,
		"reference_num": g.ReferenceNum,
		"name":          g.Name,
		"url":           g.URL,
		"created_at":    g.CreatedAt,
	}
}

// goalToRecord converts a full Goal (with custom fields) to a Record.
func goalToRecord(g *aha.Goal) result.Record {
	rec := result.Record{
		"id":              g.ID,
		"reference_num":   g.ReferenceNum,
		"name":            g.Name,
		"description":     g.Description,
		"progress":        g.Progress,
		"progress_source": g.ProgressSource,
		"status":          g.Status,
		"url":             g.URL,
		"created_at":      g.CreatedAt,
	}

	if g.StartDate != nil {
		rec["start_date"] = *g.StartDate
	}
	if g.EndDate != nil {
		rec["end_date"] = *g.EndDate
	}
	if g.UpdatedAt != nil {
		rec["updated_at"] = *g.UpdatedAt
	}
	if g.TimeFrame != nil {
		rec["time_frame"] = g.TimeFrame.Name
		rec["time_frame_id"] = g.TimeFrame.ID
	}
	if g.WorkflowStatus != nil {
		rec["workflow_status"] = g.WorkflowStatus.Name
		rec["workflow_status_id"] = g.WorkflowStatus.ID
	}

	// Add custom fields with "custom." prefix
	for _, cf := range g.CustomFields {
		key := cf.Key
		if key == "" {
			key = cf.Name
		}
		rec.SetCustomField(key, cf.Value)
	}

	return rec
}

// executeEpics fetches epics from the Aha API.
//
//nolint:dupl // Pagination pattern is similar but type-specific API calls differ
func (e *Executor) executeEpics(ctx context.Context, plan *planner.Plan) ([]result.Record, error) {
	var records []result.Record

	if plan.RequiresPagination {
		page := 1
		for {
			list, err := e.client.ListEpics(ctx, aha.WithEpicPage(page), aha.WithEpicPerPage(100))
			if err != nil {
				return nil, fmt.Errorf("listing epics: %w", err)
			}

			for _, ep := range list.Epics {
				records = append(records, epicMetaToRecord(ep))
			}

			if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
				break
			}
			page++
		}
	} else {
		list, err := e.client.ListEpics(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing epics: %w", err)
		}

		for _, ep := range list.Epics {
			records = append(records, epicMetaToRecord(ep))
		}
	}

	return records, nil
}

func epicMetaToRecord(ep aha.EpicMeta) result.Record {
	return result.Record{
		"id":            ep.ID,
		"reference_num": ep.ReferenceNum,
		"name":          ep.Name,
		"url":           ep.URL,
		"created_at":    ep.CreatedAt,
	}
}

// executeRequirements fetches requirements from the Aha API.
// Requirements require a feature_id filter in the WHERE clause.
func (e *Executor) executeRequirements(ctx context.Context, plan *planner.Plan) ([]result.Record, error) {
	// Requirements must be fetched per-feature. Look for feature_id in filters.
	var featureID string
	for _, filter := range plan.ClientFilters {
		if filter.Field == "feature_id" && filter.Op == ast.OpEQ {
			if filter.Value != nil && filter.Value.Type == ast.ValueTypeString {
				featureID = filter.Value.String
				break
			}
		}
	}

	if featureID == "" {
		return nil, fmt.Errorf("requirements query requires feature_id filter (e.g., WHERE feature_id = 'FEAT-123')")
	}

	var records []result.Record

	if plan.RequiresPagination {
		page := 1
		for {
			list, err := e.client.ListFeatureRequirements(ctx, featureID, aha.WithPage(page), aha.WithPerPage(100))
			if err != nil {
				return nil, fmt.Errorf("listing requirements: %w", err)
			}

			for _, r := range list.Requirements {
				records = append(records, requirementMetaToRecord(r, featureID))
			}

			if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
				break
			}
			page++
		}
	} else {
		list, err := e.client.ListFeatureRequirements(ctx, featureID)
		if err != nil {
			return nil, fmt.Errorf("listing requirements: %w", err)
		}

		for _, r := range list.Requirements {
			records = append(records, requirementMetaToRecord(r, featureID))
		}
	}

	return records, nil
}

func requirementMetaToRecord(r aha.RequirementMeta, featureID string) result.Record {
	return result.Record{
		"id":            r.ID,
		"reference_num": r.ReferenceNum,
		"name":          r.Name,
		"feature_id":    featureID,
		"url":           r.URL,
		"created_at":    r.CreatedAt,
	}
}

// executeComments fetches comments from the Aha API.
// Comments require a context filter: product_id, feature_id, idea_id, release_id,
// initiative_id, epic_id, or goal_id in the WHERE clause.
func (e *Executor) executeComments(ctx context.Context, plan *planner.Plan) ([]result.Record, error) {
	// Look for context filters to determine which list endpoint to use
	var (
		productID    string
		featureID    string
		ideaID       string
		releaseID    string
		initiativeID string
		epicID       string
		goalID       string
	)

	for _, filter := range plan.ClientFilters {
		if filter.Op == ast.OpEQ && filter.Value != nil && filter.Value.Type == ast.ValueTypeString {
			switch filter.Field {
			case "product_id":
				productID = filter.Value.String
			case "feature_id":
				featureID = filter.Value.String
			case "idea_id":
				ideaID = filter.Value.String
			case "release_id":
				releaseID = filter.Value.String
			case "initiative_id":
				initiativeID = filter.Value.String
			case "epic_id":
				epicID = filter.Value.String
			case "goal_id":
				goalID = filter.Value.String
			}
		}
	}

	// Determine which endpoint to use based on filters
	var records []result.Record
	var contextID string
	var contextType string

	if productID != "" {
		contextID = productID
		contextType = "product"
	} else if featureID != "" {
		contextID = featureID
		contextType = "feature"
	} else if ideaID != "" {
		contextID = ideaID
		contextType = "idea"
	} else if releaseID != "" {
		contextID = releaseID
		contextType = "release"
	} else if initiativeID != "" {
		contextID = initiativeID
		contextType = "initiative"
	} else if epicID != "" {
		contextID = epicID
		contextType = "epic"
	} else if goalID != "" {
		contextID = goalID
		contextType = "goal"
	} else {
		return nil, fmt.Errorf("comments query requires a context filter (e.g., WHERE product_id = 'PROD' or feature_id = 'FEAT-123')")
	}

	if plan.RequiresPagination {
		page := 1
		for {
			list, err := e.fetchComments(ctx, contextType, contextID, page)
			if err != nil {
				return nil, err
			}

			for _, c := range list.Comments {
				records = append(records, commentMetaToRecord(c, contextType, contextID))
			}

			if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
				break
			}
			page++
		}
	} else {
		list, err := e.fetchComments(ctx, contextType, contextID, 0)
		if err != nil {
			return nil, err
		}

		for _, c := range list.Comments {
			records = append(records, commentMetaToRecord(c, contextType, contextID))
		}
	}

	return records, nil
}

// fetchComments fetches comments based on the context type and ID.
func (e *Executor) fetchComments(ctx context.Context, contextType, contextID string, page int) (*aha.CommentList, error) {
	var opts []aha.ListOption
	if page > 0 {
		opts = append(opts, aha.WithPage(page), aha.WithPerPage(100))
	}

	switch contextType {
	case "product":
		return e.client.ListProductComments(ctx, contextID, opts...)
	case "feature":
		return e.client.ListFeatureComments(ctx, contextID, opts...)
	case "idea":
		return e.client.ListIdeaComments(ctx, contextID, opts...)
	case "release":
		return e.client.ListReleaseComments(ctx, contextID, opts...)
	case "initiative":
		return e.client.ListInitiativeComments(ctx, contextID, opts...)
	case "epic":
		return e.client.ListEpicComments(ctx, contextID, opts...)
	case "goal":
		return e.client.ListGoalComments(ctx, contextID, opts...)
	default:
		return nil, fmt.Errorf("unknown context type: %s", contextType)
	}
}

func commentMetaToRecord(c aha.CommentMeta, contextType, contextID string) result.Record {
	record := result.Record{
		"id":         c.ID,
		"body":       c.Body,
		"url":        c.URL,
		"created_at": c.CreatedAt,
	}

	// Add context field based on type
	switch contextType {
	case "product":
		record["product_id"] = contextID
	case "feature":
		record["feature_id"] = contextID
	case "idea":
		record["idea_id"] = contextID
	case "release":
		record["release_id"] = contextID
	case "initiative":
		record["initiative_id"] = contextID
	case "epic":
		record["epic_id"] = contextID
	case "goal":
		record["goal_id"] = contextID
	}

	// Add user info if available
	if c.User != nil {
		record["user_id"] = c.User.ID
	}

	return record
}

// executeSubqueries executes all subqueries in a plan and stores their results.
func (e *Executor) executeSubqueries(ctx context.Context, plan *planner.Plan) error {
	for _, subquery := range plan.Subqueries {
		// Execute the subquery
		subResult, err := e.Execute(ctx, subquery.Plan)
		if err != nil {
			return fmt.Errorf("executing subquery %d: %w", subquery.Index, err)
		}

		// Extract the result value(s)
		var subqueryResult any
		if subquery.IsScalar {
			// For scalar subqueries, extract a single value
			subqueryResult = e.extractScalarResult(subResult)
		} else {
			// For list subqueries (IN), extract all values
			subqueryResult = e.extractListResult(subResult)
		}

		// Store the result in the corresponding filter
		for i := range plan.ClientFilters {
			if plan.ClientFilters[i].SubqueryIndex == subquery.Index {
				plan.ClientFilters[i].SubqueryResult = subqueryResult
			}
		}
	}
	return nil
}

// extractScalarResult extracts a single value from a subquery result.
// This handles aggregate results (AVG, SUM, COUNT, MIN, MAX).
func (e *Executor) extractScalarResult(res *result.Result) any {
	if len(res.Records) == 0 {
		return nil
	}

	record := res.Records[0]

	// For aggregate queries, look for common aggregate aliases
	// The aggregate result is typically the first value in the record
	for _, v := range record {
		return v
	}
	return nil
}

// extractListResult extracts a list of values from a subquery result.
// This is used for IN/NOT IN subqueries.
func (e *Executor) extractListResult(res *result.Result) []any {
	var values []any
	for _, record := range res.Records {
		// Take the first column from each record
		for _, v := range record {
			values = append(values, v)
			break // only take first column per record
		}
	}
	return values
}
