# Colors
COLOR_RESET = \033[0m
COLOR_GREEN = \033[32m
COLOR_YELLOW = \033[33m

.DEFAULT_GOAL := help

help: ## Show help
	@echo "$(COLOR_YELLOW) Usage:$(COLOR_RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | sed 's/.*Makefile://' | awk 'BEGIN {FS = ":.*?## "}; {printf "$(COLOR_GREEN)\t%-60s$(COLOR_RESET) %s\n", $$1, $$2}'

## --- Infra ---

mongo-up: ## Start the local Mongo container (docker compose)
	docker compose up mongo

## --- user_service ---

build-user: ## Build user_service
	cd services/user_service && go build ./...

run-user: ## Run user_service (reads .env, defaults to Mongo on localhost:27017, HTTP :8080)
	cd services/user_service && go run ./cmd/main.go

test-user: ## Run user_service unit tests
	cd services/user_service && go test ./...

test-user-integration: ## Run user_service integration tests (requires Docker)
	cd services/user_service && go test -tags=integration ./...

## --- driver_service ---

build-driver: ## Build driver_service
	cd services/driver_service && go build ./...

run-driver: ## Run driver_service
	cd services/driver_service && go run ./cmd/main.go

test-driver: ## Run driver_service unit tests
	cd services/driver_service && go test ./...

## --- order_service ---

build-order: ## Build order_service
	cd services/order_service && go build ./...

run-order: ## Run order_service
	cd services/order_service && go run ./cmd/main.go

test-order: ## Run order_service unit tests
	cd services/order_service && go test ./...

test-order-integration: ## Run order_service integration tests (requires Docker)
	cd services/order_service && go test -tags=integration ./...

## --- proto (order_service) ---

proto-tools: ## Install protoc plugins needed for proto/gRPC-Gateway generation
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

proto-order: ## Generate Go/gRPC-Gateway/OpenAPI code from shared/proto/order_service/*.proto
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=. \
		-I . \
		-I shared/third_party/googleapis \
		shared/proto/order_service/order_service.proto shared/proto/order_service/kafka.proto

## --- Postgres migrations (order_service, via goose) ---

MIGRATIONS_DIR := services/order_service/migrations/postgres

PG_ORDER_HOST ?= localhost
PG_ORDER_PORT ?= 5433
PG_ORDER_USER ?= postgres
PG_ORDER_PASS ?= postgres
PG_ORDER_DATABASE ?= order
PG_ORDER_DSN := postgres://$(PG_ORDER_USER):$(PG_ORDER_PASS)@$(PG_ORDER_HOST):$(PG_ORDER_PORT)/$(PG_ORDER_DATABASE)?sslmode=disable

goose-tools: ## Install the goose CLI
	go install github.com/pressly/goose/v3/cmd/goose@latest

migrate-order-create: ## Create a new order_service migration (usage: make migrate-order-create name=add_foo)
	goose -dir $(MIGRATIONS_DIR) create $(name) sql

migrate-order-up: ## Apply all pending order_service migrations
	goose -dir $(MIGRATIONS_DIR) postgres "$(PG_ORDER_DSN)" up

migrate-order-down: ## Roll back the last order_service migration
	goose -dir $(MIGRATIONS_DIR) postgres "$(PG_ORDER_DSN)" down

migrate-order-status: ## Show order_service migration status
	goose -dir $(MIGRATIONS_DIR) postgres "$(PG_ORDER_DSN)" status

.PHONY: help mongo-up build-user run-user test-user test-user-integration build-driver run-driver test-driver build-order run-order test-order test-order-integration proto-tools proto-order goose-tools migrate-order-create migrate-order-up migrate-order-down migrate-order-status
