package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
)

func main() {
	// Check if colors should be disabled (e.g., when output is not a terminal)
	useColors := true
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		useColors = false
	}

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
		"create": {},
		"update": {},
		"destroy": {},
	}

	// Also track if we found a summary line
	foundSummary := false
	summaryAdds := 0
	summaryChanges := 0
	summaryDestroys := 0

	// Extract the resource type from a resource name
	extractResourceType := func(resource string) string {
		// Resource format can be one of:
		// 1. aws_s3_bucket.example
		// 2. module.network.aws_vpc.main

		// First, handle module resources by removing module prefix
		parts := strings.Split(resource, ".")
		if len(parts) > 2 && parts[0] == "module" {
			// It's a module resource, find the resource type
			for i, part := range parts {
				// Resource types usually start with provider prefix (aws_, azurerm_, google_)
				if (i > 1 && (strings.HasPrefix(part, "aws_") ||
					strings.HasPrefix(part, "azurerm_") ||
					strings.HasPrefix(part, "google_") ||
					strings.HasPrefix(part, "kubernetes_") ||
					strings.HasPrefix(part, "digitalocean_"))) {
					return part
				}
			}
		}

		// For non-module resources or if we couldn't find a type in a module resource
		if len(parts) >= 2 {
			return parts[0]
		}

		// If we can't determine type, return the full resource
		return resource
	}

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
	fmt.Println()
	if useColors {
		fmt.Println(colorBold + "=== TERRAFORM PLAN SUMMARY ===" + colorReset)
	} else {
		fmt.Println("=== TERRAFORM PLAN SUMMARY ===")
	}
	fmt.Println()

	// If we found detailed resources, use those
	resourcesFound := false

	// Group resources first by action, then by type within each action
	if len(resources["create"]) > 0 || len(resources["update"]) > 0 || len(resources["destroy"]) > 0 {
		resourcesFound = true

		// Function to process and display resources by action, then by type
		displayResourcesByActionThenType := func(action string, actionLabel string, symbolColor string, symbol string) {
			if len(resources[action]) == 0 {
				return
			}

			// Group resources by type
			typeMap := make(map[string][]string)

			for _, resource := range resources[action] {
				resourceType := extractResourceType(resource)
				typeMap[resourceType] = append(typeMap[resourceType], resource)
			}

			// Sort the types
			types := make([]string, 0, len(typeMap))
			for resourceType := range typeMap {
				types = append(types, resourceType)
			}
			sort.Strings(types)

			// Print section header
			if useColors {
				fmt.Printf("%s%s:%s\n", colorBold, actionLabel, colorReset)
			} else {
				fmt.Printf("%s:\n", actionLabel)
			}

			// For each type
			for _, resourceType := range types {
				resources := typeMap[resourceType]

				// Sort resources by name within type
				sort.Strings(resources)

				// Print type subheader
				if useColors {
					fmt.Printf("  %s# %s RESOURCES:%s\n", colorBold, strings.ToUpper(resourceType), colorReset)
				} else {
					fmt.Printf("  # %s RESOURCES:\n", strings.ToUpper(resourceType))
				}

				// Print resources of this type
				for _, resource := range resources {
					if useColors {
						fmt.Printf("    %s%s %s%s\n", symbolColor, symbol, resource, colorReset)
					} else {
						fmt.Printf("    %s %s\n", symbol, resource)
					}
				}
				fmt.Println()
			}
		}

		// Display resources in order: create, update, destroy
		displayResourcesByActionThenType("create", "RESOURCES TO CREATE", colorGreen, "+")
		displayResourcesByActionThenType("update", "RESOURCES TO UPDATE", colorYellow, "~")
		displayResourcesByActionThenType("destroy", "RESOURCES TO DESTROY", colorRed, "-")
	}

	// If we didn't find detailed resources but found a summary line, use that
	if !resourcesFound && foundSummary {
		if summaryAdds > 0 {
			if useColors {
				fmt.Printf("%sRESOURCES TO CREATE:%s %d (details not available)\n\n",
					colorBold, colorReset, summaryAdds)
			} else {
				fmt.Printf("RESOURCES TO CREATE: %d (details not available)\n\n", summaryAdds)
			}
		}

		if summaryChanges > 0 {
			if useColors {
				fmt.Printf("%sRESOURCES TO UPDATE:%s %d (details not available)\n\n",
					colorBold, colorReset, summaryChanges)
			} else {
				fmt.Printf("RESOURCES TO UPDATE: %d (details not available)\n\n", summaryChanges)
			}
		}

		if summaryDestroys > 0 {
			if useColors {
				fmt.Printf("%sRESOURCES TO DESTROY:%s %d (details not available)\n\n",
					colorBold, colorReset, summaryDestroys)
			} else {
				fmt.Printf("RESOURCES TO DESTROY: %d (details not available)\n\n", summaryDestroys)
			}
		}

		totalChanges := summaryAdds + summaryChanges + summaryDestroys
		if useColors {
			fmt.Printf("%sTOTAL CHANGES:%s %d\n", colorBold, colorReset, totalChanges)
		} else {
			fmt.Printf("TOTAL CHANGES: %d\n", totalChanges)
		}
	} else {
		totalChanges := len(resources["create"]) + len(resources["update"]) + len(resources["destroy"])
		if useColors {
			fmt.Printf("%sTOTAL CHANGES:%s %d\n", colorBold, colorReset, totalChanges)
		} else {
			fmt.Printf("TOTAL CHANGES: %d\n", totalChanges)
		}
	}

	// Add a note if we got the summary from Terraform's summary line but couldn't find resources
	if !resourcesFound && foundSummary && (summaryAdds + summaryChanges + summaryDestroys) > 0 {
		fmt.Println("\nNote: Resource details weren't found in the output.")
		fmt.Println("To see full resource details, try running with:")
		fmt.Println("terraform plan -no-color | terraform-plan-filter")
	}
}