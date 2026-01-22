package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

type Tweet struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Content    string    `json:"content"`
	LikesCount int64     `json:"likes_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ============================================
// Handler
// ============================================

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := validateToken(r)
		if err != nil {
			respondError(w, http.StatusUnauthorized, err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Name == "" || req.Password == "" {
			respondError(w, http.StatusBadRequest, "name and password are required")
			return
		}

		// ID採番 (UUID v7)
		id, err := uuid.NewV7()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		p := []byte(req.Password)
		hashedPassword, err := bcrypt.GenerateFromPassword(p, 10)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to hash password")
			return
		}

		// トランザクション開始
		tx, err := conn.Begin(ctx)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal server error")
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
			if pgErr, ok := err.(*pgconn.PgError); ok {
				switch pgErr.Code {
				case pgerrcode.UniqueViolation:
					respondError(w, http.StatusBadRequest, "user name is already used")
					return
				default:
					respondError(w, http.StatusInternalServerError, "database error")
					return
				}
			}
			respondError(w, http.StatusInternalServerError, "failed to signup")
			return
		}

		// 認証情報作成
		_, err = tx.Exec(ctx,
			"INSERT INTO user_auth (user_id, hashed_password) VALUES ($1, $2)",
			id.String(), hashedPassword,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to signup")
			return
		}

		// コミット
		if err := tx.Commit(ctx); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to signup")
			return
		}

		token := generateToken(id.String())

		resp := SignupResponse{
			User:  &user,
			Token: token,
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
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Name == "" || req.Password == "" {
			respondError(w, http.StatusBadRequest, "name and password are required")
			return
		}

		// check if there is a user data
		var user User
		err := conn.QueryRow(ctx,
			"SELECT id, name, created_at, updated_at FROM users WHERE name = $1",
			req.Name,
		).Scan(&user.ID, &user.Name, &user.CreatedAt, &user.UpdatedAt)
		if err == pgx.ErrNoRows {
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		} else if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
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
			respondError(w, http.StatusInternalServerError, "failed to find auth data")
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(userAuth.HashedPassword), []byte(req.Password))
		if err != nil {
			respondError(w, http.StatusUnauthorized, "invalid credentials")
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
			respondError(w, http.StatusUnauthorized, "token is not set")
			return
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}

			return []byte("secretKey"), nil
		})
		if err != nil {
			respondError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		if !token.Valid {
			respondError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			respondError(w, http.StatusUnauthorized, "invalid claims")
			return
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

		userID, ok := r.Context().Value(userIDKey).(string)
		if !ok {
			respondError(w, http.StatusInternalServerError, "unable to load user")
			return
		}

		var user User
		err := conn.QueryRow(
			ctx,
			"SELECT id, name, created_at, updated_at FROM users WHERE id = $1",
			userID,
		).Scan(&user.ID, &user.Name, &user.CreatedAt, &user.UpdatedAt)
		if err == pgx.ErrNoRows {
			respondError(w, http.StatusNotFound, "user not found")
			return
		} else if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
	}
}

func getUserByIDHandler(conn *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID := chi.URLParam(r, "userID")
		if userID == "" {
			respondError(w, http.StatusBadRequest, "user id is required")
			return
		}

		var user User
		err := conn.QueryRow(
			ctx,
			"SELECT id, name, created_at, updated_at FROM users WHERE id = $1",
			userID,
		).Scan(&user.ID, &user.Name, &user.CreatedAt, &user.UpdatedAt)
		if err == pgx.ErrNoRows {
			respondError(w, http.StatusNotFound, "user not found")
			return
		} else if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
	}
}

type GetTweetsRequest struct {
	Count  *int   `json:"count"`
	Cursor *int64 `json:"cursor"`
	MaxID  string `json:"max_id"`
}

type Pagination struct {
	Count      int64 `json:"count"`
	Cursor     int64 `json:"cursor"`
	NextCursor int64 `json:"next_cursor"`
}

type GetTweetsResponse struct {
	Tweets     []Tweet    `json:"tweets"`
	Pagination Pagination `json:"pagination"`
}

func getTweetsHandler(conn *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// parse query params
		count, _ := parsetIntQuery(r, "count")
		cursor, _ := parsetIntQuery(r, "cursor")
		q := r.URL.Query()
		maxIDParam := q.Get("max_id")
		fmt.Println("mip", maxIDParam)

		// cusor と maxID両方指定はだめ
		if cursor != nil && maxIDParam != "" {
			respondError(w, http.StatusBadRequest, "unable to specify both cursor and max_id")
			return
		}

		// conversion
		var maxID uuid.UUID

		if maxIDParam != "" {
			mid, err := uuid.Parse(maxIDParam)
			if err != nil {
				fmt.Println(err.Error())
			}
			maxID = mid
		}

		// default value of cout
		if count == nil {
			d := int64(20)
			count = &d
		}

		if cursor == nil {
			d := int64(-1)
			cursor = &d
		}

		// TODO: cursor らへんのゼロ値の扱いとか設計はもう少し練られるかも？

		// db access
		var tweets []Tweet
		var rowsCount int64

		if maxID == uuid.Nil {
			// cursorによる相対位置指定のクエリ
			rows, err := conn.Query(
				ctx,
				"SELECT id, user_id, content, likes_count, created_at, updated_at FROM tweets ORDER BY created_at DESC OFFSET $1 LIMIT $2",
				*cursor+1, *count,
			)
			if err == pgx.ErrNoRows {
				respondError(w, http.StatusNotFound, "not found")
				return
			} else if err != nil {
				respondError(w, http.StatusInternalServerError, err.Error())
				return
			}

			defer rows.Close()

			for rows.Next() {
				var tweet Tweet
				err := rows.Scan(&tweet.ID, &tweet.UserID, &tweet.Content, &tweet.LikesCount, &tweet.CreatedAt, &tweet.UpdatedAt)
				if err != nil {
					respondError(w, http.StatusInternalServerError, err.Error())
					return
				}

				tweets = append(tweets, tweet)
				rowsCount++
			}

		} else {
			// maxIDによる絶対値指定のクエリ
			// TODO: impl
		}

		resp := GetTweetsResponse{
			Tweets: tweets,
			Pagination: Pagination{
				Count:      rowsCount,
				Cursor:     *cursor,
				NextCursor: *cursor + rowsCount,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
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
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}))

	// PublicRoutes
	r.Group(func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello chi!"))
		})
		r.Post("/auth/signup", signupHandler(conn))
		r.Post("/auth/login", loginHandler(conn))
		r.Get("/users/{userID}", getUserByIDHandler(conn))
		r.Get("/tweets", getTweetsHandler(conn))
	})

	// PrivateRoutes(need token)
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware)
		r.Post("/auth/logout", logoutHandler(conn))
		r.Get("/me", getMeHandler(conn))
	})

	log.Println("Server starting on :8080")
	http.ListenAndServe(":8080", r)
}

// ============================================
// Utils
// ============================================

type contextKey string

const userIDKey contextKey = "userID"

func validateToken(r *http.Request) (string, error) {
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		return "", errors.New("token is not set")
	}
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}

		return []byte("secretKey"), nil
	})
	if err != nil {
		return "", errors.New("invalid token")
	}

	if !token.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", errors.New("user_id is not found in token")
	}

	return userID, nil
}

func respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func generateToken(id string) string {
	claims := jwt.MapClaims{
		"user_id": id,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secretKey := []byte("secretKey")
	tokenString, err := token.SignedString(secretKey) // error握りつぶし箇所。あとでどないかする
	if err != nil {
		fmt.Println("generateToken err", err)
	}

	return tokenString
}

// TODO: move this func into utils
func parsetIntQuery(r *http.Request, s string) (*int64, error) {
	q := r.URL.Query()
	p := q.Get(s)
	if p == "" {
		return nil, errors.New("no value")
	}
	v, err := strconv.Atoi(p)
	if err != nil {
		return nil, err
	}

	ret := int64(v)

	return &ret, nil
}
