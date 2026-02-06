.PHONY: run build test clean deps

# Run the application
run:
	go run cmd/api/main.go

# Build the binary
build:
	go build -o bin/api cmd/api/main.go

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f *.db

# Download dependencies
deps:
	go mod download
	go mod tidy

# Development with auto-reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	air

# Docker build
docker-build:
	docker build -t bingoals-api .

# Docker run
docker-run:
	docker run -p 8080:8080 bingoals-api
