# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go-based REST API for a social media application using Chi router and PostgreSQL.

## Development Commands

```bash
# Start dependencies (PostgreSQL + Swagger UI)
docker-compose up -d

# Run the application
go run main.go

# Build binary
go build -o social-media-scaling ./

# Manage dependencies
go mod tidy

# Stop dependencies
docker-compose down

# Run migrations
migrate -path ./migrations -database "postgres://user:password@localhost:5432/mydatabase?sslmode=disable" up

# Rollback one migration
migrate -path ./migrations -database "postgres://user:password@localhost:5432/mydatabase?sslmode=disable" down 1

# Create new migration
migrate create -ext sql -dir ./migrations -seq <migration_name>
```

## Architecture

Single-file Go microservice (`main.go`) with:
- Chi v5 HTTP router on port 8080
- PostgreSQL via pgx driver (localhost:5432)
- OpenAPI 3.0 spec (`openapi.yaml`) with Swagger UI on port 8081

**Database credentials (dev):** user/password, database: mydatabase

## Documentation

- `docs/spec.md` - Project requirements and features
- `docs/schema.md` - Database schema
- `openapi.yaml` - API endpoint specifications (Swagger UI on port 8081)
