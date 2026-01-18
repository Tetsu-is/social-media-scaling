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
