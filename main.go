package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type User struct {
	ID         string
	Name       string
	Email      string
	Password   string
	Created_at string
}

func main() {
	dsn := "postgres://user:password@localhost:5432/mydatabase"

	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		log.Fatal("failed to connect databaase", err)
	}
	defer conn.Close(context.Background())

	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello chi!"))
	})

	http.ListenAndServe(":8080", r)
}
