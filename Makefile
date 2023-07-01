all: help

build: ## compiles the whole code base
	@go version
	go build -v ./...

test: build ## executes all unit tests
	go clean -testcache
	go test ./...

clean: ## deletes untracked git and go cached files
	git clean -xfd
	go clean -testcache

fmt: ## uses gofmt to format the source code base
	gofmt -w $(shell find -name "*.go")

static-anal: ## executes basic static code-analysis tools
	staticcheck -f=stylish ./...
	go vet ./...
	go vet -vettool=$(shell which shadow) ./...

coverage:
	rm -f coverage.out coverage.html
	go test -v -coverprofile coverage.out
	go tool cover -html coverage.out -o coverage.html

lint: ## runs a golang source code linter
	golint -set_exit_status ./...

help: ## display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
