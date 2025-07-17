GOLANGCI_VERSION = v1.64.8

all: help

build: ## compiles the whole code base
	@go version
	go build -v ./...

test: build ## executes all unit tests
	go clean -testcache
	go test ./...

asan: build ## executes all unit tests with asan flag
	go clean -testcache
	go test -asan ./...

race: build ## executes all unit tests with race flag
	go clean -testcache
	go test -race ./...

clean: ## deletes untracked git and go cached files
	git clean -xfd
	go clean -testcache

fmt: ## uses gofmt to format the source code base
	gofmt -w $(shell find -name "*.go")

coverage: ## generates test coverage
	go test -coverpkg ./... -coverprofile coverage.out ./...
	go tool cover -html coverage.out -o coverage.html

lint: ## runs a golang source code linter
	golangci-lint run --timeout 10m -E gofmt,gofumpt

install-linters: ## install all used linters
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_VERSION}

help: ## display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
