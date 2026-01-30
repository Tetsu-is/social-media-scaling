# Database Schema

Migrations managed with [golang-migrate](https://github.com/golang-migrate/migrate). Migration files are in `/db/migrations`.

## Users Table

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_name ON users(name);
```

### Fields

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | UUID v7 (アプリ側で生成) |
| name | VARCHAR(255) | NOT NULL, UNIQUE | User's unique name |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT NOW() | Account creation time |
| updated_at | TIMESTAMP WITH TIME ZONE | DEFAULT NOW() | Account last update time |

### UUID v7について

- 128ビット（PostgreSQL UUID型: 16バイト固定）
- 時刻ベース + ランダム → ソート可能かつ分散環境でも衝突しにくい
- アプリ側（Go）で生成し、INSERTする
- 文字列表現: 36文字（ハイフン含む）例: `0190a5e4-b890-7000-8000-000000000001`


## UserAuth Table

```sql
CREATE TABLE user_auth (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    hashed_password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Fields

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| user_id | UUID | PRIMARY KEY, REFERENCES users(id) | usersテーブルとの1対1対応 |
| hashed_password | VARCHAR(255) | NOT NULL | Bcrypt hashed password |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT NOW() | Record creation time |
| updated_at | TIMESTAMP WITH TIME ZONE | DEFAULT NOW() | Record last update time |


## Tweets Table

```sql
CREATE TABLE tweets (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    content VARCHAR(255) NOT NULL,
    likes_count INTEGER NOT NULL DEFAULT 0 CHECK (likes_count >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Fields

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | UUID v7 (アプリ側で生成) |
| user_id | UUID | NOT NULL, REFERENCES users(id) | 投稿者のユーザーID |
| content | VARCHAR(255) | NOT NULL | ツイート本文（最大255文字） |
| likes_count | INTEGER | NOT NULL, DEFAULT 0, CHECK >= 0 | いいね数（非負整数） |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT NOW() | ツイート作成日時 |
| updated_at | TIMESTAMP WITH TIME ZONE | DEFAULT NOW() | ツイート更新日時 |


## Follows Table

```sql
CREATE TABLE follows (
    follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    followee_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (follower_id, followee_id),
    CHECK (follower_id != followee_id)
);

CREATE INDEX idx_follows_follower ON follows(follower_id);
CREATE INDEX idx_follows_followee ON follows(followee_id);
```

### Fields

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| follower_id | UUID | NOT NULL, REFERENCES users(id), PK | フォローする側のユーザーID |
| followee_id | UUID | NOT NULL, REFERENCES users(id), PK | フォローされる側のユーザーID |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT NOW() | フォロー日時 |

### Constraints

- **複合主キー**: `(follower_id, followee_id)` で同じペアの重複フォローを防止
- **CHECK制約**: 自分自身をフォローすることを禁止
- **ON DELETE CASCADE**: ユーザー削除時にフォロー関係も自動削除
