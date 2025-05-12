#!/bin/zsh

# Check if GitHub CLI is installed
if ! command -v gh &> /dev/null; then
  echo "ERROR: GitHub CLI (gh) not found. Please install it first:"
  echo "  brew install gh"
  exit 1
fi

# Check if the user is logged in to GitHub
echo "Checking GitHub authentication status..."
if ! gh auth status &> /dev/null; then
  echo "ERROR: You are not logged in to GitHub via the GitHub CLI."
  echo "Please authenticate first by running:"
  echo "  gh auth login"
  exit 1
fi

# Get the version from git tag
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
echo "Creating release $VERSION..."

# Build the release binaries
make release

# Check if binaries were built successfully
if [ ! -f "./build/release/terraform-plan-filter-darwin-amd64" ]; then
  echo "ERROR: Binaries were not built successfully. Check the build process."
  exit 1
fi

# Create compressed archives for each binary
echo "Creating compressed archives..."
cd "./build/release/"
zip -j "terraform-plan-filter-darwin-amd64.zip" "terraform-plan-filter-darwin-amd64"
zip -j "terraform-plan-filter-darwin-arm64.zip" "terraform-plan-filter-darwin-arm64"
zip -j "terraform-plan-filter-linux-amd64.zip" "terraform-plan-filter-linux-amd64"
zip -j "terraform-plan-filter-linux-riscv64.zip" "terraform-plan-filter-linux-riscv64"
zip -j "terraform-plan-filter-windows-amd64.zip" "terraform-plan-filter-windows-amd64.exe"
cd "../.."

# Create a GitHub release using gh CLI tool (with draft flag so you can edit it)
echo "Creating draft release $VERSION..."
gh release create $VERSION \
  --title "Terraform Plan Filter $VERSION" \
  --draft \
  "./build/release/terraform-plan-filter-linux-amd64.zip" \
  "./build/release/terraform-plan-filter-darwin-amd64.zip" \
  "./build/release/terraform-plan-filter-darwin-arm64.zip" \
  "./build/release/terraform-plan-filter-windows-amd64.zip" \
  "./build/release/terraform-plan-filter-linux-riscv64.zip"

echo "Draft release $VERSION created successfully!"
echo "Please visit GitHub to edit the release notes and publish the release."
echo "https://github.com/marc-poljak/terraform-plan-filter/releases"