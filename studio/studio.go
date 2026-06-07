// Package studio provides AQL (Aha Query Language) - a JQL-like query layer for Aha.io.
//
// Studio supports querying Ideas, Features, Releases, and Initiatives using SQL-like syntax.
//
// Example usage:
//
//	client, _ := aha.NewClient()
//	studio := studio.New(client)
//	result, _ := studio.Query(ctx, "FROM features WHERE status = 'In Progress' LIMIT 10")
//	for _, record := range result.Records {
//	    fmt.Println(record["name"])
//	}
package studio

import (
	"context"
	"fmt"

	aha "github.com/grokify/aha-go"
	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/aql/parser"
	"github.com/grokify/aha-studio/aql/validator"
	"github.com/grokify/aha-studio/executor"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/result"
)

// Version is the library version.
const Version = "0.1.0"

// Studio provides the main interface for AQL queries.
type Studio struct {
	client   *aha.Client
	executor *executor.Executor
}

// New creates a new Studio instance with the given Aha client.
func New(client *aha.Client) *Studio {
	return &Studio{
		client:   client,
		executor: executor.New(client),
	}
}

// Query executes an AQL query and returns the results.
func (s *Studio) Query(ctx context.Context, aql string) (*result.Result, error) {
	return s.QueryWithOptions(ctx, aql, QueryOptions{})
}

// QueryOptions configures query execution.
type QueryOptions struct {
	// ProductID is required for releases queries.
	ProductID string

	// ReleaseID is required for release-scoped feature queries.
	ReleaseID string
}

// QueryWithOptions executes an AQL query with the given options.
func (s *Studio) QueryWithOptions(ctx context.Context, aql string, opts QueryOptions) (*result.Result, error) {
	// Parse the query
	p := parser.New(aql)
	query, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Validate the query
	v := validator.New()
	if err := v.Validate(query); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Create execution plan
	pl := planner.New()
	plan := pl.Plan(query)

	// Apply options
	if opts.ProductID != "" {
		plan.APIParams.ProductID = opts.ProductID
	}
	if opts.ReleaseID != "" {
		plan.APIParams.ReleaseID = opts.ReleaseID
	}

	// Execute the query
	return s.executor.Execute(ctx, plan)
}

// Parse parses an AQL query and returns the AST.
// This is useful for query validation without execution.
func Parse(aql string) (*ast.Query, error) {
	p := parser.New(aql)
	return p.Parse()
}

// Validate validates an AQL query.
// Returns nil if the query is valid.
func Validate(aql string) error {
	query, err := Parse(aql)
	if err != nil {
		return err
	}

	v := validator.New()
	return v.Validate(query)
}

// Plan creates an execution plan for an AQL query.
// This is useful for inspecting how a query will be executed.
func Plan(aql string) (*planner.Plan, error) {
	query, err := Parse(aql)
	if err != nil {
		return nil, err
	}

	v := validator.New()
	if err := v.Validate(query); err != nil {
		return nil, err
	}

	pl := planner.New()
	return pl.Plan(query), nil
}

// Client returns the underlying Aha client.
func (s *Studio) Client() *aha.Client {
	return s.client
}
