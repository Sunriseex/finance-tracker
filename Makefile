SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

.PHONY: help test race lint check check-race db-up db-down db-migrate db-rollback

help:
	@echo "Targets:"
	@echo "  test   - run Go tests"
	@echo "  lint   - run golangci-lint"
	@echo "  check  - run tests and lint"
	@echo "  db-up  - start local PostgreSQL"
	@echo "  db-down - stop local PostgreSQL"
	@echo "  db-migrate - run PostgreSQL migrations"
	@echo "  db-rollback - rollback one PostgreSQL migration"

test:
	@go test ./...

lint:
	@golangci-lint run ./...

race: 
	@go test ./... -race

check-race: test lint race

check: test lint

db-up:
	@docker compose up -d postgres

db-down:
	@docker compose down

db-migrate:
	@go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$${DATABASE_URL:-postgres://finance_tracker:finance_tracker@localhost:5432/finance_tracker?sslmode=disable}" up

db-rollback:
	@go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$${DATABASE_URL:-postgres://finance_tracker:finance_tracker@localhost:5432/finance_tracker?sslmode=disable}" down
