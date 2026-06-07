package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/plexusone/w3pilot"
)

// TemplateType represents the type of strategic model template.
type TemplateType string

const (
	TemplateCapabilityStack   TemplateType = "capability-stack"
	TemplateMaturityModel     TemplateType = "maturity-model"
	TemplateOpportunityPatton TemplateType = "opportunity-patton"
	TemplateFeatureCanvas     TemplateType = "feature-canvas"
	TemplateBusinessModel     TemplateType = "business-model"
	TemplateLeanCanvas        TemplateType = "lean-canvas"
	TemplateValueProposition  TemplateType = "value-proposition"
	TemplateOST               TemplateType = "opportunity-solution-tree"
)

// TemplateConfig defines a strategic model template configuration.
type TemplateConfig struct {
	Name        string       `json:"name"`
	Type        TemplateType `json:"type"`
	Description string       `json:"description"`
	Rows        []RowConfig  `json:"rows"`
}

// RowConfig defines a row in a strategic model.
type RowConfig struct {
	Title   string       `json:"title"`
	Columns []CellConfig `json:"columns"`
}

// CellConfig defines a cell in a strategic model row.
type CellConfig struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	ColSpan     int    `json:"colSpan,omitempty"`
	Color       string `json:"color,omitempty"`
}

// PredefinedTemplates contains ready-to-use template configurations.
var PredefinedTemplates = map[TemplateType]TemplateConfig{
	TemplateCapabilityStack: {
		Name:        "Capability Stack",
		Type:        TemplateCapabilityStack,
		Description: "Map capabilities across layers of your technology or business stack",
		Rows: []RowConfig{
			{Title: "Customer Experience", Columns: []CellConfig{
				{Title: "Web Portal"}, {Title: "Mobile App"}, {Title: "API Gateway"},
			}},
			{Title: "Business Logic", Columns: []CellConfig{
				{Title: "Order Management"}, {Title: "Inventory"}, {Title: "Pricing"},
			}},
			{Title: "Data Layer", Columns: []CellConfig{
				{Title: "Customer Data"}, {Title: "Product Catalog"}, {Title: "Analytics"},
			}},
			{Title: "Infrastructure", Columns: []CellConfig{
				{Title: "Cloud Services"}, {Title: "Security"}, {Title: "Monitoring"},
			}},
		},
	},
	TemplateMaturityModel: {
		Name:        "Maturity Model",
		Type:        TemplateMaturityModel,
		Description: "Assess maturity levels across capability dimensions",
		Rows: []RowConfig{
			{Title: "Dimensions", Columns: []CellConfig{
				{Title: "Level 1: Initial"}, {Title: "Level 2: Developing"}, {Title: "Level 3: Defined"},
				{Title: "Level 4: Managed"}, {Title: "Level 5: Optimizing"},
			}},
			{Title: "Process", Columns: []CellConfig{
				{Title: "Ad-hoc"}, {Title: "Repeatable"}, {Title: "Documented"},
				{Title: "Measured"}, {Title: "Continuous"},
			}},
			{Title: "Technology", Columns: []CellConfig{
				{Title: "Manual"}, {Title: "Assisted"}, {Title: "Automated"},
				{Title: "Integrated"}, {Title: "Intelligent"},
			}},
			{Title: "People", Columns: []CellConfig{
				{Title: "Aware"}, {Title: "Learning"}, {Title: "Competent"},
				{Title: "Proficient"}, {Title: "Expert"},
			}},
		},
	},
	TemplateOpportunityPatton: {
		Name:        "Opportunity Canvas (Patton)",
		Type:        TemplateOpportunityPatton,
		Description: "Jeff Patton's Opportunity Canvas for validating product opportunities",
		Rows: []RowConfig{
			{Title: "Problem Space", Columns: []CellConfig{
				{Title: "Problems", Description: "What problems do users have?"},
				{Title: "Users & Customers", Description: "Who has these problems?"},
				{Title: "Solutions Today", Description: "How do they solve it now?"},
			}},
			{Title: "Solution Space", Columns: []CellConfig{
				{Title: "Value Proposition", Description: "What's your unique value?"},
				{Title: "User Value", Description: "Why will users adopt this?"},
				{Title: "Business Value", Description: "Why does the business care?"},
			}},
			{Title: "Validation", Columns: []CellConfig{
				{Title: "Assumptions", Description: "What must be true?"},
				{Title: "Risks", Description: "What could go wrong?"},
				{Title: "Budget & Timeline", Description: "Resources needed?"},
			}},
		},
	},
	TemplateFeatureCanvas: {
		Name:        "Feature Canvas",
		Type:        TemplateFeatureCanvas,
		Description: "Nikita Efimov's Feature Canvas for feature definition (CC BY-SA 4.0)",
		Rows: []RowConfig{
			{Title: "Idea Statement", Columns: []CellConfig{
				{Title: "Feature Idea", Description: "What is the feature?", ColSpan: 4},
			}},
			{Title: "Problem Area", Columns: []CellConfig{
				{Title: "Situations", Description: "When/where does the problem occur?"},
				{Title: "Problems", Description: "What are the user pain points?"},
				{Title: "Value", Description: "What value does solving this create?"},
				{Title: "Capabilities", Description: "What must the solution do?"},
			}},
			{Title: "Constraints", Columns: []CellConfig{
				{Title: "Restrictions", Description: "What can't we do?", ColSpan: 2},
				{Title: "Limitations", Description: "What resources are limited?", ColSpan: 2},
			}},
		},
	},
	TemplateBusinessModel: {
		Name:        "Business Model Canvas",
		Type:        TemplateBusinessModel,
		Description: "Osterwalder's Business Model Canvas for business design",
		Rows: []RowConfig{
			{Title: "Partners & Activities", Columns: []CellConfig{
				{Title: "Key Partners"},
				{Title: "Key Activities"},
				{Title: "Value Propositions", ColSpan: 2},
				{Title: "Customer Relationships"},
				{Title: "Customer Segments"},
			}},
			{Title: "Resources & Channels", Columns: []CellConfig{
				{Title: "", ColSpan: 1},
				{Title: "Key Resources"},
				{Title: "", ColSpan: 2},
				{Title: "Channels"},
				{Title: "", ColSpan: 1},
			}},
			{Title: "Finances", Columns: []CellConfig{
				{Title: "Cost Structure", ColSpan: 3},
				{Title: "Revenue Streams", ColSpan: 3},
			}},
		},
	},
	TemplateLeanCanvas: {
		Name:        "Lean Canvas",
		Type:        TemplateLeanCanvas,
		Description: "Ash Maurya's Lean Canvas for startup/product validation",
		Rows: []RowConfig{
			{Title: "Problem & Solution", Columns: []CellConfig{
				{Title: "Problem", Description: "Top 3 problems"},
				{Title: "Solution", Description: "Top 3 features"},
				{Title: "Unique Value Proposition", ColSpan: 2},
				{Title: "Unfair Advantage"},
				{Title: "Customer Segments"},
			}},
			{Title: "Channels & Metrics", Columns: []CellConfig{
				{Title: "Existing Alternatives"},
				{Title: "Key Metrics"},
				{Title: "High-Level Concept", ColSpan: 2},
				{Title: "Channels"},
				{Title: "Early Adopters"},
			}},
			{Title: "Finances", Columns: []CellConfig{
				{Title: "Cost Structure", ColSpan: 3},
				{Title: "Revenue Streams", ColSpan: 3},
			}},
		},
	},
	TemplateValueProposition: {
		Name:        "Value Proposition Canvas",
		Type:        TemplateValueProposition,
		Description: "Osterwalder's Value Proposition Canvas for product-market fit",
		Rows: []RowConfig{
			{Title: "Customer Profile", Columns: []CellConfig{
				{Title: "Customer Jobs", Description: "What are they trying to get done?"},
				{Title: "Pains", Description: "What frustrates them?"},
				{Title: "Gains", Description: "What outcomes do they want?"},
			}},
			{Title: "Value Map", Columns: []CellConfig{
				{Title: "Products & Services", Description: "What do you offer?"},
				{Title: "Pain Relievers", Description: "How do you address pains?"},
				{Title: "Gain Creators", Description: "How do you create gains?"},
			}},
		},
	},
	TemplateOST: {
		Name:        "Opportunity Solution Tree",
		Type:        TemplateOST,
		Description: "Teresa Torres's OST for continuous discovery",
		Rows: []RowConfig{
			{Title: "Outcome", Columns: []CellConfig{
				{Title: "Desired Outcome", Description: "What metric are we trying to move?", ColSpan: 4},
			}},
			{Title: "Opportunities", Columns: []CellConfig{
				{Title: "Opportunity 1"}, {Title: "Opportunity 2"}, {Title: "Opportunity 3"}, {Title: "Opportunity 4"},
			}},
			{Title: "Solutions", Columns: []CellConfig{
				{Title: "Solution A"}, {Title: "Solution B"}, {Title: "Solution C"}, {Title: "Solution D"},
			}},
			{Title: "Experiments", Columns: []CellConfig{
				{Title: "Experiment 1"}, {Title: "Experiment 2"}, {Title: "Experiment 3"}, {Title: "Experiment 4"},
			}},
		},
	},
}

// ListPredefinedTemplates returns all available predefined templates.
func ListPredefinedTemplates() []TemplateConfig {
	templates := make([]TemplateConfig, 0, len(PredefinedTemplates))
	for _, t := range PredefinedTemplates {
		templates = append(templates, t)
	}
	return templates
}

// GetPredefinedTemplate returns a predefined template by type.
func GetPredefinedTemplate(templateType TemplateType) (TemplateConfig, bool) {
	t, ok := PredefinedTemplates[templateType]
	return t, ok
}

// CreateTemplateResult contains the result of creating a template.
type CreateTemplateResult struct {
	Success    bool   `json:"success"`
	TemplateID string `json:"template_id,omitempty"`
	URL        string `json:"url,omitempty"`
	Error      string `json:"error,omitempty"`
	Screenshot []byte `json:"screenshot,omitempty"`
}

// CreateStrategicModelTemplate creates a strategic model template in Aha! via browser automation.
func (c *Client) CreateStrategicModelTemplate(ctx context.Context, productKey string, config TemplateConfig) (*CreateTemplateResult, error) {
	if !c.loggedIn {
		if err := c.Login(ctx); err != nil {
			return &CreateTemplateResult{Success: false, Error: err.Error()}, nil
		}
	}

	// Navigate to strategic models for the product
	strategicModelsURL := fmt.Sprintf("/products/%s/strategic_models", productKey)
	if err := c.NavigateTo(ctx, strategicModelsURL); err != nil {
		return &CreateTemplateResult{Success: false, Error: fmt.Sprintf("failed to navigate to strategic models: %v", err)}, nil
	}

	// Wait for page to load
	time.Sleep(2 * time.Second)

	// Click the "Add" or "New" button to create a new strategic model
	addBtn, err := c.pilot.Find(ctx, "[data-testid='add-strategic-model'], .add-strategic-model, button:has-text('Add'), a:has-text('New')", &w3pilot.FindOptions{
		Timeout: c.timeout,
	})
	if err != nil {
		return &CreateTemplateResult{Success: false, Error: fmt.Sprintf("failed to find add button: %v", err)}, nil
	}
	if err := addBtn.Click(ctx, nil); err != nil {
		return &CreateTemplateResult{Success: false, Error: fmt.Sprintf("failed to click add button: %v", err)}, nil
	}

	// Wait for the template creation modal/page
	time.Sleep(1 * time.Second)

	// Enter template name
	nameField, err := c.pilot.Find(ctx, "input[name='name'], input[placeholder*='name'], #strategic_model_name", &w3pilot.FindOptions{
		Timeout: c.timeout,
	})
	if err != nil {
		return &CreateTemplateResult{Success: false, Error: fmt.Sprintf("failed to find name field: %v", err)}, nil
	}
	if err := nameField.Fill(ctx, config.Name, nil); err != nil {
		return &CreateTemplateResult{Success: false, Error: fmt.Sprintf("failed to fill name: %v", err)}, nil
	}

	// Enter description if there's a field
	descField, err := c.pilot.Find(ctx, "textarea[name='description'], #strategic_model_description", &w3pilot.FindOptions{
		Timeout: 5 * time.Second,
	})
	if err == nil && config.Description != "" {
		_ = descField.Fill(ctx, config.Description, nil)
	}

	// Click create/save button
	saveBtn, err := c.pilot.Find(ctx, "button[type='submit'], input[type='submit'], button:has-text('Create'), button:has-text('Save')", &w3pilot.FindOptions{
		Timeout: c.timeout,
	})
	if err != nil {
		return &CreateTemplateResult{Success: false, Error: fmt.Sprintf("failed to find save button: %v", err)}, nil
	}
	if err := saveBtn.Click(ctx, nil); err != nil {
		return &CreateTemplateResult{Success: false, Error: fmt.Sprintf("failed to click save: %v", err)}, nil
	}

	// Wait for navigation
	if err := c.pilot.WaitForNavigation(ctx, c.timeout); err != nil {
		// Continue anyway, page might not navigate
	}

	// Add rows and cells to the template
	for _, row := range config.Rows {
		if err := c.addRow(ctx, row); err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to add row %s: %v\n", row.Title, err)
		}
	}

	// Get the current URL as the template URL
	url, _ := c.URL(ctx)

	// Take a screenshot
	screenshot, _ := c.Screenshot(ctx)

	return &CreateTemplateResult{
		Success:    true,
		URL:        url,
		Screenshot: screenshot,
	}, nil
}

// addRow adds a row to the strategic model.
func (c *Client) addRow(ctx context.Context, row RowConfig) error {
	// Click add row button
	addRowBtn, err := c.pilot.Find(ctx, "[data-testid='add-row'], .add-row, button:has-text('Add Row'), button:has-text('Add row')", &w3pilot.FindOptions{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to find add row button: %w", err)
	}
	if err := addRowBtn.Click(ctx, nil); err != nil {
		return fmt.Errorf("failed to click add row: %w", err)
	}

	// Wait for row to appear
	time.Sleep(500 * time.Millisecond)

	// Find and fill row title
	rowTitleField, err := c.pilot.Find(ctx, ".row-title input, [data-testid='row-title'], input[placeholder*='row']", &w3pilot.FindOptions{
		Timeout: 5 * time.Second,
	})
	if err == nil {
		_ = rowTitleField.Fill(ctx, row.Title, nil)
	}

	// Add columns/cells
	for _, cell := range row.Columns {
		if err := c.addCell(ctx, cell); err != nil {
			// Log but continue
			fmt.Printf("Warning: failed to add cell %s: %v\n", cell.Title, err)
		}
	}

	return nil
}

// addCell adds a cell to the current row.
func (c *Client) addCell(ctx context.Context, cell CellConfig) error {
	// Click add cell/column button
	addCellBtn, err := c.pilot.Find(ctx, "[data-testid='add-cell'], .add-cell, button:has-text('Add Cell'), button:has-text('Add column')", &w3pilot.FindOptions{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to find add cell button: %w", err)
	}
	if err := addCellBtn.Click(ctx, nil); err != nil {
		return fmt.Errorf("failed to click add cell: %w", err)
	}

	// Wait for cell to appear
	time.Sleep(300 * time.Millisecond)

	// Find and fill cell title
	cellTitleField, err := c.pilot.Find(ctx, ".cell-title input, [data-testid='cell-title'], input[placeholder*='cell'], input[placeholder*='title']", &w3pilot.FindOptions{
		Timeout: 5 * time.Second,
	})
	if err == nil {
		_ = cellTitleField.Fill(ctx, cell.Title, nil)
	}

	// Fill description if available
	if cell.Description != "" {
		descField, err := c.pilot.Find(ctx, ".cell-description textarea, [data-testid='cell-description']", &w3pilot.FindOptions{
			Timeout: 3 * time.Second,
		})
		if err == nil {
			_ = descField.Fill(ctx, cell.Description, nil)
		}
	}

	return nil
}
