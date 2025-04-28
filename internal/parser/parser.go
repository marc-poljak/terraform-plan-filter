package parser

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/marc-poljak/terraform-plan-filter/internal/model"
)

// patternDefinition defines a regex pattern and its associated action
type patternDefinition struct {
	regex  *regexp.Regexp
	action model.Action
	double bool // If true, this matches resources that are both created and destroyed (replaced)
}

var (
	// Patterns for different Terraform resource actions
	patterns = []patternDefinition{
		// Main resource patterns - these match the resource type and identifier
		{regexp.MustCompile(`^\s*\+\s+(.+?)\s+{`), model.ActionCreate, false},
		{regexp.MustCompile(`^\s*~\s+(.+?)\s+{`), model.ActionUpdate, false},
		{regexp.MustCompile(`^\s*-\s+(.+?)\s+{`), model.ActionDestroy, false},
		{regexp.MustCompile(`^(.+?)\s+will\s+be\s+created`), model.ActionCreate, false},
		{regexp.MustCompile(`^(.+?)\s+will\s+be\s+updated\s+in-place`), model.ActionUpdate, false},
		{regexp.MustCompile(`^(.+?)\s+will\s+be\s+destroyed`), model.ActionDestroy, false},
		{regexp.MustCompile(`^(.+?)\s+must\s+be\s+replaced`), "", true}, // Special case for replacements

		// Resource action patterns from Terraform 0.12+ format
		{regexp.MustCompile(`^\s*#\s+(.+?)\s+will\s+be\s+created`), model.ActionCreate, false},
		{regexp.MustCompile(`^\s*#\s+(.+?)\s+will\s+be\s+updated\s+in-place`), model.ActionUpdate, false},
		{regexp.MustCompile(`^\s*#\s+(.+?)\s+will\s+be\s+destroyed`), model.ActionDestroy, false},
		{regexp.MustCompile(`^\s*#\s+(.+?)\s+must\s+be\s+replaced`), "", true}, // Special case for replacements

		// Module resource patterns - requires special handling
		{regexp.MustCompile(`^\s*#\s+module\.(.+?)\s+will\s+be\s+created`), model.ActionCreate, false},
		{regexp.MustCompile(`^\s*#\s+module\.(.+?)\s+will\s+be\s+updated\s+in-place`), model.ActionUpdate, false},
		{regexp.MustCompile(`^\s*#\s+module\.(.+?)\s+will\s+be\s+destroyed`), model.ActionDestroy, false},
		{regexp.MustCompile(`^\s*#\s+module\.(.+?)\s+must\s+be\s+replaced`), "", true}, // Special case for replacements

		// Resource definition patterns
		{regexp.MustCompile(`^\s*\+\s+resource\s+"([^"]+)"\s+"([^"]+)"\s+{`), model.ActionCreate, false},
		{regexp.MustCompile(`^\s*~\s+resource\s+"([^"]+)"\s+"([^"]+)"\s+{`), model.ActionUpdate, false},
		{regexp.MustCompile(`^\s*-\s+resource\s+"([^"]+)"\s+"([^"]+)"\s+{`), model.ActionDestroy, false},
	}

	// Resource identifier patterns
	resourceIdentifierPattern = regexp.MustCompile(`^\s*(?:resource\s+)?"([^"]+)"\s+"([^"]+)"`)

	// Module resource identifier pattern
	moduleResourcePattern = regexp.MustCompile(`^\s*#\s+module\.(.+?)\.([^.]+?)\.([^.]+?)\s+will`)

	// Data resource pattern - we want to ignore data resources
	dataResourcePattern = regexp.MustCompile(`^\s*#?\s*data\s+`)

	// Pattern for the plan summary line
	planSummaryRegex = regexp.MustCompile(`Plan:\s+(\d+)\s+to\s+add,\s+(\d+)\s+to\s+change,\s+(\d+)\s+to\s+destroy`)

	// New pattern to identify resource actions in detailed format
	resourceActionPattern = regexp.MustCompile(`^\s*([\+\-~])\s+(.+?)\s+=(.*)$`)

	// Pattern to identify block openings (which we want to filter out)
	blockOpeningPattern = regexp.MustCompile(`^\s*([a-z_]+)\s+{$`)

	// Pattern to identify non-resource blocks like "statement", "action", etc.
	nonResourceBlockPattern = regexp.MustCompile(`^\s*([a-z_]+)\s*(\{|\()`)

	// Pattern to identify tag blocks
	tagPattern = regexp.MustCompile(`^\s*(tags|tags_all)\s+(=|{)`)
)

// isNonResourceBlock checks if a line matches a block that shouldn't be treated as a resource
func isNonResourceBlock(line string) bool {
	// List of block types that shouldn't be treated as resources
	nonResourceBlocks := []string{
		"action", "statement", "visibility_config", "field_to_match",
		"and_statement", "or_statement", "not_statement", "uri_path",
		"text_transformation", "allow", "block", "rule", "parameter",
		"setting", "alias", "override_action", "none", "regular_expression",
		"regex_pattern_set_reference_statement", "label_match_statement",
		"rule_group_reference_statement", "regex_match_statement", "tags",
		"tags_all",
	}

	for _, blockType := range nonResourceBlocks {
		if strings.Contains(line, blockType+" ") ||
			strings.Contains(line, blockType+"{") ||
			strings.Contains(line, blockType+"}") ||
			strings.HasSuffix(line, blockType) {
			return true
		}
	}

	// Check for tag blocks specifically
	if tagPattern.MatchString(line) {
		return true
	}

	return nonResourceBlockPattern.MatchString(line)
}

// extractResourceIdentifier tries to extract a proper resource identifier from a line
func extractResourceIdentifier(line string) (string, bool) {
	// Check for module.X.Y.Z format
	if moduleMatch := moduleResourcePattern.FindStringSubmatch(line); len(moduleMatch) > 3 {
		return "module." + moduleMatch[1] + "." + moduleMatch[2] + "." + moduleMatch[3], true
	}

	// Check for resource "type" "name" format
	if resourceMatch := resourceIdentifierPattern.FindStringSubmatch(line); len(resourceMatch) > 2 {
		return resourceMatch[1] + "." + resourceMatch[2], true
	}

	// Special case for +/- resource "type" "name"
	resourceDefPattern := regexp.MustCompile(`^\s*[\+\-~]\s+resource\s+"([^"]+)"\s+"([^"]+)"`)
	if resourceMatch := resourceDefPattern.FindStringSubmatch(line); len(resourceMatch) > 2 {
		return resourceMatch[1] + "." + resourceMatch[2], true
	}

	// If no match, return the original line
	return line, false
}

// isDataResource checks if a line represents a data resource
func isDataResource(line string) bool {
	return dataResourcePattern.MatchString(line)
}

// ParseTerraformPlan parses Terraform plan output and returns a ResourceCollection
func ParseTerraformPlan(reader io.Reader) (*model.ResourceCollection, error) {
	resources := model.NewResourceCollection()
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip data resources
		if isDataResource(line) {
			continue
		}

		// Skip non-resource blocks
		if isNonResourceBlock(line) {
			continue
		}

		// Check for resource actions
		for _, pattern := range patterns {
			if matches := pattern.regex.FindStringSubmatch(line); len(matches) > 1 {
				resourceIdentifier := matches[1]

				// Try to extract a better identifier if available
				betterIdentifier, found := extractResourceIdentifier(line)
				if found {
					resourceIdentifier = betterIdentifier
				}

				if pattern.double {
					// This is a replacement (both create and destroy)
					resources.AddReplacement(resourceIdentifier)
				} else {
					// Regular action
					resources.AddResource(pattern.action, resourceIdentifier)
				}
				break
			}
		}

		// Check for plan summary line
		if matches := planSummaryRegex.FindStringSubmatch(line); len(matches) > 1 {
			resources.FoundSummary = true
			resources.SummaryAdds, _ = strconv.Atoi(matches[1])
			resources.SummaryChanges, _ = strconv.Atoi(matches[2])
			resources.SummaryDestroys, _ = strconv.Atoi(matches[3])
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return resources, nil
}
