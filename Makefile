.PHONY: build clean test install release

# Variables
BINARY_NAME=pf
VERSION=$(shell git describe --tags --always --dirty)
COMMIT=$(shell git rev-parse --short HEAD)
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

# Build
build:
	go build ${LDFLAGS} -o bin/${BINARY_NAME} cmd/portfinder/main.go

# Build for all platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-amd64 cmd/portfinder/main.go
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-arm64 cmd/portfinder/main.go
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-linux-amd64 cmd/portfinder/main.go
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-linux-arm64 cmd/portfinder/main.go

# Clean
clean:
	go clean
	rm -rf bin/

# Test
test:
	go test ./...

# Install locally
install: build
	sudo cp bin/${BINARY_NAME} /usr/local/bin/

# Run
run: build
	./bin/${BINARY_NAME}

# Format code
fmt:
	go fmt ./...

# Lint
lint:
	golangci-lint run

# Generate release with goreleaser
release:
	goreleaser release --clean

# Snapshot release (for testing)
snapshot:
	goreleaser release --snapshot --clean