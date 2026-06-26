BINARY := disgo
PKG     := ./cmd/bot
OUT     := bin/$(BINARY)

COMPOSE := docker compose -f deployments/docker-compose.yml

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the bot binary into bin/
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o $(OUT) $(PKG)

.PHONY: run
run: ## Run the bot locally (reads ./config.yaml + env)
	go run $(PKG)

.PHONY: test
test: ## Run unit tests with the race detector
	go test -race -count=1 ./...

.PHONY: cover
cover: ## Run tests and open an HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (must be installed)
	golangci-lint run

.PHONY: fmt
fmt: ## Format all Go code
	gofmt -w .

.PHONY: tidy
tidy: ## Sync go.mod / go.sum
	go mod tidy

.PHONY: check
check: fmt vet test ## Format, vet and test in one shot

.PHONY: docker-build
docker-build: ## Build the container image
	docker build -f deployments/docker/Dockerfile -t disgo-bot:latest .

.PHONY: up
up: ## Start the full stack (Postgres + Redis + bot)
	$(COMPOSE) up --build

.PHONY: down
down: ## Stop the stack and remove volumes
	$(COMPOSE) down -v

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin coverage.out
