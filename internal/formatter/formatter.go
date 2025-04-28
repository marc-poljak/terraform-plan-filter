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

	// Header
	sb.WriteString("\n")
	if opts.UseColors {
		sb.WriteString(util.BoldText("=== TERRAFORM PLAN SUMMARY ===", opts.UseColors))
	} else {
		sb.WriteString("=== TERRAFORM PLAN SUMMARY ===")
	}
	sb.WriteString("\n\n")

	// If we have detailed resources, show them grouped by action then type
	if resources.HasDetailedResources {
		// Display resources for each action in order: create, update, destroy
		displayActionResources := func(action model.Action, actionLabel string) {
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
			types := make([]string, 0, len(typeMap))
			for t := range typeMap {
				types = append(types, t)
			}
			sort.Strings(types)

			// Print action header
			if opts.UseColors {
				sb.WriteString(util.BoldText(actionLabel+":", opts.UseColors))
			} else {
				sb.WriteString(actionLabel + ":")
			}
			sb.WriteString("\n")

			// First print module resources if they exist
			if moduleResources, ok := typeMap["module"]; ok && len(moduleResources) > 0 {
				// Print type subheader
				if opts.UseColors {
					sb.WriteString(fmt.Sprintf("  %s# MODULE RESOURCES:%s\n",
						util.ColorBold, util.ColorReset))
				} else {
					sb.WriteString("  # MODULE RESOURCES:\n")
				}

				// Print module resources
				for _, resource := range moduleResources {
					if opts.UseColors {
						sb.WriteString(fmt.Sprintf("    %s%s %s%s\n",
							color, symbol, resource, util.ColorReset))
					} else {
						sb.WriteString(fmt.Sprintf("    %s %s\n", symbol, resource))
					}
				}
				sb.WriteString("\n")

				// Remove "module" from the types list since we've already processed it
				var filteredTypes []string
				for _, t := range types {
					if t != "module" {
						filteredTypes = append(filteredTypes, t)
					}
				}
				types = filteredTypes
			}

			// Print resources grouped by type (excluding modules which we've already handled)
			for _, resourceType := range types {
				resources := typeMap[resourceType]

				// Skip empty resource types
				if len(resources) == 0 {
					continue
				}

				// Print type subheader
				if opts.UseColors {
					sb.WriteString(fmt.Sprintf("  %s# %s RESOURCES:%s\n",
						util.ColorBold, strings.ToUpper(resourceType), util.ColorReset))
				} else {
					sb.WriteString(fmt.Sprintf("  # %s RESOURCES:\n", strings.ToUpper(resourceType)))
				}

				// Print resources of this type
				for _, resource := range resources {
					if opts.UseColors {
						sb.WriteString(fmt.Sprintf("    %s%s %s%s\n",
							color, symbol, resource, util.ColorReset))
					} else {
						sb.WriteString(fmt.Sprintf("    %s %s\n", symbol, resource))
					}
				}
				sb.WriteString("\n")
			}
		}

		// Display in order: create, update, destroy
		displayActionResources(model.ActionCreate, "RESOURCES TO CREATE")
		displayActionResources(model.ActionUpdate, "RESOURCES TO UPDATE")
		displayActionResources(model.ActionDestroy, "RESOURCES TO DESTROY")
	} else if resources.FoundSummary {
		// No detailed resources, but we have a summary
		if resources.SummaryAdds > 0 {
			if opts.UseColors {
				sb.WriteString(fmt.Sprintf("%sRESOURCES TO CREATE:%s %d (details not available)\n\n",
					util.ColorBold, util.ColorReset, resources.SummaryAdds))
			} else {
				sb.WriteString(fmt.Sprintf("RESOURCES TO CREATE: %d (details not available)\n\n",
					resources.SummaryAdds))
			}
		}

		if resources.SummaryChanges > 0 {
			if opts.UseColors {
				sb.WriteString(fmt.Sprintf("%sRESOURCES TO UPDATE:%s %d (details not available)\n\n",
					util.ColorBold, util.ColorReset, resources.SummaryChanges))
			} else {
				sb.WriteString(fmt.Sprintf("RESOURCES TO UPDATE: %d (details not available)\n\n",
					resources.SummaryChanges))
			}
		}

		if resources.SummaryDestroys > 0 {
			if opts.UseColors {
				sb.WriteString(fmt.Sprintf("%sRESOURCES TO DESTROY:%s %d (details not available)\n\n",
					util.ColorBold, util.ColorReset, resources.SummaryDestroys))
			} else {
				sb.WriteString(fmt.Sprintf("RESOURCES TO DESTROY: %d (details not available)\n\n",
					resources.SummaryDestroys))
			}
		}
	}

	// Total changes
	totalChanges := resources.TotalChanges()
	if opts.UseColors {
		sb.WriteString(fmt.Sprintf("%sTOTAL CHANGES:%s %d\n",
			util.ColorBold, util.ColorReset, totalChanges))
	} else {
		sb.WriteString(fmt.Sprintf("TOTAL CHANGES: %d\n", totalChanges))
	}

	// If we have a summary directly from the plan, show it
	if resources.FoundSummary {
		planSummary := fmt.Sprintf("Plan: %d to add, %d to change, %d to destroy.",
			resources.SummaryAdds, resources.SummaryChanges, resources.SummaryDestroys)

		if opts.UseColors {
			sb.WriteString(fmt.Sprintf("\n%sPlan Summary:%s %s\n",
				util.ColorBold, util.ColorReset, planSummary))
		} else {
			sb.WriteString(fmt.Sprintf("\nPlan Summary: %s\n", planSummary))
		}
	}

	return sb.String(), nil
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
    <h1>Terraform Plan Summary</h1>
    <div class="summary">
        <p><strong>Total changes:</strong> `)

	sb.WriteString(fmt.Sprintf("%d</p>\n", resources.TotalChanges()))

	// If we have detailed resources
	if resources.HasDetailedResources {
		// Function to render a section for an action
		renderActionSection := func(action model.Action, actionName, colorClass string) {
			actionResources := resources.GetResourcesForAction(action)
			if len(actionResources) == 0 {
				return
			}

			sb.WriteString(fmt.Sprintf("    <div class=\"action-group %s\">\n", colorClass))
			sb.WriteString(fmt.Sprintf("        <h2>Resources to %s</h2>\n", strings.ToLower(actionName)))

			// Group by resource type
			typeMap := resources.ResourcesByType(action)

			// Sort types
			types := make([]string, 0, len(typeMap))
			for t := range typeMap {
				types = append(types, t)
			}
			sort.Strings(types)

			// Special handling for module resources
			if moduleResources, ok := typeMap["module"]; ok && len(moduleResources) > 0 {
				sb.WriteString("        <div class=\"resource-type\">MODULE RESOURCES</div>\n")

				for _, resource := range moduleResources {
					sb.WriteString(fmt.Sprintf("        <div class=\"resource\">%s</div>\n", resource))
				}

				// Remove module from the types list
				var filteredTypes []string
				for _, t := range types {
					if t != "module" {
						filteredTypes = append(filteredTypes, t)
					}
				}
				types = filteredTypes
			}

			// Render each type
			for _, resourceType := range types {
				resources := typeMap[resourceType]

				// Skip empty resource types
				if len(resources) == 0 {
					continue
				}

				sb.WriteString(fmt.Sprintf("        <div class=\"resource-type\">%s</div>\n",
					strings.ToUpper(resourceType)))

				for _, resource := range resources {
					sb.WriteString(fmt.Sprintf("        <div class=\"resource\">%s</div>\n", resource))
				}
			}

			sb.WriteString("    </div>\n")
		}

		renderActionSection(model.ActionCreate, "Create", "create")
		renderActionSection(model.ActionUpdate, "Update", "update")
		renderActionSection(model.ActionDestroy, "Destroy", "destroy")
	} else if resources.FoundSummary {
		// No detailed resources, but we have a summary
		sb.WriteString("    <div class=\"summary-details\">\n")

		if resources.SummaryAdds > 0 {
			sb.WriteString(fmt.Sprintf("        <p><strong>Resources to create:</strong> %d (details not available)</p>\n",
				resources.SummaryAdds))
		}

		if resources.SummaryChanges > 0 {
			sb.WriteString(fmt.Sprintf("        <p><strong>Resources to update:</strong> %d (details not available)</p>\n",
				resources.SummaryChanges))
		}

		if resources.SummaryDestroys > 0 {
			sb.WriteString(fmt.Sprintf("        <p><strong>Resources to destroy:</strong> %d (details not available)</p>\n",
				resources.SummaryDestroys))
		}

		sb.WriteString("    </div>\n")
	}

	// Plan summary
	if resources.FoundSummary {
		planSummary := fmt.Sprintf("Plan: %d to add, %d to change, %d to destroy.",
			resources.SummaryAdds, resources.SummaryChanges, resources.SummaryDestroys)

		sb.WriteString(fmt.Sprintf("    <div class=\"plan-summary\">%s</div>\n", planSummary))
	}

	// Timestamp
	currentTime := time.Now().Format("January 2, 2006 15:04:05")
	sb.WriteString(fmt.Sprintf("    <div class=\"timestamp\">Report generated on %s</div>\n", currentTime))

	sb.WriteString("</body>\n</html>")

	return sb.String(), nil
}
