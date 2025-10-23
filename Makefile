.PHONY: help build test test-unit test-integration test-coverage test-clean lint fmt vet check-deps deps clean install setup-test-env cleanup-test-env reset-test-env test-e2e test-basic-workflow test-multi-env test-error-scenarios

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the binary
	@echo "Building kargo-bootstrap..."
	go build -o bin/kargo-bootstrap ./cmd/kargo-bootstrap

install: ## Install the binary
	@echo "Installing kargo-bootstrap..."
	go install ./cmd/kargo-bootstrap

# Test targets
test: test-unit test-integration ## Run all tests

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	go test -v ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -v -tags=integration ./tests/...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-clean: ## Clean test artifacts
	@echo "Cleaning test artifacts..."
	rm -f coverage.out coverage.html
	rm -rf testdata/temp

# Linting and formatting
lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

# Dependency management
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download

check-deps: ## Check for outdated dependencies
	@echo "Checking for outdated dependencies..."
	go list -u -m all

update-deps: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Clean targets
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html

clean-all: clean test-clean ## Clean all artifacts

# Development targets
dev: ## Run in development mode
	@echo "Running in development mode..."
	go run ./cmd/kargo-bootstrap

debug: ## Run in debug mode
	@echo "Running in debug mode..."
	dlv debug ./cmd/kargo-bootstrap

# CI/CD targets
ci: fmt vet lint test-coverage ## Run CI checks

# Release targets
release: clean test build ## Create a release
	@echo "Creating release..."
	mkdir -p bin/release
	cp bin/kargo-bootstrap bin/release/
	tar -czf bin/release/kargo-bootstrap-$(shell git describe --tags --always).tar.gz -C bin/release kargo-bootstrap

# Documentation targets
docs: ## Generate documentation
	@echo "Generating documentation..."
	godoc -http=:6060

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t kargo-bootstrap:latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -it kargo-bootstrap:latest

# Test specific packages
test-k8s: ## Run k8s package tests
	@echo "Running k8s package tests..."
	go test -v ./pkg/k8s/...

test-argocd: ## Run argocd package tests
	@echo "Running argocd package tests..."
	go test -v ./pkg/argocd/...

test-git: ## Run git package tests
	@echo "Running git package tests..."
	go test -v ./pkg/git/...

test-yaml: ## Run yaml package tests
	@echo "Running yaml package tests..."
	go test -v ./pkg/yaml/...

test-config: ## Run config package tests
	@echo "Running config package tests..."
	go test -v ./pkg/config/...

test-errors: ## Run errors package tests
	@echo "Running errors package tests..."
	go test -v ./pkg/errors/...

test-prompt: ## Run prompt package tests
	@echo "Running prompt package tests..."
	go test -v ./pkg/prompt/...

# Benchmark tests
bench: ## Run benchmark tests
	@echo "Running benchmark tests..."
	go test -bench=. -benchmem ./...

# Race condition tests
test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	go test -race -v ./...

# Test with specific Go version
test-go1.19: ## Run tests with Go 1.19
	@echo "Running tests with Go 1.19..."
	go1.19 test -v ./...

test-go1.20: ## Run tests with Go 1.20
	@echo "Running tests with Go 1.20..."
	go1.20 test -v ./...

test-go1.21: ## Run tests with Go 1.21
	@echo "Running tests with Go 1.21..."
	go1.21 test -v ./...

# Test with verbose output
test-verbose: ## Run tests with verbose output
	@echo "Running tests with verbose output..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Test with short mode
test-short: ## Run tests in short mode
	@echo "Running tests in short mode..."
	go test -short -v ./...

# Test with specific timeout
test-timeout: ## Run tests with timeout
	@echo "Running tests with timeout..."
	go test -timeout=30s -v ./...

# Test with specific pattern
test-pattern: ## Run tests with pattern (usage: make test-pattern PATTERN=TestNamespace)
	@echo "Running tests with pattern: $(PATTERN)..."
	go test -run=$(PATTERN) -v ./...

# Test with specific package
test-pkg: ## Run tests for specific package (usage: make test-pkg PKG=./pkg/k8s)
	@echo "Running tests for package: $(PKG)..."
	go test -v $(PKG)

# Test with coverage for specific package
test-coverage-pkg: ## Run tests with coverage for specific package (usage: make test-coverage-pkg PKG=./pkg/k8s)
	@echo "Running tests with coverage for package: $(PKG)..."
	go test -v -coverprofile=coverage-$(shell basename $(PKG)).out $(PKG)
	go tool cover -html=coverage-$(shell basename $(PKG)).out -o coverage-$(shell basename $(PKG)).html
	@echo "Coverage report generated: coverage-$(shell basename $(PKG)).html"

# Test with specific tags
test-tags: ## Run tests with specific tags (usage: make test-tags TAGS=integration)
	@echo "Running tests with tags: $(TAGS)..."
	go test -v -tags=$(TAGS) ./...

# Test with specific environment variables
test-env: ## Run tests with specific environment variables (usage: make test-env ENV=ENV1=value1,ENV2=value2)
	@echo "Running tests with environment variables: $(ENV)..."
	$(ENV) go test -v ./...

# Test with specific build flags
test-flags: ## Run tests with specific build flags (usage: make test-flags FLAGS="-tags=integration -race")
	@echo "Running tests with build flags: $(FLAGS)..."
	go test $(FLAGS) -v ./...

# Test with specific test directory
test-dir: ## Run tests in specific directory (usage: make test-dir DIR=./tests)
	@echo "Running tests in directory: $(DIR)..."
	go test -v $(DIR)/...

# Test with specific test file
test-file: ## Run tests in specific file (usage: make test-file FILE=./pkg/k8s/namespace_test.go)
	@echo "Running tests in file: $(FILE)..."
	go test -v $(FILE)

# Test with specific test function
test-func: ## Run specific test function (usage: make test-func FUNC=TestNamespaceExists)
	@echo "Running test function: $(FUNC)..."
	go test -v -run=$(FUNC) ./...

# Test with specific test timeout
test-timeout-func: ## Run specific test function with timeout (usage: make test-timeout-func FUNC=TestNamespaceExists TIMEOUT=10s)
	@echo "Running test function: $(FUNC) with timeout: $(TIMEOUT)..."
	go test -v -run=$(FUNC) -timeout=$(TIMEOUT) ./...

# Test with specific test count
test-count: ## Run tests with specific count (usage: make test-count COUNT=1)
	@echo "Running tests with count: $(COUNT)..."
	go test -v -count=$(COUNT) ./...

# Test with specific test parallel
test-parallel: ## Run tests with specific parallel (usage: make test-parallel PARALLEL=4)
	@echo "Running tests with parallel: $(PARALLEL)..."
	go test -v -parallel=$(PARALLEL) ./...

# Test with specific test shuffle
test-shuffle: ## Run tests with shuffle (usage: make test-shuffle SEED=123)
	@echo "Running tests with shuffle seed: $(SEED)..."
	go test -v -shuffle=$(SEED) ./...

# Test with specific test failfast
test-failfast: ## Run tests with failfast (usage: make test-failfast FAILFAST=true)
	@echo "Running tests with failfast: $(FAILFAST)..."
	go test -v -failfast=$(FAILFAST) ./...

# Test with specific test json
test-json: ## Run tests with json output (usage: make test-json JSON=true)
	@echo "Running tests with json output: $(JSON)..."
	go test -v -json=$(JSON) ./...

# Test with specific test covermode
test-covermode: ## Run tests with specific covermode (usage: make test-covermode COVERMODE=count)
	@echo "Running tests with covermode: $(COVERMODE)..."
	go test -v -covermode=$(COVERMODE) -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Test with specific test coverpkg
test-coverpkg: ## Run tests with specific coverpkg (usage: make test-coverpkg COVERPKG=./pkg/...)
	@echo "Running tests with coverpkg: $(COVERPKG)..."
	go test -v -coverpkg=$(COVERPKG) -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Test with specific test args
test-args: ## Run tests with specific args (usage: make test-args ARGS="-v -race")
	@echo "Running tests with args: $(ARGS)..."
	go test $(ARGS) ./...

# Test with specific test output
test-output: ## Run tests with specific output (usage: make test-output OUTPUT=coverage.out)
	@echo "Running tests with output: $(OUTPUT)..."
	go test -v -coverprofile=$(OUTPUT) ./...
	go tool cover -html=$(OUTPUT) -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Test with specific test profile
test-profile: ## Run tests with specific profile (usage: make test-profile PROFILE=cpu)
	@echo "Running tests with profile: $(PROFILE)..."
	go test -v -cpuprofile=$(PROFILE).prof ./...

# Test with specific test trace
test-trace: ## Run tests with specific trace (usage: make test-trace TRACE=trace.out)
	@echo "Running tests with trace: $(TRACE)..."
	go test -v -trace=$(TRACE) ./...

# Test with specific test blockprofile
test-blockprofile: ## Run tests with specific blockprofile (usage: make test-blockprofile BLOCKPROFILE=block.prof)
	@echo "Running tests with blockprofile: $(BLOCKPROFILE)..."
	go test -v -blockprofile=$(BLOCKPROFILE).prof ./...

# Test with specific test memprofile
test-memprofile: ## Run tests with specific memprofile (usage: make test-memprofile MEMPROFILE=mem.prof)
	@echo "Running tests with memprofile: $(MEMPROFILE)..."
	go test -v -memprofile=$(MEMPROFILE).prof ./...

# Test with specific test mutexprofile
test-mutexprofile: ## Run tests with specific mutexprofile (usage: make test-mutexprofile MUTEXPROFILE=mutex.prof)
	@echo "Running tests with mutexprofile: $(MUTEXPROFILE)..."
	go test -v -mutexprofile=$(MUTEXPROFILE).prof ./...

# Test with specific test gcflags
test-gcflags: ## Run tests with specific gcflags (usage: make test-gcflags GCFLAGS="-N -l")
	@echo "Running tests with gcflags: $(GCFLAGS)..."
	go test -v -gcflags="$(GCFLAGS)" ./...

# Test with specific test ldflags
test-ldflags: ## Run tests with specific ldflags (usage: make test-ldflags LDFLAGS="-X main.version=1.0.0")
	@echo "Running tests with ldflags: $(LDFLAGS)..."
	go test -v -ldflags="$(LDFLAGS)" ./...

# Test with specific test asmflags
test-asmflags: ## Run tests with specific asmflags (usage: make test-asmflags ASMFLAGS="-DGOOS_linux")
	@echo "Running tests with asmflags: $(ASMFLAGS)..."
	go test -v -asmflags="$(ASMFLAGS)" ./...

# Test with specific test buildmode
test-buildmode: ## Run tests with specific buildmode (usage: make test-buildmode BUILDMODE=archive)
	@echo "Running tests with buildmode: $(BUILDMODE)..."
	go test -v -buildmode=$(BUILDMODE) ./...

# Test with specific test compiler
test-compiler: ## Run tests with specific compiler (usage: make test-compiler COMPILER=gccgo)
	@echo "Running tests with compiler: $(COMPILER)..."
	go test -v -compiler=$(COMPILER) ./...

# Test with specific test gccgoflags
test-gccgoflags: ## Run tests with specific gccgoflags (usage: make test-gccgoflags GCCGOFLAGS="-O2")
	@echo "Running tests with gccgoflags: $(GCCGOFLAGS)..."
	go test -v -gccgoflags="$(GCCGOFLAGS)" ./...

# Test with specific test tags
test-buildtags: ## Run tests with specific buildtags (usage: make test-buildtags BUILDTAGS="integration")
	@echo "Running tests with buildtags: $(BUILDTAGS)..."
	go test -v -tags="$(BUILDTAGS)" ./...

# Test with specific test toolexec
test-toolexec: ## Run tests with specific toolexec (usage: make test-toolexec TOOLEXEC="gccgo")
	@echo "Running tests with toolexec: $(TOOLEXEC)..."
	go test -v -toolexec=$(TOOLEXEC) ./...

# Test with specific test work
test-work: ## Run tests with specific work (usage: make test-work WORK=/tmp/go-work)
	@echo "Running tests with work: $(WORK)..."
	go test -v -work=$(WORK) ./...

# Test with specific test mod
test-mod: ## Run tests with specific mod (usage: make test-mod MOD=readonly)
	@echo "Running tests with mod: $(MOD)..."
	go test -v -mod=$(MOD) ./...

# Test with specific test modfile
test-modfile: ## Run tests with specific modfile (usage: make test-modfile MODFILE=go.mod)
	@echo "Running tests with modfile: $(MODFILE)..."
	go test -v -modfile=$(MODFILE) ./...

# Test with specific test overlay
test-overlay: ## Run tests with specific overlay (usage: make test-overlay OVERLAY=overlay.json)
	@echo "Running tests with overlay: $(OVERLAY)..."
	go test -v -overlay=$(OVERLAY) ./...

# Test with specific test x
test-x: ## Run tests with specific x (usage: make test-x X="vet")
	@echo "Running tests with x: $(X)..."
	go test -v -x=$(X) ./...

# Test with specific test a
test-a: ## Run tests with specific a (usage: make test-a A="package")
	@echo "Running tests with a: $(A)..."
	go test -v -a=$(A) ./...

# Test with specific test n
test-n: ## Run tests with specific n (usage: make test-n N=1)
	@echo "Running tests with n: $(N)..."
	go test -v -n=$(N) ./...

# Test with specific test p
test-p: ## Run tests with specific p (usage: make test-p P="pattern")
	@echo "Running tests with p: $(P)..."
	go test -v -p=$(P) ./...

# Test with specific test o
test-o: ## Run tests with specific o (usage: make test-o O="output")
	@echo "Running tests with o: $(O)..."
	go test -v -o=$(O) ./...

# Test with specific test i
test-i: ## Run tests with specific i (usage: make test-i I="import")
	@echo "Running tests with i: $(I)..."
	go test -v -i=$(I) ./...

# Test with specific test pkg
test-pkg-test: ## Run tests with specific pkg (usage: make test-pkg-test PKG="package")
	@echo "Running tests with pkg: $(PKG)..."
	go test -v -pkg=$(PKG) ./...

# Test with specific test bench
test-bench-test: ## Run tests with specific bench (usage: make test-bench-test BENCH="Benchmark")
	@echo "Running tests with bench: $(BENCH)..."
	go test -v -bench=$(BENCH) ./...

# Test with specific test benchtime
test-benchtime: ## Run tests with specific benchtime (usage: make test-benchtime BENCHTIME="1s")
	@echo "Running tests with benchtime: $(BENCHTIME)..."
	go test -v -benchtime=$(BENCHTIME) ./...

# Test with specific test count
test-count-test: ## Run tests with specific count (usage: make test-count-test COUNT=1)
	@echo "Running tests with count: $(COUNT)..."
	go test -v -count=$(COUNT) ./...

# Test with specific test cover
test-cover-test: ## Run tests with specific cover (usage: make test-cover-test COVER=true)
	@echo "Running tests with cover: $(COVER)..."
	go test -v -cover=$(COVER) ./...

# Test with specific test covermode
test-covermode-test: ## Run tests with specific covermode (usage: make test-covermode-test COVERMODE=count)
	@echo "Running tests with covermode: $(COVERMODE)..."
	go test -v -covermode=$(COVERMODE) ./...

# Test with specific test coverpkg
test-coverpkg-test: ## Run tests with specific coverpkg (usage: make test-coverpkg-test COVERPKG="./pkg/...")
	@echo "Running tests with coverpkg: $(COVERPKG)..."
	go test -v -coverpkg=$(COVERPKG) ./...

# Test with specific test cpu
test-cpu: ## Run tests with specific cpu (usage: make test-cpu CPU=4)
	@echo "Running tests with cpu: $(CPU)..."
	go test -v -cpu=$(CPU) ./...

# Test with specific test failfast
test-failfast-test: ## Run tests with specific failfast (usage: make test-failfast-test FAILFAST=true)
	@echo "Running tests with failfast: $(FAILFAST)..."
	go test -v -failfast=$(FAILFAST) ./...

# Test with specific test json
test-json-test: ## Run tests with specific json (usage: make test-json-test JSON=true)
	@echo "Running tests with json: $(JSON)..."
	go test -v -json=$(JSON) ./...

# Test with specific test list
test-list: ## Run tests with specific list (usage: make test-list LIST="Test")
	@echo "Running tests with list: $(LIST)..."
	go test -v -list=$(LIST) ./...

# Test with specific test parallel
test-parallel-test: ## Run tests with specific parallel (usage: make test-parallel-test PARALLEL=4)
	@echo "Running tests with parallel: $(PARALLEL)..."
	go test -v -parallel=$(PARALLEL) ./...

# Test with specific test run
test-run-test: ## Run tests with specific run (usage: make test-run-test RUN="Test")
	@echo "Running tests with run: $(RUN)..."
	go test -v -run=$(RUN) ./...

# Test with specific test short
test-short-test: ## Run tests with specific short (usage: make test-short-test SHORT=true)
	@echo "Running tests with short: $(SHORT)..."
	go test -v -short=$(SHORT) ./...

# Test with specific test shuffle
test-shuffle-test: ## Run tests with specific shuffle (usage: make test-shuffle-test SHUFFLE=on)
	@echo "Running tests with shuffle: $(SHUFFLE)..."
	go test -v -shuffle=$(SHUFFLE) ./...

# Test with specific test timeout
test-timeout-test: ## Run tests with specific timeout (usage: make test-timeout-test TIMEOUT=30s)
	@echo "Running tests with timeout: $(TIMEOUT)..."
	go test -v -timeout=$(TIMEOUT) ./...

# Test with specific test v
test-v: ## Run tests with specific v (usage: make test-v V=true)
	@echo "Running tests with v: $(V)..."
	go test -v -v=$(V) ./...

# Test with specific test vet
test-vet-test: ## Run tests with specific vet (usage: make test-vet-test VET="all")
	@echo "Running tests with vet: $(VET)..."
	go test -v -vet=$(VET) ./...

# Test with specific test x
test-x-test: ## Run tests with specific x (usage: make test-x-test X="vet")
  @echo "Running tests with x: $(X)..."
  go test -v -x=$(X) ./...

# Test environment targets
setup-test-env: ## Set up the complete test environment
	@echo "Setting up test environment..."
	./scripts/setup-env.sh

cleanup-test-env: ## Clean up the test environment
	@echo "Cleaning up test environment..."
	./scripts/cleanup-env.sh

reset-test-env: ## Reset the test environment to a clean state
	@echo "Resetting test environment..."
	./scripts/reset-env.sh

test-e2e: ## Run end-to-end tests
	@echo "Running end-to-end tests..."
	./scripts/test-basic-workflow.sh
	./scripts/test-multi-env.sh
	./scripts/test-error-scenarios.sh

test-basic-workflow: ## Test basic deployment workflow
	@echo "Testing basic deployment workflow..."
	./scripts/test-basic-workflow.sh

test-multi-env: ## Test multi-environment deployment
	@echo "Testing multi-environment deployment..."
	./scripts/test-multi-env.sh

test-error-scenarios: ## Test error handling
	@echo "Testing error scenarios..."
	./scripts/test-error-scenarios.sh