.PHONY: migrate-up migrate-down migrate-clean docker-up docker-down docker-build docker-logs

migrate-up:
	migrate -database "postgres://user:password@localhost:5432/mydatabase?sslmode=disable" -path db/migrations up

migrate-down:
	migrate -database "postgres://user:password@localhost:5432/mydatabase?sslmode=disable" -path db/migrations down 1

migrate-clean:
	migrate -database "postgres://user:password@localhost:5432/mydatabase?sslmode=disable" -path db/migrations drop -f

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api

