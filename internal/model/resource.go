package model

import (
	"regexp"
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

	// Track resources by their canonical form for deduplication
	canonicalResources map[string]struct{}
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
		canonicalResources:   map[string]struct{}{},
	}
}

// isNestedAttribute checks if a resource identifier is actually a nested attribute/block
func isNestedAttribute(resource string) bool {
	// List of common terraform nested blocks that should not be treated as resources
	nestedBlockPatterns := []string{
		"statement", "action", "visibility_config", "field_to_match",
		"and_statement", "or_statement", "not_statement", "uri_path",
		"text_transformation", "allow", "block", "rule", "parameter",
		"setting", "alias", "override_action", "none", "regular_expression",
		"regex_pattern_set_reference_statement", "label_match_statement",
		"rule_group_reference_statement", "regex_match_statement", "tags",
		"tags_all",
	}

	for _, pattern := range nestedBlockPatterns {
		if strings.HasPrefix(resource, pattern) ||
			strings.HasSuffix(resource, pattern) ||
			resource == pattern ||
			strings.Contains(resource, " "+pattern) {
			return true
		}
	}

	// Check if it's a value assignment (e.g., "tags = {}")
	if strings.Contains(resource, " = ") {
		return true
	}

	return false
}

// normalizeResourceIdentifier converts various resource formats to a consistent form
func normalizeResourceIdentifier(resource string) string {
	// Handle resource type+name format: 'resource "aws_type" "name"'
	resourceDefPattern := regexp.MustCompile(`resource\s+"([^"]+)"\s+"([^"]+)"`)
	if matches := resourceDefPattern.FindStringSubmatch(resource); len(matches) > 2 {
		return matches[1] + "." + matches[2]
	}

	return resource
}

// getCanonicalResourceIdentifier returns a canonical form of the resource identifier for deduplication
func getCanonicalResourceIdentifier(resource string) string {
	// Extract the module path and resource name
	modulePattern := regexp.MustCompile(`module\.(.+)\.([^.]+)\.([^.]+)`)
	if matches := modulePattern.FindStringSubmatch(resource); len(matches) > 3 {
		return "module." + matches[1] + "." + matches[2] + "." + matches[3]
	}

	// Handle resource type+name format
	resourceDefPattern := regexp.MustCompile(`resource\s+"([^"]+)"\s+"([^"]+)"`)
	if matches := resourceDefPattern.FindStringSubmatch(resource); len(matches) > 2 {
		return matches[1] + "." + matches[2]
	}

	return resource
}

// isModuleResource checks if a resource is a module resource
func isModuleResource(resource string) bool {
	return strings.HasPrefix(resource, "module.")
}

// AddResource adds a resource to the collection for a given action
func (rc *ResourceCollection) AddResource(action Action, resource string) {
	// Skip nested attributes/blocks
	if isNestedAttribute(resource) {
		return
	}

	// Skip resource if it's just a # (comment)
	if strings.TrimSpace(resource) == "#" {
		return
	}

	// Normalize the resource identifier
	normalizedResource := normalizeResourceIdentifier(resource)

	// Get canonical form for deduplication
	canonicalResource := getCanonicalResourceIdentifier(normalizedResource)

	// Skip if we've already seen this resource (deduplication)
	if _, exists := rc.canonicalResources[canonicalResource]; exists {
		return
	}

	// Add to canonical resources for deduplication
	rc.canonicalResources[canonicalResource] = struct{}{}

	if rc.Resources[action] == nil {
		rc.Resources[action] = make(map[string]struct{})
	}

	rc.Resources[action][normalizedResource] = struct{}{}
	rc.HasDetailedResources = true
}

// AddReplacement adds a resource that will be replaced (both destroyed and created)
func (rc *ResourceCollection) AddReplacement(resource string) {
	// Skip nested attributes/blocks
	if isNestedAttribute(resource) {
		return
	}

	// Normalize the resource identifier
	normalizedResource := normalizeResourceIdentifier(resource)

	// Get canonical form for deduplication
	canonicalResource := getCanonicalResourceIdentifier(normalizedResource)

	// Skip if we've already seen this resource (deduplication)
	if _, exists := rc.canonicalResources[canonicalResource]; exists {
		return
	}

	// Add to canonical resources for deduplication
	rc.canonicalResources[canonicalResource] = struct{}{}

	rc.AddResource(ActionDestroy, normalizedResource)
	rc.AddResource(ActionCreate, normalizedResource)
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

		// Otherwise, count unique resources across all actions
		return len(rc.canonicalResources)
	}

	// Fallback to summary if no detailed resources
	return rc.SummaryAdds + rc.SummaryChanges + rc.SummaryDestroys
}

// ResourcesByType returns a map of resources grouped by type for a given action
func (rc *ResourceCollection) ResourcesByType(action Action) map[string][]string {
	typeMap := make(map[string][]string)

	for res := range rc.Resources[action] {
		// Special handling for module resources
		if isModuleResource(res) {
			if typeMap["module"] == nil {
				typeMap["module"] = []string{}
			}
			typeMap["module"] = append(typeMap["module"], res)
			continue
		}

		resourceType := ExtractResourceType(res)
		typeMap[resourceType] = append(typeMap[resourceType], res)
	}

	// Sort resources within each type
	for _, resources := range typeMap {
		sort.Strings(resources)
	}

	return typeMap
}

// ModuleResourceRegex matches module.xxx.resource_type.resource_name pattern
var ModuleResourceRegex = regexp.MustCompile(`module\.(.+)\.([^.]+)\.([^.]+)`)

// ExtractResourceType returns the resource type from a terraform resource string
func ExtractResourceType(resource string) string {
	// Resource format can be one of:
	// 1. aws_s3_bucket.example
	// 2. module.network.aws_vpc.main
	// 3. resource "aws_s3_bucket" "example"

	// First, check for "resource "type" "name"" format
	resourceDefPattern := regexp.MustCompile(`^\s*resource\s+"([^"]+)"\s+"([^"]+)"`)
	if matches := resourceDefPattern.FindStringSubmatch(resource); len(matches) > 1 {
		return matches[1]
	}

	// Handle module resources - they go into the "module" category
	if isModuleResource(resource) {
		return "module"
	}

	// For standard resources (aws_s3_bucket.name)
	parts := strings.Split(resource, ".")
	if len(parts) >= 2 {
		return parts[0]
	}

	// Fallback
	return resource
}
