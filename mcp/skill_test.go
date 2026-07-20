package mcp

import (
	"testing"

	"github.com/plexusone/omniskill/skill"
)

func TestNewAhaSkill(t *testing.T) {
	cfg := &Config{
		Subdomain: "test-company",
		APIKey:    "test-key",
	}

	s := NewAhaSkill(cfg)
	if s == nil {
		t.Fatal("NewAhaSkill() returned nil")
	}
	if s.handlers == nil {
		t.Error("NewAhaSkill() handlers is nil")
	}
}

func TestAhaSkill_Name(t *testing.T) {
	cfg := &Config{
		Subdomain: "test-company",
		APIKey:    "test-key",
	}

	s := NewAhaSkill(cfg)
	got := s.Name()
	want := "aha"
	if got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
}

func TestAhaSkill_Description(t *testing.T) {
	cfg := &Config{
		Subdomain: "test-company",
		APIKey:    "test-key",
	}

	s := NewAhaSkill(cfg)
	got := s.Description()
	if got == "" {
		t.Error("Description() returned empty string")
	}
	// Check it contains expected keywords
	if !containsString(got, "AQL") && !containsString(got, "Aha") {
		t.Errorf("Description() = %q, expected to contain 'AQL' or 'Aha'", got)
	}
}

func TestAhaSkill_Tools(t *testing.T) {
	cfg := &Config{
		Subdomain: "test-company",
		APIKey:    "test-key",
	}

	s := NewAhaSkill(cfg)
	tools := s.Tools()

	// Verify we have the expected number of tools (71 as per documentation)
	expectedCount := 71
	if len(tools) != expectedCount {
		t.Errorf("Tools() returned %d tools, want %d", len(tools), expectedCount)
	}
}

func TestAhaSkill_ToolNames(t *testing.T) {
	cfg := &Config{
		Subdomain: "test-company",
		APIKey:    "test-key",
	}

	s := NewAhaSkill(cfg)
	tools := s.Tools()

	// Build a set of tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name()] = true
	}

	// Verify essential tools exist
	essentialTools := []string{
		// Query tools
		"query",
		"describe_aql",
		"graph_query",
		"graph_sync",
		// Get tools
		"get_feature",
		"get_idea",
		"get_release",
		"get_initiative",
		"get_epic",
		"get_goal",
		"get_current_user",
		"get_strategic_model",
		// List tools
		"list_ideas",
		"list_products",
		"list_features",
		"list_release_features",
		"list_epics",
		"list_product_epics",
		"list_goals",
		"list_product_goals",
		"list_initiatives",
		"list_product_initiatives",
		"list_feature_requirements",
		"list_users",
		"list_strategic_models",
		"list_product_strategic_models",
		"list_custom_fields",
		// Create tools
		"create_feature",
		"create_epic",
		"create_goal",
		"create_initiative",
		"create_requirement",
		"create_product",
		"create_strategic_model",
		// Update tools
		"update_initiative",
		"update_epic",
		"update_goal",
		"update_feature",
		"update_requirement",
		"update_release",
		"update_idea",
		"update_product",
		"update_strategic_model",
		// Status and assignment tools
		"change_feature_status",
		"assign_feature_release",
		// Comment tools
		"add_feature_comment",
		"add_idea_comment",
		"update_comment",
		"list_feature_comments",
		"list_idea_comments",
		"list_epic_comments",
		// Delete tools
		"delete_requirement",
		"delete_comment",
	}

	for _, name := range essentialTools {
		if !toolNames[name] {
			t.Errorf("Tools() missing essential tool %q", name)
		}
	}
}

func TestAhaSkill_ToolSchemas(t *testing.T) {
	cfg := &Config{
		Subdomain: "test-company",
		APIKey:    "test-key",
	}

	s := NewAhaSkill(cfg)
	tools := s.Tools()

	// Build a map for easy lookup
	toolMap := make(map[string]skill.Tool)
	for _, tool := range tools {
		toolMap[tool.Name()] = tool
	}

	// Test query tool schema
	queryTool, ok := toolMap["query"]
	if !ok {
		t.Fatal("query tool not found")
	}

	params := queryTool.Parameters()
	if _, ok := params["query"]; !ok {
		t.Error("query tool missing 'query' parameter")
	}
	if params["query"].Type != "string" {
		t.Errorf("query parameter type = %q, want 'string'", params["query"].Type)
	}
	if !params["query"].Required {
		t.Error("query parameter should be required")
	}

	// Test get_feature tool schema
	getFeatureTool, ok := toolMap["get_feature"]
	if !ok {
		t.Fatal("get_feature tool not found")
	}

	featureParams := getFeatureTool.Parameters()
	if _, ok := featureParams["reference"]; !ok {
		t.Error("get_feature tool missing 'reference' parameter")
	}
	if !featureParams["reference"].Required {
		t.Error("get_feature reference parameter should be required")
	}

	// Test create_feature tool schema
	createFeatureTool, ok := toolMap["create_feature"]
	if !ok {
		t.Fatal("create_feature tool not found")
	}

	createParams := createFeatureTool.Parameters()
	requiredParams := []string{"release_id", "name"}
	for _, param := range requiredParams {
		if p, ok := createParams[param]; !ok {
			t.Errorf("create_feature tool missing '%s' parameter", param)
		} else if !p.Required {
			t.Errorf("create_feature '%s' parameter should be required", param)
		}
	}

	optionalParams := []string{"description", "workflow_status", "assigned_to_user"}
	for _, param := range optionalParams {
		if p, ok := createParams[param]; !ok {
			t.Errorf("create_feature tool missing '%s' parameter", param)
		} else if p.Required {
			t.Errorf("create_feature '%s' parameter should be optional", param)
		}
	}
}

func TestAhaSkill_Close(t *testing.T) {
	cfg := &Config{
		Subdomain: "test-company",
		APIKey:    "test-key",
	}

	s := NewAhaSkill(cfg)
	err := s.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
