// Package sync provides SQLite-based offline caching for Aha Studio.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	entschema "entgo.io/ent/dialect/sql/schema"

	"github.com/grokify/aha-studio/ent"
	"github.com/grokify/aha-studio/ent/feature"
	"github.com/grokify/aha-studio/ent/relationship"
	"github.com/grokify/aha-studio/ent/savedfilter"
	"github.com/grokify/aha-studio/ent/syncmeta"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database connection for Aha data caching. The 13
// core tables (features, initiatives, etc.) are managed by the generated
// Ent client; the FTS5 virtual tables/triggers stay hand-written SQL
// against the shared *sql.DB connection (Ent has no concept of SQLite
// virtual tables). See
// /Users/johnwang/.claude/plans/sprightly-waddling-platypus.md.
type DB struct {
	db      *sql.DB
	ent     *ent.Client
	dbPath  string
	product string
}

// DefaultDBPath returns the default database path.
func DefaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ahastudio", "cache.db")
}

// Open opens or creates a SQLite database at the given path.
func Open(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	// _pragma=foreign_keys(1): required for Ent's SQLite schema inspector to
	// run Schema.Create below (it needs to read the foreign_keys pragma to
	// introspect the existing schema, independent of whether Ent is asked to
	// emit new FK constraints via WithForeignKeys(false)). This is
	// modernc.org/sqlite's DSN convention, not the mattn/go-sqlite3 cgo
	// driver's `_fk=1`.
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	// Wire Ent over the same *sql.DB connection modernc.org/sqlite opened
	// above (not ent.Open(), which hardcodes the cgo mattn/go-sqlite3
	// driver name).
	drv := entsql.OpenDB(dialect.SQLite, db)
	entClient := ent.NewClient(ent.Driver(drv))

	d := &DB{db: db, ent: entClient, dbPath: dbPath}

	// Create the 13 Ent-managed tables (append-only: adds new
	// tables/columns/indexes, never drops - safe to run against an
	// existing populated database on every Open()). No edges/FK
	// constraints are modeled (see plan non-goals), hence
	// WithForeignKeys(false).
	if err := entClient.Schema.Create(context.Background(), entschema.WithForeignKeys(false)); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("creating ent schema: %w", err)
	}

	// Create the FTS5 virtual tables/triggers, which live outside Ent's
	// schema model entirely (SQLite-specific, no Ent equivalent).
	if err := d.createFTSSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("creating FTS schema: %w", err)
	}

	return d, nil
}

// Close closes the database connection (and the Ent client sharing it).
func (d *DB) Close() error {
	if err := d.ent.Close(); err != nil {
		return err
	}
	return d.db.Close()
}

// SetProduct sets the current product context.
func (d *DB) SetProduct(product string) {
	d.product = product
}

// createFTSSchema creates the FTS5 virtual tables and their sync triggers.
// The 13 core tables (sync_meta, features, initiatives, etc.) are created
// by the Ent client in Open() instead - FTS5 virtual tables have no Ent
// equivalent, so they stay hand-written SQL against the base tables Ent
// just created.
func (d *DB) createFTSSchema() error {
	// Create FTS5 tables for full-text search
	fts := `
	-- FTS5 virtual tables for full-text search
	CREATE VIRTUAL TABLE IF NOT EXISTS features_fts USING fts5(
		id UNINDEXED,
		name,
		description,
		content='features',
		content_rowid='rowid'
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS ideas_fts USING fts5(
		id UNINDEXED,
		name,
		description,
		content='ideas',
		content_rowid='rowid'
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS initiatives_fts USING fts5(
		id UNINDEXED,
		name,
		description,
		content='initiatives',
		content_rowid='rowid'
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS epics_fts USING fts5(
		id UNINDEXED,
		name,
		description,
		content='epics',
		content_rowid='rowid'
	);

	-- Triggers to keep FTS tables in sync
	CREATE TRIGGER IF NOT EXISTS features_ai AFTER INSERT ON features BEGIN
		INSERT INTO features_fts(rowid, id, name, description) VALUES (NEW.rowid, NEW.id, NEW.name, NEW.description);
	END;
	CREATE TRIGGER IF NOT EXISTS features_ad AFTER DELETE ON features BEGIN
		INSERT INTO features_fts(features_fts, rowid, id, name, description) VALUES('delete', OLD.rowid, OLD.id, OLD.name, OLD.description);
	END;
	CREATE TRIGGER IF NOT EXISTS features_au AFTER UPDATE ON features BEGIN
		INSERT INTO features_fts(features_fts, rowid, id, name, description) VALUES('delete', OLD.rowid, OLD.id, OLD.name, OLD.description);
		INSERT INTO features_fts(rowid, id, name, description) VALUES (NEW.rowid, NEW.id, NEW.name, NEW.description);
	END;

	CREATE TRIGGER IF NOT EXISTS ideas_ai AFTER INSERT ON ideas BEGIN
		INSERT INTO ideas_fts(rowid, id, name, description) VALUES (NEW.rowid, NEW.id, NEW.name, NEW.description);
	END;
	CREATE TRIGGER IF NOT EXISTS ideas_ad AFTER DELETE ON ideas BEGIN
		INSERT INTO ideas_fts(ideas_fts, rowid, id, name, description) VALUES('delete', OLD.rowid, OLD.id, OLD.name, OLD.description);
	END;
	CREATE TRIGGER IF NOT EXISTS ideas_au AFTER UPDATE ON ideas BEGIN
		INSERT INTO ideas_fts(ideas_fts, rowid, id, name, description) VALUES('delete', OLD.rowid, OLD.id, OLD.name, OLD.description);
		INSERT INTO ideas_fts(rowid, id, name, description) VALUES (NEW.rowid, NEW.id, NEW.name, NEW.description);
	END;

	CREATE TRIGGER IF NOT EXISTS initiatives_ai AFTER INSERT ON initiatives BEGIN
		INSERT INTO initiatives_fts(rowid, id, name, description) VALUES (NEW.rowid, NEW.id, NEW.name, NEW.description);
	END;
	CREATE TRIGGER IF NOT EXISTS initiatives_ad AFTER DELETE ON initiatives BEGIN
		INSERT INTO initiatives_fts(initiatives_fts, rowid, id, name, description) VALUES('delete', OLD.rowid, OLD.id, OLD.name, OLD.description);
	END;
	CREATE TRIGGER IF NOT EXISTS initiatives_au AFTER UPDATE ON initiatives BEGIN
		INSERT INTO initiatives_fts(initiatives_fts, rowid, id, name, description) VALUES('delete', OLD.rowid, OLD.id, OLD.name, OLD.description);
		INSERT INTO initiatives_fts(rowid, id, name, description) VALUES (NEW.rowid, NEW.id, NEW.name, NEW.description);
	END;

	CREATE TRIGGER IF NOT EXISTS epics_ai AFTER INSERT ON epics BEGIN
		INSERT INTO epics_fts(rowid, id, name, description) VALUES (NEW.rowid, NEW.id, NEW.name, NEW.description);
	END;
	CREATE TRIGGER IF NOT EXISTS epics_ad AFTER DELETE ON epics BEGIN
		INSERT INTO epics_fts(epics_fts, rowid, id, name, description) VALUES('delete', OLD.rowid, OLD.id, OLD.name, OLD.description);
	END;
	CREATE TRIGGER IF NOT EXISTS epics_au AFTER UPDATE ON epics BEGIN
		INSERT INTO epics_fts(epics_fts, rowid, id, name, description) VALUES('delete', OLD.rowid, OLD.id, OLD.name, OLD.description);
		INSERT INTO epics_fts(rowid, id, name, description) VALUES (NEW.rowid, NEW.id, NEW.name, NEW.description);
	END;
	`

	_, err := d.db.Exec(fts)
	return err
}

// GetLastSync returns the last sync time for an entity/product combination.
func (d *DB) GetLastSync(ctx context.Context, entity, product string) (time.Time, error) {
	meta, err := d.ent.SyncMeta.Query().
		Where(syncmeta.Entity(entity), syncmeta.Product(product)).
		Only(ctx)
	if ent.IsNotFound(err) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return meta.LastSync, nil
}

// SetLastSync updates the last sync time for an entity/product combination.
func (d *DB) SetLastSync(ctx context.Context, entity, product string, t time.Time, count int) error {
	return d.upsertSyncMetaEnt(ctx, entity, product, t, count)
}

// UpsertFeature inserts or updates a feature record.
func (d *DB) UpsertFeature(ctx context.Context, product string, data map[string]any) error {
	return d.upsertFeatureEnt(ctx, product, data)
}

// UpsertIdea inserts or updates an idea record.
func (d *DB) UpsertIdea(ctx context.Context, product string, data map[string]any) error {
	return d.upsertIdeaEnt(ctx, product, data)
}

// UpsertRelease inserts or updates a release record.
func (d *DB) UpsertRelease(ctx context.Context, product string, data map[string]any) error {
	return d.upsertReleaseEnt(ctx, product, data)
}

// UpsertInitiative inserts or updates an initiative record.
func (d *DB) UpsertInitiative(ctx context.Context, product string, data map[string]any) error {
	return d.upsertInitiativeEnt(ctx, product, data)
}

// UpsertGoal inserts or updates a goal record.
func (d *DB) UpsertGoal(ctx context.Context, product string, data map[string]any) error {
	return d.upsertGoalEnt(ctx, product, data)
}

// UpsertEpic inserts or updates an epic record.
func (d *DB) UpsertEpic(ctx context.Context, product string, data map[string]any) error {
	return d.upsertEpicEnt(ctx, product, data)
}

// UpsertUser inserts or updates a user record.
func (d *DB) UpsertUser(ctx context.Context, data map[string]any) error {
	return d.upsertUserEnt(ctx, data)
}

// UpsertProduct inserts or updates a product record.
func (d *DB) UpsertProduct(ctx context.Context, data map[string]any) error {
	return d.upsertProductEnt(ctx, data)
}

// UpsertIdeaUser inserts or updates an idea user (voter identity) record.
func (d *DB) UpsertIdeaUser(ctx context.Context, data map[string]any) error {
	return d.upsertIdeaUserEnt(ctx, data)
}

// UpsertIdeaOrganization inserts or updates an idea organization (customer/account) record.
func (d *DB) UpsertIdeaOrganization(ctx context.Context, data map[string]any) error {
	return d.upsertIdeaOrganizationEnt(ctx, data)
}

// tagsToString converts a tags value to a comma-separated string.
func tagsToString(v any) string {
	if v == nil {
		return ""
	}
	switch tags := v.(type) {
	case []string:
		return joinStrings(tags, ",")
	case string:
		return tags
	default:
		return fmt.Sprintf("%v", v)
	}
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += sep + s
	}
	return result
}

// UpsertComment inserts or updates a comment record.
func (d *DB) UpsertComment(ctx context.Context, product string, data map[string]any) error {
	return d.upsertCommentEnt(ctx, product, data)
}

// UpsertIdeaEndorsement inserts or updates an idea endorsement (vote) record.
func (d *DB) UpsertIdeaEndorsement(ctx context.Context, product string, data map[string]any) error {
	return d.upsertIdeaEndorsementEnt(ctx, product, data)
}

// UpsertRequirement inserts or updates a requirement record.
func (d *DB) UpsertRequirement(ctx context.Context, product string, data map[string]any) error {
	return d.upsertRequirementEnt(ctx, product, data)
}

// UpsertRelationship inserts or updates a relationship record.
func (d *DB) UpsertRelationship(ctx context.Context, fromType, fromID, relType, toType, toID, product string) error {
	return d.upsertRelationshipEnt(ctx, fromType, fromID, relType, toType, toID, product)
}

// FullTextSearch searches across all FTS5 tables.
func (d *DB) FullTextSearch(ctx context.Context, query string, entityTypes []string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 100
	}

	var results []map[string]any

	// If no entity types specified, search all
	if len(entityTypes) == 0 {
		entityTypes = []string{"features", "ideas", "initiatives", "epics"}
	}

	for _, et := range entityTypes {
		// Allow-list: et ends up interpolated into a table name below (FTS5
		// virtual tables have no Ent/query-builder equivalent to bind this
		// safely). httpserver's /api/search passes this straight from a
		// query-string parameter, so this has to be checked here, not
		// upstream.
		switch et {
		case "features", "ideas", "initiatives", "epics":
			// valid
		default:
			continue
		}
		ftsTable := et + "_fts"
		rows, err := d.db.QueryContext(ctx, fmt.Sprintf(`
			SELECT id, name, description, bm25(%s) as score
			FROM %s
			WHERE %s MATCH ?
			ORDER BY score
			LIMIT ?
		`, ftsTable, ftsTable, ftsTable), query, limit)
		if err != nil {
			continue // Skip if FTS table doesn't exist
		}

		for rows.Next() {
			var id, name, desc string
			var score float64
			if err := rows.Scan(&id, &name, &desc, &score); err != nil {
				continue
			}
			results = append(results, map[string]any{
				"entity":      et,
				"id":          id,
				"name":        name,
				"description": desc,
				"score":       score,
			})
		}
		_ = rows.Close()
	}

	return results, nil
}

// GetRelationships returns relationships for an entity.
func (d *DB) GetRelationships(ctx context.Context, entityType, entityID string) ([]map[string]any, error) {
	rels, err := d.ent.Relationship.Query().
		Where(relationship.Or(
			relationship.And(relationship.FromType(entityType), relationship.FromID(entityID)),
			relationship.And(relationship.ToType(entityType), relationship.ToID(entityID)),
		)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0, len(rels))
	for _, r := range rels {
		results = append(results, map[string]any{
			"from_type": r.FromType,
			"from_id":   r.FromID,
			"rel_type":  r.RelType,
			"to_type":   r.ToType,
			"to_id":     r.ToID,
			"product":   r.Product,
		})
	}
	return results, nil
}

// GetFeatureIDs returns all feature IDs for a product.
func (d *DB) GetFeatureIDs(ctx context.Context, product string) ([]string, error) {
	return d.ent.Feature.Query().Where(feature.Product(product)).IDs(ctx)
}

// GetIdeaVoteCounts returns each cached idea's current vote count, keyed by
// idea ID, for a product. Used by syncIdeaEndorsements to decide which
// ideas' endorsements actually need re-fetching.
func (d *DB) GetIdeaVoteCounts(ctx context.Context, product string) (map[string]int, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT id, votes FROM ideas WHERE product = ?`, product)
	if err != nil {
		return nil, fmt.Errorf("querying idea vote counts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	counts := make(map[string]int)
	for rows.Next() {
		var id string
		var votes int
		if err := rows.Scan(&id, &votes); err != nil {
			return nil, err
		}
		counts[id] = votes
	}
	return counts, rows.Err()
}

// CountIdeaEndorsements returns the number of endorsement rows already
// cached for an idea.
func (d *DB) CountIdeaEndorsements(ctx context.Context, ideaID string) (int, error) {
	var count int
	err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM idea_endorsements WHERE idea_id = ?`, ideaID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting cached endorsements for idea %s: %w", ideaID, err)
	}
	return count, nil
}

// GetIdeas returns cached ideas for a product, optionally filtered to those
// created at or after since. Used by the omnisignalcache provider, which
// builds signals from the cache rather than live API calls.
func (d *DB) GetIdeas(ctx context.Context, product string, since time.Time) ([]map[string]any, error) {
	query := `SELECT id, reference_num, name, COALESCE(status, ''), COALESCE(description, ''), votes, created_at, updated_at
		FROM ideas WHERE product = ?`
	args := []any{product}
	if !since.IsZero() {
		query += ` AND created_at >= ?`
		args = append(args, since)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying ideas for product %s: %w", product, err)
	}
	defer func() { _ = rows.Close() }()

	var results []map[string]any
	for rows.Next() {
		var id, refNum, name, status, description string
		var votes int
		var createdAt, updatedAt sql.NullTime
		if err := rows.Scan(&id, &refNum, &name, &status, &description, &votes, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		rec := map[string]any{
			"id": id, "reference_num": refNum, "name": name,
			"status": status, "description": description, "votes": votes,
		}
		if createdAt.Valid {
			rec["created_at"] = createdAt.Time
		}
		if updatedAt.Valid {
			rec["updated_at"] = updatedAt.Time
		}
		results = append(results, rec)
	}
	return results, rows.Err()
}

// GetIdeaEndorsementsByIdea returns cached endorsements (votes) for an idea,
// including the voter's portal email.
func (d *DB) GetIdeaEndorsementsByIdea(ctx context.Context, ideaID string) ([]map[string]any, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, COALESCE(portal_user_email, ''), COALESCE(portal_user_name, '')
		FROM idea_endorsements WHERE idea_id = ?
	`, ideaID)
	if err != nil {
		return nil, fmt.Errorf("querying endorsements for idea %s: %w", ideaID, err)
	}
	defer func() { _ = rows.Close() }()

	var results []map[string]any
	for rows.Next() {
		var id, email, name string
		if err := rows.Scan(&id, &email, &name); err != nil {
			return nil, err
		}
		results = append(results, map[string]any{"id": id, "portal_user_email": email, "portal_user_name": name})
	}
	return results, rows.Err()
}

// IdeaOrganizationSummary is the subset of a cached IdeaOrganization needed
// to resolve a voter's email domain to a customer reference.
type IdeaOrganizationSummary struct {
	ID           string
	Name         string
	ReferenceNum string
}

// GetIdeaOrganizationsByDomain returns a domain -> organization lookup built
// from every cached IdeaOrganization's (possibly comma-separated)
// email_domains field. Built once per omnisignalcache.Fetch() call rather
// than once per voter.
func (d *DB) GetIdeaOrganizationsByDomain(ctx context.Context) (map[string]IdeaOrganizationSummary, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, COALESCE(name, ''), COALESCE(reference_num, ''), COALESCE(email_domains, '')
		FROM idea_organizations WHERE email_domains IS NOT NULL AND email_domains != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("querying idea organizations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	byDomain := make(map[string]IdeaOrganizationSummary)
	for rows.Next() {
		var id, name, refNum, domains string
		if err := rows.Scan(&id, &name, &refNum, &domains); err != nil {
			return nil, err
		}
		summary := IdeaOrganizationSummary{ID: id, Name: name, ReferenceNum: refNum}
		for _, domain := range strings.Split(domains, ",") {
			domain = strings.ToLower(strings.TrimSpace(domain))
			if domain != "" {
				byDomain[domain] = summary
			}
		}
	}
	return byDomain, rows.Err()
}

// SavedFilter represents a saved AQL query.
type SavedFilter struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	AQL         string    `json:"aql"`
	Product     string    `json:"product,omitempty"`
	Description string    `json:"description,omitempty"`
	IsFavorite  bool      `json:"is_favorite"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListFilters returns all saved filters.
func (d *DB) ListFilters(ctx context.Context) ([]SavedFilter, error) {
	rows, err := d.ent.SavedFilter.Query().
		Order(ent.Desc(savedfilter.FieldIsFavorite), ent.Asc(savedfilter.FieldName)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	filters := make([]SavedFilter, 0, len(rows))
	for _, r := range rows {
		filters = append(filters, savedFilterFromEnt(r))
	}
	return filters, nil
}

// GetFilter returns a saved filter by ID.
func (d *DB) GetFilter(ctx context.Context, id string) (*SavedFilter, error) {
	r, err := d.ent.SavedFilter.Get(ctx, id)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f := savedFilterFromEnt(r)
	return &f, nil
}

// GetFilterByName returns a saved filter by name.
func (d *DB) GetFilterByName(ctx context.Context, name string) (*SavedFilter, error) {
	r, err := d.ent.SavedFilter.Query().Where(savedfilter.Name(name)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f := savedFilterFromEnt(r)
	return &f, nil
}

// CreateFilter creates a new saved filter.
func (d *DB) CreateFilter(ctx context.Context, f *SavedFilter) error {
	if f.ID == "" {
		f.ID = generateFilterID()
	}
	now := time.Now()
	f.CreatedAt = now
	f.UpdatedAt = now

	r, err := d.ent.SavedFilter.Create().
		SetID(f.ID).
		SetName(f.Name).
		SetAql(f.AQL).
		SetNillableProduct(mapStringPtr2(f.Product)).
		SetNillableDescription(mapStringPtr2(f.Description)).
		SetIsFavorite(f.IsFavorite).
		SetCreatedAt(f.CreatedAt).
		SetUpdatedAt(f.UpdatedAt).
		Save(ctx)
	if err != nil {
		return err
	}
	*f = savedFilterFromEnt(r)
	return nil
}

// UpdateFilter updates an existing saved filter.
func (d *DB) UpdateFilter(ctx context.Context, f *SavedFilter) error {
	f.UpdatedAt = time.Now()

	r, err := d.ent.SavedFilter.UpdateOneID(f.ID).
		SetName(f.Name).
		SetAql(f.AQL).
		SetNillableProduct(mapStringPtr2(f.Product)).
		SetNillableDescription(mapStringPtr2(f.Description)).
		SetIsFavorite(f.IsFavorite).
		SetUpdatedAt(f.UpdatedAt).
		Save(ctx)
	if ent.IsNotFound(err) {
		return sql.ErrNoRows
	}
	if err != nil {
		return err
	}
	*f = savedFilterFromEnt(r)
	return nil
}

// DeleteFilter deletes a saved filter by ID.
func (d *DB) DeleteFilter(ctx context.Context, id string) error {
	err := d.ent.SavedFilter.DeleteOneID(id).Exec(ctx)
	if ent.IsNotFound(err) {
		return sql.ErrNoRows
	}
	return err
}

// generateFilterID generates a unique filter ID.
func generateFilterID() string {
	return fmt.Sprintf("filter_%d", time.Now().UnixNano())
}

// savedFilterFromEnt converts a generated Ent entity to the API-facing
// SavedFilter DTO (kept distinct from the Ent type so the HTTP/MCP layers
// aren't coupled to the storage schema).
func savedFilterFromEnt(r *ent.SavedFilter) SavedFilter {
	return SavedFilter{
		ID:          r.ID,
		Name:        r.Name,
		AQL:         r.Aql,
		Product:     r.Product,
		Description: r.Description,
		IsFavorite:  r.IsFavorite,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// GetFeaturesByReleaseID returns all features for a given release ID.
func (d *DB) GetFeaturesByReleaseID(ctx context.Context, product, releaseID string) ([]map[string]any, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, reference_num, name, status, assigned_to, start_date, due_date,
			   release, release_id, created_at, updated_at
		FROM features
		WHERE product = ? AND release_id = ?
		ORDER BY reference_num
	`, product, releaseID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanFeatureRows(rows)
}

// GetFeaturesByReleaseName returns all features for a given release name.
func (d *DB) GetFeaturesByReleaseName(ctx context.Context, product, releaseName string) ([]map[string]any, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, reference_num, name, status, assigned_to, start_date, due_date,
			   release, release_id, created_at, updated_at
		FROM features
		WHERE product = ? AND release = ?
		ORDER BY reference_num
	`, product, releaseName)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanFeatureRows(rows)
}

// GetFeaturesByReleaseDate returns all features for releases matching a specific date.
// The date should be in YYYY-MM-DD format.
func (d *DB) GetFeaturesByReleaseDate(ctx context.Context, product, releaseDate string) ([]map[string]any, error) {
	// Join with releases table to match by release_date
	rows, err := d.db.QueryContext(ctx, `
		SELECT f.id, f.reference_num, f.name, f.status, f.assigned_to, f.start_date, f.due_date,
			   f.release, f.release_id, f.created_at, f.updated_at
		FROM features f
		JOIN releases r ON f.release_id = r.id
		WHERE f.product = ? AND r.release_date = ?
		ORDER BY f.reference_num
	`, product, releaseDate)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanFeatureRows(rows)
}

// GetFeaturesByReleaseDateRange returns all features for releases within a date range.
// Both dates should be in YYYY-MM-DD format. Either can be empty for open-ended range.
func (d *DB) GetFeaturesByReleaseDateRange(ctx context.Context, product, startDate, endDate string) ([]map[string]any, error) {
	var query string
	var args []any

	if startDate != "" && endDate != "" {
		query = `
			SELECT f.id, f.reference_num, f.name, f.status, f.assigned_to, f.start_date, f.due_date,
				   f.release, f.release_id, f.created_at, f.updated_at
			FROM features f
			JOIN releases r ON f.release_id = r.id
			WHERE f.product = ? AND r.release_date >= ? AND r.release_date <= ?
			ORDER BY r.release_date, f.reference_num
		`
		args = []any{product, startDate, endDate}
	} else if startDate != "" {
		query = `
			SELECT f.id, f.reference_num, f.name, f.status, f.assigned_to, f.start_date, f.due_date,
				   f.release, f.release_id, f.created_at, f.updated_at
			FROM features f
			JOIN releases r ON f.release_id = r.id
			WHERE f.product = ? AND r.release_date >= ?
			ORDER BY r.release_date, f.reference_num
		`
		args = []any{product, startDate}
	} else if endDate != "" {
		query = `
			SELECT f.id, f.reference_num, f.name, f.status, f.assigned_to, f.start_date, f.due_date,
				   f.release, f.release_id, f.created_at, f.updated_at
			FROM features f
			JOIN releases r ON f.release_id = r.id
			WHERE f.product = ? AND r.release_date <= ?
			ORDER BY r.release_date, f.reference_num
		`
		args = []any{product, endDate}
	} else {
		// No date filter, return all features with releases
		query = `
			SELECT f.id, f.reference_num, f.name, f.status, f.assigned_to, f.start_date, f.due_date,
				   f.release, f.release_id, f.created_at, f.updated_at
			FROM features f
			WHERE f.product = ? AND f.release_id IS NOT NULL
			ORDER BY f.reference_num
		`
		args = []any{product}
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanFeatureRows(rows)
}

// scanFeatureRows scans feature rows into a slice of maps.
func scanFeatureRows(rows *sql.Rows) ([]map[string]any, error) {
	var results []map[string]any
	for rows.Next() {
		var id, refNum, name string
		var status, assignedTo, startDate, dueDate, release, releaseID sql.NullString
		var createdAt, updatedAt sql.NullTime

		if err := rows.Scan(&id, &refNum, &name, &status, &assignedTo, &startDate, &dueDate,
			&release, &releaseID, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		rec := map[string]any{
			"id":            id,
			"reference_num": refNum,
			"name":          name,
		}
		if status.Valid {
			rec["status"] = status.String
		}
		if assignedTo.Valid {
			rec["assigned_to"] = assignedTo.String
		}
		if startDate.Valid {
			rec["start_date"] = startDate.String
		}
		if dueDate.Valid {
			rec["due_date"] = dueDate.String
		}
		if release.Valid {
			rec["release"] = release.String
		}
		if releaseID.Valid {
			rec["release_id"] = releaseID.String
		}
		if createdAt.Valid {
			rec["created_at"] = createdAt.Time
		}
		if updatedAt.Valid {
			rec["updated_at"] = updatedAt.Time
		}

		results = append(results, rec)
	}

	return results, rows.Err()
}

// GetReleaseByDate returns a release matching a specific date.
func (d *DB) GetReleaseByDate(ctx context.Context, product, releaseDate string) (*map[string]any, error) {
	return d.getReleaseByField(ctx, product, "release_date", releaseDate)
}

// GetReleaseByName returns a release matching a specific name.
func (d *DB) GetReleaseByName(ctx context.Context, product, releaseName string) (*map[string]any, error) {
	return d.getReleaseByField(ctx, product, "name", releaseName)
}

// getReleaseByField is a helper that returns a release matching a field value.
// The field parameter must be a known column name (name or release_date).
func (d *DB) getReleaseByField(ctx context.Context, product, field, value string) (*map[string]any, error) {
	// Validate field to prevent SQL injection (only allow known columns)
	switch field {
	case "name", "release_date":
		// valid
	default:
		return nil, fmt.Errorf("invalid field: %s", field)
	}

	var id, refNum, name string
	var startDate, relDate sql.NullString
	var released, parkingLot int
	var createdAt sql.NullTime

	//nolint:gosec // G201: field is validated above to known column names
	query := fmt.Sprintf(`
		SELECT id, reference_num, name, start_date, release_date, released, parking_lot, created_at
		FROM releases
		WHERE product = ? AND %s = ?
		LIMIT 1
	`, field)

	err := d.db.QueryRowContext(ctx, query, product, value).Scan(
		&id, &refNum, &name, &startDate, &relDate, &released, &parkingLot, &createdAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rec := map[string]any{
		"id":            id,
		"reference_num": refNum,
		"name":          name,
		"released":      released == 1,
		"parking_lot":   parkingLot == 1,
	}
	if startDate.Valid {
		rec["start_date"] = startDate.String
	}
	if relDate.Valid {
		rec["release_date"] = relDate.String
	}
	if createdAt.Valid {
		rec["created_at"] = createdAt.Time
	}

	return &rec, nil
}

// =============================================================================
// Statistics Methods (Phase 5 Analytics Tools)
// =============================================================================

// IdeaStatistics holds aggregated statistics for ideas.
type IdeaStatistics struct {
	TotalCount    int            `json:"total_count"`
	ByStatus      map[string]int `json:"by_status"`
	TotalVotes    int            `json:"total_votes"`
	AvgVotes      float64        `json:"avg_votes"`
	MaxVotes      int            `json:"max_votes"`
	TopIdeas      []IdeaSummary  `json:"top_ideas"`
	RecentCount   int            `json:"recent_count"`   // ideas in last 30 days
	UpdatedRecent int            `json:"updated_recent"` // updated in last 7 days
}

// IdeaSummary is a brief summary of an idea for statistics.
type IdeaSummary struct {
	ID           string `json:"id"`
	ReferenceNum string `json:"reference_num"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	Votes        int    `json:"votes"`
}

// GetIdeasStatistics returns aggregated statistics for ideas in a product.
func (d *DB) GetIdeasStatistics(ctx context.Context, product string, limit int) (*IdeaStatistics, error) {
	if limit <= 0 {
		limit = 10
	}

	stats := &IdeaStatistics{
		ByStatus: make(map[string]int),
	}

	// Total count and vote stats
	err := d.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(votes), 0), COALESCE(AVG(votes), 0), COALESCE(MAX(votes), 0)
		FROM ideas WHERE product = ?
	`, product).Scan(&stats.TotalCount, &stats.TotalVotes, &stats.AvgVotes, &stats.MaxVotes)
	if err != nil {
		return nil, fmt.Errorf("querying idea totals: %w", err)
	}

	// Count by status
	rows, err := d.db.QueryContext(ctx, `
		SELECT COALESCE(status, 'unknown'), COUNT(*)
		FROM ideas WHERE product = ?
		GROUP BY status
	`, product)
	if err != nil {
		return nil, fmt.Errorf("querying idea status counts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.ByStatus[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Recent ideas (last 30 days)
	err = d.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM ideas
		WHERE product = ? AND created_at >= datetime('now', '-30 days')
	`, product).Scan(&stats.RecentCount)
	if err != nil {
		return nil, fmt.Errorf("querying recent ideas: %w", err)
	}

	// Recently updated (last 7 days)
	err = d.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM ideas
		WHERE product = ? AND updated_at >= datetime('now', '-7 days')
	`, product).Scan(&stats.UpdatedRecent)
	if err != nil {
		return nil, fmt.Errorf("querying updated ideas: %w", err)
	}

	// Top ideas by votes
	topRows, err := d.db.QueryContext(ctx, `
		SELECT id, reference_num, name, COALESCE(status, ''), votes
		FROM ideas WHERE product = ?
		ORDER BY votes DESC
		LIMIT ?
	`, product, limit)
	if err != nil {
		return nil, fmt.Errorf("querying top ideas: %w", err)
	}
	defer func() { _ = topRows.Close() }()

	for topRows.Next() {
		var idea IdeaSummary
		if err := topRows.Scan(&idea.ID, &idea.ReferenceNum, &idea.Name, &idea.Status, &idea.Votes); err != nil {
			return nil, err
		}
		stats.TopIdeas = append(stats.TopIdeas, idea)
	}

	return stats, topRows.Err()
}

// VoterDomainStatistics holds an email-domain histogram of idea voters.
type VoterDomainStatistics struct {
	TotalVoters   int           `json:"total_voters"`
	UniqueDomains int           `json:"unique_domains"`
	ByDomain      []DomainCount `json:"by_domain"`
}

// DomainCount is the voter count for a single email domain.
type DomainCount struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

// GetVoterEmailDomainStatistics returns a histogram of idea voters grouped
// by email domain (the part of portal_user_email after "@"). If ideaID is
// empty, the histogram covers every idea in the product; otherwise it's
// scoped to that one idea. AQL's executor has no string/computed-expression
// functions (no SUBSTR equivalent), so this can't be a generic AQL GROUP
// BY -- it needs the same hand-written-SQL approach as GetIdeasStatistics.
func (d *DB) GetVoterEmailDomainStatistics(ctx context.Context, product, ideaID string, limit int) (*VoterDomainStatistics, error) {
	if limit <= 0 {
		limit = 10
	}

	stats := &VoterDomainStatistics{}

	args := []any{product}
	where := `product = ? AND portal_user_email IS NOT NULL AND portal_user_email != ''`
	if ideaID != "" {
		where += ` AND idea_id = ?`
		args = append(args, ideaID)
	}

	err := d.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COUNT(DISTINCT substr(portal_user_email, instr(portal_user_email,'@')+1))
		FROM idea_endorsements WHERE `+where, args...,
	).Scan(&stats.TotalVoters, &stats.UniqueDomains)
	if err != nil {
		return nil, fmt.Errorf("querying voter domain totals: %w", err)
	}

	rows, err := d.db.QueryContext(ctx, `
		SELECT substr(portal_user_email, instr(portal_user_email,'@')+1) AS domain, COUNT(*)
		FROM idea_endorsements WHERE `+where+`
		GROUP BY domain
		ORDER BY COUNT(*) DESC, domain ASC
		LIMIT ?
	`, append(args, limit)...)
	if err != nil {
		return nil, fmt.Errorf("querying voter domain histogram: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var dc DomainCount
		if err := rows.Scan(&dc.Domain, &dc.Count); err != nil {
			return nil, err
		}
		stats.ByDomain = append(stats.ByDomain, dc)
	}
	return stats, rows.Err()
}

// FeatureStatistics holds aggregated statistics for features.
type FeatureStatistics struct {
	TotalCount       int              `json:"total_count"`
	ByStatus         map[string]int   `json:"by_status"`
	ByRelease        map[string]int   `json:"by_release"`
	WithRelease      int              `json:"with_release"`
	WithoutRelease   int              `json:"without_release"`
	RecentCount      int              `json:"recent_count"`   // features in last 30 days
	UpdatedRecent    int              `json:"updated_recent"` // updated in last 7 days
	UpcomingReleases []ReleaseSummary `json:"upcoming_releases"`
}

// ReleaseSummary is a brief summary of a release for statistics.
type ReleaseSummary struct {
	ID           string `json:"id"`
	ReferenceNum string `json:"reference_num"`
	Name         string `json:"name"`
	ReleaseDate  string `json:"release_date"`
	FeatureCount int    `json:"feature_count"`
}

// GetFeaturesStatistics returns aggregated statistics for features in a product.
func (d *DB) GetFeaturesStatistics(ctx context.Context, product string, limit int) (*FeatureStatistics, error) {
	if limit <= 0 {
		limit = 5
	}

	stats := &FeatureStatistics{
		ByStatus:  make(map[string]int),
		ByRelease: make(map[string]int),
	}

	// Total count
	err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM features WHERE product = ?`, product).Scan(&stats.TotalCount)
	if err != nil {
		return nil, fmt.Errorf("querying feature total: %w", err)
	}

	// With/without release
	err = d.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM features WHERE product = ? AND release_id IS NOT NULL AND release_id != ''
	`, product).Scan(&stats.WithRelease)
	if err != nil {
		return nil, fmt.Errorf("querying features with release: %w", err)
	}
	stats.WithoutRelease = stats.TotalCount - stats.WithRelease

	// Count by status
	rows, err := d.db.QueryContext(ctx, `
		SELECT COALESCE(status, 'unknown'), COUNT(*)
		FROM features WHERE product = ?
		GROUP BY status
	`, product)
	if err != nil {
		return nil, fmt.Errorf("querying feature status counts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.ByStatus[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Count by release
	relRows, err := d.db.QueryContext(ctx, `
		SELECT COALESCE(release, 'unassigned'), COUNT(*)
		FROM features WHERE product = ?
		GROUP BY release
	`, product)
	if err != nil {
		return nil, fmt.Errorf("querying feature release counts: %w", err)
	}
	defer func() { _ = relRows.Close() }()

	for relRows.Next() {
		var release string
		var count int
		if err := relRows.Scan(&release, &count); err != nil {
			return nil, err
		}
		stats.ByRelease[release] = count
	}
	if err := relRows.Err(); err != nil {
		return nil, err
	}

	// Recent features (last 30 days)
	err = d.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM features
		WHERE product = ? AND created_at >= datetime('now', '-30 days')
	`, product).Scan(&stats.RecentCount)
	if err != nil {
		return nil, fmt.Errorf("querying recent features: %w", err)
	}

	// Recently updated (last 7 days)
	err = d.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM features
		WHERE product = ? AND updated_at >= datetime('now', '-7 days')
	`, product).Scan(&stats.UpdatedRecent)
	if err != nil {
		return nil, fmt.Errorf("querying updated features: %w", err)
	}

	// Upcoming releases with feature counts
	upRows, err := d.db.QueryContext(ctx, `
		SELECT r.id, r.reference_num, r.name, COALESCE(r.release_date, ''),
			   (SELECT COUNT(*) FROM features f WHERE f.release_id = r.id)
		FROM releases r
		WHERE r.product = ? AND r.released = 0 AND r.parking_lot = 0
			  AND r.release_date >= date('now')
		ORDER BY r.release_date ASC
		LIMIT ?
	`, product, limit)
	if err != nil {
		return nil, fmt.Errorf("querying upcoming releases: %w", err)
	}
	defer func() { _ = upRows.Close() }()

	for upRows.Next() {
		var rel ReleaseSummary
		if err := upRows.Scan(&rel.ID, &rel.ReferenceNum, &rel.Name, &rel.ReleaseDate, &rel.FeatureCount); err != nil {
			return nil, err
		}
		stats.UpcomingReleases = append(stats.UpcomingReleases, rel)
	}

	return stats, upRows.Err()
}
