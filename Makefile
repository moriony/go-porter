.PHONY: test
test: ## Run tests
	@go test ./...

.PHONY: test-race
test-race: ## Run test with race detection
	@CGO_ENABLED=1 go test -race ./...

.PHONY: cover
cover: ## Run tests with cover
	@go test -covermode=count -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | grep total:

.PHONY: coverr
coverr: cover ## Run tests with cover and open report
	@go tool cover -html=coverage.out

.PHONY: bench
bench: ## Run benchmarks
	@go test -bench=. -benchmem -run "^Benchmark"

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run -v ./...
