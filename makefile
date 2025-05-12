.PHONY: build test clean install release all

# Build variables
BINARY_NAME = terraform-plan-filter
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOPATH ?= $(shell go env GOPATH)
BUILD_DIR ?= ./build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) -ldflags "-X main.version=$(VERSION)" ./cmd/terraform-plan-filter
	@echo "Build complete. Binary is located at $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	
# Run tests
test:
	go test -v ./...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out

# Install the binary
install: build
	@echo "Installing to $(GOPATH)/bin/$(BINARY_NAME)"
	@mkdir -p $(GOPATH)/bin
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

# Format the code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint the code
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Running golangci-lint..."; \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin; \
		$(GOPATH)/bin/golangci-lint run ./...; \
	fi

# Build for all platforms
release:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)/release
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 -ldflags "-X main.version=$(VERSION)" ./cmd/terraform-plan-filter
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 -ldflags "-X main.version=$(VERSION)" ./cmd/terraform-plan-filter
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-arm64 -ldflags "-X main.version=$(VERSION)" ./cmd/terraform-plan-filter
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/release/$(BINARY_NAME)-windows-amd64.exe -ldflags "-X main.version=$(VERSION)" ./cmd/terraform-plan-filter
	GOOS=linux GOARCH=riscv64 go build -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-riscv64 -ldflags "-X main.version=$(VERSION)" ./cmd/terraform-plan-filter
	@echo "Build complete. Binaries available in $(BUILD_DIR)/release/"
	@ls -la $(BUILD_DIR)/release/

# Default target - build and test
all: fmt lint build test