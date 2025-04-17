package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
)

func main() {
	// Regular expressions to match terraform plan output lines
	createRegex := regexp.MustCompile(`^\s*\+\s(.+?)\s+{`)
	updateRegex := regexp.MustCompile(`^\s*~\s(.+?)\s+{`)
	destroyRegex := regexp.MustCompile(`^\s*-\s(.+?)\s+{`)

	// Additional regex to match plan summary line
	// Example: "Plan: 0 to add, 1 to change, 0 to destroy."
	planSummaryRegex := regexp.MustCompile(`Plan:\s+(\d+)\s+to\s+add,\s+(\d+)\s+to\s+change,\s+(\d+)\s+to\s+destroy`)

	// Match Terraform 0.12+ detailed resource lines
	// Examples:
	// "module.rds_daily_backup.aws_backup_plan.this will be updated in-place"
	// "aws_instance.example will be created"
	detailedCreateRegex := regexp.MustCompile(`^(.+?)\s+will\s+be\s+created`)
	detailedUpdateRegex := regexp.MustCompile(`^(.+?)\s+will\s+be\s+updated\s+in-place`)
	detailedDestroyRegex := regexp.MustCompile(`^(.+?)\s+will\s+be\s+destroyed`)
	detailedReplaceRegex := regexp.MustCompile(`^(.+?)\s+must\s+be\s+replaced`)

	// Track resources by action type
	resources := map[string][]string{
		"create":  {},
		"update":  {},
		"destroy": {},
	}

	// Also track if we found a summary line
	foundSummary := false
	summaryAdds := 0
	summaryChanges := 0
	summaryDestroys := 0

	// Read from stdin line by line
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for resource creation
		if matches := createRegex.FindStringSubmatch(line); len(matches) > 1 {
			resources["create"] = append(resources["create"], matches[1])
		}

		// Check for resource updates
		if matches := updateRegex.FindStringSubmatch(line); len(matches) > 1 {
			resources["update"] = append(resources["update"], matches[1])
		}

		// Check for resource destruction
		if matches := destroyRegex.FindStringSubmatch(line); len(matches) > 1 {
			resources["destroy"] = append(resources["destroy"], matches[1])
		}

		// Check for detailed resource descriptions (Terraform 0.12+)
		if matches := detailedCreateRegex.FindStringSubmatch(line); len(matches) > 1 {
			resources["create"] = append(resources["create"], matches[1])
		}

		if matches := detailedUpdateRegex.FindStringSubmatch(line); len(matches) > 1 {
			resources["update"] = append(resources["update"], matches[1])
		}

		if matches := detailedDestroyRegex.FindStringSubmatch(line); len(matches) > 1 {
			resources["destroy"] = append(resources["destroy"], matches[1])
		}

		if matches := detailedReplaceRegex.FindStringSubmatch(line); len(matches) > 1 {
			// Resources being replaced are both destroyed and created
			resources["destroy"] = append(resources["destroy"], matches[1])
			resources["create"] = append(resources["create"], matches[1])
		}

		// Check for plan summary line
		if matches := planSummaryRegex.FindStringSubmatch(line); len(matches) > 1 {
			foundSummary = true
			summaryAdds, _ = strconv.Atoi(matches[1])
			summaryChanges, _ = strconv.Atoi(matches[2])
			summaryDestroys, _ = strconv.Atoi(matches[3])
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Println("\n=== TERRAFORM PLAN SUMMARY ===")
	fmt.Println()

	// If we found detailed resources, use those
	resourcesFound := false

	if len(resources["create"]) > 0 {
		resourcesFound = true
		fmt.Println("RESOURCES TO CREATE:")
		for _, resource := range resources["create"] {
			fmt.Printf("  + %s\n", resource)
		}
		fmt.Println()
	}

	if len(resources["update"]) > 0 {
		resourcesFound = true
		fmt.Println("RESOURCES TO UPDATE:")
		for _, resource := range resources["update"] {
			fmt.Printf("  ~ %s\n", resource)
		}
		fmt.Println()
	}

	if len(resources["destroy"]) > 0 {
		resourcesFound = true
		fmt.Println("RESOURCES TO DESTROY:")
		for _, resource := range resources["destroy"] {
			fmt.Printf("  - %s\n", resource)
		}
		fmt.Println()
	}

	// If we didn't find detailed resources but found a summary line, use that
	if !resourcesFound && foundSummary {
		if summaryAdds > 0 {
			fmt.Printf("RESOURCES TO CREATE: %d (details not available)\n\n", summaryAdds)
		}

		if summaryChanges > 0 {
			fmt.Printf("RESOURCES TO UPDATE: %d (details not available)\n\n", summaryChanges)
		}

		if summaryDestroys > 0 {
			fmt.Printf("RESOURCES TO DESTROY: %d (details not available)\n\n", summaryDestroys)
		}

		fmt.Printf("TOTAL CHANGES: %d\n", summaryAdds+summaryChanges+summaryDestroys)
	} else {
		totalChanges := len(resources["create"]) + len(resources["update"]) + len(resources["destroy"])
		fmt.Printf("TOTAL CHANGES: %d\n", totalChanges)
	}

	// Add a note if we got the summary from Terraform's summary line but couldn't find resources
	if !resourcesFound && foundSummary && (summaryAdds+summaryChanges+summaryDestroys) > 0 {
		fmt.Println("\nNote: Resource details weren't found in the output.")
		fmt.Println("To see full resource details, try running with:")
		fmt.Println("terraform plan -no-color | terraform-plan-filter")
	}
}
