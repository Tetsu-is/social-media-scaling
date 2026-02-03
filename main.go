package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Tetsu-is/social-media-scaling/internal/auth"
	"github.com/Tetsu-is/social-media-scaling/internal/domain"
	"github.com/Tetsu-is/social-media-scaling/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ============================================
// Handlers
// ============================================

func signupHandler(userRepo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req domain.SignupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Name == "" || req.Password == "" {
			respondError(w, http.StatusBadRequest, "name and password are required")
			return
		}

		id, err := uuid.NewV7()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		user, err := userRepo.CreateUser(ctx, id.String(), req.Name, req.Password)
		if err != nil {
			if err == repository.ErrDuplicateUser {
				respondError(w, http.StatusConflict, "user name is already used")
				return
			}
			respondError(w, http.StatusInternalServerError, "failed to signup")
			return
		}

		token := auth.GenerateToken(id.String())

		resp := domain.SignupResponse{
			User:  user,
			Token: token,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func loginHandler(userRepo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req domain.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Name == "" || req.Password == "" {
			respondError(w, http.StatusBadRequest, "name and password are required")
			return
		}

		user, err := userRepo.GetUserByName(ctx, req.Name)
		if err == repository.ErrUserNotFound {
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		} else if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
			return
		}

		userAuth, err := userRepo.GetUserAuth(ctx, user.ID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to find auth data")
			return
		}

		err = userRepo.VerifyPassword(userAuth.HashedPassword, req.Password)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		token := auth.GenerateToken(user.ID)

		resp := domain.LoginResponse{
			User:  user,
			Token: token,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

func logoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		fmt.Println(claims)
		w.WriteHeader(http.StatusNoContent)
	}
}

func getMeHandler(userRepo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, ok := ctx.Value(auth.UserIDKey).(string)
		if !ok {
			respondError(w, http.StatusInternalServerError, "unable to load user")
			return
		}

		user, err := userRepo.GetUserByID(ctx, userID)
		if err == repository.ErrUserNotFound {
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

func getUserByIDHandler(userRepo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID := chi.URLParam(r, "userID")
		if userID == "" {
			respondError(w, http.StatusBadRequest, "user id is required")
			return
		}

		user, err := userRepo.GetUserByID(ctx, userID)
		if err == repository.ErrUserNotFound {
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

func getTweetsHandler(tweetRepo *repository.TweetRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		limit, _ := parseIntQuery(r, "limit")
		offset, _ := parseIntQuery(r, "offset")
		q := r.URL.Query()
		maxIDParam := q.Get("max_id")

		if offset != nil && maxIDParam != "" {
			respondError(w, http.StatusBadRequest, "unable to specify both offset and max_id")
			return
		}

		var maxID uuid.UUID
		if maxIDParam != "" {
			mid, err := uuid.Parse(maxIDParam)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid max_id")
				return
			}
			maxID = mid
		}

		if limit == nil {
			d := int64(20)
			limit = &d
		}
		if *limit < 1 || *limit > 100 {
			respondError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}

		if offset == nil {
			d := int64(0)
			offset = &d
		}
		if *offset < 0 {
			respondError(w, http.StatusBadRequest, "offset must be 0 or greater")
			return
		}

		var tweets []domain.Tweet

		if maxID == uuid.Nil {
			var err error
			// limit + 1 件取得して次のページがあるか確認する
			tweets, err = tweetRepo.GetTweets(ctx, *offset, *limit+1)
			if err != nil {
				respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
		} else {
			respondError(w, http.StatusBadRequest, "max_id parameter is not supported yet")
			return
		}

		// 次のページがあるか確認
		var nextOffset *int64
		if int64(len(tweets)) > *limit {
			tweets = tweets[:*limit]
			no := *offset + *limit
			nextOffset = &no
		}

		resp := domain.GetTweetsResponse{
			Tweets: tweets,
			Pagination: domain.Pagination{
				Offset:     *offset,
				Limit:      *limit,
				NextOffset: nextOffset,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

func postTweetHandler(tweetRepo *repository.TweetRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, ok := ctx.Value(auth.UserIDKey).(string)
		if !ok {
			respondError(w, http.StatusInternalServerError, "unable to load user")
			return
		}

		var req domain.PostTweetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Content == "" {
			respondError(w, http.StatusBadRequest, "content is blank")
			return
		}
		if len(req.Content) > 255 {
			respondError(w, http.StatusBadRequest, "content exceeds 255 characters")
			return
		}

		id, err := uuid.NewV7()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		tweet, err := tweetRepo.CreateTweet(ctx, id.String(), userID, req.Content)
		if err != nil {
			if err == repository.ErrDuplicateTweet {
				respondError(w, http.StatusBadRequest, "duplicate tweet")
				return
			}
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		resp := domain.PostTweetResponse{
			ID:         tweet.ID,
			UserID:     tweet.UserID,
			Content:    tweet.Content,
			LikesCount: tweet.LikesCount,
			CreatedAt:  tweet.CreatedAt,
			UpdatedAt:  tweet.UpdatedAt,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func followHandler(userRepo *repository.UserRepository, followRepo *repository.FollowRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		followerID, ok := ctx.Value(auth.UserIDKey).(string)
		if !ok {
			respondError(w, http.StatusInternalServerError, "unable to load user")
			return
		}

		followeeID := chi.URLParam(r, "userID")
		if followeeID == "" {
			respondError(w, http.StatusBadRequest, "user id is required")
			return
		}

		if followerID == followeeID {
			respondError(w, http.StatusBadRequest, "cannot follow yourself")
			return
		}

		exists, err := userRepo.CheckUserExists(ctx, followeeID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
			return
		}
		if !exists {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}

		err = followRepo.CreateFollow(ctx, followerID, followeeID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to follow")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func unfollowHandler(followRepo *repository.FollowRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		followerID, ok := ctx.Value(auth.UserIDKey).(string)
		if !ok {
			respondError(w, http.StatusInternalServerError, "unable to load user")
			return
		}

		followeeID := chi.URLParam(r, "userID")
		if followeeID == "" {
			respondError(w, http.StatusBadRequest, "user id is required")
			return
		}

		err := followRepo.DeleteFollow(ctx, followerID, followeeID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to unfollow")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func getFollowersHandler(userRepo *repository.UserRepository, followRepo *repository.FollowRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID := chi.URLParam(r, "userID")
		if userID == "" {
			respondError(w, http.StatusBadRequest, "user id is required")
			return
		}

		exists, err := userRepo.CheckUserExists(ctx, userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
			return
		}
		if !exists {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}

		users, err := followRepo.GetFollowers(ctx, userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
			return
		}

		if users == nil {
			users = []domain.User{}
		}

		resp := domain.GetUsersResponse{Users: users}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

func getFolloweesHandler(userRepo *repository.UserRepository, followRepo *repository.FollowRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID := chi.URLParam(r, "userID")
		if userID == "" {
			respondError(w, http.StatusBadRequest, "user id is required")
			return
		}

		exists, err := userRepo.CheckUserExists(ctx, userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
			return
		}
		if !exists {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}

		users, err := followRepo.GetFollowees(ctx, userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
			return
		}

		if users == nil {
			users = []domain.User{}
		}

		resp := domain.GetUsersResponse{Users: users}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

func getFeedHandler(feedRepo *repository.FeedRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, ok := ctx.Value(auth.UserIDKey).(string)
		if !ok {
			respondError(w, http.StatusInternalServerError, "unable to load user")
			return
		}

		limit, _ := parseIntQuery(r, "limit")
		offset, _ := parseIntQuery(r, "offset")

		if limit == nil {
			d := int64(20)
			limit = &d
		}
		if *limit < 1 || *limit > 100 {
			respondError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}

		if offset == nil {
			d := int64(0)
			offset = &d
		}
		if *offset < 0 {
			respondError(w, http.StatusBadRequest, "offset must be 0 or greater")
			return
		}

		// limit + 1 件取得して次のページがあるか確認する
		tweets, err := feedRepo.GetFeedTweets(ctx, userID, *offset, *limit+1)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch feed")
			return
		}

		if tweets == nil {
			tweets = []domain.TweetWithUser{}
		}

		// 次のページがあるか確認
		var nextOffset *int64
		if int64(len(tweets)) > *limit {
			tweets = tweets[:*limit]
			no := *offset + *limit
			nextOffset = &no
		}

		resp := domain.GetFeedResponse{
			Tweets: tweets,
			Pagination: domain.Pagination{
				Offset:     *offset,
				Limit:      *limit,
				NextOffset: nextOffset,
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
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5432/mydatabase"
	}

	pgxConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal(err)
		return
	}
	pgxConfig.MaxConns = 1
	pgxConfig.MinConns = 1
	pgxConfig.MaxConnLifetime = 1 * time.Hour
	pgxConfig.MaxConnIdleTime = 30 * time.Minute
	pgxConfig.HealthCheckPeriod = 1 * time.Minute

	// conn, err := pgx.Connect(context.Background(), dsn)
	conn, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}
	defer conn.Close()

	userRepo := repository.NewUserRepository(conn)
	tweetRepo := repository.NewTweetRepository(conn)
	followRepo := repository.NewFollowRepository(conn)
	feedRepo := repository.NewFeedRepository(conn)

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:8081"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}))

	r.Group(func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello chi!"))
		})
		r.Post("/auth/signup", signupHandler(userRepo))
		r.Post("/auth/login", loginHandler(userRepo))
		r.Get("/users/{userID}", getUserByIDHandler(userRepo))
		r.Get("/users/{userID}/followers", getFollowersHandler(userRepo, followRepo))
		r.Get("/users/{userID}/followees", getFolloweesHandler(userRepo, followRepo))
		r.Get("/tweets", getTweetsHandler(tweetRepo))
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware)
		r.Post("/auth/logout", logoutHandler())
		r.Get("/users/me", getMeHandler(userRepo))
		r.Get("/users/me/feed", getFeedHandler(feedRepo))
		r.Put("/users/{userID}/follow", followHandler(userRepo, followRepo))
		r.Delete("/users/{userID}/follow", unfollowHandler(followRepo))
		r.Post("/tweets", postTweetHandler(tweetRepo))
	})

	// PPROF_ENABLED=1 で :6060 に pprof API を公開（ベンチマーク用）
	if os.Getenv("PPROF_ENABLED") == "1" {
		go func() {
			log.Println("pprof listening on :6060")
			http.ListenAndServe(":6060", nil)
		}()
	}

	log.Println("Server starting on :8080")
	http.ListenAndServe(":8080", r)
}

// ============================================
// Utils
// ============================================

func respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(domain.ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func parseIntQuery(r *http.Request, s string) (*int64, error) {
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
