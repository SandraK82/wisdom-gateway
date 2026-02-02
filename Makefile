.PHONY: build test run clean

# Default binary name
BINARY=gateway

# Build the gateway
build:
	go build -o bin/$(BINARY) ./cmd/gateway

# Run tests
test:
	go test -v ./...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run the gateway
run:
	go run ./cmd/gateway

# Run with custom port
run-dev:
	go run ./cmd/gateway -addr :8080 -db dev.db

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f *.db
	rm -f coverage.out coverage.html

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod tidy
