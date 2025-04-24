package parser

import (
	"strings"
	"testing"

	"github.com/marc-poljak/terraform-plan-filter/internal/model"
)

func TestParseTerraformPlan(t *testing.T) {
	tests := []struct {
		name                 string
		input                string
		expectedCreateCount  int
		expectedUpdateCount  int
		expectedDestroyCount int
		expectSummary        bool
		summaryAdds          int
		summaryChanges       int
		summaryDestroys      int
	}{
		{
			name: "Simple plan with all action types",
			input: `
+ aws_s3_bucket.new_bucket {
~ aws_instance.web_server {
- aws_cloudfront_distribution.legacy_cdn {
Plan: 1 to add, 1 to change, 1 to destroy.
`,
			expectedCreateCount:  1,
			expectedUpdateCount:  1,
			expectedDestroyCount: 1,
			expectSummary:        true,
			summaryAdds:          1,
			summaryChanges:       1,
			summaryDestroys:      1,
		},
		{
			name: "Modern format with 'will be' phrases",
			input: `
aws_s3_bucket.logs will be created
module.network.aws_vpc.main will be updated in-place
aws_lambda_function.old_function will be destroyed
Plan: 1 to add, 1 to change, 1 to destroy.
`,
			expectedCreateCount:  1,
			expectedUpdateCount:  1,
			expectedDestroyCount: 1,
			expectSummary:        true,
			summaryAdds:          1,
			summaryChanges:       1,
			summaryDestroys:      1,
		},
		{
			name: "Resource replacement",
			input: `
aws_instance.web_server must be replaced
Plan: 1 to add, 0 to change, 1 to destroy.
`,
			expectedCreateCount:  1,
			expectedUpdateCount:  0,
			expectedDestroyCount: 1,
			expectSummary:        true,
			summaryAdds:          1,
			summaryChanges:       0,
			summaryDestroys:      1,
		},
		{
			name: "Empty plan",
			input: `
No changes. Your infrastructure matches the configuration.
`,
			expectedCreateCount:  0,
			expectedUpdateCount:  0,
			expectedDestroyCount: 0,
			expectSummary:        false,
		},
		{
			name: "Summary only",
			input: `
Plan: 2 to add, 3 to change, 1 to destroy.
`,
			expectedCreateCount:  0,
			expectedUpdateCount:  0,
			expectedDestroyCount: 0,
			expectSummary:        true,
			summaryAdds:          2,
			summaryChanges:       3,
			summaryDestroys:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			resources, err := ParseTerraformPlan(reader)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if count := resources.CountResourcesForAction(model.ActionCreate); count != tt.expectedCreateCount {
				t.Errorf("Expected %d create resources, got %d", tt.expectedCreateCount, count)
			}

			if count := resources.CountResourcesForAction(model.ActionUpdate); count != tt.expectedUpdateCount {
				t.Errorf("Expected %d update resources, got %d", tt.expectedUpdateCount, count)
			}

			if count := resources.CountResourcesForAction(model.ActionDestroy); count != tt.expectedDestroyCount {
				t.Errorf("Expected %d destroy resources, got %d", tt.expectedDestroyCount, count)
			}

			if resources.FoundSummary != tt.expectSummary {
				t.Errorf("Expected FoundSummary to be %v, got %v", tt.expectSummary, resources.FoundSummary)
			}

			if tt.expectSummary {
				if resources.SummaryAdds != tt.summaryAdds {
					t.Errorf("Expected SummaryAdds to be %d, got %d", tt.summaryAdds, resources.SummaryAdds)
				}

				if resources.SummaryChanges != tt.summaryChanges {
					t.Errorf("Expected SummaryChanges to be %d, got %d", tt.summaryChanges, resources.SummaryChanges)
				}

				if resources.SummaryDestroys != tt.summaryDestroys {
					t.Errorf("Expected SummaryDestroys to be %d, got %d", tt.summaryDestroys, resources.SummaryDestroys)
				}
			}
		})
	}
}

func TestExtractResourceType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple resource",
			input:    "aws_s3_bucket.example",
			expected: "aws_s3_bucket",
		},
		{
			name:     "Module resource",
			input:    "module.network.aws_vpc.main",
			expected: "aws_vpc",
		},
		{
			name:     "Nested module resource",
			input:    "module.network.module.subnets.aws_subnet.private",
			expected: "aws_subnet",
		},
		{
			name:     "Non-standard resource",
			input:    "custom_provider_resource.example",
			expected: "custom_provider_resource",
		},
		{
			name:     "Fallback for unrecognized format",
			input:    "some_weird_format",
			expected: "some_weird_format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := model.ExtractResourceType(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
