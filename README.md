# 🚀 Terraform Plan Filter

[![Go Report Card](https://goreportcard.com/badge/github.com/marc-poljak/terraform-plan-filter)](https://goreportcard.com/report/github.com/marc-poljak/terraform-plan-filter)
[![License](https://img.shields.io/github/license/marc-poljak/terraform-plan-filter)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/marc-poljak/terraform-plan-filter)](go.mod)

A lightweight CLI tool that streamlines Terraform plan output by filtering and displaying only the resource titles that will be created, updated, or destroyed, without all the verbose details.

## 📋 Overview

When working with large Terraform projects, the plan output can be overwhelming with detailed attribute changes. This tool provides a clean, concise summary showing only the resource titles organized first by action type (create, update, destroy) and then by resource type, making it easier to quickly review what will change.

## ✨ Features

- 🔍 Filters Terraform plan output to show only resource titles
- 🎯 Categorizes resources first by action type (create, update, destroy)
- 🎨 Groups resources by resource type (aws_s3_bucket, aws_instance, etc.)
- 🌈 Colorized output (green for creations, yellow for updates, red for deletions)
- 📊 Provides a total count of changes
- 📱 Multiple output formats (text, JSON, HTML)
- 🧰 Simple to use with Terraform JSON plan output

## ⚠️ Disclaimer

**USE AT YOUR OWN RISK**. This tool is provided "as is", without warranty of any kind, express or implied. Neither the authors nor contributors shall be liable for any damages or consequences arising from the use of this tool. Always:

- 🧪 Test in a non-production environment first
- ✓ Verify results manually before taking action
- 💾 Maintain proper backups
- 🔒 Follow your organization's security policies

## 🛠️ Installation

### Build from source

```bash
# Clone the repository
git clone https://github.com/marc-poljak/terraform-plan-filter.git
cd terraform-plan-filter

# Build the project
make build

# Install to your GOPATH/bin
make install
```

## 🚀 Usage

This tool processes Terraform plans in JSON format. Here's how to use it:

### Basic Usage

```bash
# Step 1: Create a plan file
terraform plan -out=tfplan

# Step 2: Convert the plan to JSON and pipe to terraform-plan-filter
terraform show -json tfplan | terraform-plan-filter
```

You can also save the JSON to a file and process it:

```bash
# Save the JSON plan to a file
terraform show -json tfplan > tfplan.json

# Process the JSON plan file
terraform-plan-filter --plan tfplan.json
```

### Using with Terraform Variable Files (tfvars)

When using variable files with your Terraform plans:

```bash
# Using with a tfvars file
terraform plan -var-file=environments/prod.tfvars -out=tfplan
terraform show -json tfplan | terraform-plan-filter
```

#### Multiple Variable Files

```bash
# Using multiple variable files
terraform plan -var-file=environments/prod.tfvars -var-file=overrides.tfvars -out=tfplan
terraform show -json tfplan | terraform-plan-filter
```

#### Saving JSON Output to a File

```bash
# Create a plan with variable files and save JSON output
terraform plan -var-file=environments/prod.tfvars -out=tfplan
terraform show -json tfplan > tfplan.json

# Process the saved JSON file
terraform-plan-filter --plan tfplan.json
```

### Command Line Flags

```
Usage: terraform show -json tfplan | terraform-plan-filter [options]

Options:
  -no-color        Disable colored output
  -json            Output in JSON format
  -html            Output in HTML format
  -plan string     Terraform JSON plan file (default: stdin)
  -output string   Output file (default: stdout)
  -verbose         Show verbose output
```

### Additional Examples

Generate JSON output to a file:
```bash
terraform show -json tfplan | terraform-plan-filter --json --output plan.json
```

Generate HTML report:
```bash
terraform show -json tfplan | terraform-plan-filter --html --output plan.html
```

Process a saved JSON plan file and output as HTML:
```bash
terraform-plan-filter --plan tfplan.json --output plan.html --html
```

### Example Output

```
=== TERRAFORM PLAN SUMMARY ===

RESOURCES TO CREATE:
  # AWS_S3_BUCKET RESOURCES:
    + aws_s3_bucket.logs
    + aws_s3_bucket.data

RESOURCES TO UPDATE:
  # AWS_INSTANCE RESOURCES:
    ~ aws_instance.web_server

  # AWS_SECURITY_GROUP RESOURCES:
    ~ aws_security_group.allow_http

RESOURCES TO DESTROY:
  # AWS_CLOUDFRONT_DISTRIBUTION RESOURCES:
    - aws_cloudfront_distribution.legacy_cdn

TOTAL CHANGES: 5

Plan Summary: Plan: 2 to add, 2 to change, 1 to destroy.
```

### Alternative Workflows

#### One-liner for quick checks

```bash
terraform plan -out=tfplan && terraform show -json tfplan | terraform-plan-filter
```

#### Save both plan and summary

```bash
terraform plan -out=tfplan && terraform show -json tfplan | tee tfplan.json | terraform-plan-filter
```

#### Create a shell function/alias

Add this to your shell config (~/.zshrc for zsh):

```bash
# Usage: tfp -var-file=prod.tfvars

# Function to create and filter terraform plan with workaround for JSON parsing issues
tfp() {
  # First, save the current plan to a file
  echo "📝 Generating Terraform plan..."
  terraform plan -out=tfplan $@ || return 1
  
  # Convert the plan to JSON and save it to a file
  echo "💾 Converting plan to JSON..."
  terraform show -json tfplan > tfplan.json || return 1
  
  # Generate the HTML summary directly using jq to pre-process the JSON
  # This filters out the problematic provider_config section
  echo "📊 Generating HTML summary..."
  if command -v jq &>/dev/null; then
    jq 'del(.configuration.provider_config)' tfplan.json > tfplan-filtered.json
    cat tfplan-filtered.json | terraform-plan-filter --html --output tfplan-summary.html
  else
    # Fallback if jq is not installed
    cat tfplan.json | terraform-plan-filter --html --output tfplan-summary.html || echo "⚠️ HTML summary generation failed, but continuing..."
  fi
  
  # Generate text summary
  echo "📋 Text summary:"
  if command -v jq &>/dev/null; then
    cat tfplan-filtered.json | terraform-plan-filter
  else
    cat tfplan.json | terraform-plan-filter || {
      echo "⚠️ Text summary generation failed, trying to extract basic information..."
      echo "Plan:" $(grep -A 1 "\"summary\":" tfplan.json | grep "\"total\":" | grep -o "[0-9]*") "changes total"
    }
  fi
  
  echo "✅ Done! HTML summary saved to: tfplan-summary.html"
}

# Function to apply the plan
tfapply() {
  echo "🚀 Applying Terraform plan..."
  terraform apply "$@" tfplan
}

# Function to show the HTML summary in the browser
tfopen() {
  if [[ -f tfplan-summary.html ]]; then
    echo "🌐 Opening HTML summary in browser..."
    open tfplan-summary.html
  else
    echo "❌ Summary file not found! Run tfp first."
  fi
}
```

## 📦 Project Structure

```
terraform-plan-filter/
├── cmd/
│   └── terraform-plan-filter/    # Command line application
├── internal/
│   ├── formatter/                # Output formatting
│   ├── model/                    # Data structures
│   ├── parser/                   # Terraform plan parsing
│   └── util/                     # Utility functions
└── ...
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 👏 Acknowledgments

- Created with assistance from [Claude](https://anthropic.com/claude) by Anthropic
- Inspired by the need for cleaner Terraform planning workflows