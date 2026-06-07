package graph

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// NodeLabel represents a Neo4j node label.
type NodeLabel string

const (
	NodeProduct     NodeLabel = "Product"
	NodeRelease     NodeLabel = "Release"
	NodeFeature     NodeLabel = "Feature"
	NodeRequirement NodeLabel = "Requirement"
	NodeEpic        NodeLabel = "Epic"
	NodeInitiative  NodeLabel = "Initiative"
	NodeGoal        NodeLabel = "Goal"
	NodeIdea        NodeLabel = "Idea"
	NodeUser        NodeLabel = "User"
	NodeComment     NodeLabel = "Comment"
	NodeTag         NodeLabel = "Tag"
)

// RelationType represents a Neo4j relationship type.
type RelationType string

const (
	// Containment relationships
	RelContains         RelationType = "CONTAINS"           // Product → Release, Feature, Epic, etc.
	RelBelongsTo        RelationType = "BELONGS_TO"         // Feature → Product
	RelInRelease        RelationType = "IN_RELEASE"         // Feature, Epic → Release
	RelHasRequirement   RelationType = "HAS_REQUIREMENT"    // Feature → Requirement
	RelPartOfInitiative RelationType = "PART_OF_INITIATIVE" // Feature, Epic → Initiative
	RelSupportsGoal     RelationType = "SUPPORTS_GOAL"      // Initiative, Feature → Goal

	// People relationships
	RelAssignedTo RelationType = "ASSIGNED_TO" // Feature, Requirement → User
	RelCreatedBy  RelationType = "CREATED_BY"  // Idea, Comment → User
	RelOwnedBy    RelationType = "OWNED_BY"    // Product, Initiative → User

	// Dependency/linking relationships
	RelDependsOn  RelationType = "DEPENDS_ON" // Feature → Feature
	RelBlocks     RelationType = "BLOCKS"     // Feature → Feature
	RelImplements RelationType = "IMPLEMENTS" // Feature → Idea
	RelRelatedTo  RelationType = "RELATED_TO" // Generic relationship
	RelParentOf   RelationType = "PARENT_OF"  // Epic → Feature, Initiative → Epic
	RelChildOf    RelationType = "CHILD_OF"   // Feature → Epic

	// Tagging
	RelTaggedWith RelationType = "TAGGED_WITH" // Feature, Epic → Tag

	// Comments
	RelCommentOn RelationType = "COMMENT_ON" // Comment → Feature, Idea, etc.

	// Time-based
	RelPrecedes RelationType = "PRECEDES" // Release → Release
	RelFollows  RelationType = "FOLLOWS"  // Release → Release
)

// SchemaConstraints returns Cypher statements to create schema constraints.
func SchemaConstraints() []string {
	return []string{
		// Unique constraints on ID
		"CREATE CONSTRAINT product_id IF NOT EXISTS FOR (n:Product) REQUIRE n.id IS UNIQUE",
		"CREATE CONSTRAINT release_id IF NOT EXISTS FOR (n:Release) REQUIRE n.id IS UNIQUE",
		"CREATE CONSTRAINT feature_id IF NOT EXISTS FOR (n:Feature) REQUIRE n.id IS UNIQUE",
		"CREATE CONSTRAINT requirement_id IF NOT EXISTS FOR (n:Requirement) REQUIRE n.id IS UNIQUE",
		"CREATE CONSTRAINT epic_id IF NOT EXISTS FOR (n:Epic) REQUIRE n.id IS UNIQUE",
		"CREATE CONSTRAINT initiative_id IF NOT EXISTS FOR (n:Initiative) REQUIRE n.id IS UNIQUE",
		"CREATE CONSTRAINT goal_id IF NOT EXISTS FOR (n:Goal) REQUIRE n.id IS UNIQUE",
		"CREATE CONSTRAINT idea_id IF NOT EXISTS FOR (n:Idea) REQUIRE n.id IS UNIQUE",
		"CREATE CONSTRAINT user_id IF NOT EXISTS FOR (n:User) REQUIRE n.id IS UNIQUE",
		"CREATE CONSTRAINT comment_id IF NOT EXISTS FOR (n:Comment) REQUIRE n.id IS UNIQUE",
		"CREATE CONSTRAINT tag_name IF NOT EXISTS FOR (n:Tag) REQUIRE n.name IS UNIQUE",

		// Reference number uniqueness
		"CREATE CONSTRAINT feature_ref IF NOT EXISTS FOR (n:Feature) REQUIRE n.reference_num IS UNIQUE",
		"CREATE CONSTRAINT release_ref IF NOT EXISTS FOR (n:Release) REQUIRE n.reference_num IS UNIQUE",
		"CREATE CONSTRAINT epic_ref IF NOT EXISTS FOR (n:Epic) REQUIRE n.reference_num IS UNIQUE",
		"CREATE CONSTRAINT initiative_ref IF NOT EXISTS FOR (n:Initiative) REQUIRE n.reference_num IS UNIQUE",
		"CREATE CONSTRAINT goal_ref IF NOT EXISTS FOR (n:Goal) REQUIRE n.reference_num IS UNIQUE",
		"CREATE CONSTRAINT idea_ref IF NOT EXISTS FOR (n:Idea) REQUIRE n.reference_num IS UNIQUE",
	}
}

// SchemaIndexes returns Cypher statements to create indexes for query performance.
func SchemaIndexes() []string {
	return []string{
		// Full-text search indexes
		"CREATE FULLTEXT INDEX feature_search IF NOT EXISTS FOR (n:Feature) ON EACH [n.name, n.description]",
		"CREATE FULLTEXT INDEX idea_search IF NOT EXISTS FOR (n:Idea) ON EACH [n.name, n.description]",
		"CREATE FULLTEXT INDEX initiative_search IF NOT EXISTS FOR (n:Initiative) ON EACH [n.name, n.description]",
		"CREATE FULLTEXT INDEX epic_search IF NOT EXISTS FOR (n:Epic) ON EACH [n.name, n.description]",

		// Property indexes for common filters
		"CREATE INDEX feature_status IF NOT EXISTS FOR (n:Feature) ON (n.status)",
		"CREATE INDEX feature_updated IF NOT EXISTS FOR (n:Feature) ON (n.updated_at)",
		"CREATE INDEX idea_status IF NOT EXISTS FOR (n:Idea) ON (n.status)",
		"CREATE INDEX idea_votes IF NOT EXISTS FOR (n:Idea) ON (n.votes)",
		"CREATE INDEX release_date IF NOT EXISTS FOR (n:Release) ON (n.release_date)",
		"CREATE INDEX release_released IF NOT EXISTS FOR (n:Release) ON (n.released)",
		"CREATE INDEX initiative_status IF NOT EXISTS FOR (n:Initiative) ON (n.status)",
		"CREATE INDEX goal_status IF NOT EXISTS FOR (n:Goal) ON (n.status)",
		"CREATE INDEX epic_status IF NOT EXISTS FOR (n:Epic) ON (n.status)",
		"CREATE INDEX user_email IF NOT EXISTS FOR (n:User) ON (n.email)",
	}
}

// CreateSchema creates all constraints and indexes in the database.
func (c *Client) CreateSchema(ctx context.Context) error {
	session := c.Session(ctx)
	defer session.Close(ctx)

	// Create constraints
	for _, constraint := range SchemaConstraints() {
		if _, err := session.Run(ctx, constraint, nil); err != nil {
			return fmt.Errorf("creating constraint: %w", err)
		}
	}

	// Create indexes
	for _, index := range SchemaIndexes() {
		if _, err := session.Run(ctx, index, nil); err != nil {
			return fmt.Errorf("creating index: %w", err)
		}
	}

	return nil
}

// DropSchema drops all constraints and indexes (use with caution).
func (c *Client) DropSchema(ctx context.Context) error {
	session := c.Session(ctx)
	defer session.Close(ctx)

	// Get all constraints
	result, err := session.Run(ctx, "SHOW CONSTRAINTS", nil)
	if err != nil {
		return fmt.Errorf("listing constraints: %w", err)
	}

	records, err := result.Collect(ctx)
	if err != nil {
		return fmt.Errorf("collecting constraints: %w", err)
	}

	for _, record := range records {
		name, _ := record.Get("name")
		if name != nil {
			dropQuery := fmt.Sprintf("DROP CONSTRAINT %s IF EXISTS", name)
			if _, err := session.Run(ctx, dropQuery, nil); err != nil {
				return fmt.Errorf("dropping constraint %s: %w", name, err)
			}
		}
	}

	// Get all indexes
	result, err = session.Run(ctx, "SHOW INDEXES", nil)
	if err != nil {
		return fmt.Errorf("listing indexes: %w", err)
	}

	records, err = result.Collect(ctx)
	if err != nil {
		return fmt.Errorf("collecting indexes: %w", err)
	}

	for _, record := range records {
		name, _ := record.Get("name")
		indexType, _ := record.Get("type")
		// Don't drop lookup indexes
		if name != nil && indexType != "LOOKUP" {
			dropQuery := fmt.Sprintf("DROP INDEX %s IF EXISTS", name)
			if _, err := session.Run(ctx, dropQuery, nil); err != nil {
				return fmt.Errorf("dropping index %s: %w", name, err)
			}
		}
	}

	return nil
}

// MergeNode creates or updates a node with the given label and properties.
func MergeNode(label NodeLabel, id string, props map[string]any) string {
	return fmt.Sprintf(`
		MERGE (n:%s {id: $id})
		SET n += $props
		RETURN n
	`, label)
}

// MergeRelationship creates or updates a relationship between two nodes.
func MergeRelationship(fromLabel NodeLabel, toLabel NodeLabel, relType RelationType) string {
	return fmt.Sprintf(`
		MATCH (a:%s {id: $fromId})
		MATCH (b:%s {id: $toId})
		MERGE (a)-[r:%s]->(b)
		RETURN r
	`, fromLabel, toLabel, relType)
}

// CreateRelationshipIfExists creates a relationship only if both nodes exist.
func (c *Client) CreateRelationshipIfExists(ctx context.Context, fromLabel NodeLabel, fromID string, toLabel NodeLabel, toID string, relType RelationType) error {
	if fromID == "" || toID == "" {
		return nil // Skip if either ID is empty
	}

	_, err := c.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, MergeRelationship(fromLabel, toLabel, relType), map[string]any{
			"fromId": fromID,
			"toId":   toID,
		})
		if err != nil {
			return nil, err
		}
		return result.Consume(ctx)
	})
	return err
}
