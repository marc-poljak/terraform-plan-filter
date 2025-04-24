package util

import (
	"fmt"

	"github.com/marc-poljak/terraform-plan-filter/internal/model"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

// GetColorForAction returns the ANSI color code for a given action
func GetColorForAction(action model.Action) string {
	switch action {
	case model.ActionCreate:
		return ColorGreen
	case model.ActionUpdate:
		return ColorYellow
	case model.ActionDestroy:
		return ColorRed
	default:
		return ColorReset
	}
}

// GetSymbolForAction returns the symbol for a given action
func GetSymbolForAction(action model.Action) string {
	switch action {
	case model.ActionCreate:
		return "+"
	case model.ActionUpdate:
		return "~"
	case model.ActionDestroy:
		return "-"
	default:
		return "?"
	}
}

// ColorizeText wraps text with color codes if useColors is true
func ColorizeText(text, color string, useColors bool) string {
	if useColors {
		return color + text + ColorReset
	}
	return text
}

// BoldText makes text bold if useColors is true
func BoldText(text string, useColors bool) string {
	return ColorizeText(text, ColorBold, useColors)
}

// PrintDebugInfo prints debug information if verbose mode is enabled
func PrintDebugInfo(resources *model.ResourceCollection, verbose bool) {
	if !verbose {
		return
	}

	fmt.Println("\n=== DEBUG INFO ===")
	fmt.Printf("Found summary: %v\n", resources.FoundSummary)
	fmt.Printf("Has detailed resources: %v\n", resources.HasDetailedResources)

	if resources.FoundSummary {
		fmt.Printf("Summary adds: %d\n", resources.SummaryAdds)
		fmt.Printf("Summary changes: %d\n", resources.SummaryChanges)
		fmt.Printf("Summary destroys: %d\n", resources.SummaryDestroys)
	}

	fmt.Printf("Total changes detected: %d\n", resources.TotalChanges())
	fmt.Println("=================")
}
