# Performance Tuning Records

このドキュメントでは、パフォーマンスチューニングの実施内容と結果を記録します。

## 目的

- パフォーマンス改善の履歴を残す
- 問題と解決策を文書化する
- 今後の参考資料とする
- ベンチマーク結果を比較可能にする

## ベンチマークの実施方法

### Go標準のベンチマーク
```bash
# 全体のベンチマーク実行
go test -bench=. -benchmem

# 特定のベンチマーク実行
go test -bench=BenchmarkXXX -benchmem

# CPU/メモリプロファイリング
go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof
go tool pprof cpu.prof
```

### 負荷テスト (wrk/k6など)
```bash
# wrkを使用した負荷テスト例
wrk -t4 -c100 -d30s http://localhost:8080/api/endpoint

# Apache Benchを使用した例
ab -n 10000 -c 100 http://localhost:8080/api/endpoint
```

### PostgreSQL クエリ分析
```sql
-- クエリ実行計画の確認
EXPLAIN ANALYZE SELECT ...;

-- 実行中のクエリ確認
SELECT * FROM pg_stat_activity;

-- スロークエリの特定
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
```

---

## チューニング履歴

### テンプレート
```markdown
### [YYYY-MM-DD] タイトル

**問題・課題**
- 問題の詳細を記載

**原因分析**
- 問題の根本原因

**実施した対策**
1. 対策1
2. 対策2

**結果**
- Before: 具体的な数値
- After: 具体的な数値
- 改善率: XX%

**ベンチマーク結果**
\`\`\`
ベンチマーク結果を貼り付け
\`\`\`

**学び・備考**
- 今後の参考になる情報
```

---

## チューニング実施記録

### [2026-02-02] tweets(created_at, id) 複合インデックスの追加

**問題・課題**
- ツイート一覧取得API (GET /tweets) のレスポンスタイム改善の検討
- ORDER BY created_at DESC, id DESC を使用したクエリの最適化
- データ量増加に備えたインデックス戦略の策定

**原因分析**
- tweetsテーブルに created_at と id のソート用インデックスが存在しない
- ORDER BY created_at DESC, id DESC の実行時にフルテーブルスキャン + ソートが発生する可能性
- ページネーション（OFFSET/LIMIT）の深いページで性能劣化が予想される

**実施した対策**
1. テストデータ生成（Users: 1,000人、Tweets: 100,000件、Follows: 10,000件）
2. インデックス追加前後でApache Benchによるベンチマーク実施
   - リクエスト数: 100、同時接続数: 10
3. 以下のインデックスを追加:
   ```sql
   CREATE INDEX idx_tweets_created_at_id ON tweets(created_at DESC, id DESC);
   ```
4. EXPLAIN ANALYZEで実行計画を確認

**結果**

| テストケース | Before (req/sec) | After (req/sec) | Before (ms) | After (ms) | 改善率 |
|-------------|------------------|-----------------|-------------|-----------|--------|
| Test 1: GET /tweets (count=20, cursor=0) | 6,116.96 | 11,277.77 | 1.635 | 0.887 | **+84.4%** ✓ |
| Test 2: GET /tweets (count=20, cursor=1000) | 11,784.12 | 11,941.72 | 0.849 | 0.837 | +1.3% |
| Test 3: GET /tweets (count=100, cursor=0) | 13,312.03 | 12,310.72 | 0.751 | 0.812 | -7.5% ✗ |
| Test 4: GET /users/me/feed (count=20, cursor=0) | 13,262.60 | 13,424.62 | 0.754 | 0.745 | +1.2% |
| Test 5: GET /users/me/feed (count=20, cursor=100) | 12,894.91 | 11,981.79 | 0.775 | 0.835 | -7.1% ✗ |

**EXPLAIN ANALYZE 結果**

インデックスが正しく使用されていることを確認:

```sql
-- Test 1: 最初のページ取得（LIMIT 20）
EXPLAIN ANALYZE
SELECT id, user_id, content, likes_count, created_at, updated_at
FROM tweets
ORDER BY created_at DESC, id DESC
LIMIT 20;

-- 結果: Index Scan using idx_tweets_created_at_id
-- Execution Time: 0.112 ms
```

```sql
-- Test 2: 深いページネーション（OFFSET 1001）
EXPLAIN ANALYZE
SELECT id, user_id, content, likes_count, created_at, updated_at
FROM tweets
ORDER BY created_at DESC, id DESC
OFFSET 1001 LIMIT 20;

-- 結果: Index Scan using idx_tweets_created_at_id
-- Execution Time: 0.607 ms （1021行をスキャン）
```

```sql
-- Test 3: フィードクエリ（JOINあり）
EXPLAIN ANALYZE
SELECT t.id, t.user_id, t.content, t.likes_count, t.created_at, t.updated_at,
       u.id, u.name, u.created_at, u.updated_at
FROM tweets t
INNER JOIN follows f ON t.user_id = f.followee_id
INNER JOIN users u ON t.user_id = u.id
WHERE f.follower_id = '019c1ec4-7504-70f2-9ded-511c2a5ef1e6'
ORDER BY t.created_at DESC, t.id DESC
LIMIT 20;

-- 結果: Index Scan using idx_tweets_created_at_id （Nested Loop内）
-- Execution Time: 3.500 ms （1999行をスキャン）
```

**学び・備考**

**ポジティブな結果:**
- 最初のページ取得（Test 1）で84.4%の大幅な高速化を達成
- インデックスは正しく使用されており、EXPLAIN ANALYZEで確認済み
- LIMIT値が小さい場合（20件程度）では明確な効果がある

**ネガティブな結果:**
- 大量取得（LIMIT 100）では逆に7.5%性能が低下
- 深いページのフィード取得でも7.1%性能が低下
- データ量10万件では効果が限定的なケースがある

**原因の考察:**
1. **データ量が少ない**: 10万件では、PostgreSQLのキャッシュが効きやすく、フルスキャンでも高速
2. **キャッシュの影響**: ベンチマーク1回目でデータがメモリに乗り、2回目は有利な状態だった可能性
3. **OFFSETの非効率性**: OFFSET + LIMITは深いページほど非効率（カーソルベースページネーションの検討が必要）
4. **JOINクエリの最適化不足**: フィードクエリは1999行をスキャンしており、クエリ自体の改善余地あり

**今後の対応:**
- ✓ インデックスは追加（最初のページで効果確認）
- データ量を100万件〜1000万件に増やして再測定を検討
- tweets(user_id) 単一インデックスの追加を検討（フィードクエリの最適化）
- カーソルベースページネーション（max_id方式）の実装を検討
- フィードクエリのJOIN順序最適化を検討

**詳細レポート:**
- benchmark_results/comparison.md
- docs/BENCHMARK_GUIDE.md

_(ここに今後のチューニング結果を追記していきます)_

<!--
最新のものを上に追記してください

例:
### [2026-02-02] データベースインデックスの追加

**問題・課題**
- ユーザー検索APIのレスポンスタイムが500ms以上かかっていた
- 同時接続数が100を超えるとタイムアウトが発生

**原因分析**
- users テーブルの username カラムにインデックスが設定されていなかった
- フルテーブルスキャンが発生していた

**実施した対策**
1. username カラムに B-tree インデックスを追加
2. 検索クエリの最適化

**結果**
- Before: 平均レスポンスタイム 520ms
- After: 平均レスポンスタイム 45ms
- 改善率: 91.3%

**ベンチマーク結果**
```
wrk -t4 -c100 -d30s http://localhost:8080/api/users/search?username=test

Before:
  Requests/sec: 192.34
  Latency: avg 520ms, max 2.1s

After:
  Requests/sec: 2234.56
  Latency: avg 45ms, max 180ms
```

**学び・備考**
- 頻繁に検索されるカラムには必ずインデックスを設定する
- EXPLAIN ANALYZE で実行計画を確認する習慣をつける
-->
