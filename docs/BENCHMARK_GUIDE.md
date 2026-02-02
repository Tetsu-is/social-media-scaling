# tweets(created_at, id) インデックス ベンチマークガイド

このドキュメントでは、tweetsテーブルの`created_at DESC, id DESC`インデックスの効果を測定する手順を説明します。

## 測定対象

### 対象インデックス
```sql
CREATE INDEX idx_tweets_created_at_id ON tweets(created_at DESC, id DESC);
```

### 影響を受けるAPI
1. **GET /tweets** - ツイート一覧取得
   - クエリ: `ORDER BY created_at DESC OFFSET $1 LIMIT $2`
   - repository/tweet_repository.go:41

2. **GET /users/me/feed** - フィード取得
   - クエリ: `ORDER BY t.created_at DESC, t.id DESC`
   - repository/feed_repository.go:37

## 前提条件

- PostgreSQLが起動している（docker-compose up -d）
- アプリケーションが起動している（go run main.go）
- Apache Bench (ab) がインストールされている
  - macOS: `brew install apache2`
  - Linux: `apt-get install apache2-utils`
- jq がインストールされている
  - macOS: `brew install jq`
  - Linux: `apt-get install jq`

## ベンチマーク手順

### Step 1: テストデータ生成

10万件のツイートを生成します。

```bash
# スクリプトディレクトリに移動
cd /Users/tetsuro/dev/social-media-scaling

# テストデータ生成スクリプトを実行
go run scripts/generate_test_data.go
```

実行後、以下のデータが生成されます：
- Users: 1,000人
- Tweets: 100,000件
- Follows: 約10,000件

### Step 2: インデックスなしでベンチマーク実行

```bash
# アプリケーションが起動していることを確認
# 別ターミナルで: go run main.go

# ベンチマークスクリプトを実行
./scripts/benchmark_tweets.sh
```

結果は `benchmark_results/YYYYMMDD_HHMMSS/` ディレクトリに保存されます。

### Step 3: 現在のインデックス状況を確認

```bash
psql postgres://user:password@localhost:5432/mydatabase -c "
SELECT
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'tweets'
ORDER BY indexname;"
```

### Step 4: インデックスを追加

```bash
psql postgres://user:password@localhost:5432/mydatabase -c "
CREATE INDEX idx_tweets_created_at_id ON tweets(created_at DESC, id DESC);"
```

### Step 5: インデックスありでベンチマーク実行

```bash
# 再度ベンチマークを実行
./scripts/benchmark_tweets.sh
```

### Step 6: 結果を比較

2つのベンチマーク結果を比較します：

```bash
# 最新の2つのベンチマーク結果を表示
ls -lt benchmark_results/ | head -3
```

各ディレクトリの結果ファイルを確認し、以下の指標を比較：
- **Requests/sec**: リクエスト処理数（高いほど良い）
- **Mean time**: 平均レスポンスタイム（低いほど良い）
- **P50/P95/P99**: パーセンタイルレスポンスタイム（低いほど良い）

### Step 7: EXPLAIN ANALYZEで実行計画を確認

インデックスが実際に使用されているか確認します。

```bash
# インデックスなしの実行計画（インデックスを削除後）
psql postgres://user:password@localhost:5432/mydatabase -c "
EXPLAIN ANALYZE
SELECT id, user_id, content, likes_count, created_at, updated_at
FROM tweets
ORDER BY created_at DESC, id DESC
LIMIT 20;"

# インデックスありの実行計画（インデックス追加後）
psql postgres://user:password@localhost:5432/mydatabase -c "
EXPLAIN ANALYZE
SELECT id, user_id, content, likes_count, created_at, updated_at
FROM tweets
ORDER BY created_at DESC, id DESC
LIMIT 20;"
```

期待される変化：
- **Before**: `Sort` + `Seq Scan` (フルテーブルスキャン + ソート)
- **After**: `Index Scan using idx_tweets_created_at_id` (インデックススキャン)

### Step 8: 結果をPERFORMANCE.mdに記録

ベンチマーク結果を `docs/PERFORMANCE.md` に記録します。

## ベンチマーク結果の見方

### 良い結果の例
```
Before (インデックスなし):
  Requests/sec: 150
  Mean time: 66ms
  P95: 120ms

After (インデックスあり):
  Requests/sec: 800
  Mean time: 12ms
  P95: 25ms

改善率: 5.3倍高速化
```

### 主な評価指標

| 指標 | 説明 | 目標 |
|------|------|------|
| Requests/sec | 1秒あたりの処理可能リクエスト数 | 増加 |
| Mean time | 平均レスポンスタイム | 減少 |
| P95 | 95%のリクエストが完了する時間 | 減少 |
| P99 | 99%のリクエストが完了する時間 | 減少 |

## トラブルシューティング

### テストデータ生成が失敗する
- PostgreSQLが起動しているか確認: `docker-compose ps`
- 既存のマイグレーションが適用されているか確認

### ベンチマークでトークン取得に失敗する
- user0が存在しない場合、以下のコマンドでサインアップ:
```bash
curl -X POST http://localhost:8080/auth/signup \
  -H 'Content-Type: application/json' \
  -d '{"name":"user0","password":"password"}'
```

### Apache Benchがインストールされていない
```bash
# macOS
brew install apache2

# Linux (Ubuntu/Debian)
sudo apt-get install apache2-utils
```

## 次のステップ

1. インデックスの効果が確認できたら、マイグレーションファイルを作成
2. 本番環境への適用計画を立てる
3. 他のインデックス候補（tweets(user_id)など）も同様に検証
