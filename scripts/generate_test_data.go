package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	numUsers       = 1000
	tweetsPerUser  = 100 // 合計 100,000 件
	followsPerUser = 50  // 合計 ~50,000 件
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5432/mydatabase"
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal("parse dsn:", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Fatal("connect:", err)
	}
	defer pool.Close()

	if len(os.Args) > 1 && os.Args[1] == "--clean" {
		cleanTestData(ctx, pool)
		return
	}

	generateTestData(ctx, pool)
}

func generateTestData(ctx context.Context, pool *pgxpool.Pool) {
	// 既存テストデータチェック
	var existing int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE name LIKE 'user_%'").Scan(&existing); err != nil {
		log.Fatal("check:", err)
	}
	if existing > 0 {
		fmt.Printf("テストデータが既に存在します (%d users)。スキップ。\n", existing)
		fmt.Println("再生成する場合は `make clean-test-data` を実行してください。")
		return
	}

	// bcryptハッシュは1回だけ計算（cost=10で約100ms）
	hashedPw, err := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	if err != nil {
		log.Fatal("bcrypt:", err)
	}

	// ユーザーID生成
	userIDs := make([]string, numUsers)
	for i := range userIDs {
		id, err := uuid.NewV7()
		if err != nil {
			log.Fatal("uuid:", err)
		}
		userIDs[i] = id.String()
	}

	// --- 1. users ---
	fmt.Printf("[1/4] users (%d件)...", numUsers)
	n, err := pool.CopyFrom(ctx,
		pgx.Identifier{"users"},
		[]string{"id", "name", "created_at", "updated_at"},
		pgx.CopyFromSlice(numUsers, func(i int) ([]any, error) {
			now := time.Now()
			return []any{userIDs[i], fmt.Sprintf("user_%04d", i), now, now}, nil
		}),
	)
	if err != nil {
		log.Fatal("users:", err)
	}
	fmt.Printf(" %d件\n", n)

	// --- 2. user_auth ---
	fmt.Printf("[2/4] user_auth (%d件)...", numUsers)
	n, err = pool.CopyFrom(ctx,
		pgx.Identifier{"user_auth"},
		[]string{"user_id", "hashed_password", "created_at", "updated_at"},
		pgx.CopyFromSlice(numUsers, func(i int) ([]any, error) {
			now := time.Now()
			return []any{userIDs[i], string(hashedPw), now, now}, nil
		}),
	)
	if err != nil {
		log.Fatal("user_auth:", err)
	}
	fmt.Printf(" %d件\n", n)

	// --- 3. tweets ---
	// ラウンドロビンでユーザーに割り当て → 時系列にユーザーが均等に混ぜられる
	// 30日分に均等に分散 → created_at の分布がリアルなり
	totalTweets := numUsers * tweetsPerUser
	fmt.Printf("[3/4] tweets (%d件)...", totalTweets)

	baseTime := time.Now().Add(-30 * 24 * time.Hour)
	span := 30 * 24 * time.Hour

	n, err = pool.CopyFrom(ctx,
		pgx.Identifier{"tweets"},
		[]string{"id", "user_id", "content", "likes_count", "created_at", "updated_at"},
		pgx.CopyFromSlice(totalTweets, func(i int) ([]any, error) {
			id, err := uuid.NewV7()
			if err != nil {
				return nil, err
			}
			userIdx := i % numUsers
			offset := time.Duration(float64(span) * float64(i) / float64(totalTweets))
			createdAt := baseTime.Add(offset)
			content := fmt.Sprintf("Tweet #%d from user_%04d", i, userIdx)
			return []any{id.String(), userIDs[userIdx], content, rand.Intn(100), createdAt, createdAt}, nil
		}),
	)
	if err != nil {
		log.Fatal("tweets:", err)
	}
	fmt.Printf(" %d件\n", n)

	// --- 4. follows ---
	// 各ユーザーが rand.Perm で重複なしに followsPerUser 人をフォロー
	fmt.Printf("[4/4] follows (~%d件)...", numUsers*followsPerUser)

	type followRow struct {
		follower  string
		followee  string
		createdAt time.Time
	}
	follows := make([]followRow, 0, numUsers*followsPerUser)

	for i := range numUsers {
		perm := rand.Perm(numUsers)
		added := 0
		for _, j := range perm {
			if j == i {
				continue // 自己フォロー不可
			}
			follows = append(follows, followRow{userIDs[i], userIDs[j], time.Now()})
			added++
			if added >= followsPerUser {
				break
			}
		}
	}

	n, err = pool.CopyFrom(ctx,
		pgx.Identifier{"follows"},
		[]string{"follower_id", "followee_id", "created_at"},
		pgx.CopyFromSlice(len(follows), func(i int) ([]any, error) {
			return []any{follows[i].follower, follows[i].followee, follows[i].createdAt}, nil
		}),
	)
	if err != nil {
		log.Fatal("follows:", err)
	}
	fmt.Printf(" %d件\n", n)

	fmt.Println("\nDone!")
	fmt.Printf("  users: %d, tweets: %d, follows: %d\n", numUsers, totalTweets, len(follows))
}

func cleanTestData(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Print("テストデータを削除する...")

	// tweets は user_id に CASCADE がないため先に削除
	// users 削除後に follows・user_auth は CASCADE で自動削除
	queries := []string{
		"DELETE FROM tweets WHERE user_id IN (SELECT id FROM users WHERE name LIKE 'user_%')",
		"DELETE FROM users WHERE name LIKE 'user_%'",
	}

	for _, q := range queries {
		ct, err := pool.Exec(ctx, q)
		if err != nil {
			log.Fatal("clean:", err)
		}
		fmt.Printf(" %s (%d件)", ct.String(), ct.RowsAffected())
	}
	fmt.Println("\nDone!")
}
