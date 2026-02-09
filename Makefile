RUST_TARGETARCH ?= x86_64
TARGET_DIR := ./target
OUTPUT := $(TARGET_DIR)/$(RUST_TARGETARCH)-unknown-linux-musl/release/pg-maintenance

# Default target: build the application
all: build

# Build the static binary
build:
	rustup target add $(RUST_TARGETARCH)-unknown-linux-musl
	RUSTFLAGS='-C relocation-model=static -C strip=symbols' cargo build --release --target $(RUST_TARGETARCH)-unknown-linux-musl --target-dir $(TARGET_DIR)

# Compress the binary with UPX
compress: build
	strip $(OUTPUT)
	upx --brute $(OUTPUT)

# Build and compress for release
release: build compress
	cp $(OUTPUT) $(TARGET_DIR)/

# Run the application
run: build
	$(OUTPUT)

clean:
	cargo clean

# Display help
help:
	@echo "Makefile commands:"
	@echo "  make           Build the static binary"
	@echo "  make build     Build the static binary"
	@echo "  make compress  Compress the binary with UPX"
	@echo "  make release   Build and compress the binary"
	@echo "  make clean     Remove build artifacts"
