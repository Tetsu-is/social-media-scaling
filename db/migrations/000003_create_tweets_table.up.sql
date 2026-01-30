CREATE TABLE IF NOT EXISTS tweets (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  content VARCHAR(255) NOT NULL,
  likes_count INTEGER NOT NULL DEFAULT 0 CHECK (likes_count >= 0),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- CREATE INDEX idx_tweets_posted_by ON tweets(user_id)
-- 後でインデックス張ってみて性能を比べてみる
