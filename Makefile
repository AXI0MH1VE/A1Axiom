# RADM - Real-Time Anomaly Detection Microservice
# Build, test, and deployment automation

.PHONY: help build test clean docker-build docker-push deploy lint format deps

# Default target
help: ## Show this help message
	@echo "RADM Build System"
	@echo "================="
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build variables
BINARY_NAME=radm
DOCKER_REGISTRY=axiomhive
DOCKER_IMAGE=radm-detector
VERSION?=latest
GO_FILES=$(shell find . -name "*.go" -not -path "./vendor/*")

# Build targets
build: ## Build the application binary
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/$(BINARY_NAME) ./cmd/radm
	@echo "Binary built: bin/$(BINARY_NAME)"

build-local: ## Build for local development
	@echo "Building $(BINARY_NAME) for local development..."
	go build -o bin/$(BINARY_NAME) ./cmd/radm
	@echo "Binary built: bin/$(BINARY_NAME)"

test: ## Run all tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Test coverage report: coverage.html"

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test -v ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

format: ## Format Go code
	@echo "Formatting code..."
	gofmt -w $(GO_FILES)
	goimports -w $(GO_FILES)

deps: ## Download and tidy dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	go mod verify

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(VERSION)..."
	docker build -t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(VERSION) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest

docker-push: ## Push Docker image to registry
	@echo "Pushing Docker image $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(VERSION)..."
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest

docker-run: ## Run Docker container locally
	@echo "Running Docker container..."
	docker run -p 8080:8080 --name $(BINARY_NAME) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(VERSION)

# Kubernetes targets
deploy: ## Deploy to Kubernetes
	@echo "Deploying to Kubernetes..."
	kubectl apply -f infra/k8s/configmap.yaml
	kubectl apply -f infra/k8s/deployment.yaml
	kubectl apply -f infra/k8s/service.yaml
	kubectl apply -f infra/k8s/ingress.yaml
	kubectl apply -f infra/k8s/hpa.yaml
	kubectl apply -f infra/k8s/pdb.yaml

deploy-rollback: ## Rollback deployment
	@echo "Rolling back deployment..."
	kubectl rollout undo deployment/radm-detector

deploy-status: ## Check deployment status
	@echo "Deployment status:"
	kubectl rollout status deployment/radm-detector
	kubectl get pods -l app=radm-detector
	kubectl get services -l app=radm-detector

# Development targets
dev: ## Run in development mode
	@echo "Starting development server..."
	go run ./cmd/radm

dev-watch: ## Run with hot reload (requires air)
	@echo "Starting development server with hot reload..."
	air

# Monitoring targets
logs: ## View application logs
	kubectl logs -l app=radm-detector --tail=100 -f

metrics: ## View metrics
	kubectl port-forward svc/radm-detector-service 8080:8080
	@echo "Metrics available at http://localhost:8080/metrics"

# Security targets
security-scan: ## Run security scan
	@echo "Running security scan..."
	gosec ./...
	docker scan $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(VERSION)

# CI/CD targets
ci: deps lint test build docker-build ## Run CI pipeline
	@echo "CI pipeline completed"

# Release targets
release-patch: ## Create patch release
	@echo "Creating patch release..."
	git tag -a v$$(date +%Y%m%d).0.1 -m "Patch release"
	git push origin v$$(date +%Y%m%d).0.1

release-minor: ## Create minor release
	@echo "Creating minor release..."
	git tag -a v$$(date +%Y%m%d).1.0 -m "Minor release"
	git push origin v$$(date +%Y%m%d).1.0

# Validation targets
validate-config: ## Validate Kubernetes manifests
	@echo "Validating Kubernetes manifests..."
	kubectl apply --dry-run=client -f infra/k8s/

validate-dockerfile: ## Validate Dockerfile
	@echo "Validating Dockerfile..."
	docker build --dry-run .

# Performance targets
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

profile: ## Generate CPU profile
	@echo "Generating CPU profile..."
	go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./anomaly
	go tool pprof cpu.prof

# Documentation targets
docs-serve: ## Serve documentation locally
	@echo "Serving documentation..."
	swag init -g cmd/radm/main.go
	swag fmt

# Cleanup targets
uninstall: ## Remove from Kubernetes
	@echo "Removing from Kubernetes..."
	kubectl delete -f infra/k8s/ --ignore-not-found=true

reset: clean ## Reset development environment
	@echo "Resetting development environment..."
	docker system prune -f
	go clean -modcache

# Environment setup
setup: ## Setup development environment
	@echo "Setting up development environment..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install github.com/swaggo/swag/cmd/swag@latest

# Version and status
version: ## Show version information
	@echo "RADM Version Information"
	@echo "======================="
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "Build Time: $$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
	@echo "Go Version: $$(go version)"

status: ## Show project status
	@echo "Project Status"
	@echo "=============="
	@echo "Go Modules:"
	@go list -m all | head -10
	@echo "\nKubernetes Resources:"
	@kubectl get pods,services,ingress -l app=radm-detector 2>/dev/null || echo "No Kubernetes resources found"