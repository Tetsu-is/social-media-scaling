package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ============================================
// Domain Objects
// ============================================

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ============================================
// Handler
// ============================================

type SignupRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type SignupResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}

func signupHandler(conn *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// リクエストパース
		var req SignupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Password == "" {
			http.Error(w, "name and password are required", http.StatusBadRequest)
			return
		}

		// ID採番 (UUID v7)
		id, err := uuid.NewV7()
		if err != nil {
			log.Printf("failed to generate uuid: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// TODO: パスワードハッシュ化
		p := []byte(req.Password)
		hashedPassword, err := bcrypt.GenerateFromPassword(p, 10)
		if err != nil {
			http.Error(w, "failed to hash pasword", http.StatusInternalServerError)
			return
		}

		// トランザクション開始
		tx, err := conn.Begin(ctx)
		if err != nil {
			log.Printf("failed to begin transaction: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		// ユーザー作成
		var user User
		err = tx.QueryRow(ctx,
			"INSERT INTO users (id, name) VALUES ($1, $2) RETURNING id, name, created_at, updated_at",
			id.String(), req.Name,
		).Scan(&user.ID, &user.Name, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			log.Printf("failed to create user: %v", err)
			http.Error(w, "failed to signup", http.StatusInternalServerError)
			return
		}

		// 認証情報作成
		_, err = tx.Exec(ctx,
			"INSERT INTO user_auth (user_id, hashed_password) VALUES ($1, $2)",
			id.String(), hashedPassword,
		)
		if err != nil {
			log.Printf("failed to create user auth: %v", err)
			http.Error(w, "failed to signup", http.StatusInternalServerError)
			return
		}

		// コミット
		if err := tx.Commit(ctx); err != nil {
			log.Printf("failed to commit transaction: %v", err)
			http.Error(w, "failed to signup", http.StatusInternalServerError)
			return
		}

		// TODO: JWT トークン生成
		token := "dummy-token"

		resp := SignupResponse{
			User:  &user,
			Token: token,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

// ============================================
// Main
// ============================================

func main() {
	dsn := "postgres://user:password@localhost:5432/mydatabase"

	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}
	defer conn.Close(context.Background())

	r := chi.NewRouter()

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:8081"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))

	// Routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello chi!"))
	})

	r.Post("/auth/signup", signupHandler(conn))

	log.Println("Server starting on :8080")
	http.ListenAndServe(":8080", r)
}
