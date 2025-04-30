package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/marc-poljak/terraform-plan-filter/internal/formatter"
	"github.com/marc-poljak/terraform-plan-filter/internal/parser"
	"github.com/marc-poljak/terraform-plan-filter/internal/util"
)

func main() {
	var (
		noColor    bool
		jsonOut    bool
		htmlOut    bool
		planFile   string
		outputFile string
		verbose    bool
	)

	// Parse command-line flags
	flag.BoolVar(&noColor, "no-color", false, "Disable colored output")
	flag.BoolVar(&jsonOut, "json", false, "Output in JSON format")
	flag.BoolVar(&htmlOut, "html", false, "Output in HTML format")
	flag.StringVar(&planFile, "plan", "", "Terraform JSON plan file (default: stdin)")
	flag.StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	flag.BoolVar(&verbose, "verbose", false, "Show verbose output")
	flag.Parse()

	// Force no-color if environment variable is set
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		noColor = true
	}

	// Set up input source
	var err error
	var inputFile *os.File

	if planFile != "" {
		// Check if the file exists
		if _, err := os.Stat(planFile); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: Plan file %s does not exist\n", planFile)
			os.Exit(1)
		}

		// Open the specified plan file
		inputFile, err = os.Open(planFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening plan file: %v\n", err)
			os.Exit(1)
		}
		// Properly handle Close() error with anonymous function
		defer func() {
			if closeErr := inputFile.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Error closing input file: %v\n", closeErr)
			}
		}()
	} else {
		// Use stdin
		inputFile = os.Stdin
	}

	// Parse the Terraform plan
	result, err := parser.ParseTerraformPlan(inputFile)
	if err != nil {
		if strings.Contains(err.Error(), "input appears to be text format") {
			fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
			fmt.Fprintf(os.Stderr, "This tool now only supports JSON-formatted Terraform plans.\n")
			fmt.Fprintf(os.Stderr, "Please use the following commands:\n")
			fmt.Fprintf(os.Stderr, "  terraform plan -out=tfplan\n")
			fmt.Fprintf(os.Stderr, "  terraform show -json tfplan | terraform-plan-filter\n")
			os.Exit(1)
		} else {
			fmt.Fprintf(os.Stderr, "Error parsing Terraform plan: %v\n", err)
			os.Exit(1)
		}
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
		// Properly handle Close() error with anonymous function
		defer func() {
			if closeErr := outputWriter.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Error closing output file: %v\n", closeErr)
			}
		}()
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
	if _, err := writer.WriteString(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	if err := writer.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "Error flushing writer: %v\n", err)
		os.Exit(1)
	}

	// Print debug information if verbose
	util.PrintDebugInfo(result, verbose)
}
