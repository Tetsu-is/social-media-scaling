# Database Schema

Migrations managed with [golang-migrate](https://github.com/golang-migrate/migrate). Migration files are in `/migrations`.

## Users Table

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
```

### Fields

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | BIGSERIAL | PRIMARY KEY | Auto-incrementing user ID |
| username | VARCHAR(255) | NOT NULL, UNIQUE | User's unique username |
| password | VARCHAR(255) | NOT NULL | Hashed password |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT NOW() | Account creation time |
