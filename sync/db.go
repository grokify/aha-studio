// Package sync provides SQLite-based offline caching for Aha Studio.
package sync

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database connection for Aha data caching.
type DB struct {
	db      *sql.DB
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

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	d := &DB{db: db, dbPath: dbPath}

	// Create schema
	if err := d.createSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	return d, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// SetProduct sets the current product context.
func (d *DB) SetProduct(product string) {
	d.product = product
}

// createSchema creates the database tables if they don't exist.
func (d *DB) createSchema() error {
	schema := `
	-- Sync metadata
	CREATE TABLE IF NOT EXISTS sync_meta (
		entity TEXT NOT NULL,
		product TEXT NOT NULL,
		last_sync DATETIME NOT NULL,
		record_count INTEGER DEFAULT 0,
		PRIMARY KEY (entity, product)
	);

	-- Features
	CREATE TABLE IF NOT EXISTS features (
		id TEXT PRIMARY KEY,
		product TEXT NOT NULL,
		reference_num TEXT,
		name TEXT,
		description TEXT,
		status TEXT,
		assigned_to TEXT,
		start_date TEXT,
		due_date TEXT,
		release TEXT,
		tags TEXT,
		url TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		data JSON
	);
	CREATE INDEX IF NOT EXISTS idx_features_product ON features(product);
	CREATE INDEX IF NOT EXISTS idx_features_status ON features(status);
	CREATE INDEX IF NOT EXISTS idx_features_updated ON features(updated_at);

	-- Ideas
	CREATE TABLE IF NOT EXISTS ideas (
		id TEXT PRIMARY KEY,
		product TEXT NOT NULL,
		reference_num TEXT,
		name TEXT,
		description TEXT,
		status TEXT,
		votes INTEGER DEFAULT 0,
		tags TEXT,
		url TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		data JSON
	);
	CREATE INDEX IF NOT EXISTS idx_ideas_product ON ideas(product);
	CREATE INDEX IF NOT EXISTS idx_ideas_status ON ideas(status);
	CREATE INDEX IF NOT EXISTS idx_ideas_votes ON ideas(votes);

	-- Releases
	CREATE TABLE IF NOT EXISTS releases (
		id TEXT PRIMARY KEY,
		product TEXT NOT NULL,
		reference_num TEXT,
		name TEXT,
		start_date TEXT,
		release_date TEXT,
		released INTEGER DEFAULT 0,
		parking_lot INTEGER DEFAULT 0,
		url TEXT,
		created_at DATETIME,
		data JSON
	);
	CREATE INDEX IF NOT EXISTS idx_releases_product ON releases(product);

	-- Initiatives
	CREATE TABLE IF NOT EXISTS initiatives (
		id TEXT PRIMARY KEY,
		product TEXT NOT NULL,
		reference_num TEXT,
		name TEXT,
		description TEXT,
		status TEXT,
		value REAL,
		effort REAL,
		progress REAL,
		start_date TEXT,
		end_date TEXT,
		url TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		data JSON
	);
	CREATE INDEX IF NOT EXISTS idx_initiatives_product ON initiatives(product);

	-- Goals
	CREATE TABLE IF NOT EXISTS goals (
		id TEXT PRIMARY KEY,
		product TEXT NOT NULL,
		reference_num TEXT,
		name TEXT,
		description TEXT,
		status TEXT,
		progress REAL,
		start_date TEXT,
		end_date TEXT,
		url TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		data JSON
	);
	CREATE INDEX IF NOT EXISTS idx_goals_product ON goals(product);

	-- Epics
	CREATE TABLE IF NOT EXISTS epics (
		id TEXT PRIMARY KEY,
		product TEXT NOT NULL,
		reference_num TEXT,
		name TEXT,
		description TEXT,
		status TEXT,
		progress REAL,
		start_date TEXT,
		due_date TEXT,
		release TEXT,
		url TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		data JSON
	);
	CREATE INDEX IF NOT EXISTS idx_epics_product ON epics(product);

	-- Users
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		first_name TEXT,
		last_name TEXT,
		email TEXT,
		role TEXT,
		created_at DATETIME,
		data JSON
	);

	-- Products
	CREATE TABLE IF NOT EXISTS products (
		id TEXT PRIMARY KEY,
		reference_prefix TEXT,
		name TEXT,
		product_line INTEGER DEFAULT 0,
		has_ideas INTEGER DEFAULT 0,
		created_at DATETIME,
		data JSON
	);

	-- Comments
	CREATE TABLE IF NOT EXISTS comments (
		id TEXT PRIMARY KEY,
		product TEXT,
		commentable_type TEXT,
		commentable_id TEXT,
		body TEXT,
		user_id TEXT,
		url TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		data JSON
	);
	CREATE INDEX IF NOT EXISTS idx_comments_product ON comments(product);
	CREATE INDEX IF NOT EXISTS idx_comments_commentable ON comments(commentable_type, commentable_id);
	CREATE INDEX IF NOT EXISTS idx_comments_user ON comments(user_id);

	-- Requirements
	CREATE TABLE IF NOT EXISTS requirements (
		id TEXT PRIMARY KEY,
		product TEXT NOT NULL,
		feature_id TEXT,
		reference_num TEXT,
		name TEXT,
		description TEXT,
		status TEXT,
		assigned_to TEXT,
		position INTEGER,
		original_estimate REAL,
		remaining_estimate REAL,
		work_done REAL,
		url TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		data JSON
	);
	CREATE INDEX IF NOT EXISTS idx_requirements_product ON requirements(product);
	CREATE INDEX IF NOT EXISTS idx_requirements_feature ON requirements(feature_id);
	CREATE INDEX IF NOT EXISTS idx_requirements_status ON requirements(status);

	-- Saved filters (user-defined AQL queries)
	CREATE TABLE IF NOT EXISTS saved_filters (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		aql TEXT NOT NULL,
		product TEXT,
		description TEXT,
		is_favorite INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_filters_favorite ON saved_filters(is_favorite);
	CREATE INDEX IF NOT EXISTS idx_filters_product ON saved_filters(product);

	-- Relationships (for offline joins)
	CREATE TABLE IF NOT EXISTS relationships (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		from_type TEXT NOT NULL,
		from_id TEXT NOT NULL,
		rel_type TEXT NOT NULL,
		to_type TEXT NOT NULL,
		to_id TEXT NOT NULL,
		product TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(from_type, from_id, rel_type, to_type, to_id)
	);
	CREATE INDEX IF NOT EXISTS idx_rel_from ON relationships(from_type, from_id);
	CREATE INDEX IF NOT EXISTS idx_rel_to ON relationships(to_type, to_id);
	CREATE INDEX IF NOT EXISTS idx_rel_type ON relationships(rel_type);
	CREATE INDEX IF NOT EXISTS idx_rel_product ON relationships(product);
	`

	_, err := d.db.Exec(schema)
	if err != nil {
		return err
	}

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

	_, err = d.db.Exec(fts)
	return err
}

// GetLastSync returns the last sync time for an entity/product combination.
func (d *DB) GetLastSync(entity, product string) (time.Time, error) {
	var lastSync time.Time
	err := d.db.QueryRow(
		"SELECT last_sync FROM sync_meta WHERE entity = ? AND product = ?",
		entity, product,
	).Scan(&lastSync)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	return lastSync, err
}

// SetLastSync updates the last sync time for an entity/product combination.
func (d *DB) SetLastSync(entity, product string, t time.Time, count int) error {
	_, err := d.db.Exec(`
		INSERT INTO sync_meta (entity, product, last_sync, record_count)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(entity, product) DO UPDATE SET
			last_sync = excluded.last_sync,
			record_count = excluded.record_count
	`, entity, product, t, count)
	return err
}

// UpsertFeature inserts or updates a feature record.
func (d *DB) UpsertFeature(product string, data map[string]any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
		INSERT INTO features (id, product, reference_num, name, description, status,
			assigned_to, start_date, due_date, release, tags, url, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			reference_num = excluded.reference_num,
			name = excluded.name,
			description = excluded.description,
			status = excluded.status,
			assigned_to = excluded.assigned_to,
			start_date = excluded.start_date,
			due_date = excluded.due_date,
			release = excluded.release,
			tags = excluded.tags,
			url = excluded.url,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			data = excluded.data
	`,
		data["id"], product, data["reference_num"], data["name"], data["description"],
		data["status"], data["assigned_to"], data["start_date"], data["due_date"],
		data["release"], tagsToString(data["tags"]), data["url"],
		data["created_at"], data["updated_at"], string(jsonData),
	)
	return err
}

// UpsertIdea inserts or updates an idea record.
func (d *DB) UpsertIdea(product string, data map[string]any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	votes, _ := data["votes"].(int64)

	_, err = d.db.Exec(`
		INSERT INTO ideas (id, product, reference_num, name, description, status,
			votes, tags, url, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			reference_num = excluded.reference_num,
			name = excluded.name,
			description = excluded.description,
			status = excluded.status,
			votes = excluded.votes,
			tags = excluded.tags,
			url = excluded.url,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			data = excluded.data
	`,
		data["id"], product, data["reference_num"], data["name"], data["description"],
		data["status"], votes, tagsToString(data["tags"]), data["url"],
		data["created_at"], data["updated_at"], string(jsonData),
	)
	return err
}

// UpsertRelease inserts or updates a release record.
func (d *DB) UpsertRelease(product string, data map[string]any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	released := boolToInt(data["released"])
	parkingLot := boolToInt(data["parking_lot"])

	_, err = d.db.Exec(`
		INSERT INTO releases (id, product, reference_num, name, start_date, release_date,
			released, parking_lot, url, created_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			reference_num = excluded.reference_num,
			name = excluded.name,
			start_date = excluded.start_date,
			release_date = excluded.release_date,
			released = excluded.released,
			parking_lot = excluded.parking_lot,
			url = excluded.url,
			created_at = excluded.created_at,
			data = excluded.data
	`,
		data["id"], product, data["reference_num"], data["name"],
		data["start_date"], data["release_date"], released, parkingLot,
		data["url"], data["created_at"], string(jsonData),
	)
	return err
}

// UpsertInitiative inserts or updates an initiative record.
func (d *DB) UpsertInitiative(product string, data map[string]any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	value, _ := data["value"].(float64)
	effort, _ := data["effort"].(float64)
	progress, _ := data["progress"].(float64)

	_, err = d.db.Exec(`
		INSERT INTO initiatives (id, product, reference_num, name, description, status,
			value, effort, progress, start_date, end_date, url, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			reference_num = excluded.reference_num,
			name = excluded.name,
			description = excluded.description,
			status = excluded.status,
			value = excluded.value,
			effort = excluded.effort,
			progress = excluded.progress,
			start_date = excluded.start_date,
			end_date = excluded.end_date,
			url = excluded.url,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			data = excluded.data
	`,
		data["id"], product, data["reference_num"], data["name"], data["description"],
		data["status"], value, effort, progress, data["start_date"], data["end_date"],
		data["url"], data["created_at"], data["updated_at"], string(jsonData),
	)
	return err
}

// UpsertGoal inserts or updates a goal record.
func (d *DB) UpsertGoal(product string, data map[string]any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	progress, _ := data["progress"].(float64)

	_, err = d.db.Exec(`
		INSERT INTO goals (id, product, reference_num, name, description, status,
			progress, start_date, end_date, url, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			reference_num = excluded.reference_num,
			name = excluded.name,
			description = excluded.description,
			status = excluded.status,
			progress = excluded.progress,
			start_date = excluded.start_date,
			end_date = excluded.end_date,
			url = excluded.url,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			data = excluded.data
	`,
		data["id"], product, data["reference_num"], data["name"], data["description"],
		data["status"], progress, data["start_date"], data["end_date"],
		data["url"], data["created_at"], data["updated_at"], string(jsonData),
	)
	return err
}

// UpsertEpic inserts or updates an epic record.
func (d *DB) UpsertEpic(product string, data map[string]any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	progress, _ := data["progress"].(float64)

	_, err = d.db.Exec(`
		INSERT INTO epics (id, product, reference_num, name, description, status,
			progress, start_date, due_date, release, url, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			reference_num = excluded.reference_num,
			name = excluded.name,
			description = excluded.description,
			status = excluded.status,
			progress = excluded.progress,
			start_date = excluded.start_date,
			due_date = excluded.due_date,
			release = excluded.release,
			url = excluded.url,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			data = excluded.data
	`,
		data["id"], product, data["reference_num"], data["name"], data["description"],
		data["status"], progress, data["start_date"], data["due_date"],
		data["release"], data["url"], data["created_at"], data["updated_at"], string(jsonData),
	)
	return err
}

// UpsertUser inserts or updates a user record.
func (d *DB) UpsertUser(data map[string]any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, created_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			first_name = excluded.first_name,
			last_name = excluded.last_name,
			email = excluded.email,
			role = excluded.role,
			created_at = excluded.created_at,
			data = excluded.data
	`,
		data["id"], data["first_name"], data["last_name"], data["email"],
		data["role"], data["created_at"], string(jsonData),
	)
	return err
}

// UpsertProduct inserts or updates a product record.
func (d *DB) UpsertProduct(data map[string]any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	productLine := boolToInt(data["product_line"])
	hasIdeas := boolToInt(data["has_ideas"])

	_, err = d.db.Exec(`
		INSERT INTO products (id, reference_prefix, name, product_line, has_ideas, created_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			reference_prefix = excluded.reference_prefix,
			name = excluded.name,
			product_line = excluded.product_line,
			has_ideas = excluded.has_ideas,
			created_at = excluded.created_at,
			data = excluded.data
	`,
		data["id"], data["reference_prefix"], data["name"],
		productLine, hasIdeas, data["created_at"], string(jsonData),
	)
	return err
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

func boolToInt(v any) int {
	if v == nil {
		return 0
	}
	switch b := v.(type) {
	case bool:
		if b {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// UpsertComment inserts or updates a comment record.
func (d *DB) UpsertComment(product string, data map[string]any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
		INSERT INTO comments (id, product, commentable_type, commentable_id, body, user_id,
			url, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			commentable_type = excluded.commentable_type,
			commentable_id = excluded.commentable_id,
			body = excluded.body,
			user_id = excluded.user_id,
			url = excluded.url,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			data = excluded.data
	`,
		data["id"], product, data["commentable_type"], data["commentable_id"],
		data["body"], data["user_id"], data["url"],
		data["created_at"], data["updated_at"], string(jsonData),
	)
	return err
}

// UpsertRequirement inserts or updates a requirement record.
func (d *DB) UpsertRequirement(product string, data map[string]any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	position, _ := data["position"].(int64)
	origEst, _ := data["original_estimate"].(float64)
	remainEst, _ := data["remaining_estimate"].(float64)
	workDone, _ := data["work_done"].(float64)

	_, err = d.db.Exec(`
		INSERT INTO requirements (id, product, feature_id, reference_num, name, description,
			status, assigned_to, position, original_estimate, remaining_estimate, work_done,
			url, created_at, updated_at, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			feature_id = excluded.feature_id,
			reference_num = excluded.reference_num,
			name = excluded.name,
			description = excluded.description,
			status = excluded.status,
			assigned_to = excluded.assigned_to,
			position = excluded.position,
			original_estimate = excluded.original_estimate,
			remaining_estimate = excluded.remaining_estimate,
			work_done = excluded.work_done,
			url = excluded.url,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			data = excluded.data
	`,
		data["id"], product, data["feature_id"], data["reference_num"], data["name"],
		data["description"], data["status"], data["assigned_to"], position,
		origEst, remainEst, workDone, data["url"],
		data["created_at"], data["updated_at"], string(jsonData),
	)
	return err
}

// UpsertRelationship inserts or updates a relationship record.
func (d *DB) UpsertRelationship(fromType, fromID, relType, toType, toID, product string) error {
	_, err := d.db.Exec(`
		INSERT INTO relationships (from_type, from_id, rel_type, to_type, to_id, product)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(from_type, from_id, rel_type, to_type, to_id) DO UPDATE SET
			product = excluded.product
	`,
		fromType, fromID, relType, toType, toID, product,
	)
	return err
}

// FullTextSearch searches across all FTS5 tables.
func (d *DB) FullTextSearch(query string, entityTypes []string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 100
	}

	var results []map[string]any

	// If no entity types specified, search all
	if len(entityTypes) == 0 {
		entityTypes = []string{"features", "ideas", "initiatives", "epics"}
	}

	for _, et := range entityTypes {
		ftsTable := et + "_fts"
		rows, err := d.db.Query(fmt.Sprintf(`
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
func (d *DB) GetRelationships(entityType, entityID string) ([]map[string]any, error) {
	rows, err := d.db.Query(`
		SELECT from_type, from_id, rel_type, to_type, to_id, product
		FROM relationships
		WHERE (from_type = ? AND from_id = ?) OR (to_type = ? AND to_id = ?)
	`, entityType, entityID, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []map[string]any
	for rows.Next() {
		var fromType, fromID, relType, toType, toID, product string
		if err := rows.Scan(&fromType, &fromID, &relType, &toType, &toID, &product); err != nil {
			return nil, err
		}
		results = append(results, map[string]any{
			"from_type": fromType,
			"from_id":   fromID,
			"rel_type":  relType,
			"to_type":   toType,
			"to_id":     toID,
			"product":   product,
		})
	}

	return results, rows.Err()
}

// GetFeatureIDs returns all feature IDs for a product.
func (d *DB) GetFeatureIDs(product string) ([]string, error) {
	rows, err := d.db.Query("SELECT id FROM features WHERE product = ?", product)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
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
func (d *DB) ListFilters() ([]SavedFilter, error) {
	rows, err := d.db.Query(`
		SELECT id, name, aql, product, description, is_favorite, created_at, updated_at
		FROM saved_filters
		ORDER BY is_favorite DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var filters []SavedFilter
	for rows.Next() {
		var f SavedFilter
		var product, description sql.NullString
		var isFav int
		if err := rows.Scan(&f.ID, &f.Name, &f.AQL, &product, &description, &isFav, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		f.Product = product.String
		f.Description = description.String
		f.IsFavorite = isFav == 1
		filters = append(filters, f)
	}

	return filters, rows.Err()
}

// GetFilter returns a saved filter by ID.
func (d *DB) GetFilter(id string) (*SavedFilter, error) {
	var f SavedFilter
	var product, description sql.NullString
	var isFav int
	err := d.db.QueryRow(`
		SELECT id, name, aql, product, description, is_favorite, created_at, updated_at
		FROM saved_filters
		WHERE id = ?
	`, id).Scan(&f.ID, &f.Name, &f.AQL, &product, &description, &isFav, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f.Product = product.String
	f.Description = description.String
	f.IsFavorite = isFav == 1
	return &f, nil
}

// GetFilterByName returns a saved filter by name.
func (d *DB) GetFilterByName(name string) (*SavedFilter, error) {
	var f SavedFilter
	var product, description sql.NullString
	var isFav int
	err := d.db.QueryRow(`
		SELECT id, name, aql, product, description, is_favorite, created_at, updated_at
		FROM saved_filters
		WHERE name = ?
	`, name).Scan(&f.ID, &f.Name, &f.AQL, &product, &description, &isFav, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f.Product = product.String
	f.Description = description.String
	f.IsFavorite = isFav == 1
	return &f, nil
}

// CreateFilter creates a new saved filter.
func (d *DB) CreateFilter(f *SavedFilter) error {
	if f.ID == "" {
		f.ID = generateFilterID()
	}
	now := time.Now()
	f.CreatedAt = now
	f.UpdatedAt = now

	isFav := 0
	if f.IsFavorite {
		isFav = 1
	}

	_, err := d.db.Exec(`
		INSERT INTO saved_filters (id, name, aql, product, description, is_favorite, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, f.ID, f.Name, f.AQL, nullString(f.Product), nullString(f.Description), isFav, f.CreatedAt, f.UpdatedAt)
	return err
}

// UpdateFilter updates an existing saved filter.
func (d *DB) UpdateFilter(f *SavedFilter) error {
	f.UpdatedAt = time.Now()

	isFav := 0
	if f.IsFavorite {
		isFav = 1
	}

	result, err := d.db.Exec(`
		UPDATE saved_filters
		SET name = ?, aql = ?, product = ?, description = ?, is_favorite = ?, updated_at = ?
		WHERE id = ?
	`, f.Name, f.AQL, nullString(f.Product), nullString(f.Description), isFav, f.UpdatedAt, f.ID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteFilter deletes a saved filter by ID.
func (d *DB) DeleteFilter(id string) error {
	result, err := d.db.Exec("DELETE FROM saved_filters WHERE id = ?", id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// generateFilterID generates a unique filter ID.
func generateFilterID() string {
	return fmt.Sprintf("filter_%d", time.Now().UnixNano())
}

// nullString returns a sql.NullString for empty strings.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
