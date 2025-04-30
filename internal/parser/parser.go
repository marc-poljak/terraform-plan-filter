package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/marc-poljak/terraform-plan-filter/internal/model"
)

// TerraformPlanJSON represents the structure of a Terraform plan in JSON format
type TerraformPlanJSON struct {
	FormatVersion    string `json:"format_version"`
	TerraformVersion string `json:"terraform_version"`
	Variables        map[string]struct {
		Value json.RawMessage `json:"value"`
	} `json:"variables"`
	PlannedValues struct {
		RootModule struct {
			Resources []struct {
				Address      string                 `json:"address"`
				Mode         string                 `json:"mode"`
				Type         string                 `json:"type"`
				Name         string                 `json:"name"`
				ProviderName string                 `json:"provider_name"`
				Values       map[string]interface{} `json:"values"`
			} `json:"resources"`
			ChildModules []struct {
				Address   string `json:"address"`
				Resources []struct {
					Address      string                 `json:"address"`
					Mode         string                 `json:"mode"`
					Type         string                 `json:"type"`
					Name         string                 `json:"name"`
					ProviderName string                 `json:"provider_name"`
					Values       map[string]interface{} `json:"values"`
				} `json:"resources"`
			} `json:"child_modules"`
		} `json:"root_module"`
	} `json:"planned_values"`
	ResourceChanges []struct {
		Address      string `json:"address"`
		Mode         string `json:"mode"`
		Type         string `json:"type"`
		Name         string `json:"name"`
		ProviderName string `json:"provider_name"`
		Change       struct {
			Actions []string    `json:"actions"`
			Before  interface{} `json:"before"`
			After   interface{} `json:"after"`
		} `json:"change"`
	} `json:"resource_changes"`
	OutputChanges map[string]struct {
		Change struct {
			Actions []string    `json:"actions"`
			Before  interface{} `json:"before"`
			After   interface{} `json:"after"`
		} `json:"change"`
	} `json:"output_changes"`
	PriorState struct {
		FormatVersion string `json:"format_version"`
	} `json:"prior_state"`
	// Modify this part to handle more flexible JSON structures
	Config struct {
		ProviderConfig map[string]struct {
			Name        string `json:"name"`
			Expressions map[string]struct {
				// Use json.RawMessage instead of string to handle any JSON type
				ConstantValue json.RawMessage `json:"constant_value"`
			} `json:"expressions"`
		} `json:"provider_config"`
	} `json:"configuration"`
}

// ParseTerraformPlan parses a Terraform plan in JSON format
func ParseTerraformPlan(reader io.Reader) (*model.ResourceCollection, error) {
	resources := model.NewResourceCollection()

	// Read the entire input
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Check if this is a text plan instead of JSON
	if isTextPlan(data) {
		return nil, fmt.Errorf("input appears to be text format, not JSON. Please use 'terraform show -json tfplan' to generate JSON output")
	}

	// Attempt to parse the full JSON structure
	var plan TerraformPlanJSON
	if err := json.Unmarshal(data, &plan); err != nil {
		// Try a more flexible approach if full parsing fails
		return parseResourceChangesOnly(data)
	}

	// Process the resource changes
	processResourceChanges(resources, plan.ResourceChanges)

	// Set the summary flags and counters
	calculateSummaryValues(resources, plan.ResourceChanges)

	resources.HasDetailedResources = true
	return resources, nil
}

// isTextPlan checks if the input data is in text format rather than JSON
func isTextPlan(data []byte) bool {
	return strings.HasPrefix(string(data), "Terraform will perform")
}

// processResourceChanges processes the resource changes from the Terraform plan
func processResourceChanges(resources *model.ResourceCollection, resourceChanges []struct {
	Address      string `json:"address"`
	Mode         string `json:"mode"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	ProviderName string `json:"provider_name"`
	Change       struct {
		Actions []string    `json:"actions"`
		Before  interface{} `json:"before"`
		After   interface{} `json:"after"`
	} `json:"change"`
}) {
	for _, resource := range resourceChanges {
		// Skip data resources
		if resource.Mode == "data" {
			continue
		}

		// Check for special case: "no-op" actions (read-only)
		if isNoOpAction(resource.Change.Actions) {
			continue
		}

		// Process replacement resources specially
		if isReplacement(resource.Change.Actions) {
			resources.AddResource(model.ActionCreate, resource.Address)
			resources.AddResource(model.ActionDestroy, resource.Address)
			continue
		}

		// Process standard resources based on their actions
		processStandardActions(resources, resource.Address, resource.Change.Actions)
	}
}

// isNoOpAction checks if the actions list contains only "no-op"
func isNoOpAction(actions []string) bool {
	return len(actions) == 1 && actions[0] == "no-op"
}

// isReplacement checks if the actions list contains "replace"
func isReplacement(actions []string) bool {
	for _, action := range actions {
		if action == "replace" {
			return true
		}
	}
	return false
}

// processStandardActions processes standard create/update/delete actions
func processStandardActions(resources *model.ResourceCollection, address string, actions []string) {
	for _, action := range actions {
		switch action {
		case "create":
			resources.AddResource(model.ActionCreate, address)
		case "update":
			resources.AddResource(model.ActionUpdate, address)
		case "delete":
			resources.AddResource(model.ActionDestroy, address)
		}
	}
}

// calculateSummaryValues sets the summary values in the resource collection
func calculateSummaryValues(resources *model.ResourceCollection, resourceChanges []struct {
	Address      string `json:"address"`
	Mode         string `json:"mode"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	ProviderName string `json:"provider_name"`
	Change       struct {
		Actions []string    `json:"actions"`
		Before  interface{} `json:"before"`
		After   interface{} `json:"after"`
	} `json:"change"`
}) {
	resources.FoundSummary = true

	// Reset summary counters
	resources.SummaryAdds = 0
	resources.SummaryChanges = 0
	resources.SummaryDestroys = 0

	// Count resources by action type
	for _, resource := range resourceChanges {
		if resource.Mode == "data" {
			continue
		}

		// Special handling for replacements
		if isReplacement(resource.Change.Actions) {
			resources.SummaryAdds++
			resources.SummaryDestroys++
			continue
		}

		// Count standard actions
		countActionsForSummary(resources, resource.Change.Actions)
	}
}

// countActionsForSummary counts actions for the summary counters
func countActionsForSummary(resources *model.ResourceCollection, actions []string) {
	for _, action := range actions {
		switch action {
		case "create":
			resources.SummaryAdds++
		case "update":
			resources.SummaryChanges++
		case "delete":
			resources.SummaryDestroys++
		}
	}
}

// parseResourceChangesOnly attempts to extract just the resource changes from the JSON
// This is a fallback when the full JSON structure doesn't match our expectations
func parseResourceChangesOnly(data []byte) (*model.ResourceCollection, error) {
	resources := model.NewResourceCollection()

	// Try to extract just the resource_changes array
	var jsonMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	// Check if resource_changes exists
	resourceChangesJSON, ok := jsonMap["resource_changes"]
	if !ok {
		return nil, fmt.Errorf("couldn't find resource_changes in the JSON")
	}

	// Parse the resource changes
	var resourceChanges []struct {
		Address string `json:"address"`
		Mode    string `json:"mode"`
		Change  struct {
			Actions []string `json:"actions"`
		} `json:"change"`
	}

	if err := json.Unmarshal(resourceChangesJSON, &resourceChanges); err != nil {
		return nil, fmt.Errorf("error parsing resource changes: %w", err)
	}

	// Process the resource changes
	processSimplifiedResourceChanges(resources, resourceChanges)

	resources.FoundSummary = true
	resources.HasDetailedResources = true
	return resources, nil
}

// processSimplifiedResourceChanges processes the simplified resource changes structure
func processSimplifiedResourceChanges(resources *model.ResourceCollection, resourceChanges []struct {
	Address string `json:"address"`
	Mode    string `json:"mode"`
	Change  struct {
		Actions []string `json:"actions"`
	} `json:"change"`
}) {
	for _, resource := range resourceChanges {
		// Skip data resources
		if resource.Mode == "data" {
			continue
		}

		// Check for replacements
		if isReplacement(resource.Change.Actions) {
			resources.AddResource(model.ActionCreate, resource.Address)
			resources.AddResource(model.ActionDestroy, resource.Address)
			resources.SummaryAdds++
			resources.SummaryDestroys++
			continue
		}

		// Process standard actions
		for _, action := range resource.Change.Actions {
			switch action {
			case "create":
				resources.AddResource(model.ActionCreate, resource.Address)
				resources.SummaryAdds++
			case "update":
				resources.AddResource(model.ActionUpdate, resource.Address)
				resources.SummaryChanges++
			case "delete":
				resources.AddResource(model.ActionDestroy, resource.Address)
				resources.SummaryDestroys++
			}
		}
	}
}
