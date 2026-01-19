package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/golang-jwt/jwt/v5"
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

type UserAuth struct {
	UserID         string    `json:"-"` // 外に出さない予定だけど念の為パースできないように
	HashedPassword string    `json:"-"`
	CreatedAt      time.Time `json:"-"`
	UpdatedAt      time.Time `json:"-"`
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

		// JWT トークン生成
		claims := jwt.MapClaims{
			"user_id": id,
			"exp":     time.Now().Add(time.Hour * 24).Unix(),
			"iat":     time.Now().Unix(),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		secretKey := []byte("secretKey")
		tokenString, _ := token.SignedString(secretKey) // error握りつぶし箇所。あとでどないかする

		resp := SignupResponse{
			User:  &user,
			Token: tokenString,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}

func loginHandler(conn *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// parse request data
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Password == "" {
			http.Error(w, "name and password are required", http.StatusBadRequest)
			return
		}

		// check if there is a user data
		var user User
		err := conn.QueryRow(ctx,
			"SELECT id, name, created_at, updated_at FROM users WHERE name = $1",
			req.Name,
		).Scan(&user.ID, &user.Name, &user.CreatedAt, &user.UpdatedAt)
		if err == pgx.ErrNoRows {
			http.Error(w, "user not found", http.StatusBadRequest)
			return
		} else if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}

		// check if the password match with one in db
		var userAuth UserAuth
		err = conn.QueryRow(ctx,
			"SELECT user_id, hashed_password, created_at, updated_at from user_auth WHERE user_id = $1",
			user.ID,
		).Scan(&userAuth.UserID, &userAuth.HashedPassword, &userAuth.CreatedAt, &userAuth.UpdatedAt)
		if err != nil {
			// userは見つかったのに認証情報がないのはサインアップのトランザクションに不具合の可能性あるかも
			http.Error(w, "failed to find auth data", http.StatusInternalServerError)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(userAuth.HashedPassword), []byte(req.Password))
		if err != nil {
			http.Error(w, "fail to match password", http.StatusUnauthorized)
			return
		}

		// JWT トークン生成
		claims := jwt.MapClaims{
			"user_id": user.ID,
			"exp":     time.Now().Add(time.Hour * 24).Unix(),
			"iat":     time.Now().Unix(),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		secretKey := []byte("secretKey")
		tokenString, _ := token.SignedString(secretKey) // error握りつぶし箇所。あとでどないかする

		resp := LoginResponse{
			User:  &user,
			Token: tokenString,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

func logoutHandler(conn *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ctx := r.Context()

		// jwtを取り出す。
		tokenString := r.Header.Get("Authorization")
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		if tokenString == "" {
			http.Error(w, "token is not set", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}

			return []byte("secretKey"), nil
		})
		if err != nil {
			http.Error(w, "failed to parse jwt", http.StatusBadRequest)
			return
		}

		if !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "invalid claims", http.StatusUnauthorized)
		}

		// userID, ok := claims["user_id"].(string)
		//if !ok {
		//	http.Error(w, "user_id not found in token", http.StatusUnauthorized)
		//	return
		//}

		// refreshTokenなどは後ほど実装。
		fmt.Println(claims)
		w.WriteHeader(http.StatusNoContent)
	}
}

func getMeHandler(conn *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		tokenString := r.Header.Get("Authorization")
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		if tokenString == "" {
			http.Error(w, "token is not set", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}

			return []byte("secretKey"), nil
		})
		if err != nil {
			http.Error(w, "failed to parse jwt", http.StatusBadRequest)
			return
		}

		if !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "invalid claims", http.StatusUnauthorized)
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			http.Error(w, "user_id not found in token", http.StatusUnauthorized)
			return
		}

		var user User
		err = conn.QueryRow(
			ctx,
			"SELECT id, name, created_at, updated_at FROM users WHERE id = $1",
			userID,
		).Scan(&user.ID, &user.Name, &user.CreatedAt, &user.UpdatedAt)
		if err == pgx.ErrNoRows {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "err", http.StatusInternalServerError)
			fmt.Println(err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
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
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}))

	// Routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello chi!"))
	})

	r.Post("/auth/signup", signupHandler(conn))
	r.Post("/auth/login", loginHandler(conn))
	r.Post("/auth/logout", logoutHandler(conn))

	r.Get("/me", getMeHandler(conn))

	log.Println("Server starting on :8080")
	http.ListenAndServe(":8080", r)
}
