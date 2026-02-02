-- インデックス追加マイグレーション（性能比較用）
-- ニュースフィードAPIのパフォーマンステストのため、現時点ではコメントアウト

-- ツイートの作成日時降順インデックス
-- ニュースフィードで最新のツイートを効率的に取得するために使用
-- CREATE INDEX idx_tweets_created_at ON tweets(created_at DESC);

-- ツイートのユーザーID + 作成日時の複合インデックス
-- 特定ユーザーのツイートを作成日時順で取得する際に最適化
-- CREATE INDEX idx_tweets_user_created ON tweets(user_id, created_at DESC);

-- 性能比較の手順:
-- 1. インデックスなしでフィードAPIのレスポンスタイムを計測
-- 2. 上記のCREATE INDEX文のコメントを外して実行
-- 3. インデックスありでフィードAPIのレスポンスタイムを計測
-- 4. 結果を比較
