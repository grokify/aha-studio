// Package schema defines entity and field metadata for AQL queries.
package schema

import "github.com/grokify/aha-studio/aql/ast"

// Entity represents an Aha entity type with its metadata.
type Entity struct {
	Name        ast.EntityType
	DisplayName string
	Fields      map[string]*Field
}

// Field returns the field definition for the given name, or nil if not found.
func (e *Entity) Field(name string) *Field {
	return e.Fields[name]
}

// HasField returns true if the entity has the given field.
func (e *Entity) HasField(name string) bool {
	_, ok := e.Fields[name]
	return ok
}

// entities holds the schema for all supported entities.
var entities = map[ast.EntityType]*Entity{
	ast.EntityComments:     commentEntity,
	ast.EntityEpics:        epicEntity,
	ast.EntityFeatures:     featureEntity,
	ast.EntityGoals:        goalEntity,
	ast.EntityIdeas:        ideaEntity,
	ast.EntityReleases:     releaseEntity,
	ast.EntityRequirements: requirementEntity,
	ast.EntityInitiatives:  initiativeEntity,
	ast.EntityProducts:     productEntity,
	ast.EntityTags:         tagEntity,
	ast.EntityUsers:        userEntity,
}

// GetEntity returns the entity schema for the given entity type.
func GetEntity(entityType ast.EntityType) *Entity {
	return entities[entityType]
}

// AllEntities returns all entity schemas.
func AllEntities() map[ast.EntityType]*Entity {
	return entities
}

// epicEntity defines the Epic entity schema.
var epicEntity = &Entity{
	Name:        ast.EntityEpics,
	DisplayName: "Epic",
	Fields: map[string]*Field{
		"id":              {Name: "id", Type: FieldTypeString, Filterable: true, Sortable: true},
		"reference_num":   {Name: "reference_num", Type: FieldTypeString, Filterable: true, Sortable: true},
		"name":            {Name: "name", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "q"},
		"description":     {Name: "description", Type: FieldTypeString, Filterable: false, Sortable: false},
		"progress":        {Name: "progress", Type: FieldTypeFloat, Filterable: true, Sortable: true},
		"progress_source": {Name: "progress_source", Type: FieldTypeString, Filterable: true, Sortable: false},
		"position":        {Name: "position", Type: FieldTypeInt, Filterable: true, Sortable: true},
		"color":           {Name: "color", Type: FieldTypeString, Filterable: true, Sortable: false},
		"status":          {Name: "status", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "workflow_status"},
		"start_date":      {Name: "start_date", Type: FieldTypeDate, Filterable: true, Sortable: true},
		"due_date":        {Name: "due_date", Type: FieldTypeDate, Filterable: true, Sortable: true},
		"created_at":      {Name: "created_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true},
		"updated_at":      {Name: "updated_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true, APIParam: "updated_since"},
		"release":         {Name: "release", Type: FieldTypeString, Filterable: true, Sortable: true},
		"initiative":      {Name: "initiative", Type: FieldTypeString, Filterable: true, Sortable: true},
		"url":             {Name: "url", Type: FieldTypeString, Filterable: false, Sortable: false},
	},
}

// featureEntity defines the Feature entity schema.
var featureEntity = &Entity{
	Name:        ast.EntityFeatures,
	DisplayName: "Feature",
	Fields: map[string]*Field{
		"id":              {Name: "id", Type: FieldTypeString, Filterable: true, Sortable: true},
		"reference_num":   {Name: "reference_num", Type: FieldTypeString, Filterable: true, Sortable: true},
		"name":            {Name: "name", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "q"},
		"description":     {Name: "description", Type: FieldTypeString, Filterable: false, Sortable: false},
		"product_id":      {Name: "product_id", Type: FieldTypeString, Filterable: true, Sortable: true},
		"position":        {Name: "position", Type: FieldTypeInt, Filterable: true, Sortable: true},
		"workspace":       {Name: "workspace", Type: FieldTypeString, Filterable: true, Sortable: true},
		"status":          {Name: "status", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "workflow_status"},
		"workflow_status": {Name: "workflow_status", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "workflow_status"},
		"assigned_to":     {Name: "assigned_to", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "assigned_to_user"},
		"tags":            {Name: "tags", Type: FieldTypeStringArray, Filterable: true, Sortable: false, APIParam: "tag"},
		"tag":             {Name: "tag", Type: FieldTypeString, Filterable: true, Sortable: false, APIParam: "tag"},
		"tag_list":        {Name: "tag_list", Type: FieldTypeString, Filterable: true, Sortable: false},
		"start_date":      {Name: "start_date", Type: FieldTypeDate, Filterable: true, Sortable: true},
		"due_date":        {Name: "due_date", Type: FieldTypeDate, Filterable: true, Sortable: true},
		"created_at":      {Name: "created_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true},
		"updated_at":      {Name: "updated_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true, APIParam: "updated_since"},
		"release":         {Name: "release", Type: FieldTypeString, Filterable: true, Sortable: true},
		"release_id":      {Name: "release_id", Type: FieldTypeString, Filterable: true, Sortable: true},
		"release_date":    {Name: "release_date", Type: FieldTypeDate, Filterable: true, Sortable: true},
		"initiative":      {Name: "initiative", Type: FieldTypeString, Filterable: true, Sortable: true},
		"url":             {Name: "url", Type: FieldTypeString, Filterable: false, Sortable: false},
	},
}

// ideaEntity defines the Idea entity schema.
var ideaEntity = &Entity{
	Name:        ast.EntityIdeas,
	DisplayName: "Idea",
	Fields: map[string]*Field{
		"id":              {Name: "id", Type: FieldTypeString, Filterable: true, Sortable: true},
		"reference_num":   {Name: "reference_num", Type: FieldTypeString, Filterable: true, Sortable: true},
		"name":            {Name: "name", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "q"},
		"description":     {Name: "description", Type: FieldTypeString, Filterable: false, Sortable: false},
		"status":          {Name: "status", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "workflow_status"},
		"workflow_status": {Name: "workflow_status", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "workflow_status"},
		"votes":           {Name: "votes", Type: FieldTypeInt, Filterable: true, Sortable: true},
		"tags":            {Name: "tags", Type: FieldTypeStringArray, Filterable: true, Sortable: false, APIParam: "tag"},
		"tag":             {Name: "tag", Type: FieldTypeString, Filterable: true, Sortable: false, APIParam: "tag"},
		"created_at":      {Name: "created_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true, APIParam: "created_since"},
		"created_before":  {Name: "created_before", Type: FieldTypeDatetime, Filterable: true, Sortable: false, APIParam: "created_before"},
		"updated_at":      {Name: "updated_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true, APIParam: "updated_since"},
		"user_id":         {Name: "user_id", Type: FieldTypeString, Filterable: true, Sortable: false, APIParam: "user_id"},
		"spam":            {Name: "spam", Type: FieldTypeBool, Filterable: true, Sortable: false, APIParam: "spam"},
		"url":             {Name: "url", Type: FieldTypeString, Filterable: false, Sortable: false},
	},
}

// goalEntity defines the Goal entity schema.
var goalEntity = &Entity{
	Name:        ast.EntityGoals,
	DisplayName: "Goal",
	Fields: map[string]*Field{
		"id":              {Name: "id", Type: FieldTypeString, Filterable: true, Sortable: true},
		"reference_num":   {Name: "reference_num", Type: FieldTypeString, Filterable: true, Sortable: true},
		"name":            {Name: "name", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "q"},
		"description":     {Name: "description", Type: FieldTypeString, Filterable: false, Sortable: false},
		"progress":        {Name: "progress", Type: FieldTypeFloat, Filterable: true, Sortable: true},
		"progress_source": {Name: "progress_source", Type: FieldTypeString, Filterable: true, Sortable: false},
		"status":          {Name: "status", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "workflow_status"},
		"start_date":      {Name: "start_date", Type: FieldTypeDate, Filterable: true, Sortable: true},
		"end_date":        {Name: "end_date", Type: FieldTypeDate, Filterable: true, Sortable: true},
		"created_at":      {Name: "created_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true},
		"updated_at":      {Name: "updated_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true, APIParam: "updated_since"},
		"url":             {Name: "url", Type: FieldTypeString, Filterable: false, Sortable: false},
	},
}

// releaseEntity defines the Release entity schema.
var releaseEntity = &Entity{
	Name:        ast.EntityReleases,
	DisplayName: "Release",
	Fields: map[string]*Field{
		"id":            {Name: "id", Type: FieldTypeString, Filterable: true, Sortable: true},
		"reference_num": {Name: "reference_num", Type: FieldTypeString, Filterable: true, Sortable: true},
		"name":          {Name: "name", Type: FieldTypeString, Filterable: true, Sortable: true},
		"start_date":    {Name: "start_date", Type: FieldTypeDate, Filterable: true, Sortable: true},
		"release_date":  {Name: "release_date", Type: FieldTypeDate, Filterable: true, Sortable: true},
		"released":      {Name: "released", Type: FieldTypeBool, Filterable: true, Sortable: true},
		"parking_lot":   {Name: "parking_lot", Type: FieldTypeBool, Filterable: true, Sortable: true},
		"url":           {Name: "url", Type: FieldTypeString, Filterable: false, Sortable: false},
	},
}

// requirementEntity defines the Requirement entity schema.
// Note: Requirements belong to features and must be queried with a feature_id filter.
var requirementEntity = &Entity{
	Name:        ast.EntityRequirements,
	DisplayName: "Requirement",
	Fields: map[string]*Field{
		"id":                 {Name: "id", Type: FieldTypeString, Filterable: true, Sortable: true},
		"reference_num":      {Name: "reference_num", Type: FieldTypeString, Filterable: true, Sortable: true},
		"name":               {Name: "name", Type: FieldTypeString, Filterable: true, Sortable: true},
		"description":        {Name: "description", Type: FieldTypeString, Filterable: false, Sortable: false},
		"feature_id":         {Name: "feature_id", Type: FieldTypeString, Filterable: true, Sortable: false},
		"status":             {Name: "status", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "workflow_status"},
		"assigned_to":        {Name: "assigned_to", Type: FieldTypeString, Filterable: true, Sortable: true},
		"position":           {Name: "position", Type: FieldTypeInt, Filterable: true, Sortable: true},
		"original_estimate":  {Name: "original_estimate", Type: FieldTypeFloat, Filterable: true, Sortable: true},
		"remaining_estimate": {Name: "remaining_estimate", Type: FieldTypeFloat, Filterable: true, Sortable: true},
		"work_done":          {Name: "work_done", Type: FieldTypeFloat, Filterable: true, Sortable: true},
		"created_at":         {Name: "created_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true},
		"updated_at":         {Name: "updated_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true},
		"url":                {Name: "url", Type: FieldTypeString, Filterable: false, Sortable: false},
	},
}

// initiativeEntity defines the Initiative entity schema.
var initiativeEntity = &Entity{
	Name:        ast.EntityInitiatives,
	DisplayName: "Initiative",
	Fields: map[string]*Field{
		"id":            {Name: "id", Type: FieldTypeString, Filterable: true, Sortable: true},
		"reference_num": {Name: "reference_num", Type: FieldTypeString, Filterable: true, Sortable: true},
		"name":          {Name: "name", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "q"},
		"description":   {Name: "description", Type: FieldTypeString, Filterable: false, Sortable: false},
		"status":        {Name: "status", Type: FieldTypeString, Filterable: true, Sortable: true, APIParam: "workflow_status"},
		"color":         {Name: "color", Type: FieldTypeString, Filterable: true, Sortable: false},
		"value":         {Name: "value", Type: FieldTypeFloat, Filterable: true, Sortable: true},
		"effort":        {Name: "effort", Type: FieldTypeFloat, Filterable: true, Sortable: true},
		"progress":      {Name: "progress", Type: FieldTypeFloat, Filterable: true, Sortable: true},
		"presented":     {Name: "presented", Type: FieldTypeBool, Filterable: true, Sortable: false},
		"start_date":    {Name: "start_date", Type: FieldTypeDate, Filterable: true, Sortable: true},
		"end_date":      {Name: "end_date", Type: FieldTypeDate, Filterable: true, Sortable: true},
		"created_at":    {Name: "created_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true},
		"updated_at":    {Name: "updated_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true, APIParam: "updated_since"},
		"url":           {Name: "url", Type: FieldTypeString, Filterable: false, Sortable: false},
	},
}

// productEntity defines the Product entity schema.
var productEntity = &Entity{
	Name:        ast.EntityProducts,
	DisplayName: "Product",
	Fields: map[string]*Field{
		"id":                  {Name: "id", Type: FieldTypeString, Filterable: true, Sortable: true},
		"reference_prefix":    {Name: "reference_prefix", Type: FieldTypeString, Filterable: true, Sortable: true},
		"name":                {Name: "name", Type: FieldTypeString, Filterable: true, Sortable: true},
		"product_line":        {Name: "product_line", Type: FieldTypeBool, Filterable: true, Sortable: true},
		"has_ideas":           {Name: "has_ideas", Type: FieldTypeBool, Filterable: true, Sortable: false},
		"has_master_features": {Name: "has_master_features", Type: FieldTypeBool, Filterable: true, Sortable: false},
		"created_at":          {Name: "created_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true},
		"url":                 {Name: "url", Type: FieldTypeString, Filterable: false, Sortable: false},
	},
}

// userEntity defines the User entity schema.
var userEntity = &Entity{
	Name:        ast.EntityUsers,
	DisplayName: "User",
	Fields: map[string]*Field{
		"id":         {Name: "id", Type: FieldTypeString, Filterable: true, Sortable: true},
		"first_name": {Name: "first_name", Type: FieldTypeString, Filterable: true, Sortable: true},
		"last_name":  {Name: "last_name", Type: FieldTypeString, Filterable: true, Sortable: true},
		"email":      {Name: "email", Type: FieldTypeString, Filterable: true, Sortable: true},
		"role":       {Name: "role", Type: FieldTypeString, Filterable: true, Sortable: true},
		"created_at": {Name: "created_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true},
	},
}

// commentEntity defines the Comment entity schema.
// Comments can be listed by product_id, feature_id, idea_id, release_id, initiative_id, epic_id, or goal_id.
var commentEntity = &Entity{
	Name:        ast.EntityComments,
	DisplayName: "Comment",
	Fields: map[string]*Field{
		"id":            {Name: "id", Type: FieldTypeString, Filterable: true, Sortable: true},
		"body":          {Name: "body", Type: FieldTypeString, Filterable: true, Sortable: false},
		"created_at":    {Name: "created_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true},
		"updated_at":    {Name: "updated_at", Type: FieldTypeDatetime, Filterable: true, Sortable: true},
		"user_id":       {Name: "user_id", Type: FieldTypeString, Filterable: true, Sortable: false},
		"product_id":    {Name: "product_id", Type: FieldTypeString, Filterable: true, Sortable: false},
		"feature_id":    {Name: "feature_id", Type: FieldTypeString, Filterable: true, Sortable: false},
		"idea_id":       {Name: "idea_id", Type: FieldTypeString, Filterable: true, Sortable: false},
		"release_id":    {Name: "release_id", Type: FieldTypeString, Filterable: true, Sortable: false},
		"initiative_id": {Name: "initiative_id", Type: FieldTypeString, Filterable: true, Sortable: false},
		"epic_id":       {Name: "epic_id", Type: FieldTypeString, Filterable: true, Sortable: false},
		"goal_id":       {Name: "goal_id", Type: FieldTypeString, Filterable: true, Sortable: false},
		"url":           {Name: "url", Type: FieldTypeString, Filterable: false, Sortable: false},
	},
}

// tagEntity defines the Tag entity schema.
// Tags are a derived entity aggregated from features and epics.
// Note: The Aha API does not have a dedicated tags endpoint, so tags are
// extracted from features and epics and deduplicated.
var tagEntity = &Entity{
	Name:        ast.EntityTags,
	DisplayName: "Tag",
	Fields: map[string]*Field{
		"name":          {Name: "name", Type: FieldTypeString, Filterable: true, Sortable: true},
		"feature_count": {Name: "feature_count", Type: FieldTypeInt, Filterable: true, Sortable: true},
		"epic_count":    {Name: "epic_count", Type: FieldTypeInt, Filterable: true, Sortable: true},
		"total_count":   {Name: "total_count", Type: FieldTypeInt, Filterable: true, Sortable: true},
	},
}
