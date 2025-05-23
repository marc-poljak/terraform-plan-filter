package parser

import (
	"strings"
	"testing"

	"github.com/marc-poljak/terraform-plan-filter/internal/model"
)

func TestParseTerraformPlan(t *testing.T) {
	// Sample JSON plan with different resource actions
	jsonPlan := createSampleJSONPlan()

	reader := strings.NewReader(jsonPlan)
	resources, err := ParseTerraformPlan(reader)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Test create resources
	testCreateResources(t, resources)

	// Test update resources
	testUpdateResources(t, resources)

	// Test destroy resources
	testDestroyResources(t, resources)

	// Test summary counts
	testSummaryCounts(t, resources)

	// Test that data resources are excluded
	testDataResourcesExcluded(t, resources)
}

// createSampleJSONPlan returns a sample JSON plan for testing
func createSampleJSONPlan() string {
	return `{
		"format_version": "1.0",
		"terraform_version": "1.4.6",
		"resource_changes": [
			{
				"address": "aws_s3_bucket.logs",
				"mode": "managed",
				"type": "aws_s3_bucket",
				"name": "logs",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["create"],
					"before": null,
					"after": {}
				}
			},
			{
				"address": "aws_instance.web_server",
				"mode": "managed",
				"type": "aws_instance",
				"name": "web_server",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["update"],
					"before": {},
					"after": {}
				}
			},
			{
				"address": "aws_cloudfront_distribution.legacy_cdn",
				"mode": "managed",
				"type": "aws_cloudfront_distribution",
				"name": "legacy_cdn",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["delete"],
					"before": {},
					"after": null
				}
			},
			{
				"address": "aws_instance.replacement_server",
				"mode": "managed",
				"type": "aws_instance",
				"name": "replacement_server",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["replace"],
					"before": {},
					"after": {}
				}
			},
			{
				"address": "data.aws_ami.latest",
				"mode": "data",
				"type": "aws_ami",
				"name": "latest",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["no-op"],
					"before": {},
					"after": {}
				}
			}
		]
	}`
}

// hasResource checks if a resource is in a list
func hasResource(list []string, resource string) bool {
	for _, r := range list {
		if r == resource {
			return true
		}
	}
	return false
}

// testCreateResources tests the create resources
func testCreateResources(t *testing.T, resources *model.ResourceCollection) {
	createResources := resources.GetResourcesForAction(model.ActionCreate)
	if len(createResources) != 2 {
		t.Errorf("Expected 2 create resources, got %d", len(createResources))
		t.Logf("Create resources: %v", createResources)
	}

	if !hasResource(createResources, "aws_s3_bucket.logs") {
		t.Errorf("Expected to find aws_s3_bucket.logs in create resources")
	}

	if !hasResource(createResources, "aws_instance.replacement_server") {
		t.Errorf("Expected to find aws_instance.replacement_server in create resources (replacement)")
	}
}

// testUpdateResources tests the update resources
func testUpdateResources(t *testing.T, resources *model.ResourceCollection) {
	updateResources := resources.GetResourcesForAction(model.ActionUpdate)
	if len(updateResources) != 1 {
		t.Errorf("Expected 1 update resource, got %d", len(updateResources))
	}

	if !hasResource(updateResources, "aws_instance.web_server") {
		t.Errorf("Expected to find aws_instance.web_server in update resources")
	}
}

// testDestroyResources tests the destroy resources
func testDestroyResources(t *testing.T, resources *model.ResourceCollection) {
	destroyResources := resources.GetResourcesForAction(model.ActionDestroy)
	if len(destroyResources) != 2 {
		t.Errorf("Expected 2 destroy resources, got %d", len(destroyResources))
	}

	if !hasResource(destroyResources, "aws_cloudfront_distribution.legacy_cdn") {
		t.Errorf("Expected to find aws_cloudfront_distribution.legacy_cdn in destroy resources")
	}

	if !hasResource(destroyResources, "aws_instance.replacement_server") {
		t.Errorf("Expected to find aws_instance.replacement_server in destroy resources (replacement)")
	}
}

// testSummaryCounts tests the summary counts
func testSummaryCounts(t *testing.T, resources *model.ResourceCollection) {
	if resources.SummaryAdds != 2 {
		t.Errorf("Expected SummaryAdds to be 2, got %d", resources.SummaryAdds)
	}

	if resources.SummaryChanges != 1 {
		t.Errorf("Expected SummaryChanges to be 1, got %d", resources.SummaryChanges)
	}

	if resources.SummaryDestroys != 2 {
		t.Errorf("Expected SummaryDestroys to be 2, got %d", resources.SummaryDestroys)
	}
}

// testDataResourcesExcluded tests that data resources are excluded
func testDataResourcesExcluded(t *testing.T, resources *model.ResourceCollection) {
	createResources := resources.GetResourcesForAction(model.ActionCreate)
	updateResources := resources.GetResourcesForAction(model.ActionUpdate)
	destroyResources := resources.GetResourcesForAction(model.ActionDestroy)

	for _, resource := range createResources {
		if strings.HasPrefix(resource, "data.") {
			t.Errorf("Found data resource in create list: %s", resource)
		}
	}

	for _, resource := range updateResources {
		if strings.HasPrefix(resource, "data.") {
			t.Errorf("Found data resource in update list: %s", resource)
		}
	}

	for _, resource := range destroyResources {
		if strings.HasPrefix(resource, "data.") {
			t.Errorf("Found data resource in destroy list: %s", resource)
		}
	}
}

func TestParseTerraformPlanWithInvalidJSON(t *testing.T) {
	// Test with invalid JSON
	testWithInvalidJSON(t)

	// Test with text format instead of JSON
	testWithTextFormat(t)
}

// testWithInvalidJSON tests the behavior with invalid JSON
func testWithInvalidJSON(t *testing.T) {
	invalidJSON := `{ This is not valid JSON }`
	reader := strings.NewReader(invalidJSON)
	_, err := ParseTerraformPlan(reader)

	if err == nil {
		t.Errorf("Expected error for invalid JSON, but got none")
	}
}

// testWithTextFormat tests the behavior with text format
func testWithTextFormat(t *testing.T) {
	textFormat := `Terraform will perform the following actions:
  + aws_s3_bucket.logs
  ~ aws_instance.web_server
  - aws_cloudfront_distribution.legacy_cdn`
	reader := strings.NewReader(textFormat)
	_, err := ParseTerraformPlan(reader)

	if err == nil {
		t.Errorf("Expected error for text format, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "input appears to be text format") {
		t.Errorf("Expected error about text format, got: %v", err)
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
			expected: "module",
		},
		{
			name:     "Nested module resource",
			input:    "module.network.module.subnets.aws_subnet.private",
			expected: "module",
		},
		{
			name:     "Non-standard resource",
			input:    "custom_provider_resource.example",
			expected: "custom_provider_resource",
		},
		{
			name:     "Fallback for unrecognized format",
			input:    "some_weird_format",
			expected: "unknown",
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
