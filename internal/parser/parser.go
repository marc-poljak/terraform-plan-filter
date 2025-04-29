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

	// Parse the JSON
	var plan TerraformPlanJSON
	if err := json.Unmarshal(data, &plan); err != nil {
		// Handle more gracefully if the JSON structure doesn't match our expectations
		// First, check if this is a text plan instead of JSON
		if strings.HasPrefix(string(data), "Terraform will perform") {
			return nil, fmt.Errorf("input appears to be text format, not JSON. Please use 'terraform show -json tfplan' to generate JSON output")
		}

		// Try a more flexible approach to at least extract resource changes
		return parseResourceChangesOnly(data)
	}

	// Process resource changes from the JSON plan
	for _, resource := range plan.ResourceChanges {
		// Skip data resources
		if resource.Mode == "data" {
			continue
		}

		// Check for special case: "no-op" actions (read-only)
		if len(resource.Change.Actions) == 1 && resource.Change.Actions[0] == "no-op" {
			continue
		}

		// Track if this is a replacement (will be handled specially)
		isReplacement := false
		for _, action := range resource.Change.Actions {
			if action == "replace" {
				isReplacement = true
				break
			}
		}

		// Process replacement resources specially to ensure they appear in both create and destroy
		if isReplacement {
			resources.AddResource(model.ActionCreate, resource.Address)
			resources.AddResource(model.ActionDestroy, resource.Address)
			continue
		}

		// Process standard resources based on their actions
		for _, action := range resource.Change.Actions {
			switch action {
			case "create":
				resources.AddResource(model.ActionCreate, resource.Address)
			case "update":
				resources.AddResource(model.ActionUpdate, resource.Address)
			case "delete":
				resources.AddResource(model.ActionDestroy, resource.Address)
			}
		}
	}

	// Set summary values by counting the resource changes
	resources.FoundSummary = true

	// Reset summary counters
	resources.SummaryAdds = 0
	resources.SummaryChanges = 0
	resources.SummaryDestroys = 0

	// Count resources by action type
	for _, resource := range plan.ResourceChanges {
		if resource.Mode == "data" {
			continue
		}

		// Special handling for replacements
		isReplacement := false
		for _, action := range resource.Change.Actions {
			if action == "replace" {
				isReplacement = true
				break
			}
		}

		if isReplacement {
			resources.SummaryAdds++
			resources.SummaryDestroys++
			continue
		}

		// Count standard actions
		for _, action := range resource.Change.Actions {
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

	resources.HasDetailedResources = true
	return resources, nil
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
	for _, resource := range resourceChanges {
		// Skip data resources
		if resource.Mode == "data" {
			continue
		}

		// Check for replacements
		isReplacement := false
		for _, action := range resource.Change.Actions {
			if action == "replace" {
				isReplacement = true
				resources.AddResource(model.ActionCreate, resource.Address)
				resources.AddResource(model.ActionDestroy, resource.Address)
				resources.SummaryAdds++
				resources.SummaryDestroys++
				break
			}
		}

		if isReplacement {
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

	resources.FoundSummary = true
	resources.HasDetailedResources = true
	return resources, nil
}
