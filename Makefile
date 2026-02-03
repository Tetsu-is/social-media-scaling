.PHONY: migrate-up migrate-down migrate-clean docker-up docker-down docker-build docker-logs seed-test-data clean-test-data load-test

TS := $(shell date +%Y%m%d_%H%M%S)

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

seed-test-data:
	go run ./scripts/generate_test_data.go

clean-test-data:
	go run ./scripts/generate_test_data.go --clean

load-test:
	@mkdir -p perf
	k6 run --out csv=perf/results_$(TS).csv perf/script.js 2>&1 | tee perf/run_$(TS).log

