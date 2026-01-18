package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// domain objects

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
}

// define request data & response data

type CreateUserRequest struct {
	Name     string
	Password string
}

var conn *pgx.Conn

func main() {
	dsn := "postgres://user:password@localhost:5432/mydatabase"

	var err error
	conn, err = pgx.Connect(context.Background(), dsn)
	if err != nil {
		log.Fatal("failed to connect databaase", err)
	}
	defer conn.Close(context.Background())

	r := chi.NewRouter()

	// cors middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:8081"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("get /")
		w.Write([]byte("Hello chi!"))
	})

	r.Post("/users", CreateUser)
	r.Get("/users", ListUsers)
	// r.Put("/users", UpdateUser)
	// r.Delete("/users", DeleteUser)

	http.ListenAndServe(":8080", r)
}

// controller for /users

func CreateUser(w http.ResponseWriter, r *http.Request) {
	fmt.Println("post /users")
	var req CreateUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "decode error", http.StatusBadRequest)
		return
	}

	user := User{
		Name:     req.Name,
		Password: req.Password,
	}

	created, err := RepoCreateUser(user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				http.Error(w, "username is already used", http.StatusBadRequest)
				return
			}
		} else {
			http.Error(w, "failed to create user", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func ListUsers(w http.ResponseWriter, r *http.Request) {
	fmt.Println("get /users")

	users, err := ReposListUsers()
	if err != nil {
		http.Error(w, "failed to read list user", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(users)

	return
}

// repository for users

func RepoCreateUser(user User) (*User, error) {
	var created User
	err := conn.QueryRow(
		context.Background(),
		"insert into users(name, password) values($1, $2) returning id, name, password, created_at",
		user.Name, user.Password,
	).Scan(&created.ID, &created.Name, &created.Password, &created.CreatedAt)
	if err != nil {
		fmt.Println("db err", err)
		return nil, err
	}

	return &created, nil
}

func RepoReadUser(id string) (*User, error) {
	row, err := conn.Query(context.Background(), "select * from users where id = $1", id)
	if err != nil {
		return nil, err
	}
	var u User
	err = row.Scan(&u.ID, &u.Name, &u.Password, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &u, nil
}

func ReposListUsers() ([]User, error) {
	rows, err := conn.Query(context.Background(), "select * from users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Password, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
