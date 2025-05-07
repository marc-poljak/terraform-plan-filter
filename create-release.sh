#!/bin/zsh

# Get the version from git tag
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
echo "Creating release $VERSION..."

# Build the release binaries
make release

# Create a GitHub release using gh CLI tool
gh release create $VERSION \
  --title "Terraform Plan Filter $VERSION" \
  --notes "Release of Terraform Plan Filter with RISC-V support." \
  ./build/release/terraform-plan-filter-linux-amd64 \
  ./build/release/terraform-plan-filter-darwin-amd64 \
  ./build/release/terraform-plan-filter-darwin-arm64 \
  ./build/release/terraform-plan-filter-windows-amd64.exe \
  ./build/release/terraform-plan-filter-linux-riscv64