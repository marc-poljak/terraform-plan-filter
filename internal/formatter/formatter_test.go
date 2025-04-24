package formatter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/marc-poljak/terraform-plan-filter/internal/model"
)

func TestFormatText(t *testing.T) {
	// Create a sample resource collection
	resources := model.NewResourceCollection()

	// Add some test resources
	resources.AddResource(model.ActionCreate, "aws_s3_bucket.logs")
	resources.AddResource(model.ActionCreate, "aws_s3_bucket.data")
	resources.AddResource(model.ActionUpdate, "aws_instance.web")
	resources.AddResource(model.ActionDestroy, "aws_cloudfront_distribution.old")

	// Test with colors disabled
	opts := Options{
		UseColors: false,
		Verbose:   false,
	}

	result, err := FormatText(resources, opts)
	if err != nil {
		t.Fatalf("FormatText returned an error: %v", err)
	}

	// Check for expected content
	expectedPhrases := []string{
		"TERRAFORM PLAN SUMMARY",
		"RESOURCES TO CREATE:",
		"# AWS_S3_BUCKET RESOURCES:",
		"+ aws_s3_bucket.data",
		"+ aws_s3_bucket.logs",
		"RESOURCES TO UPDATE:",
		"# AWS_INSTANCE RESOURCES:",
		"~ aws_instance.web",
		"RESOURCES TO DESTROY:",
		"# AWS_CLOUDFRONT_DISTRIBUTION RESOURCES:",
		"- aws_cloudfront_distribution.old",
		"TOTAL CHANGES: 4",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(result, phrase) {
			t.Errorf("Expected output to contain %q, but it didn't", phrase)
		}
	}
}

func TestFormatJSON(t *testing.T) {
	// Create a sample resource collection
	resources := model.NewResourceCollection()

	// Add some test resources
	resources.AddResource(model.ActionCreate, "aws_s3_bucket.logs")
	resources.AddResource(model.ActionUpdate, "aws_instance.web")
	resources.AddResource(model.ActionDestroy, "aws_cloudfront_distribution.old")

	jsonOutput, err := FormatJSON(resources)
	if err != nil {
		t.Fatalf("FormatJSON returned an error: %v", err)
	}

	// Parse the JSON to verify it's valid
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Check that the expected fields are present
	expectedFields := []string{"create", "update", "destroy", "summary", "has_detailed_resources", "found_summary", "timestamp"}
	for _, field := range expectedFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("Expected JSON to contain field %q, but it didn't", field)
		}
	}

	// Check the array lengths
	create, ok := parsed["create"].([]interface{})
	if !ok || len(create) != 1 {
		t.Errorf("Expected 'create' to be an array of length 1, got %v", parsed["create"])
	}

	update, ok := parsed["update"].([]interface{})
	if !ok || len(update) != 1 {
		t.Errorf("Expected 'update' to be an array of length 1, got %v", parsed["update"])
	}

	destroy, ok := parsed["destroy"].([]interface{})
	if !ok || len(destroy) != 1 {
		t.Errorf("Expected 'destroy' to be an array of length 1, got %v", parsed["destroy"])
	}
}

func TestFormatHTML(t *testing.T) {
	// Create a sample resource collection
	resources := model.NewResourceCollection()

	// Add some test resources
	resources.AddResource(model.ActionCreate, "aws_s3_bucket.logs")
	resources.AddResource(model.ActionUpdate, "aws_instance.web")

	html, err := FormatHTML(resources)
	if err != nil {
		t.Fatalf("FormatHTML returned an error: %v", err)
	}

	// Check for expected HTML elements
	expectedElements := []string{
		"<!DOCTYPE html>",
		"<html",
		"<head>",
		"<title>Terraform Plan Summary</title>",
		"<style>",
		"<body>",
		"<h1>Terraform Plan Summary</h1>",
		"<div class=\"summary\">",
		"aws_s3_bucket.logs",
		"aws_instance.web",
		"<div class=\"timestamp\">",
	}

	for _, el := range expectedElements {
		if !strings.Contains(html, el) {
			t.Errorf("Expected HTML to contain %q, but it didn't", el)
		}
	}
}

func TestFormatSummaryOnly(t *testing.T) {
	// Create a sample resource collection with only summary info
	resources := model.NewResourceCollection()
	resources.FoundSummary = true
	resources.SummaryAdds = 2
	resources.SummaryChanges = 1
	resources.SummaryDestroys = 3

	// Test with colors disabled
	opts := Options{
		UseColors: false,
		Verbose:   false,
	}

	result, err := FormatText(resources, opts)
	if err != nil {
		t.Fatalf("FormatText returned an error: %v", err)
	}

	// Check for expected content
	expectedPhrases := []string{
		"TERRAFORM PLAN SUMMARY",
		"RESOURCES TO CREATE: 2 (details not available)",
		"RESOURCES TO UPDATE: 1 (details not available)",
		"RESOURCES TO DESTROY: 3 (details not available)",
		"TOTAL CHANGES: 6",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(result, phrase) {
			t.Errorf("Expected output to contain %q, but it didn't", phrase)
		}
	}
}
