# 🚀 Terraform Plan Filter

[![Go Report Card](https://goreportcard.com/badge/github.com/marc-poljak/terraform-plan-filter)](https://goreportcard.com/report/github.com/marc-poljak/terraform-plan-filter)
[![License](https://img.shields.io/github/license/yourusername/terraform-plan-filter)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yourusername/terraform-plan-filter)](go.mod)

A lightweight CLI tool that streamlines Terraform plan output by filtering and displaying only the resource titles that will be created, updated, or destroyed, without all the verbose details.

## 📋 Overview

When working with large Terraform projects, the plan output can be overwhelming with detailed attribute changes. This tool provides a clean, concise summary showing only the resource titles organized first by action type (create, update, destroy) and then by resource type, making it easier to quickly review what will change.

## ✨ Features

- 🔍 Filters Terraform `plan` output to show only resource titles
- 🎯 Categorizes resources first by action type (create, update, destroy)
- 🎨 Groups resources by resource type (aws_s3_bucket, aws_instance, etc.)
- 🌈 Colorized output (green for creations, yellow for updates, red for deletions)
- 📊 Provides a total count of changes
- 📱 Multiple output formats (text, JSON, HTML)
- 🧰 Simple to use - just pipe your Terraform output to the tool

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

Simply pipe your Terraform plan output to the tool:

```bash
terraform plan | terraform-plan-filter
```

### Command Line Flags

```
Usage: terraform plan | terraform-plan-filter [options]

Options:
  -no-color       Disable colored output
  -json           Output in JSON format
  -html           Output in HTML format
  -output string  Output file (default: stdout)
  -verbose        Show verbose output
```

### Examples

Basic usage with colored text output:
```bash
terraform plan | terraform-plan-filter
```

Generate JSON output to a file:
```bash
terraform plan -no-color | terraform-plan-filter --json --output plan.json
```

Generate HTML report:
```bash
terraform plan | terraform-plan-filter --html --output plan.html
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