package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/marc-poljak/terraform-plan-filter/internal/formatter"
	"github.com/marc-poljak/terraform-plan-filter/internal/model"
	"github.com/marc-poljak/terraform-plan-filter/internal/parser"
	"github.com/marc-poljak/terraform-plan-filter/internal/util"
)

// version is set during build using -ldflags
var version = "dev"

func main() {
	// Parse command-line flags and set up configuration
	config := parseCommandLineFlags()

	// Process input file
	inputFile, err := setupInputSource(config.planFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if inputFile != os.Stdin {
		defer safeClose(inputFile, "input file")
	}

	// Parse the Terraform plan
	result, err := parseTerraformPlan(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Set up output file
	outputWriter, err := setupOutputDestination(config.outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if outputWriter != os.Stdout {
		defer safeClose(outputWriter, "output file")
	}

	// Generate and write output
	if err := generateAndWriteOutput(result, outputWriter, config); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Print debug information if verbose
	if config.verbose {
		util.PrintDebugInfo(result, config.verbose)
	}
}

// Config holds the command-line configuration options
type Config struct {
	noColor    bool
	jsonOut    bool
	htmlOut    bool
	planFile   string
	outputFile string
	verbose    bool
}

// parseCommandLineFlags parses command-line flags and returns a Config
func parseCommandLineFlags() Config {
	var config Config
	var showVersion bool

	flag.BoolVar(&config.noColor, "no-color", false, "Disable colored output")
	flag.BoolVar(&config.jsonOut, "json", false, "Output in JSON format")
	flag.BoolVar(&config.htmlOut, "html", false, "Output in HTML format")
	flag.StringVar(&config.planFile, "plan", "", "Terraform JSON plan file (default: stdin)")
	flag.StringVar(&config.outputFile, "output", "", "Output file (default: stdout)")
	flag.BoolVar(&config.verbose, "verbose", false, "Show verbose output")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Parse()

	// Show version if requested
	if showVersion {
		fmt.Printf("terraform-plan-filter version %s\n", version)
		os.Exit(0)
	}

	// Force no-color if environment variable is set
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		config.noColor = true
	}

	return config
}

// setupInputSource sets up the input source based on configuration
func setupInputSource(planFile string) (*os.File, error) {
	if planFile == "" {
		return os.Stdin, nil
	}

	// Check if the file exists
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("plan file %s does not exist", planFile)
	}

	// Open the specified plan file
	inputFile, err := os.Open(planFile)
	if err != nil {
		return nil, fmt.Errorf("error opening plan file: %v", err)
	}

	return inputFile, nil
}

// parseTerraformPlan parses the Terraform plan and handles errors
func parseTerraformPlan(inputFile *os.File) (*model.ResourceCollection, error) {
	result, err := parser.ParseTerraformPlan(inputFile)
	if err != nil {
		if strings.Contains(err.Error(), "input appears to be text format") {
			return nil, fmt.Errorf("error: %v\n\nThis tool now only supports JSON-formatted Terraform plans.\nPlease use the following commands:\n  terraform plan -out=tfplan\n  terraform show -json tfplan | terraform-plan-filter", err)
		}
		return nil, fmt.Errorf("error parsing Terraform plan: %v", err)
	}
	return result, nil
}

// setupOutputDestination sets up the output destination based on configuration
func setupOutputDestination(outputFile string) (*os.File, error) {
	if outputFile == "" {
		return os.Stdout, nil
	}

	outputWriter, err := os.Create(outputFile)
	if err != nil {
		return nil, fmt.Errorf("error creating output file: %v", err)
	}

	return outputWriter, nil
}

// generateAndWriteOutput generates the formatted output and writes it
func generateAndWriteOutput(result *model.ResourceCollection, outputWriter *os.File, config Config) error {
	// Configure formatter options
	opts := formatter.Options{
		UseColors: !config.noColor,
		Verbose:   config.verbose,
	}

	// Format output based on requested format
	var output string
	var err error

	if config.jsonOut {
		output, err = formatter.FormatJSON(result)
	} else if config.htmlOut {
		output, err = formatter.FormatHTML(result)
	} else {
		output, err = formatter.FormatText(result, opts)
	}

	if err != nil {
		return fmt.Errorf("error formatting output: %v", err)
	}

	// Write the output
	writer := bufio.NewWriter(outputWriter)
	if _, err := writer.WriteString(output); err != nil {
		return fmt.Errorf("error writing output: %v", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("error flushing writer: %v", err)
	}

	return nil
}

// safeClose safely closes a file and logs any error
func safeClose(file *os.File, description string) {
	if err := file.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error closing %s: %v\n", description, err)
	}
}
