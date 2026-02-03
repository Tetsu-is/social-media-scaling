# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go-based REST API for a social media application using Chi router and PostgreSQL.

## Development Commands

```bash
# Start dependencies (PostgreSQL + Swagger UI)
docker-compose up -d

# Run the application
go run main.go

# Build binary
go build -o social-media-scaling ./

# Manage dependencies
go mod tidy

# Stop dependencies
docker-compose down

# Run migrations
migrate -path ./migrations -database "postgres://user:password@localhost:5432/mydatabase?sslmode=disable" up

# Rollback one migration
migrate -path ./migrations -database "postgres://user:password@localhost:5432/mydatabase?sslmode=disable" down 1

# Create new migration
migrate create -ext sql -dir ./migrations -seq <migration_name>
```

## Architecture

Single-file Go microservice (`main.go`) with:
- Chi v5 HTTP router on port 8080
- PostgreSQL via pgx driver (localhost:5432)
- OpenAPI 3.0 spec (`openapi.yaml`) with Swagger UI on port 8081

**Database credentials (dev):** user/password, database: mydatabase

## Documentation

- `docs/spec.md` - Project requirements and features
- `docs/schema.md` - Database schema
- `docs/openapi.yaml` - API endpoint specifications (Swagger UI on port 8081)
- `docs/PERFORMANCE.md` - Performance optimization notes
- `docs/BENCHMARK_GUIDE.md` - Benchmark testing guide

## File Map

```
.
├── main.go                         # エントリポイント・ルーティング・ハンドラ
├── compose.yaml                    # Docker Compose (PostgreSQL + Swagger UI)
├── Makefile                        # ビルド・実行のタスク定義
├── go.mod / go.sum                 # Go モジュール定義・依存関係ロック
│
├── auth/
│   └── jwt.go                      # JWT認証のロジック
│
├── domain/
│   └── models.go                   # ドメインモデル（構造体・型定義）
│
├── repository/
│   ├── errors.go                   # リポジトリ共通エラー定義
│   ├── user_repository.go          # ユーザー関連のDB操作
│   ├── tweet_repository.go         # ツイート関連のDB操作
│   ├── follow_repository.go        # フォロー関連のDB操作
│   └── feed_repository.go          # フィード関連のDB操作
│
├── db/
│   └── migrations/                 # マイグレーション（000001〜000006）
│       ├── 000001  users テーブル
│       ├── 000002  user_auth テーブル
│       ├── 000003  tweets テーブル
│       ├── 000004  シードデータ
│       ├── 000005  follows テーブル
│       └── 000006  フィード用インデックス
│
├── docs/
│   ├── spec.md                     # プロジェクト仕様
│   ├── schema.md                   # DB スキーマ設計書
│   ├── openapi.yaml                # OpenAPI 3.0 仕様
│   ├── PERFORMANCE.md              # パフォーマンス最適化メモ
│   └── BENCHMARK_GUIDE.md          # ベンチマーク実行ガイド
│
├── scripts/
│   ├── benchmark_tweets.sh         # ベンチマーク実行スクリプト
│   └── generate_test_data.go       # テストデータ生成
│
└── benchmark_results/              # ベンチマーク結果（日時別）
```


## How to implement a new endpoint
1. 2-3個の設計パターンを考案
2. 既存APIとの設計の一貫性を保てる形で仕様を確定する
3. docs/openapi.yamlに仕様を記述する
4. 仕様にしたがってgoでAPIのエンドポイントを実装する
5. 修正が生じた場合はopenapi.yamlが矛盾しないように修正する
