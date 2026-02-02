package repository

import (
	"context"

	"github.com/Tetsu-is/social-media-scaling/domain"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository struct {
	conn *pgx.Conn
}

func NewUserRepository(conn *pgx.Conn) *UserRepository {
	return &UserRepository{conn: conn}
}

func (r *UserRepository) CreateUser(ctx context.Context, userID, name, password string) (*domain.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, err
	}

	tx, err := r.conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var user domain.User
	err = tx.QueryRow(ctx,
		"INSERT INTO users (id, name) VALUES ($1, $2) RETURNING id, name, created_at, updated_at",
		userID, name,
	).Scan(&user.ID, &user.Name, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return nil, ErrDuplicateUser
			}
		}
		return nil, err
	}

	_, err = tx.Exec(ctx,
		"INSERT INTO user_auth (user_id, hashed_password) VALUES ($1, $2)",
		userID, hashedPassword,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetUserByName(ctx context.Context, name string) (*domain.User, error) {
	var user domain.User
	err := r.conn.QueryRow(ctx,
		"SELECT id, name, created_at, updated_at FROM users WHERE name = $1",
		name,
	).Scan(&user.ID, &user.Name, &user.CreatedAt, &user.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	} else if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	var user domain.User
	err := r.conn.QueryRow(ctx,
		"SELECT id, name, created_at, updated_at FROM users WHERE id = $1",
		userID,
	).Scan(&user.ID, &user.Name, &user.CreatedAt, &user.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	} else if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetUserAuth(ctx context.Context, userID string) (*domain.UserAuth, error) {
	var userAuth domain.UserAuth
	err := r.conn.QueryRow(ctx,
		"SELECT user_id, hashed_password, created_at, updated_at FROM user_auth WHERE user_id = $1",
		userID,
	).Scan(&userAuth.UserID, &userAuth.HashedPassword, &userAuth.CreatedAt, &userAuth.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &userAuth, nil
}

func (r *UserRepository) CheckUserExists(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := r.conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	return exists, err
}

func (r *UserRepository) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
