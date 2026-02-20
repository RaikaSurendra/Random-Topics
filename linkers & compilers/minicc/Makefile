.PHONY: build test vet clean docker-build docker-test

# Go module root
MODULE_DIR := $(shell pwd)

# Binaries
MINICC  := $(MODULE_DIR)/bin/minicc
MINILINK := $(MODULE_DIR)/bin/minilink

# Default target
all: build

# Build both binaries
build:
	go build -o $(MINICC)  ./cmd/minicc
	go build -o $(MINILINK) ./cmd/minilink

# Run all tests
test:
	go test ./...

# Run go vet
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f /tmp/minicc_*.s /tmp/minicc_*.o /tmp/minicc_*_out

# --- macOS end-to-end test (ch07+) ---
# Compile a .mc file to assembly, assemble with clang, run
# Usage: make run-macos FILE=testdata/valid/06_factorial.mc
run-macos: build
	$(MINICC) -platform macos $(FILE) -o /tmp/minicc_out.s
	clang /tmp/minicc_out.s runtime/runtime.c -o /tmp/minicc_out -arch x86_64
	/tmp/minicc_out; echo "Exit code: $$?"

# --- Docker (Linux ELF pipeline, ch09+) ---
docker-build:
	docker build --platform linux/amd64 -t minicc .

docker-test: docker-build
	docker run --platform linux/amd64 --rm minicc make test

# --- Convenience ---
fmt:
	gofmt -w .
