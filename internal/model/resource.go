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

// AddReplacement adds a resource that will be replaced (both destroyed and created)
func (rc *ResourceCollection) AddReplacement(resource string) {
	rc.AddResource(ActionDestroy, resource)
	rc.AddResource(ActionCreate, resource)
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
		total := 0
		for _, action := range []Action{ActionCreate, ActionUpdate, ActionDestroy} {
			total += len(rc.Resources[action])
		}
		return total
	}
	return rc.SummaryAdds + rc.SummaryChanges + rc.SummaryDestroys
}

// ResourcesByType returns a map of resources grouped by type for a given action
func (rc *ResourceCollection) ResourcesByType(action Action) map[string][]string {
	typeMap := make(map[string][]string)

	for res := range rc.Resources[action] {
		resourceType := ExtractResourceType(res)
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
	// Resource format can be one of:
	// 1. aws_s3_bucket.example
	// 2. module.network.aws_vpc.main

	parts := strings.Split(resource, ".")
	if len(parts) > 2 && parts[0] == "module" {
		// It's a module resource, find the resource type
		for i, part := range parts {
			// Resource types usually start with provider prefix (aws_, azurerm_, google_)
			if i > 1 && (strings.HasPrefix(part, "aws_") ||
				strings.HasPrefix(part, "azurerm_") ||
				strings.HasPrefix(part, "google_") ||
				strings.HasPrefix(part, "kubernetes_") ||
				strings.HasPrefix(part, "digitalocean_")) {
				return part
			}
		}
	}

	// For non-module resources
	if len(parts) >= 2 {
		return parts[0]
	}

	// Fallback
	return resource
}
