package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	numUsers  = 1000    // 1,000ユーザー
	numTweets = 100000  // 100,000ツイート（ユーザーあたり平均100件）
	batchSize = 1000    // バッチサイズ
)

func main() {
	dsn := "postgres://user:password@localhost:5432/mydatabase"

	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}
	defer conn.Close(context.Background())

	ctx := context.Background()

	// 既存データをクリア
	fmt.Println("Clearing existing data...")
	_, err = conn.Exec(ctx, "TRUNCATE tweets, follows, user_auth, users CASCADE")
	if err != nil {
		log.Fatal("failed to truncate tables: ", err)
	}

	// ユーザー生成
	fmt.Printf("Generating %d users with auth...\n", numUsers)
	userIDs := make([]string, numUsers)

	// 全ユーザーのパスワードは "password" でハッシュ化
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("failed to hash password: ", err)
	}

	for i := 0; i < numUsers; i++ {
		id, _ := uuid.NewV7()
		userIDs[i] = id.String()

		// usersテーブルに挿入
		_, err := conn.Exec(ctx,
			"INSERT INTO users (id, name, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())",
			id.String(), fmt.Sprintf("user%d", i),
		)
		if err != nil {
			log.Printf("failed to insert user %d: %v", i, err)
			continue
		}

		// user_authテーブルに挿入
		_, err = conn.Exec(ctx,
			"INSERT INTO user_auth (user_id, hashed_password, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())",
			id.String(), string(hashedPassword),
		)
		if err != nil {
			log.Printf("failed to insert user_auth %d: %v", i, err)
			continue
		}

		if (i+1)%100 == 0 {
			fmt.Printf("  Created %d users\n", i+1)
		}
	}
	fmt.Printf("✓ Created %d users\n", numUsers)

	// ツイート生成（バッチ処理）
	fmt.Printf("Generating %d tweets in batches of %d...\n", numTweets, batchSize)

	for batch := 0; batch < numTweets/batchSize; batch++ {
		// トランザクション開始
		tx, err := conn.Begin(ctx)
		if err != nil {
			log.Fatal("failed to begin transaction: ", err)
		}

		for i := 0; i < batchSize; i++ {
			tweetID, _ := uuid.NewV7()
			userID := userIDs[i%numUsers] // ユーザーをラウンドロビンで選択
			content := fmt.Sprintf("This is test tweet #%d from user index %d", batch*batchSize+i, i%numUsers)

			// 作成日時を少しずつずらす（古いツイートから新しいツイートへ）
			createdAt := time.Now().Add(-time.Duration(numTweets-(batch*batchSize+i)) * time.Second)

			_, err := tx.Exec(ctx,
				"INSERT INTO tweets (id, user_id, content, likes_count, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
				tweetID.String(), userID, content, 0, createdAt, createdAt,
			)
			if err != nil {
				tx.Rollback(ctx)
				log.Fatalf("failed to insert tweet: %v", err)
			}
		}

		// トランザクションコミット
		err = tx.Commit(ctx)
		if err != nil {
			log.Fatal("failed to commit transaction: ", err)
		}

		fmt.Printf("  Created %d tweets\n", (batch+1)*batchSize)
	}
	fmt.Printf("✓ Created %d tweets\n", numTweets)

	// フォロー関係生成（各ユーザーが10人をフォロー）
	fmt.Println("Generating follow relationships...")
	followCount := 0
	for i := 0; i < numUsers; i++ {
		for j := 1; j <= 10; j++ {
			followerID := userIDs[i]
			followeeID := userIDs[(i+j)%numUsers] // 自分以外をフォロー

			if followerID == followeeID {
				continue
			}

			_, err := conn.Exec(ctx,
				"INSERT INTO follows (follower_id, followee_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT DO NOTHING",
				followerID, followeeID,
			)
			if err != nil {
				log.Printf("failed to insert follow: %v", err)
				continue
			}
			followCount++
		}

		if (i+1)%100 == 0 {
			fmt.Printf("  Created follows for %d users\n", i+1)
		}
	}
	fmt.Printf("✓ Created %d follow relationships\n", followCount)

	// 統計情報を表示
	fmt.Println("\n=== Test Data Summary ===")
	var userCount, tweetCount, followsCount int64
	conn.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	conn.QueryRow(ctx, "SELECT COUNT(*) FROM tweets").Scan(&tweetCount)
	conn.QueryRow(ctx, "SELECT COUNT(*) FROM follows").Scan(&followsCount)

	fmt.Printf("Users: %d\n", userCount)
	fmt.Printf("Tweets: %d\n", tweetCount)
	fmt.Printf("Follows: %d\n", followsCount)
	fmt.Println("=========================")
}
