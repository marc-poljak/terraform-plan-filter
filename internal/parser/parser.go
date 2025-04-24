package parser

import (
	"bufio"
	"io"
	"regexp"
	"strconv"

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
		{regexp.MustCompile(`^\s*\+\s(.+?)\s+{`), model.ActionCreate, false},
		{regexp.MustCompile(`^\s*~\s(.+?)\s+{`), model.ActionUpdate, false},
		{regexp.MustCompile(`^\s*-\s(.+?)\s+{`), model.ActionDestroy, false},
		{regexp.MustCompile(`^(.+?)\s+will\s+be\s+created`), model.ActionCreate, false},
		{regexp.MustCompile(`^(.+?)\s+will\s+be\s+updated\s+in-place`), model.ActionUpdate, false},
		{regexp.MustCompile(`^(.+?)\s+will\s+be\s+destroyed`), model.ActionDestroy, false},
		{regexp.MustCompile(`^(.+?)\s+must\s+be\s+replaced`), "", true}, // Special case for replacements
	}

	// Pattern for the plan summary line
	planSummaryRegex = regexp.MustCompile(`Plan:\s+(\d+)\s+to\s+add,\s+(\d+)\s+to\s+change,\s+(\d+)\s+to\s+destroy`)
)

// ParseTerraformPlan parses Terraform plan output and returns a ResourceCollection
func ParseTerraformPlan(reader io.Reader) (*model.ResourceCollection, error) {
	resources := model.NewResourceCollection()
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		// Check for resource actions
		for _, pattern := range patterns {
			if matches := pattern.regex.FindStringSubmatch(line); len(matches) > 1 {
				if pattern.double {
					// This is a replacement (both create and destroy)
					resources.AddReplacement(matches[1])
				} else {
					// Regular action
					resources.AddResource(pattern.action, matches[1])
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
