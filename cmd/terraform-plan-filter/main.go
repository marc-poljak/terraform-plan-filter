package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/marc-poljak/terraform-plan-filter/internal/formatter"
	"github.com/marc-poljak/terraform-plan-filter/internal/parser"
	"github.com/marc-poljak/terraform-plan-filter/internal/util"
)

func main() {
	var (
		noColor    bool
		jsonOut    bool
		htmlOut    bool
		outputFile string
		verbose    bool
	)

	// Parse command-line flags
	flag.BoolVar(&noColor, "no-color", false, "Disable colored output")
	flag.BoolVar(&jsonOut, "json", false, "Output in JSON format")
	flag.BoolVar(&htmlOut, "html", false, "Output in HTML format")
	flag.StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	flag.BoolVar(&verbose, "verbose", false, "Show verbose output")
	flag.Parse()

	// Force no-color if environment variable is set
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		noColor = true
	}

	// Set up input, default to stdin
	reader := bufio.NewReader(os.Stdin)

	// Parse the Terraform plan
	result, err := parser.ParseTerraformPlan(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing Terraform plan: %v\n", err)
		os.Exit(1)
	}

	// Determine output writer
	var outputWriter *os.File
	if outputFile != "" {
		var err error
		outputWriter, err = os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer outputWriter.Close()
	} else {
		outputWriter = os.Stdout
	}

	// Configure formatter options
	opts := formatter.Options{
		UseColors: !noColor,
		Verbose:   verbose,
	}

	// Format and write output
	var output string
	if jsonOut {
		output, err = formatter.FormatJSON(result)
	} else if htmlOut {
		output, err = formatter.FormatHTML(result)
	} else {
		output, err = formatter.FormatText(result, opts)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}

	writer := bufio.NewWriter(outputWriter)
	writer.WriteString(output)
	writer.Flush()

	// Print summary message if using verbose and details weren't found
	if verbose && !result.HasDetailedResources && result.FoundSummary && result.TotalChanges() > 0 {
		fmt.Fprintln(os.Stderr, "\nNote: Resource details weren't found in the output.")
		fmt.Fprintln(os.Stderr, "To see full resource details, try running with:")
		fmt.Fprintln(os.Stderr, "terraform plan -no-color | terraform-plan-filter")
	}

	util.PrintDebugInfo(result, verbose)
}
