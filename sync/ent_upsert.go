package sync

import (
	"context"
	"time"

	"github.com/grokify/aha-studio/ent/comment"
	"github.com/grokify/aha-studio/ent/epic"
	"github.com/grokify/aha-studio/ent/feature"
	"github.com/grokify/aha-studio/ent/goal"
	"github.com/grokify/aha-studio/ent/idea"
	"github.com/grokify/aha-studio/ent/ideaendorsement"
	"github.com/grokify/aha-studio/ent/ideaorganization"
	"github.com/grokify/aha-studio/ent/ideauser"
	"github.com/grokify/aha-studio/ent/initiative"
	"github.com/grokify/aha-studio/ent/product"
	"github.com/grokify/aha-studio/ent/relationship"
	"github.com/grokify/aha-studio/ent/release"
	"github.com/grokify/aha-studio/ent/requirement"
	"github.com/grokify/aha-studio/ent/syncmeta"
	"github.com/grokify/aha-studio/ent/user"
)

// One hand-written translation function per entity (rather than a
// reflection/tag-based generic mapper): with 13 stable, small-field
// entities, explicit functions stay directly diffable against the SQL
// they replace. Each mirrors its corresponding CREATE TABLE ... ON
// CONFLICT(id) DO UPDATE SET ... that used to live in db.go.

func (d *DB) upsertFeatureEnt(ctx context.Context, product_ string, data map[string]any) error {
	return d.ent.Feature.Create().
		SetID(mapID(data)).
		SetProduct(product_).
		SetNillableReferenceNum(mapStringPtr(data, "reference_num")).
		SetNillableName(mapStringPtr(data, "name")).
		SetNillableDescription(mapStringPtr(data, "description")).
		SetNillableStatus(mapStringPtr(data, "status")).
		SetNillableAssignedTo(mapStringPtr(data, "assigned_to")).
		SetNillableStartDate(mapStringPtr(data, "start_date")).
		SetNillableDueDate(mapStringPtr(data, "due_date")).
		SetNillableRelease(mapStringPtr(data, "release")).
		SetNillableReleaseID(mapStringPtr(data, "release_id")).
		SetNillableReleaseReferenceNum(mapStringPtr(data, "release_reference_num")).
		SetTags(tagsToString(data["tags"])).
		SetNillableURL(mapStringPtr(data, "url")).
		SetNillableCreatedAt(mapTimePtr(data, "created_at")).
		SetNillableUpdatedAt(mapTimePtr(data, "updated_at")).
		SetData(data).
		OnConflictColumns(feature.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertIdeaEnt(ctx context.Context, product_ string, data map[string]any) error {
	return d.ent.Idea.Create().
		SetID(mapID(data)).
		SetProduct(product_).
		SetNillableReferenceNum(mapStringPtr(data, "reference_num")).
		SetNillableName(mapStringPtr(data, "name")).
		SetNillableDescription(mapStringPtr(data, "description")).
		SetNillableStatus(mapStringPtr(data, "status")).
		SetVotes(mapInt(data, "votes")).
		SetTags(tagsToString(data["tags"])).
		SetNillableURL(mapStringPtr(data, "url")).
		SetNillableCreatedAt(mapTimePtr(data, "created_at")).
		SetNillableUpdatedAt(mapTimePtr(data, "updated_at")).
		SetData(data).
		OnConflictColumns(idea.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertReleaseEnt(ctx context.Context, product_ string, data map[string]any) error {
	return d.ent.Release.Create().
		SetID(mapID(data)).
		SetProduct(product_).
		SetNillableReferenceNum(mapStringPtr(data, "reference_num")).
		SetNillableName(mapStringPtr(data, "name")).
		SetNillableStartDate(mapStringPtr(data, "start_date")).
		SetNillableReleaseDate(mapStringPtr(data, "release_date")).
		SetReleased(boolFromAny(data["released"])).
		SetParkingLot(boolFromAny(data["parking_lot"])).
		SetNillableURL(mapStringPtr(data, "url")).
		SetNillableCreatedAt(mapTimePtr(data, "created_at")).
		SetData(data).
		OnConflictColumns(release.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertInitiativeEnt(ctx context.Context, product_ string, data map[string]any) error {
	return d.ent.Initiative.Create().
		SetID(mapID(data)).
		SetProduct(product_).
		SetNillableReferenceNum(mapStringPtr(data, "reference_num")).
		SetNillableName(mapStringPtr(data, "name")).
		SetNillableDescription(mapStringPtr(data, "description")).
		SetNillableStatus(mapStringPtr(data, "status")).
		SetNillableValue(mapFloat64Ptr(data, "value")).
		SetNillableEffort(mapFloat64Ptr(data, "effort")).
		SetNillableProgress(mapFloat64Ptr(data, "progress")).
		SetNillableStartDate(mapStringPtr(data, "start_date")).
		SetNillableEndDate(mapStringPtr(data, "end_date")).
		SetNillableURL(mapStringPtr(data, "url")).
		SetNillableCreatedAt(mapTimePtr(data, "created_at")).
		SetNillableUpdatedAt(mapTimePtr(data, "updated_at")).
		SetData(data).
		OnConflictColumns(initiative.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertGoalEnt(ctx context.Context, product_ string, data map[string]any) error {
	return d.ent.Goal.Create().
		SetID(mapID(data)).
		SetProduct(product_).
		SetNillableReferenceNum(mapStringPtr(data, "reference_num")).
		SetNillableName(mapStringPtr(data, "name")).
		SetNillableDescription(mapStringPtr(data, "description")).
		SetNillableStatus(mapStringPtr(data, "status")).
		SetNillableProgress(mapFloat64Ptr(data, "progress")).
		SetNillableStartDate(mapStringPtr(data, "start_date")).
		SetNillableEndDate(mapStringPtr(data, "end_date")).
		SetNillableURL(mapStringPtr(data, "url")).
		SetNillableCreatedAt(mapTimePtr(data, "created_at")).
		SetNillableUpdatedAt(mapTimePtr(data, "updated_at")).
		SetData(data).
		OnConflictColumns(goal.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertEpicEnt(ctx context.Context, product_ string, data map[string]any) error {
	return d.ent.Epic.Create().
		SetID(mapID(data)).
		SetProduct(product_).
		SetNillableReferenceNum(mapStringPtr(data, "reference_num")).
		SetNillableName(mapStringPtr(data, "name")).
		SetNillableDescription(mapStringPtr(data, "description")).
		SetNillableStatus(mapStringPtr(data, "status")).
		SetNillableProgress(mapFloat64Ptr(data, "progress")).
		SetNillableStartDate(mapStringPtr(data, "start_date")).
		SetNillableDueDate(mapStringPtr(data, "due_date")).
		SetNillableRelease(mapStringPtr(data, "release")).
		SetNillableURL(mapStringPtr(data, "url")).
		SetNillableCreatedAt(mapTimePtr(data, "created_at")).
		SetNillableUpdatedAt(mapTimePtr(data, "updated_at")).
		SetData(data).
		OnConflictColumns(epic.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertUserEnt(ctx context.Context, data map[string]any) error {
	return d.ent.User.Create().
		SetID(mapID(data)).
		SetNillableFirstName(mapStringPtr(data, "first_name")).
		SetNillableLastName(mapStringPtr(data, "last_name")).
		SetNillableEmail(mapStringPtr(data, "email")).
		SetNillableRole(mapStringPtr(data, "role")).
		SetNillableCreatedAt(mapTimePtr(data, "created_at")).
		SetData(data).
		OnConflictColumns(user.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertProductEnt(ctx context.Context, data map[string]any) error {
	return d.ent.Product.Create().
		SetID(mapID(data)).
		SetNillableReferencePrefix(mapStringPtr(data, "reference_prefix")).
		SetNillableName(mapStringPtr(data, "name")).
		SetProductLine(boolFromAny(data["product_line"])).
		SetHasIdeas(boolFromAny(data["has_ideas"])).
		SetNillableCreatedAt(mapTimePtr(data, "created_at")).
		SetData(data).
		OnConflictColumns(product.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertIdeaEndorsementEnt(ctx context.Context, product_ string, data map[string]any) error {
	return d.ent.IdeaEndorsement.Create().
		SetID(mapID(data)).
		SetNillableProduct(mapStringPtr2(product_)).
		SetNillableIdeaID(mapStringPtr(data, "idea_id")).
		SetNillableWeight(mapIntPtr(data, "weight")).
		SetNillableValue(mapStringPtr(data, "value")).
		SetNillableLink(mapStringPtr(data, "link")).
		SetNillablePortalUserID(mapStringPtr(data, "portal_user_id")).
		SetNillablePortalUserName(mapStringPtr(data, "portal_user_name")).
		SetNillablePortalUserEmail(mapStringPtr(data, "portal_user_email")).
		SetNillablePortalUserCreatedAt(mapTimePtr(data, "portal_user_created_at")).
		SetNillableIdeaUserID(mapStringPtr(data, "idea_user_id")).
		SetNillableIdeaUserName(mapStringPtr(data, "idea_user_name")).
		SetNillableIdeaUserEmail(mapStringPtr(data, "idea_user_email")).
		SetNillableIdeaUserTitle(mapStringPtr(data, "idea_user_title")).
		SetNillableIdeaUserCreatedAt(mapTimePtr(data, "idea_user_created_at")).
		SetNillableCreatedAt(mapTimePtr(data, "created_at")).
		SetNillableUpdatedAt(mapTimePtr(data, "updated_at")).
		OnConflictColumns(ideaendorsement.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertIdeaUserEnt(ctx context.Context, data map[string]any) error {
	return d.ent.IdeaUser.Create().
		SetID(mapID(data)).
		SetNillableName(mapStringPtr(data, "name")).
		SetNillableEmail(mapStringPtr(data, "email")).
		SetNillableCreatedAt(mapTimePtr(data, "created_at")).
		SetIdeaOrganizations(mapMapSlice(data, "idea_organizations")).
		OnConflictColumns(ideauser.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertIdeaOrganizationEnt(ctx context.Context, data map[string]any) error {
	return d.ent.IdeaOrganization.Create().
		SetID(mapID(data)).
		SetNillableName(mapStringPtr(data, "name")).
		SetNillableReferenceNum(mapStringPtr(data, "reference_num")).
		SetNillableURL(mapStringPtr(data, "url")).
		SetNillableEmailDomains(mapStringPtr(data, "email_domains")).
		SetNillableRevenue(mapFloat64Ptr(data, "revenue")).
		SetNillableEndorsementsCount(mapIntPtr(data, "endorsements_count")).
		SetNillableCreatedAt(mapTimePtr(data, "created_at")).
		SetNillableUpdatedAt(mapTimePtr(data, "updated_at")).
		OnConflictColumns(ideaorganization.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertCommentEnt(ctx context.Context, product_ string, data map[string]any) error {
	return d.ent.Comment.Create().
		SetID(mapID(data)).
		SetNillableProduct(mapStringPtr2(product_)).
		SetNillableCommentableType(mapStringPtr(data, "commentable_type")).
		SetNillableCommentableID(mapStringPtr(data, "commentable_id")).
		SetNillableBody(mapStringPtr(data, "body")).
		SetNillableUserID(mapStringPtr(data, "user_id")).
		SetNillableURL(mapStringPtr(data, "url")).
		SetNillableCreatedAt(mapTimePtr(data, "created_at")).
		SetNillableUpdatedAt(mapTimePtr(data, "updated_at")).
		SetData(data).
		OnConflictColumns(comment.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertRequirementEnt(ctx context.Context, product_ string, data map[string]any) error {
	return d.ent.Requirement.Create().
		SetID(mapID(data)).
		SetProduct(product_).
		SetNillableFeatureID(mapStringPtr(data, "feature_id")).
		SetNillableReferenceNum(mapStringPtr(data, "reference_num")).
		SetNillableName(mapStringPtr(data, "name")).
		SetNillableDescription(mapStringPtr(data, "description")).
		SetNillableStatus(mapStringPtr(data, "status")).
		SetNillableAssignedTo(mapStringPtr(data, "assigned_to")).
		SetNillablePosition(mapIntPtr(data, "position")).
		SetNillableOriginalEstimate(mapFloat64Ptr(data, "original_estimate")).
		SetNillableRemainingEstimate(mapFloat64Ptr(data, "remaining_estimate")).
		SetNillableWorkDone(mapFloat64Ptr(data, "work_done")).
		SetNillableURL(mapStringPtr(data, "url")).
		SetNillableCreatedAt(mapTimePtr(data, "created_at")).
		SetNillableUpdatedAt(mapTimePtr(data, "updated_at")).
		SetData(data).
		OnConflictColumns(requirement.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertRelationshipEnt(ctx context.Context, fromType, fromID, relType, toType, toID, product_ string) error {
	return d.ent.Relationship.Create().
		SetFromType(fromType).
		SetFromID(fromID).
		SetRelType(relType).
		SetToType(toType).
		SetToID(toID).
		SetNillableProduct(mapStringPtr2(product_)).
		OnConflictColumns(
			relationship.FieldFromType, relationship.FieldFromID, relationship.FieldRelType,
			relationship.FieldToType, relationship.FieldToID,
		).
		UpdateNewValues().
		Exec(ctx)
}

func (d *DB) upsertSyncMetaEnt(ctx context.Context, entity, product_ string, t time.Time, count int) error {
	return d.ent.SyncMeta.Create().
		SetEntity(entity).
		SetProduct(product_).
		SetLastSync(t).
		SetRecordCount(count).
		OnConflictColumns(syncmeta.FieldEntity, syncmeta.FieldProduct).
		UpdateNewValues().
		Exec(ctx)
}

// boolFromAny mirrors the old boolToInt helper's semantics (default false
// for nil/wrong-type) for the non-nillable, Default(false) boolean
// columns (released, parking_lot, product_line, has_ideas).
func boolFromAny(v any) bool {
	b, _ := v.(bool)
	return b
}

// mapStringPtr2 is for required-elsewhere-but-optional-on-this-table
// columns like comments.product and relationships.product, which are
// passed as a plain (already-validated) string parameter rather than
// pulled from a data map - nil only if the caller passed "".
func mapStringPtr2(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
