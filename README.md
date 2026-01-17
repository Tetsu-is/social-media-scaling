# Social Media Scaling

A learning project to explore and understand how to build scalable backend systems using a social media API as the example application.

## Purpose

This project is designed for learning:

- Building REST APIs in Go
- Database design and optimization with PostgreSQL
- Horizontal scaling patterns for web applications
- Connection pooling and concurrent request handling
- API documentation with OpenAPI/Swagger

## Tech Stack

| Technology | Purpose |
|------------|---------|
| **Go 1.25** | Backend language - chosen for performance and concurrency |
| **Chi v5** | Lightweight HTTP router |
| **PostgreSQL** | Primary database |
| **pgx v5** | PostgreSQL driver with connection pooling |
| **Docker Compose** | Local development environment |
| **Swagger UI** | API documentation and testing |
| **golang-migrate** | Database migrations |

## Getting Started

```bash
# Start PostgreSQL and Swagger UI
docker-compose up -d

# Run database migrations
migrate -path ./db/migrations -database "postgres://user:password@localhost:5432/mydatabase?sslmode=disable" up

# Run the application
go run main.go
```

## Endpoints

- API: http://localhost:8080
- Swagger UI: http://localhost:8081

## Documentation

- [API Specification](docs/spec.md)
- [Database Schema](docs/schema.md)
- [OpenAPI Spec](docs/openapi.yaml)
