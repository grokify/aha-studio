# Write Tools

The MCP server provides the following tools for creating and modifying Aha.io data.

## Available Tools

### Create Tools

| Feature | Tool Name | Description |
|---------|-----------|-------------|
| Create a feature | `create_feature` | Create a new feature in a release |
| Create an epic | `create_epic` | Create a new epic in a release |
| Create a goal | `create_goal` | Create a new goal in a product |
| Create an initiative | `create_initiative` | Create a new initiative in a product |
| Create a requirement | `create_requirement` | Create a new requirement for a feature |

### Update Tools

| Feature | Tool Name | Description |
|---------|-----------|-------------|
| Update feature | `update_feature` | Update feature fields (name, description, status, dates) |
| Update epic | `update_epic` | Update epic fields (name, description, status, progress) |
| Update goal | `update_goal` | Update goal fields (name, description, status, progress) |
| Update initiative | `update_initiative` | Update initiative fields (name, description, dates, scores) |
| Update requirement | `update_requirement` | Update requirement fields (name, description, status) |
| Update release | `update_release` | Update release fields (name, dates, parking_lot, description) |
| Update idea | `update_idea` | Update idea fields (name, description, status, visibility) |

### Status & Assignment Tools

| Feature | Tool Name | Description |
|---------|-----------|-------------|
| Assign user to feature | `assign_user_to_feature` | Assign a user to a feature |
| Change status | `change_feature_status` | Change the workflow status of a feature |
| Assign release | `assign_feature_release` | Assign a feature to a different release |
| Lookup statuses | `list_workflow_statuses` | List all workflow statuses for a product (lookup IDs by name) |
| Lookup releases | `list_releases` | List all releases for a product (lookup IDs by name) |

### Comment Tools

| Feature | Tool Name | Description |
|---------|-----------|-------------|
| Add feature comments | `add_feature_comment` | Add a comment to a feature |
| Add idea comments | `add_idea_comment` | Add a comment to an idea |

## Usage Examples

### Create a Feature

```json
{
  "tool": "create_feature",
  "parameters": {
    "release_id": "PROD-R-1",
    "name": "User Authentication",
    "description": "Implement OAuth2 login flow",
    "workflow_status": "Ready for Development",
    "assigned_to_user": "user@example.com"
  }
}
```

### Change Feature Status

First, lookup available statuses:

```json
{
  "tool": "list_workflow_statuses",
  "parameters": {
    "product_id": "PROD"
  }
}
```

Then change the status:

```json
{
  "tool": "change_feature_status",
  "parameters": {
    "feature_id": "PROD-123",
    "status": "In Progress"
  }
}
```

### Assign Feature to Release

First, lookup available releases:

```json
{
  "tool": "list_releases",
  "parameters": {
    "product_id": "PROD"
  }
}
```

Then assign the feature:

```json
{
  "tool": "assign_feature_release",
  "parameters": {
    "feature_id": "PROD-123",
    "release_id": "PROD-R-2"
  }
}
```

### Assign User to Feature

```json
{
  "tool": "assign_user_to_feature",
  "parameters": {
    "feature_id": "PROD-123",
    "user": "user@example.com"
  }
}
```

### Add Comment to Feature

```json
{
  "tool": "add_feature_comment",
  "parameters": {
    "feature_id": "PROD-123",
    "body": "Updated the requirements based on customer feedback."
  }
}
```

## Workflow Example

A typical agent workflow for creating and configuring a feature:

1. **List releases** to find the target release ID
2. **Create feature** in that release
3. **List workflow statuses** to find the desired status ID
4. **Change feature status** to the desired state
5. **Assign user** to the feature
6. **Add comment** documenting the creation
