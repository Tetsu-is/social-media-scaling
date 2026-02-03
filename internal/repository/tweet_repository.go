package repository

import (
	"context"

	"github.com/Tetsu-is/social-media-scaling/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TweetRepository struct {
	conn *pgxpool.Pool
}

func NewTweetRepository(conn *pgxpool.Pool) *TweetRepository {
	return &TweetRepository{conn: conn}
}

func (r *TweetRepository) CreateTweet(ctx context.Context, tweetID, userID, content string) (*domain.Tweet, error) {
	var tweet domain.Tweet
	err := r.conn.QueryRow(ctx,
		"INSERT INTO tweets (id, user_id, content) VALUES ($1, $2, $3) RETURNING id, user_id, content, likes_count, created_at, updated_at",
		tweetID, userID, content,
	).Scan(&tweet.ID, &tweet.UserID, &tweet.Content, &tweet.LikesCount, &tweet.CreatedAt, &tweet.UpdatedAt)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return nil, ErrDuplicateTweet
			}
		}
		return nil, err
	}

	return &tweet, nil
}

func (r *TweetRepository) GetTweetsByCursor(ctx context.Context, cursor, count int64) ([]domain.Tweet, error) {
	rows, err := r.conn.Query(ctx,
		"SELECT id, user_id, content, likes_count, created_at, updated_at FROM tweets ORDER BY created_at DESC OFFSET $1 LIMIT $2",
		cursor+1, count,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tweets []domain.Tweet
	for rows.Next() {
		var tweet domain.Tweet
		err := rows.Scan(&tweet.ID, &tweet.UserID, &tweet.Content, &tweet.LikesCount, &tweet.CreatedAt, &tweet.UpdatedAt)
		if err != nil {
			return nil, err
		}
		tweets = append(tweets, tweet)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tweets, nil
}

func (r *TweetRepository) GetTweetsByMaxID(ctx context.Context, maxID uuid.UUID, count int64) ([]domain.Tweet, error) {
	// 未実装
	return nil, ErrNotImplemented
}
