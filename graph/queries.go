package graph

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// QueryResult holds the result of a graph query.
type QueryResult struct {
	Nodes         []map[string]any   `json:"nodes,omitempty"`
	Relationships []map[string]any   `json:"relationships,omitempty"`
	Paths         [][]map[string]any `json:"paths,omitempty"`
	Records       []map[string]any   `json:"records,omitempty"`
}

// FindConnectedFeatures finds features connected to a given entity.
func (c *Client) FindConnectedFeatures(ctx context.Context, entityType NodeLabel, entityID string, depth int) (*QueryResult, error) {
	if depth <= 0 {
		depth = 2
	}

	query := fmt.Sprintf(`
		MATCH (start:%s {id: $entityId})
		MATCH path = (start)-[*1..%d]-(f:Feature)
		RETURN DISTINCT f AS feature, length(path) AS distance
		ORDER BY distance
		LIMIT 100
	`, entityType, depth)

	result, err := c.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, query, map[string]any{"entityId": entityID})
		if err != nil {
			return nil, err
		}
		return records.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}

	qr := &QueryResult{Records: make([]map[string]any, 0)}
	for _, record := range result.([]*neo4j.Record) {
		node, _ := record.Get("feature")
		distance, _ := record.Get("distance")
		if n, ok := node.(neo4j.Node); ok {
			qr.Records = append(qr.Records, map[string]any{
				"feature":  nodeToMap(n),
				"distance": distance,
			})
		}
	}

	return qr, nil
}

// FindPath finds the shortest path between two entities.
func (c *Client) FindPath(ctx context.Context, fromType NodeLabel, fromID string, toType NodeLabel, toID string) (*QueryResult, error) {
	query := fmt.Sprintf(`
		MATCH (start:%s {id: $fromId}), (end:%s {id: $toId})
		MATCH path = shortestPath((start)-[*..10]-(end))
		RETURN path
	`, fromType, toType)

	result, err := c.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, query, map[string]any{
			"fromId": fromID,
			"toId":   toID,
		})
		if err != nil {
			return nil, err
		}
		return records.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}

	qr := &QueryResult{Paths: make([][]map[string]any, 0)}
	for _, record := range result.([]*neo4j.Record) {
		path, _ := record.Get("path")
		if p, ok := path.(neo4j.Path); ok {
			pathNodes := make([]map[string]any, 0)
			for _, node := range p.Nodes {
				pathNodes = append(pathNodes, nodeToMap(node))
			}
			qr.Paths = append(qr.Paths, pathNodes)
		}
	}

	return qr, nil
}

// GetReleaseDependencies returns all features in a release and their dependencies.
func (c *Client) GetReleaseDependencies(ctx context.Context, releaseID string) (*QueryResult, error) {
	query := `
		MATCH (r:Release {id: $releaseId})<-[:IN_RELEASE]-(f:Feature)
		OPTIONAL MATCH (f)-[:DEPENDS_ON]->(dep:Feature)
		OPTIONAL MATCH (f)<-[:DEPENDS_ON]-(depBy:Feature)
		RETURN f AS feature,
		       collect(DISTINCT dep) AS dependsOn,
		       collect(DISTINCT depBy) AS dependedBy
	`

	result, err := c.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, query, map[string]any{"releaseId": releaseID})
		if err != nil {
			return nil, err
		}
		return records.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}

	qr := &QueryResult{Records: make([]map[string]any, 0)}
	for _, record := range result.([]*neo4j.Record) {
		feature, _ := record.Get("feature")
		dependsOn, _ := record.Get("dependsOn")
		dependedBy, _ := record.Get("dependedBy")

		rec := map[string]any{}
		if f, ok := feature.(neo4j.Node); ok {
			rec["feature"] = nodeToMap(f)
		}
		if deps, ok := dependsOn.([]any); ok {
			depsArr := make([]map[string]any, 0)
			for _, d := range deps {
				if n, ok := d.(neo4j.Node); ok {
					depsArr = append(depsArr, nodeToMap(n))
				}
			}
			rec["depends_on"] = depsArr
		}
		if deps, ok := dependedBy.([]any); ok {
			depsArr := make([]map[string]any, 0)
			for _, d := range deps {
				if n, ok := d.(neo4j.Node); ok {
					depsArr = append(depsArr, nodeToMap(n))
				}
			}
			rec["depended_by"] = depsArr
		}
		qr.Records = append(qr.Records, rec)
	}

	return qr, nil
}

// GetInitiativeImpact returns all entities connected to an initiative.
func (c *Client) GetInitiativeImpact(ctx context.Context, initiativeID string) (*QueryResult, error) {
	query := `
		MATCH (i:Initiative {id: $initiativeId})
		OPTIONAL MATCH (i)<-[:PART_OF_INITIATIVE]-(f:Feature)
		OPTIONAL MATCH (i)<-[:PART_OF_INITIATIVE]-(e:Epic)
		OPTIONAL MATCH (i)-[:SUPPORTS_GOAL]->(g:Goal)
		OPTIONAL MATCH (f)-[:IN_RELEASE]->(r:Release)
		RETURN i AS initiative,
		       collect(DISTINCT f) AS features,
		       collect(DISTINCT e) AS epics,
		       collect(DISTINCT g) AS goals,
		       collect(DISTINCT r) AS releases
	`

	result, err := c.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, query, map[string]any{"initiativeId": initiativeID})
		if err != nil {
			return nil, err
		}
		return records.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}

	qr := &QueryResult{Records: make([]map[string]any, 0)}
	for _, record := range result.([]*neo4j.Record) {
		rec := map[string]any{}

		if init, _ := record.Get("initiative"); init != nil {
			if n, ok := init.(neo4j.Node); ok {
				rec["initiative"] = nodeToMap(n)
			}
		}

		for _, key := range []string{"features", "epics", "goals", "releases"} {
			if val, _ := record.Get(key); val != nil {
				if arr, ok := val.([]any); ok {
					items := make([]map[string]any, 0)
					for _, item := range arr {
						if n, ok := item.(neo4j.Node); ok {
							items = append(items, nodeToMap(n))
						}
					}
					rec[key] = items
				}
			}
		}

		qr.Records = append(qr.Records, rec)
	}

	return qr, nil
}

// GetIdeaImplementations returns features that implement ideas.
func (c *Client) GetIdeaImplementations(ctx context.Context, minVotes int) (*QueryResult, error) {
	query := `
		MATCH (idea:Idea)
		WHERE idea.votes >= $minVotes
		OPTIONAL MATCH (idea)<-[:IMPLEMENTS]-(f:Feature)
		RETURN idea, collect(f) AS features
		ORDER BY idea.votes DESC
		LIMIT 100
	`

	result, err := c.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, query, map[string]any{"minVotes": minVotes})
		if err != nil {
			return nil, err
		}
		return records.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}

	qr := &QueryResult{Records: make([]map[string]any, 0)}
	for _, record := range result.([]*neo4j.Record) {
		rec := map[string]any{}

		if idea, _ := record.Get("idea"); idea != nil {
			if n, ok := idea.(neo4j.Node); ok {
				rec["idea"] = nodeToMap(n)
			}
		}

		if features, _ := record.Get("features"); features != nil {
			if arr, ok := features.([]any); ok {
				items := make([]map[string]any, 0)
				for _, item := range arr {
					if n, ok := item.(neo4j.Node); ok {
						items = append(items, nodeToMap(n))
					}
				}
				rec["features"] = items
			}
		}

		qr.Records = append(qr.Records, rec)
	}

	return qr, nil
}

// FullTextSearch performs full-text search across entities.
func (c *Client) FullTextSearch(ctx context.Context, searchTerm string, entityTypes []NodeLabel) (*QueryResult, error) {
	// Build index names based on entity types
	var indexes []string
	for _, et := range entityTypes {
		switch et {
		case NodeFeature:
			indexes = append(indexes, "feature_search")
		case NodeIdea:
			indexes = append(indexes, "idea_search")
		case NodeInitiative:
			indexes = append(indexes, "initiative_search")
		case NodeEpic:
			indexes = append(indexes, "epic_search")
		}
	}

	if len(indexes) == 0 {
		indexes = []string{"feature_search", "idea_search", "initiative_search", "epic_search"}
	}

	qr := &QueryResult{Records: make([]map[string]any, 0)}

	for _, indexName := range indexes {
		query := fmt.Sprintf(`
			CALL db.index.fulltext.queryNodes('%s', $searchTerm)
			YIELD node, score
			RETURN node, score
			ORDER BY score DESC
			LIMIT 25
		`, indexName)

		result, err := c.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			records, err := tx.Run(ctx, query, map[string]any{"searchTerm": searchTerm})
			if err != nil {
				return nil, err
			}
			return records.Collect(ctx)
		})
		if err != nil {
			continue // Skip failed indexes
		}

		for _, record := range result.([]*neo4j.Record) {
			node, _ := record.Get("node")
			score, _ := record.Get("score")
			if n, ok := node.(neo4j.Node); ok {
				rec := nodeToMap(n)
				rec["_score"] = score
				rec["_labels"] = n.Labels
				qr.Records = append(qr.Records, rec)
			}
		}
	}

	return qr, nil
}

// GetProductOverview returns a summary of all entities for a product.
func (c *Client) GetProductOverview(ctx context.Context, productID string) (*QueryResult, error) {
	query := `
		MATCH (p:Product {id: $productId})
		OPTIONAL MATCH (p)-[:CONTAINS]->(r:Release)
		OPTIONAL MATCH (p)-[:CONTAINS]->(f:Feature)
		OPTIONAL MATCH (p)-[:CONTAINS]->(e:Epic)
		OPTIONAL MATCH (p)-[:CONTAINS]->(i:Initiative)
		OPTIONAL MATCH (p)-[:CONTAINS]->(g:Goal)
		RETURN p AS product,
		       count(DISTINCT r) AS releaseCount,
		       count(DISTINCT f) AS featureCount,
		       count(DISTINCT e) AS epicCount,
		       count(DISTINCT i) AS initiativeCount,
		       count(DISTINCT g) AS goalCount
	`

	result, err := c.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, query, map[string]any{"productId": productID})
		if err != nil {
			return nil, err
		}
		return records.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}

	qr := &QueryResult{Records: make([]map[string]any, 0)}
	for _, record := range result.([]*neo4j.Record) {
		rec := map[string]any{}

		if prod, _ := record.Get("product"); prod != nil {
			if n, ok := prod.(neo4j.Node); ok {
				rec["product"] = nodeToMap(n)
			}
		}

		for _, key := range []string{"releaseCount", "featureCount", "epicCount", "initiativeCount", "goalCount"} {
			if val, _ := record.Get(key); val != nil {
				rec[key] = val
			}
		}

		qr.Records = append(qr.Records, rec)
	}

	return qr, nil
}

// RunCypher executes a raw Cypher query.
func (c *Client) RunCypher(ctx context.Context, cypher string, params map[string]any) (*QueryResult, error) {
	result, err := c.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		return records.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}

	qr := &QueryResult{Records: make([]map[string]any, 0)}
	for _, record := range result.([]*neo4j.Record) {
		rec := make(map[string]any)
		for _, key := range record.Keys {
			val, _ := record.Get(key)
			switch v := val.(type) {
			case neo4j.Node:
				rec[key] = nodeToMap(v)
			case neo4j.Relationship:
				rec[key] = relToMap(v)
			case neo4j.Path:
				pathNodes := make([]map[string]any, 0)
				for _, node := range v.Nodes {
					pathNodes = append(pathNodes, nodeToMap(node))
				}
				rec[key] = pathNodes
			default:
				rec[key] = val
			}
		}
		qr.Records = append(qr.Records, rec)
	}

	return qr, nil
}

// Helper functions

func nodeToMap(n neo4j.Node) map[string]any {
	m := make(map[string]any)
	m["_id"] = n.GetElementId()
	m["_labels"] = n.Labels
	for k, v := range n.Props {
		m[k] = v
	}
	return m
}

func relToMap(r neo4j.Relationship) map[string]any {
	m := make(map[string]any)
	m["_id"] = r.GetElementId()
	m["_type"] = r.Type
	m["_start"] = r.StartElementId
	m["_end"] = r.EndElementId
	for k, v := range r.Props {
		m[k] = v
	}
	return m
}
