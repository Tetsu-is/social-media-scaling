package repository

import (
	"context"

	"github.com/Tetsu-is/social-media-scaling/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FeedRepository struct {
	conn *pgxpool.Pool
}

func NewFeedRepository(conn *pgxpool.Pool) *FeedRepository {
	return &FeedRepository{conn: conn}
}

// GetFeedTweets はログインユーザーがフォローしているユーザーのツイートを取得する
// Pull型ニュースフィード実装（OFFSET/LIMITベースページネーション）
func (r *FeedRepository) GetFeedTweets(ctx context.Context, userID string, offset int64, count int64) ([]domain.TweetWithUser, error) {
	query := `
		SELECT
			t.id,
			t.user_id,
			t.content,
			t.likes_count,
			t.created_at,
			t.updated_at,
			u.id,
			u.name,
			u.created_at,
			u.updated_at
		FROM tweets t
		INNER JOIN follows f ON t.user_id = f.followee_id
		INNER JOIN users u ON t.user_id = u.id
		WHERE f.follower_id = $1
		ORDER BY t.created_at DESC, t.id DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.conn.Query(ctx, query, userID, count, offset)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tweets []domain.TweetWithUser
	for rows.Next() {
		var tweet domain.TweetWithUser
		err := rows.Scan(
			&tweet.ID,
			&tweet.UserID,
			&tweet.Content,
			&tweet.LikesCount,
			&tweet.CreatedAt,
			&tweet.UpdatedAt,
			&tweet.User.ID,
			&tweet.User.Name,
			&tweet.User.CreatedAt,
			&tweet.User.UpdatedAt,
		)
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
