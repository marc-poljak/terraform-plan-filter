package formatter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/marc-poljak/terraform-plan-filter/internal/model"
	"github.com/marc-poljak/terraform-plan-filter/internal/util"
)

// Options configures the output formatter
type Options struct {
	UseColors bool
	Verbose   bool
}

// FormatText formats the resource collection as colored text
func FormatText(resources *model.ResourceCollection, opts Options) (string, error) {
	var sb strings.Builder

	// Format the header
	formatTextHeader(&sb, opts)

	// If we have detailed resources, show them grouped by action then type
	if resources.HasDetailedResources {
		// Display resources for each action type in order
		formatActionResourcesText(&sb, resources, model.ActionCreate, "RESOURCES TO CREATE", opts)
		formatActionResourcesText(&sb, resources, model.ActionUpdate, "RESOURCES TO UPDATE", opts)
		formatActionResourcesText(&sb, resources, model.ActionDestroy, "RESOURCES TO DESTROY", opts)
	} else if resources.FoundSummary {
		// No detailed resources, but we have a summary
		formatSummaryOnlyText(&sb, resources, opts)
	}

	// Format total changes
	formatTotalChangesText(&sb, resources, opts)

	// If we have a summary directly from the plan, show it
	if resources.FoundSummary {
		formatPlanSummaryText(&sb, resources, opts)
	}

	return sb.String(), nil
}

// formatTextHeader adds the header section to the text output
func formatTextHeader(sb *strings.Builder, opts Options) {
	sb.WriteString("\n")
	if opts.UseColors {
		sb.WriteString(util.BoldText("=== TERRAFORM PLAN SUMMARY ===", opts.UseColors))
	} else {
		sb.WriteString("=== TERRAFORM PLAN SUMMARY ===")
	}
	sb.WriteString("\n\n")
}

// formatActionResourcesText formats resources for a specific action type
func formatActionResourcesText(sb *strings.Builder, resources *model.ResourceCollection, action model.Action, actionLabel string, opts Options) {
	actionResources := resources.GetResourcesForAction(action)
	if len(actionResources) == 0 {
		return
	}

	// Get color and symbol for this action
	color := util.GetColorForAction(action)
	symbol := util.GetSymbolForAction(action)

	// Group resources by type
	typeMap := resources.ResourcesByType(action)

	// Get sorted type list
	types := getSortedResourceTypes(typeMap)

	// Print action header
	if opts.UseColors {
		sb.WriteString(util.BoldText(actionLabel+":", opts.UseColors))
	} else {
		sb.WriteString(actionLabel + ":")
	}
	sb.WriteString("\n")

	// Handle module resources first if they exist
	if hasModuleResources(typeMap) {
		formatModuleResourcesText(sb, typeMap, color, symbol, opts)
		// Filter out "module" from types since we've already processed it
		types = filterOutModuleType(types)
	}

	// Format resources by type (excluding modules which we've already handled)
	formatResourcesByTypeText(sb, types, typeMap, color, symbol, opts)
}

// getSortedResourceTypes returns a sorted slice of resource types
func getSortedResourceTypes(typeMap map[string][]string) []string {
	types := make([]string, 0, len(typeMap))
	for t := range typeMap {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// hasModuleResources checks if there are module resources in the type map
func hasModuleResources(typeMap map[string][]string) bool {
	moduleResources, ok := typeMap["module"]
	return ok && len(moduleResources) > 0
}

// filterOutModuleType removes "module" from the types slice
func filterOutModuleType(types []string) []string {
	var filteredTypes []string
	for _, t := range types {
		if t != "module" {
			filteredTypes = append(filteredTypes, t)
		}
	}
	return filteredTypes
}

// formatModuleResourcesText formats module resources
func formatModuleResourcesText(sb *strings.Builder, typeMap map[string][]string, color, symbol string, opts Options) {
	moduleResources := typeMap["module"]

	// Print type subheader
	if opts.UseColors {
		fmt.Fprintf(sb, "  %s# MODULE RESOURCES:%s\n",
			util.ColorBold, util.ColorReset)
	} else {
		sb.WriteString("  # MODULE RESOURCES:\n")
	}

	// Print module resources
	for _, resource := range moduleResources {
		if opts.UseColors {
			fmt.Fprintf(sb, "    %s%s %s%s\n",
				color, symbol, resource, util.ColorReset)
		} else {
			fmt.Fprintf(sb, "    %s %s\n", symbol, resource)
		}
	}
	sb.WriteString("\n")
}

// formatResourcesByTypeText formats resources grouped by type
func formatResourcesByTypeText(sb *strings.Builder, types []string, typeMap map[string][]string, color, symbol string, opts Options) {
	for _, resourceType := range types {
		resources := typeMap[resourceType]

		// Skip empty resource types
		if len(resources) == 0 {
			continue
		}

		// Print type subheader
		if opts.UseColors {
			fmt.Fprintf(sb, "  %s# %s RESOURCES:%s\n",
				util.ColorBold, strings.ToUpper(resourceType), util.ColorReset)
		} else {
			fmt.Fprintf(sb, "  # %s RESOURCES:\n", strings.ToUpper(resourceType))
		}

		// Print resources of this type
		for _, resource := range resources {
			if opts.UseColors {
				fmt.Fprintf(sb, "    %s%s %s%s\n",
					color, symbol, resource, util.ColorReset)
			} else {
				fmt.Fprintf(sb, "    %s %s\n", symbol, resource)
			}
		}
		sb.WriteString("\n")
	}
}

// formatSummaryOnlyText formats the summary when no detailed resources are available
func formatSummaryOnlyText(sb *strings.Builder, resources *model.ResourceCollection, opts Options) {
	if resources.SummaryAdds > 0 {
		if opts.UseColors {
			fmt.Fprintf(sb, "%sRESOURCES TO CREATE:%s %d (details not available)\n\n",
				util.ColorBold, util.ColorReset, resources.SummaryAdds)
		} else {
			fmt.Fprintf(sb, "RESOURCES TO CREATE: %d (details not available)\n\n",
				resources.SummaryAdds)
		}
	}

	if resources.SummaryChanges > 0 {
		if opts.UseColors {
			fmt.Fprintf(sb, "%sRESOURCES TO UPDATE:%s %d (details not available)\n\n",
				util.ColorBold, util.ColorReset, resources.SummaryChanges)
		} else {
			fmt.Fprintf(sb, "RESOURCES TO UPDATE: %d (details not available)\n\n",
				resources.SummaryChanges)
		}
	}

	if resources.SummaryDestroys > 0 {
		if opts.UseColors {
			fmt.Fprintf(sb, "%sRESOURCES TO DESTROY:%s %d (details not available)\n\n",
				util.ColorBold, util.ColorReset, resources.SummaryDestroys)
		} else {
			fmt.Fprintf(sb, "RESOURCES TO DESTROY: %d (details not available)\n\n",
				resources.SummaryDestroys)
		}
	}
}

// formatTotalChangesText formats the total number of changes
func formatTotalChangesText(sb *strings.Builder, resources *model.ResourceCollection, opts Options) {
	totalChanges := resources.TotalChanges()
	if opts.UseColors {
		fmt.Fprintf(sb, "%sTOTAL CHANGES:%s %d\n",
			util.ColorBold, util.ColorReset, totalChanges)
	} else {
		fmt.Fprintf(sb, "TOTAL CHANGES: %d\n", totalChanges)
	}
}

// formatPlanSummaryText formats the plan summary line
func formatPlanSummaryText(sb *strings.Builder, resources *model.ResourceCollection, opts Options) {
	planSummary := fmt.Sprintf("Plan: %d to add, %d to change, %d to destroy.",
		resources.SummaryAdds, resources.SummaryChanges, resources.SummaryDestroys)

	if opts.UseColors {
		fmt.Fprintf(sb, "\n%sPlan Summary:%s %s\n",
			util.ColorBold, util.ColorReset, planSummary)
	} else {
		fmt.Fprintf(sb, "\nPlan Summary: %s\n", planSummary)
	}
}

// FormatJSON formats the resource collection as JSON
func FormatJSON(resources *model.ResourceCollection) (string, error) {
	type jsonOutput struct {
		Create  []string `json:"create"`
		Update  []string `json:"update"`
		Destroy []string `json:"destroy"`
		Summary struct {
			Total    int `json:"total"`
			Adds     int `json:"adds"`
			Changes  int `json:"changes"`
			Destroys int `json:"destroys"`
		} `json:"summary"`
		HasDetailedResources bool      `json:"has_detailed_resources"`
		FoundSummary         bool      `json:"found_summary"`
		Timestamp            time.Time `json:"timestamp"`
	}

	output := jsonOutput{
		Create:               resources.GetResourcesForAction(model.ActionCreate),
		Update:               resources.GetResourcesForAction(model.ActionUpdate),
		Destroy:              resources.GetResourcesForAction(model.ActionDestroy),
		HasDetailedResources: resources.HasDetailedResources,
		FoundSummary:         resources.FoundSummary,
		Timestamp:            time.Now(),
	}

	output.Summary.Adds = resources.SummaryAdds
	output.Summary.Changes = resources.SummaryChanges
	output.Summary.Destroys = resources.SummaryDestroys
	output.Summary.Total = resources.TotalChanges()

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// FormatHTML formats the resource collection as HTML
func FormatHTML(resources *model.ResourceCollection) (string, error) {
	var sb strings.Builder

	// Write HTML header and styles
	writeHTMLHeader(&sb)

	// Write the main content
	sb.WriteString("    <h1>Terraform Plan Summary</h1>\n")
	sb.WriteString("    <div class=\"summary\">\n")
	sb.WriteString(fmt.Sprintf("        <p><strong>Total changes:</strong> %d</p>\n", resources.TotalChanges()))

	// If we have detailed resources
	if resources.HasDetailedResources {
		// Render sections for create, update, destroy actions
		renderHTMLActionSection(&sb, resources, model.ActionCreate, "Create", "create")
		renderHTMLActionSection(&sb, resources, model.ActionUpdate, "Update", "update")
		renderHTMLActionSection(&sb, resources, model.ActionDestroy, "Destroy", "destroy")
	} else if resources.FoundSummary {
		// No detailed resources, but we have a summary
		writeHTMLSummaryOnly(&sb, resources)
	}

	// Write plan summary if available
	if resources.FoundSummary {
		writeHTMLPlanSummary(&sb, resources)
	}

	// Add timestamp and close HTML
	writeHTMLFooter(&sb)

	return sb.String(), nil
}

// writeHTMLHeader writes the HTML header and styles
func writeHTMLHeader(sb *strings.Builder) {
	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Terraform Plan Summary</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            line-height: 1.6;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            color: #333;
        }
        h1 {
            color: #0f4c81;
            border-bottom: 2px solid #0f4c81;
            padding-bottom: 10px;
        }
        .summary {
            background: #f5f5f5;
            padding: 15px;
            border-radius: 4px;
            margin-bottom: 20px;
        }
        .action-group {
            margin-bottom: 30px;
        }
        .create h2 {
            color: #2a9d8f;
        }
        .update h2 {
            color: #e9c46a;
        }
        .destroy h2 {
            color: #e76f51;
        }
        .resource-type {
            background: #f9f9f9;
            padding: 8px 12px;
            margin-bottom: 10px;
            border-radius: 4px;
            font-weight: bold;
        }
        .resource {
            background: white;
            border-left: 4px solid #ddd;
            padding: 10px 15px;
            margin-bottom: 10px;
            border-radius: 0 4px 4px 0;
        }
        .create .resource {
            border-left-color: #2a9d8f;
        }
        .update .resource {
            border-left-color: #e9c46a;
        }
        .destroy .resource {
            border-left-color: #e76f51;
        }
        .timestamp {
            font-size: 0.8em;
            color: #666;
            margin-top: 30px;
        }
        .plan-summary {
            margin-top: 20px;
            font-weight: bold;
            padding: 10px;
            background-color: #f0f8ff;
            border-radius: 4px;
        }
    </style>
</head>
<body>
`)
}

// renderHTMLActionSection renders an HTML section for a specific action
func renderHTMLActionSection(sb *strings.Builder, resources *model.ResourceCollection, action model.Action, actionName, colorClass string) {
	actionResources := resources.GetResourcesForAction(action)
	if len(actionResources) == 0 {
		return
	}

	fmt.Fprintf(sb, "    <div class=\"action-group %s\">\n", colorClass)
	fmt.Fprintf(sb, "        <h2>Resources to %s</h2>\n", strings.ToLower(actionName))

	// Group by resource type
	typeMap := resources.ResourcesByType(action)

	// Sort types
	types := getSortedResourceTypes(typeMap)

	// Special handling for module resources
	if hasModuleResources(typeMap) {
		writeHTMLModuleResources(sb, typeMap)
		types = filterOutModuleType(types)
	}

	// Write resources by type
	writeHTMLResourcesByType(sb, types, typeMap)

	sb.WriteString("    </div>\n")
}

// writeHTMLModuleResources writes HTML for module resources
func writeHTMLModuleResources(sb *strings.Builder, typeMap map[string][]string) {
	moduleResources := typeMap["module"]
	sb.WriteString("        <div class=\"resource-type\">MODULE RESOURCES</div>\n")

	for _, resource := range moduleResources {
		fmt.Fprintf(sb, "        <div class=\"resource\">%s</div>\n", resource)
	}
}

// writeHTMLResourcesByType writes HTML for resources grouped by type
func writeHTMLResourcesByType(sb *strings.Builder, types []string, typeMap map[string][]string) {
	for _, resourceType := range types {
		resources := typeMap[resourceType]

		// Skip empty resource types
		if len(resources) == 0 {
			continue
		}

		fmt.Fprintf(sb, "        <div class=\"resource-type\">%s</div>\n",
			strings.ToUpper(resourceType))

		for _, resource := range resources {
			fmt.Fprintf(sb, "        <div class=\"resource\">%s</div>\n", resource)
		}
	}
}

// writeHTMLSummaryOnly writes HTML for when only summary information is available
func writeHTMLSummaryOnly(sb *strings.Builder, resources *model.ResourceCollection) {
	sb.WriteString("    <div class=\"summary-details\">\n")

	if resources.SummaryAdds > 0 {
		fmt.Fprintf(sb, "        <p><strong>Resources to create:</strong> %d (details not available)</p>\n",
			resources.SummaryAdds)
	}

	if resources.SummaryChanges > 0 {
		fmt.Fprintf(sb, "        <p><strong>Resources to update:</strong> %d (details not available)</p>\n",
			resources.SummaryChanges)
	}

	if resources.SummaryDestroys > 0 {
		fmt.Fprintf(sb, "        <p><strong>Resources to destroy:</strong> %d (details not available)</p>\n",
			resources.SummaryDestroys)
	}

	sb.WriteString("    </div>\n")
}

// writeHTMLPlanSummary writes the HTML plan summary line
func writeHTMLPlanSummary(sb *strings.Builder, resources *model.ResourceCollection) {
	planSummary := fmt.Sprintf("Plan: %d to add, %d to change, %d to destroy.",
		resources.SummaryAdds, resources.SummaryChanges, resources.SummaryDestroys)

	fmt.Fprintf(sb, "    <div class=\"plan-summary\">%s</div>\n", planSummary)
}

// writeHTMLFooter writes the timestamp and closing HTML tags
func writeHTMLFooter(sb *strings.Builder) {
	currentTime := time.Now().Format("January 2, 2006 15:04:05")
	fmt.Fprintf(sb, "    <div class=\"timestamp\">Report generated on %s</div>\n", currentTime)
	sb.WriteString("</body>\n</html>")
}
