package model

import (
	"sort"
	"strings"
)

// Action represents a Terraform resource action type
type Action string

const (
	ActionCreate  Action = "create"
	ActionUpdate  Action = "update"
	ActionDestroy Action = "destroy"
)

// ResourceCollection represents resources grouped by action
type ResourceCollection struct {
	Resources            map[Action]map[string]struct{} // Maps action to a set of resource identifiers
	FoundSummary         bool                           // Whether a plan summary line was found
	SummaryAdds          int                            // Count of additions from summary line
	SummaryChanges       int                            // Count of changes from summary line
	SummaryDestroys      int                            // Count of deletions from summary line
	HasDetailedResources bool                           // Whether the plan includes detailed resource info
}

// NewResourceCollection creates a new ResourceCollection
func NewResourceCollection() *ResourceCollection {
	return &ResourceCollection{
		Resources: map[Action]map[string]struct{}{
			ActionCreate:  {},
			ActionUpdate:  {},
			ActionDestroy: {},
		},
		FoundSummary:         false,
		HasDetailedResources: false,
	}
}

// AddResource adds a resource to the collection for a given action
func (rc *ResourceCollection) AddResource(action Action, resource string) {
	if rc.Resources[action] == nil {
		rc.Resources[action] = make(map[string]struct{})
	}
	rc.Resources[action][resource] = struct{}{}
	rc.HasDetailedResources = true
}

// GetResourcesForAction returns a sorted slice of resources for a given action
func (rc *ResourceCollection) GetResourcesForAction(action Action) []string {
	var resources []string
	for r := range rc.Resources[action] {
		resources = append(resources, r)
	}
	sort.Strings(resources)
	return resources
}

// CountResourcesForAction returns the number of resources for a given action
func (rc *ResourceCollection) CountResourcesForAction(action Action) int {
	return len(rc.Resources[action])
}

// TotalChanges returns the total number of changes across all actions
func (rc *ResourceCollection) TotalChanges() int {
	if rc.HasDetailedResources {
		if rc.FoundSummary {
			// If we have the summary from the plan output, use that
			return rc.SummaryAdds + rc.SummaryChanges + rc.SummaryDestroys
		}

		// Otherwise, count resources
		total := 0
		for _, action := range []Action{ActionCreate, ActionUpdate, ActionDestroy} {
			total += len(rc.Resources[action])
		}
		return total
	}

	// Fallback to summary if no detailed resources
	return rc.SummaryAdds + rc.SummaryChanges + rc.SummaryDestroys
}

// isModuleResource checks if a resource is a module resource
func isModuleResource(resource string) bool {
	return strings.HasPrefix(resource, "module.")
}

// ResourcesByType returns a map of resources grouped by type for a given action
func (rc *ResourceCollection) ResourcesByType(action Action) map[string][]string {
	typeMap := make(map[string][]string)

	for res := range rc.Resources[action] {
		// Special handling for module resources
		if isModuleResource(res) {
			// Put it in the "module" category
			if typeMap["module"] == nil {
				typeMap["module"] = []string{}
			}
			typeMap["module"] = append(typeMap["module"], res)
			continue
		}

		// For non-module resources, use the resource type as the key
		resourceType := ExtractResourceType(res)

		if typeMap[resourceType] == nil {
			typeMap[resourceType] = []string{}
		}
		typeMap[resourceType] = append(typeMap[resourceType], res)
	}

	// Sort resources within each type
	for _, resources := range typeMap {
		sort.Strings(resources)
	}

	return typeMap
}

// ExtractResourceType returns the resource type from a terraform resource string
func ExtractResourceType(resource string) string {
	// For module resources, it should be handled by isModuleResource function
	if isModuleResource(resource) {
		return "module"
	}

	// For standard resources (aws_s3_bucket.name)
	parts := strings.Split(resource, ".")
	if len(parts) >= 2 {
		return parts[0]
	}

	// Fallback
	return "unknown"
}
