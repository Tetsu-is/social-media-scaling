package repository

import (
	"context"

	"github.com/Tetsu-is/social-media-scaling/domain"
	"github.com/jackc/pgx/v5"
)

type FollowRepository struct {
	conn *pgx.Conn
}

func NewFollowRepository(conn *pgx.Conn) *FollowRepository {
	return &FollowRepository{conn: conn}
}

func (r *FollowRepository) CreateFollow(ctx context.Context, followerID, followeeID string) error {
	_, err := r.conn.Exec(ctx,
		"INSERT INTO follows (follower_id, followee_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		followerID, followeeID,
	)
	return err
}

func (r *FollowRepository) DeleteFollow(ctx context.Context, followerID, followeeID string) error {
	_, err := r.conn.Exec(ctx,
		"DELETE FROM follows WHERE follower_id = $1 AND followee_id = $2",
		followerID, followeeID,
	)
	return err
}

func (r *FollowRepository) GetFollowers(ctx context.Context, userID string) ([]domain.User, error) {
	rows, err := r.conn.Query(ctx,
		`SELECT u.id, u.name, u.created_at, u.updated_at
		 FROM users u
		 INNER JOIN follows f ON u.id = f.follower_id
		 WHERE f.followee_id = $1
		 ORDER BY f.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.ID, &user.Name, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *FollowRepository) GetFollowees(ctx context.Context, userID string) ([]domain.User, error) {
	rows, err := r.conn.Query(ctx,
		`SELECT u.id, u.name, u.created_at, u.updated_at
		 FROM users u
		 INNER JOIN follows f ON u.id = f.followee_id
		 WHERE f.follower_id = $1
		 ORDER BY f.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.ID, &user.Name, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
