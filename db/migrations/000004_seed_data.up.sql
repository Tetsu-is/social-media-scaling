-- Seed users
INSERT INTO users (id, name, created_at, updated_at) VALUES
  ('11111111-1111-1111-1111-111111111111', 'alice', NOW(), NOW()),
  ('22222222-2222-2222-2222-222222222222', 'bob', NOW(), NOW()),
  ('33333333-3333-3333-3333-333333333333', 'charlie', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Seed user_auth (password: "password123" hashed with bcrypt)
INSERT INTO user_auth (user_id, hashed_password, created_at, updated_at) VALUES
  ('11111111-1111-1111-1111-111111111111', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', NOW(), NOW()),
  ('22222222-2222-2222-2222-222222222222', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', NOW(), NOW()),
  ('33333333-3333-3333-3333-333333333333', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', NOW(), NOW())
ON CONFLICT (user_id) DO NOTHING;

-- Seed tweets
INSERT INTO tweets (id, user_id, content, likes_count, created_at, updated_at) VALUES
  ('aaaaaaa1-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'Hello, this is my first tweet!', 5, NOW(), NOW()),
  ('aaaaaaa2-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'Learning Go is fun!', 10, NOW(), NOW()),
  ('bbbbbbb1-0000-0000-0000-000000000001', '22222222-2222-2222-2222-222222222222', 'Just joined this platform!', 3, NOW(), NOW()),
  ('bbbbbbb2-0000-0000-0000-000000000002', '22222222-2222-2222-2222-222222222222', 'Building something cool today.', 8, NOW(), NOW()),
  ('ccccccc1-0000-0000-0000-000000000001', '33333333-3333-3333-3333-333333333333', 'PostgreSQL is awesome!', 15, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

