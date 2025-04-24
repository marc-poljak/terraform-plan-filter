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
	colorBold   = "\033[1m"
)

func main() {
	useColors := !(os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb")

	// Prepare regexes
	var patterns = []struct {
		re     *regexp.Regexp
		action string
		double bool // if true, this action applies to both create and destroy (e.g., replace)
	}{
		{regexp.MustCompile(`^\s*\+\s(.+?)\s+{`), "create", false},
		{regexp.MustCompile(`^\s*~\s(.+?)\s+{`), "update", false},
		{regexp.MustCompile(`^\s*-\s(.+?)\s+{`), "destroy", false},
		{regexp.MustCompile(`^(.+?)\s+will\s+be\s+created`), "create", false},
		{regexp.MustCompile(`^(.+?)\s+will\s+be\s+updated\s+in-place`), "update", false},
		{regexp.MustCompile(`^(.+?)\s+will\s+be\s+destroyed`), "destroy", false},
		{regexp.MustCompile(`^(.+?)\s+must\s+be\s+replaced`), "replace", true},
	}

	planSummaryRegex := regexp.MustCompile(`Plan:\s+(\d+)\s+to\s+add,\s+(\d+)\s+to\s+change,\s+(\d+)\s+to\s+destroy`)

	resources := map[string]map[string]struct{}{
		"create":  {},
		"update":  {},
		"destroy": {},
	}

	var summaryAdds, summaryChanges, summaryDestroys int
	foundSummary := false

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()

		// Check all resource-related patterns
		for _, pat := range patterns {
			if matches := pat.re.FindStringSubmatch(line); len(matches) > 1 {
				if pat.double {
					resources["create"][matches[1]] = struct{}{}
					resources["destroy"][matches[1]] = struct{}{}
				} else {
					resources[pat.action][matches[1]] = struct{}{}
				}
				break
			}
		}

		// Check for plan summary
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

	// Buffered writer for faster output
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	write := func(s string) { writer.WriteString(s + "\n") }

	write("")
	if useColors {
		write(colorBold + "=== TERRAFORM PLAN SUMMARY ===" + colorReset)
	} else {
		write("=== TERRAFORM PLAN SUMMARY ===")
	}
	write("")

	// Display grouped resources
	displayResources := func(action, label, color, symbol string) {
		if len(resources[action]) == 0 {
			return
		}

		typeMap := make(map[string][]string)
		for res := range resources[action] {
			rtype := extractResourceType(res)
			typeMap[rtype] = append(typeMap[rtype], res)
		}

		if useColors {
			write(fmt.Sprintf("%s%s:%s", colorBold, label, colorReset))
		} else {
			write(label + ":")
		}

		var types []string
		for t := range typeMap {
			types = append(types, t)
		}
		sort.Strings(types)

		for _, t := range types {
			sort.Strings(typeMap[t])
			if useColors {
				write(fmt.Sprintf("  %s# %s RESOURCES:%s", colorBold, strings.ToUpper(t), colorReset))
			} else {
				write(fmt.Sprintf("  # %s RESOURCES:", strings.ToUpper(t)))
			}
			for _, r := range typeMap[t] {
				if useColors {
					write(fmt.Sprintf("    %s%s %s%s", color, symbol, r, colorReset))
				} else {
					write(fmt.Sprintf("    %s %s", symbol, r))
				}
			}
			write("")
		}
	}

	totalChanges := 0
	hasDetailed := false

	for _, k := range []string{"create", "update", "destroy"} {
		totalChanges += len(resources[k])
	}

	if totalChanges > 0 {
		hasDetailed = true
		displayResources("create", "RESOURCES TO CREATE", colorGreen, "+")
		displayResources("update", "RESOURCES TO UPDATE", colorYellow, "~")
		displayResources("destroy", "RESOURCES TO DESTROY", colorRed, "-")
	}

	if !hasDetailed && foundSummary {
		if summaryAdds > 0 {
			write(fmt.Sprintf("RESOURCES TO CREATE: %d (details not available)", summaryAdds))
		}
		if summaryChanges > 0 {
			write(fmt.Sprintf("RESOURCES TO UPDATE: %d (details not available)", summaryChanges))
		}
		if summaryDestroys > 0 {
			write(fmt.Sprintf("RESOURCES TO DESTROY: %d (details not available)", summaryDestroys))
		}
		totalChanges = summaryAdds + summaryChanges + summaryDestroys
	}

	if useColors {
		write(fmt.Sprintf("%sTOTAL CHANGES:%s %d", colorBold, colorReset, totalChanges))
	} else {
		write(fmt.Sprintf("TOTAL CHANGES: %d", totalChanges))
	}

	if !hasDetailed && foundSummary && totalChanges > 0 {
		write("\nNote: Resource details weren't found in the output.")
		write("To see full resource details, try running with:")
		write("terraform plan -no-color | terraform-plan-filter")
	}
}

// Extract the resource type from Terraform resource path
func extractResourceType(resource string) string {
	parts := strings.Split(resource, ".")
	if len(parts) > 2 && parts[0] == "module" {
		for i, part := range parts {
			if i > 1 && (strings.HasPrefix(part, "aws_") || strings.HasPrefix(part, "azurerm_") ||
				strings.HasPrefix(part, "google_") || strings.HasPrefix(part, "kubernetes_") ||
				strings.HasPrefix(part, "digitalocean_")) {
				return part
			}
		}
	}
	if len(parts) >= 2 {
		return parts[0]
	}
	return resource
}
