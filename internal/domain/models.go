package domain

import "time"

// ============================================
// Domain Models
// ============================================

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserAuth struct {
	UserID         string    `json:"-"`
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

type TweetWithUser struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Content    string    `json:"content"`
	LikesCount int64     `json:"likes_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	User       User      `json:"user"`
}

// ============================================
// Request/Response Models
// ============================================

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type SignupRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type SignupResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}

type PostTweetRequest struct {
	Content string `json:"content"`
}

type PostTweetResponse struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Content    string    `json:"content"`
	LikesCount int64     `json:"likes_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type GetTweetsRequest struct {
	Count  *int   `json:"count"`
	Cursor *int64 `json:"cursor"`
	MaxID  string `json:"max_id"`
}

type Pagination struct {
	Count      int64  `json:"count"`
	Cursor     int64  `json:"cursor"`
	NextCursor *int64 `json:"next_cursor"`
}

type GetTweetsResponse struct {
	Tweets     []Tweet    `json:"tweets"`
	Pagination Pagination `json:"pagination"`
}

type GetUsersResponse struct {
	Users []User `json:"users"`
}

type FeedPagination struct {
	Count      int64  `json:"count"`
	NextCursor *int64 `json:"next_cursor"`
}

type GetFeedResponse struct {
	Tweets     []TweetWithUser `json:"tweets"`
	Pagination FeedPagination  `json:"pagination"`
}
