# Variables
MODULE_NAME := github.com/bixority/pg_maintenance
APP_NAME := pg_maintenance
MAIN_FILE := ./cmd/main/main.go

# Build settings
GOFLAGS := -gcflags="all=-l" -ldflags="-s -w -extldflags '-static'"
BUILD_DIR := ./bin
OUTPUT := $(BUILD_DIR)/$(APP_NAME)

# Default target: build the application
all: build

# Build the static binary
build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) -o $(OUTPUT) $(MAIN_FILE)

# Compress the binary with UPX
compress: build
	strip $(OUTPUT)
	upx --brute $(OUTPUT)

# Build and compress for release
release: build compress

# Run the application
run: build
	$(OUTPUT)

# Test the application
test:
	go test ./...

# Format code
fmt:
	go fmt ./...

# Tidy up dependencies
deps:
	go mod tidy

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Display help
help:
	@echo "Makefile commands:"
	@echo "  make           Build the static binary"
	@echo "  make build     Build the static binary"
	@echo "  make compress  Compress the binary with UPX"
	@echo "  make release   Build and compress the binary"
	@echo "  make run       Build and run the application"
	@echo "  make test      Run tests"
	@echo "  make fmt       Format the code"
	@echo "  make deps      Install and tidy dependencies"
	@echo "  make clean     Remove build artifacts"
