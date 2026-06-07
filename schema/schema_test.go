package schema

import (
	"testing"

	"github.com/grokify/aha-studio/aql/ast"
)

func TestGetEntity(t *testing.T) {
	tests := []struct {
		entityType ast.EntityType
		exists     bool
	}{
		{ast.EntityFeatures, true},
		{ast.EntityIdeas, true},
		{ast.EntityReleases, true},
		{ast.EntityInitiatives, true},
		{ast.EntityType("unknown"), false},
	}

	for _, tt := range tests {
		entity := GetEntity(tt.entityType)
		if tt.exists {
			if entity == nil {
				t.Errorf("GetEntity(%s) returned nil, expected entity", tt.entityType)
			} else if entity.Name != tt.entityType {
				t.Errorf("GetEntity(%s).Name = %s, want %s", tt.entityType, entity.Name, tt.entityType)
			}
		} else {
			if entity != nil {
				t.Errorf("GetEntity(%s) = %v, expected nil", tt.entityType, entity)
			}
		}
	}
}

func TestAllEntities(t *testing.T) {
	entities := AllEntities()

	// Should have 11 entities
	if len(entities) != 11 {
		t.Errorf("expected 11 entities, got %d", len(entities))
	}

	// Check each expected entity exists
	expected := []ast.EntityType{
		ast.EntityComments,
		ast.EntityEpics,
		ast.EntityFeatures,
		ast.EntityGoals,
		ast.EntityIdeas,
		ast.EntityReleases,
		ast.EntityRequirements,
		ast.EntityInitiatives,
		ast.EntityProducts,
		ast.EntityTags,
		ast.EntityUsers,
	}

	for _, e := range expected {
		if _, ok := entities[e]; !ok {
			t.Errorf("missing entity %s", e)
		}
	}
}

func TestEntityField(t *testing.T) {
	entity := GetEntity(ast.EntityFeatures)
	if entity == nil {
		t.Fatal("GetEntity(features) returned nil")
	}

	// Test existing fields
	field := entity.Field("name")
	if field == nil {
		t.Error("Field('name') returned nil")
	} else if field.Name != "name" {
		t.Errorf("Field('name').Name = %s, want 'name'", field.Name)
	}

	// Test non-existent field
	field = entity.Field("nonexistent")
	if field != nil {
		t.Errorf("Field('nonexistent') = %v, expected nil", field)
	}
}

func TestEntityHasField(t *testing.T) {
	entity := GetEntity(ast.EntityFeatures)
	if entity == nil {
		t.Fatal("GetEntity(features) returned nil")
	}

	tests := []struct {
		fieldName string
		exists    bool
	}{
		{"name", true},
		{"status", true},
		{"id", true},
		{"reference_num", true},
		{"nonexistent", false},
		{"", false},
	}

	for _, tt := range tests {
		if entity.HasField(tt.fieldName) != tt.exists {
			t.Errorf("HasField(%q) = %v, want %v", tt.fieldName, !tt.exists, tt.exists)
		}
	}
}

func TestFeatureEntityFields(t *testing.T) {
	entity := GetEntity(ast.EntityFeatures)
	if entity == nil {
		t.Fatal("GetEntity(features) returned nil")
	}

	// Check display name
	if entity.DisplayName != "Feature" {
		t.Errorf("DisplayName = %s, want 'Feature'", entity.DisplayName)
	}

	// Check some expected fields
	expectedFields := []string{
		"id", "reference_num", "name", "description", "status",
		"assigned_to", "tags", "tag", "created_at", "updated_at", "url",
	}

	for _, f := range expectedFields {
		if !entity.HasField(f) {
			t.Errorf("missing expected field %s", f)
		}
	}
}

func TestIdeaEntityFields(t *testing.T) {
	entity := GetEntity(ast.EntityIdeas)
	if entity == nil {
		t.Fatal("GetEntity(ideas) returned nil")
	}

	// Check display name
	if entity.DisplayName != "Idea" {
		t.Errorf("DisplayName = %s, want 'Idea'", entity.DisplayName)
	}

	// Check votes field (specific to ideas)
	votesField := entity.Field("votes")
	if votesField == nil {
		t.Fatal("Field('votes') returned nil")
	}
	if votesField.Type != FieldTypeInt {
		t.Errorf("votes.Type = %s, want FieldTypeInt", votesField.Type)
	}
	if !votesField.Filterable {
		t.Error("votes should be filterable")
	}
	if !votesField.Sortable {
		t.Error("votes should be sortable")
	}
}

func TestFieldTypes(t *testing.T) {
	entity := GetEntity(ast.EntityFeatures)
	if entity == nil {
		t.Fatal("GetEntity(features) returned nil")
	}

	typeTests := []struct {
		fieldName    string
		expectedType FieldType
	}{
		{"id", FieldTypeString},
		{"name", FieldTypeString},
		{"start_date", FieldTypeDate},
		{"created_at", FieldTypeDatetime},
		{"tags", FieldTypeStringArray},
	}

	for _, tt := range typeTests {
		field := entity.Field(tt.fieldName)
		if field == nil {
			t.Errorf("Field(%q) returned nil", tt.fieldName)
			continue
		}
		if field.Type != tt.expectedType {
			t.Errorf("Field(%q).Type = %s, want %s", tt.fieldName, field.Type, tt.expectedType)
		}
	}
}

func TestFieldFilterable(t *testing.T) {
	entity := GetEntity(ast.EntityFeatures)
	if entity == nil {
		t.Fatal("GetEntity(features) returned nil")
	}

	filterableFields := []string{
		"id", "name", "status", "assigned_to", "tag", "created_at",
	}
	nonFilterableFields := []string{
		"description", "url",
	}

	for _, f := range filterableFields {
		field := entity.Field(f)
		if field == nil {
			t.Errorf("Field(%q) returned nil", f)
			continue
		}
		if !field.Filterable {
			t.Errorf("Field(%q) should be filterable", f)
		}
	}

	for _, f := range nonFilterableFields {
		field := entity.Field(f)
		if field == nil {
			t.Errorf("Field(%q) returned nil", f)
			continue
		}
		if field.Filterable {
			t.Errorf("Field(%q) should not be filterable", f)
		}
	}
}

func TestFieldSortable(t *testing.T) {
	entity := GetEntity(ast.EntityFeatures)
	if entity == nil {
		t.Fatal("GetEntity(features) returned nil")
	}

	sortableFields := []string{
		"id", "name", "status", "created_at", "updated_at",
	}
	nonSortableFields := []string{
		"description", "tags", "url",
	}

	for _, f := range sortableFields {
		field := entity.Field(f)
		if field == nil {
			t.Errorf("Field(%q) returned nil", f)
			continue
		}
		if !field.Sortable {
			t.Errorf("Field(%q) should be sortable", f)
		}
	}

	for _, f := range nonSortableFields {
		field := entity.Field(f)
		if field == nil {
			t.Errorf("Field(%q) returned nil", f)
			continue
		}
		if field.Sortable {
			t.Errorf("Field(%q) should not be sortable", f)
		}
	}
}

func TestFieldAPIParam(t *testing.T) {
	entity := GetEntity(ast.EntityFeatures)
	if entity == nil {
		t.Fatal("GetEntity(features) returned nil")
	}

	apiParamTests := []struct {
		fieldName string
		apiParam  string
	}{
		{"name", "q"},
		{"status", "workflow_status"},
		{"tag", "tag"},
		{"assigned_to", "assigned_to_user"},
		{"updated_at", "updated_since"},
		{"id", ""}, // no API param
	}

	for _, tt := range apiParamTests {
		field := entity.Field(tt.fieldName)
		if field == nil {
			t.Errorf("Field(%q) returned nil", tt.fieldName)
			continue
		}
		if field.APIParam != tt.apiParam {
			t.Errorf("Field(%q).APIParam = %q, want %q", tt.fieldName, field.APIParam, tt.apiParam)
		}
	}
}

func TestFieldIsPushable(t *testing.T) {
	entity := GetEntity(ast.EntityFeatures)
	if entity == nil {
		t.Fatal("GetEntity(features) returned nil")
	}

	pushableFields := []string{
		"name", "status", "tag", "assigned_to", "updated_at",
	}
	nonPushableFields := []string{
		"id", "description", "url",
	}

	for _, f := range pushableFields {
		field := entity.Field(f)
		if field == nil {
			t.Errorf("Field(%q) returned nil", f)
			continue
		}
		if !field.IsPushable() {
			t.Errorf("Field(%q) should be pushable (has APIParam=%q)", f, field.APIParam)
		}
	}

	for _, f := range nonPushableFields {
		field := entity.Field(f)
		if field == nil {
			t.Errorf("Field(%q) returned nil", f)
			continue
		}
		if field.IsPushable() {
			t.Errorf("Field(%q) should not be pushable", f)
		}
	}
}

func TestIsCustomFieldName(t *testing.T) {
	tests := []struct {
		fieldName string
		isCustom  bool
	}{
		{"custom.priority", true},
		{"custom.anything", true},
		{"name", false},
		{"status", false},
		{"", false},
		{"custom", false}, // just "custom" without dot is not custom
	}

	for _, tt := range tests {
		got := IsCustomFieldName(tt.fieldName)
		if got != tt.isCustom {
			t.Errorf("IsCustomFieldName(%q) = %v, want %v", tt.fieldName, got, tt.isCustom)
		}
	}
}

func TestReleaseEntityFields(t *testing.T) {
	entity := GetEntity(ast.EntityReleases)
	if entity == nil {
		t.Fatal("GetEntity(releases) returned nil")
	}

	// Check display name
	if entity.DisplayName != "Release" {
		t.Errorf("DisplayName = %s, want 'Release'", entity.DisplayName)
	}

	// Check release-specific fields
	releasedField := entity.Field("released")
	if releasedField == nil {
		t.Fatal("Field('released') returned nil")
	}
	if releasedField.Type != FieldTypeBool {
		t.Errorf("released.Type = %s, want FieldTypeBool", releasedField.Type)
	}

	parkingLotField := entity.Field("parking_lot")
	if parkingLotField == nil {
		t.Fatal("Field('parking_lot') returned nil")
	}
	if parkingLotField.Type != FieldTypeBool {
		t.Errorf("parking_lot.Type = %s, want FieldTypeBool", parkingLotField.Type)
	}
}

func TestInitiativeEntityFields(t *testing.T) {
	entity := GetEntity(ast.EntityInitiatives)
	if entity == nil {
		t.Fatal("GetEntity(initiatives) returned nil")
	}

	// Check display name
	if entity.DisplayName != "Initiative" {
		t.Errorf("DisplayName = %s, want 'Initiative'", entity.DisplayName)
	}

	// Check initiative-specific fields
	valueField := entity.Field("value")
	if valueField == nil {
		t.Fatal("Field('value') returned nil")
	}
	if valueField.Type != FieldTypeFloat {
		t.Errorf("value.Type = %s, want FieldTypeFloat", valueField.Type)
	}

	effortField := entity.Field("effort")
	if effortField == nil {
		t.Fatal("Field('effort') returned nil")
	}
	if effortField.Type != FieldTypeFloat {
		t.Errorf("effort.Type = %s, want FieldTypeFloat", effortField.Type)
	}

	progressField := entity.Field("progress")
	if progressField == nil {
		t.Fatal("Field('progress') returned nil")
	}
	if progressField.Type != FieldTypeFloat {
		t.Errorf("progress.Type = %s, want FieldTypeFloat", progressField.Type)
	}
}

func TestProductEntityFields(t *testing.T) {
	entity := GetEntity(ast.EntityProducts)
	if entity == nil {
		t.Fatal("GetEntity(products) returned nil")
	}

	// Check display name
	if entity.DisplayName != "Product" {
		t.Errorf("DisplayName = %s, want 'Product'", entity.DisplayName)
	}

	// Check required fields
	requiredFields := []string{
		"id", "reference_prefix", "name", "product_line", "has_ideas", "created_at",
	}
	for _, f := range requiredFields {
		if !entity.HasField(f) {
			t.Errorf("missing required field: %s", f)
		}
	}

	// Check product-specific fields
	productLineField := entity.Field("product_line")
	if productLineField == nil {
		t.Fatal("Field('product_line') returned nil")
	}
	if productLineField.Type != FieldTypeBool {
		t.Errorf("product_line.Type = %s, want FieldTypeBool", productLineField.Type)
	}

	hasIdeasField := entity.Field("has_ideas")
	if hasIdeasField == nil {
		t.Fatal("Field('has_ideas') returned nil")
	}
	if hasIdeasField.Type != FieldTypeBool {
		t.Errorf("has_ideas.Type = %s, want FieldTypeBool", hasIdeasField.Type)
	}
}

func TestUserEntityFields(t *testing.T) {
	entity := GetEntity(ast.EntityUsers)
	if entity == nil {
		t.Fatal("GetEntity(users) returned nil")
	}

	// Check display name
	if entity.DisplayName != "User" {
		t.Errorf("DisplayName = %s, want 'User'", entity.DisplayName)
	}

	// Check required fields
	requiredFields := []string{
		"id", "first_name", "last_name", "email", "role", "created_at",
	}
	for _, f := range requiredFields {
		if !entity.HasField(f) {
			t.Errorf("missing required field: %s", f)
		}
	}

	// Check user-specific fields
	emailField := entity.Field("email")
	if emailField == nil {
		t.Fatal("Field('email') returned nil")
	}
	if emailField.Type != FieldTypeString {
		t.Errorf("email.Type = %s, want FieldTypeString", emailField.Type)
	}
	if !emailField.Filterable {
		t.Error("email should be filterable")
	}

	roleField := entity.Field("role")
	if roleField == nil {
		t.Fatal("Field('role') returned nil")
	}
	if roleField.Type != FieldTypeString {
		t.Errorf("role.Type = %s, want FieldTypeString", roleField.Type)
	}
}
