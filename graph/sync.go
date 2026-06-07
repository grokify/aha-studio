package graph

import (
	"context"
	"fmt"
	"time"

	aha "github.com/grokify/aha-go"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Syncer synchronizes Aha data to Neo4j graph database.
type Syncer struct {
	client    *Client
	ahaClient *aha.Client
}

// NewSyncer creates a new graph syncer.
func NewSyncer(client *Client, ahaClient *aha.Client) *Syncer {
	return &Syncer{
		client:    client,
		ahaClient: ahaClient,
	}
}

// SyncOptions configures a sync operation.
type SyncOptions struct {
	Product     string   // Product ID to sync
	Entities    []string // Specific entities to sync (nil = all)
	Incremental bool     // Only sync changes since last sync
}

// SyncResult contains the result of a sync operation.
type SyncResult struct {
	Entity       string
	NodesCreated int
	RelsCreated  int
	Duration     time.Duration
	Error        error
}

// SyncAll syncs all entities for a product.
func (s *Syncer) SyncAll(ctx context.Context, opts SyncOptions) ([]SyncResult, error) {
	entities := opts.Entities
	if len(entities) == 0 {
		entities = []string{"products", "users", "releases", "features", "epics", "initiatives", "goals", "ideas"}
	}

	var results []SyncResult
	for _, entity := range entities {
		start := time.Now()
		nodes, rels, err := s.syncEntity(ctx, entity, opts)
		results = append(results, SyncResult{
			Entity:       entity,
			NodesCreated: nodes,
			RelsCreated:  rels,
			Duration:     time.Since(start),
			Error:        err,
		})
	}

	return results, nil
}

// syncEntity syncs a single entity type.
func (s *Syncer) syncEntity(ctx context.Context, entity string, opts SyncOptions) (int, int, error) {
	switch entity {
	case "products":
		return s.syncProducts(ctx)
	case "users":
		return s.syncUsers(ctx)
	case "releases":
		return s.syncReleases(ctx, opts.Product)
	case "features":
		return s.syncFeatures(ctx, opts.Product)
	case "epics":
		return s.syncEpics(ctx, opts.Product)
	case "initiatives":
		return s.syncInitiatives(ctx, opts.Product)
	case "goals":
		return s.syncGoals(ctx, opts.Product)
	case "ideas":
		return s.syncIdeas(ctx, opts.Product)
	default:
		return 0, 0, fmt.Errorf("unknown entity: %s", entity)
	}
}

// syncProducts syncs all products.
func (s *Syncer) syncProducts(ctx context.Context) (int, int, error) {
	var nodeCount int
	page := 1

	for {
		list, err := s.ahaClient.ListProducts(ctx, aha.WithPage(page), aha.WithPerPage(100))
		if err != nil {
			return nodeCount, 0, fmt.Errorf("listing products: %w", err)
		}

		for _, p := range list.Products {
			if err := s.upsertProduct(ctx, p); err != nil {
				return nodeCount, 0, err
			}
			nodeCount++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return nodeCount, 0, nil
}

// syncUsers syncs all users.
func (s *Syncer) syncUsers(ctx context.Context) (int, int, error) {
	var nodeCount int
	page := 1

	for {
		list, err := s.ahaClient.ListUsers(ctx, aha.WithPage(page), aha.WithPerPage(100))
		if err != nil {
			return nodeCount, 0, fmt.Errorf("listing users: %w", err)
		}

		for _, u := range list.Users {
			if err := s.upsertUser(ctx, u); err != nil {
				return nodeCount, 0, err
			}
			nodeCount++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return nodeCount, 0, nil
}

// syncReleases syncs releases for a product.
func (s *Syncer) syncReleases(ctx context.Context, productID string) (int, int, error) {
	var nodeCount, relCount int
	page := 1

	for {
		list, err := s.ahaClient.ListProductReleases(ctx, productID, aha.WithPage(page), aha.WithPerPage(100))
		if err != nil {
			return nodeCount, relCount, fmt.Errorf("listing releases: %w", err)
		}

		for _, r := range list.Releases {
			if err := s.upsertRelease(ctx, r, productID); err != nil {
				return nodeCount, relCount, err
			}
			nodeCount++

			// Create relationship to product
			if err := s.client.CreateRelationshipIfExists(ctx, NodeProduct, productID, NodeRelease, r.ID, RelContains); err != nil {
				return nodeCount, relCount, err
			}
			relCount++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return nodeCount, relCount, nil
}

// syncFeatures syncs features.
//
//nolint:dupl // Pagination pattern is similar but type-specific API calls differ
func (s *Syncer) syncFeatures(ctx context.Context, productID string) (int, int, error) {
	var nodeCount, relCount int
	page := 1

	for {
		opts := []aha.ListFeaturesOption{
			aha.WithFeaturePage(page),
			aha.WithFeaturePerPage(100),
		}

		list, err := s.ahaClient.ListFeatures(ctx, opts...)
		if err != nil {
			return nodeCount, relCount, fmt.Errorf("listing features: %w", err)
		}

		for _, f := range list.Features {
			if err := s.upsertFeature(ctx, f, productID); err != nil {
				return nodeCount, relCount, err
			}
			nodeCount++

			// Create relationships
			if productID != "" {
				if err := s.client.CreateRelationshipIfExists(ctx, NodeProduct, productID, NodeFeature, f.ID, RelContains); err != nil {
					return nodeCount, relCount, err
				}
				relCount++
			}
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return nodeCount, relCount, nil
}

// syncEpics syncs epics.
//
//nolint:dupl // Pagination pattern is similar but type-specific API calls differ
func (s *Syncer) syncEpics(ctx context.Context, productID string) (int, int, error) {
	var nodeCount, relCount int
	page := 1

	for {
		opts := []aha.ListEpicsOption{
			aha.WithEpicPage(page),
			aha.WithEpicPerPage(100),
		}

		list, err := s.ahaClient.ListEpics(ctx, opts...)
		if err != nil {
			return nodeCount, relCount, fmt.Errorf("listing epics: %w", err)
		}

		for _, e := range list.Epics {
			if err := s.upsertEpic(ctx, e, productID); err != nil {
				return nodeCount, relCount, err
			}
			nodeCount++

			if productID != "" {
				if err := s.client.CreateRelationshipIfExists(ctx, NodeProduct, productID, NodeEpic, e.ID, RelContains); err != nil {
					return nodeCount, relCount, err
				}
				relCount++
			}
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return nodeCount, relCount, nil
}

// syncInitiatives syncs initiatives.
func (s *Syncer) syncInitiatives(ctx context.Context, productID string) (int, int, error) {
	var nodeCount, relCount int
	page := 1

	for {
		opts := []aha.ListInitiativesOption{
			aha.WithInitiativePage(page),
			aha.WithInitiativePerPage(100),
		}

		list, err := s.ahaClient.ListInitiatives(ctx, opts...)
		if err != nil {
			return nodeCount, relCount, fmt.Errorf("listing initiatives: %w", err)
		}

		for _, i := range list.Initiatives {
			if err := s.upsertInitiative(ctx, i, productID); err != nil {
				return nodeCount, relCount, err
			}
			nodeCount++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return nodeCount, relCount, nil
}

// syncGoals syncs goals.
//
//nolint:dupl // Pagination pattern is similar but type-specific API calls differ
func (s *Syncer) syncGoals(ctx context.Context, productID string) (int, int, error) {
	var nodeCount, relCount int
	page := 1

	for {
		opts := []aha.ListGoalsOption{
			aha.WithGoalPage(page),
			aha.WithGoalPerPage(100),
		}

		list, err := s.ahaClient.ListGoals(ctx, opts...)
		if err != nil {
			return nodeCount, relCount, fmt.Errorf("listing goals: %w", err)
		}

		for _, g := range list.Goals {
			if err := s.upsertGoal(ctx, g, productID); err != nil {
				return nodeCount, relCount, err
			}
			nodeCount++

			if productID != "" {
				if err := s.client.CreateRelationshipIfExists(ctx, NodeProduct, productID, NodeGoal, g.ID, RelContains); err != nil {
					return nodeCount, relCount, err
				}
				relCount++
			}
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return nodeCount, relCount, nil
}

// syncIdeas syncs ideas.
func (s *Syncer) syncIdeas(ctx context.Context, productID string) (int, int, error) {
	var nodeCount, relCount int
	page := 1

	for {
		opts := []aha.ListIdeasOption{
			aha.WithIdeaPage(page),
			aha.WithIdeaPerPage(100),
		}

		list, err := s.ahaClient.ListIdeas(ctx, opts...)
		if err != nil {
			return nodeCount, relCount, fmt.Errorf("listing ideas: %w", err)
		}

		for _, i := range list.Ideas {
			if err := s.upsertIdea(ctx, i, productID); err != nil {
				return nodeCount, relCount, err
			}
			nodeCount++
		}

		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || list.Pagination.TotalPages == 0 {
			break
		}
		page++
	}

	return nodeCount, relCount, nil
}

// Upsert functions

func (s *Syncer) upsertProduct(ctx context.Context, p aha.ProductMeta) error {
	_, err := s.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, MergeNode(NodeProduct, p.ID, nil), map[string]any{
			"id": p.ID,
			"props": map[string]any{
				"reference_prefix": p.ReferencePrefix,
				"name":             p.Name,
				"product_line":     p.ProductLine,
				"created_at":       timeValToStr(p.CreatedAt),
			},
		})
	})
	return err
}

func (s *Syncer) upsertUser(ctx context.Context, u aha.User) error {
	_, err := s.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, MergeNode(NodeUser, u.ID, nil), map[string]any{
			"id": u.ID,
			"props": map[string]any{
				"first_name": u.FirstName,
				"last_name":  u.LastName,
				"email":      u.Email,
				"role":       u.Role,
			},
		})
	})
	return err
}

func (s *Syncer) upsertRelease(ctx context.Context, r aha.Release, productID string) error {
	props := map[string]any{
		"reference_num": r.ReferenceNum,
		"name":          r.Name,
		"url":           r.URL,
		"released":      r.Released,
		"parking_lot":   r.ParkingLot,
		"product_id":    productID,
	}
	if r.StartDate != nil {
		props["start_date"] = *r.StartDate
	}
	if r.ReleaseDate != nil {
		props["release_date"] = *r.ReleaseDate
	}

	_, err := s.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, MergeNode(NodeRelease, r.ID, nil), map[string]any{
			"id":    r.ID,
			"props": props,
		})
	})
	return err
}

func (s *Syncer) upsertFeature(ctx context.Context, f aha.FeatureMeta, productID string) error {
	props := map[string]any{
		"reference_num": f.ReferenceNum,
		"name":          f.Name,
		"url":           f.URL,
		"product_id":    productID,
		"created_at":    timeValToStr(f.CreatedAt),
	}

	_, err := s.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, MergeNode(NodeFeature, f.ID, nil), map[string]any{
			"id":    f.ID,
			"props": props,
		})
	})
	return err
}

func (s *Syncer) upsertEpic(ctx context.Context, e aha.EpicMeta, productID string) error {
	props := map[string]any{
		"reference_num": e.ReferenceNum,
		"name":          e.Name,
		"url":           e.URL,
		"product_id":    productID,
		"created_at":    timeValToStr(e.CreatedAt),
	}

	_, err := s.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, MergeNode(NodeEpic, e.ID, nil), map[string]any{
			"id":    e.ID,
			"props": props,
		})
	})
	return err
}

func (s *Syncer) upsertInitiative(ctx context.Context, i aha.InitiativeMeta, _ string) error {
	props := map[string]any{
		"reference_num": i.ReferenceNum,
		"name":          i.Name,
		"url":           i.URL,
		"created_at":    timeValToStr(i.CreatedAt),
	}

	_, err := s.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, MergeNode(NodeInitiative, i.ID, nil), map[string]any{
			"id":    i.ID,
			"props": props,
		})
	})
	return err
}

func (s *Syncer) upsertGoal(ctx context.Context, g aha.GoalMeta, productID string) error {
	props := map[string]any{
		"reference_num": g.ReferenceNum,
		"name":          g.Name,
		"url":           g.URL,
		"product_id":    productID,
		"created_at":    timeValToStr(g.CreatedAt),
	}

	_, err := s.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, MergeNode(NodeGoal, g.ID, nil), map[string]any{
			"id":    g.ID,
			"props": props,
		})
	})
	return err
}

func (s *Syncer) upsertIdea(ctx context.Context, i aha.Idea, productID string) error {
	props := map[string]any{
		"reference_num": i.ReferenceNum,
		"name":          i.Name,
		"votes":         i.Votes,
		"product_id":    productID,
		"created_at":    timeValToStr(i.CreatedAt),
		"updated_at":    timeValToStr(i.UpdatedAt),
	}

	_, err := s.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, MergeNode(NodeIdea, i.ID, nil), map[string]any{
			"id":    i.ID,
			"props": props,
		})
	})
	return err
}

func timeValToStr(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
