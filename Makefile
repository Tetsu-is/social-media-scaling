.PHONY: migrate-up migrate-down migrate-clean 

migrate-up:
	migrate -database "postgres://user:password@localhost:5432/mydatabase?sslmode=disable" -path db/migrations up

migrate-down:
	migrate -database "postgres://user:password@localhost:5432/mydatabase?sslmode=disable" -path db/migrations down 1

migrate-clean:
	migrate -database "postgres://user:password@localhost:5432/mydatabase?sslmode=disable" -path db/migrations drop -f

