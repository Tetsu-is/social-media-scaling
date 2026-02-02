#!/bin/bash

# tweets インデックスのベンチマークスクリプト

set -e

echo "================================"
echo "Tweets Index Benchmark"
echo "================================"
echo ""

# 色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ベンチマーク設定
API_BASE="http://localhost:8080"
REQUESTS=100
CONCURRENCY=10

# 認証トークン取得（user0でログイン）
echo "Step 1: Getting authentication token..."
TOKEN=$(curl -s -X POST "${API_BASE}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"name":"user0","password":"password"}' | jq -r '.token')

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
  echo "Failed to get token. Make sure user0 exists with password 'password'"
  echo "You may need to signup first:"
  echo "  curl -X POST ${API_BASE}/auth/signup -H 'Content-Type: application/json' -d '{\"name\":\"user0\",\"password\":\"password\"}'"
  exit 1
fi

echo "✓ Token obtained"
echo ""

# Apache Benchがインストールされているか確認
if ! command -v ab &> /dev/null; then
    echo -e "${RED}Apache Bench (ab) is not installed.${NC}"
    echo "Install it with: brew install apache2 (macOS) or apt-get install apache2-utils (Linux)"
    exit 1
fi

echo "Step 2: Warming up server..."
curl -s "${API_BASE}/tweets?count=20&cursor=0" > /dev/null
echo "✓ Server warmed up"
echo ""

# ベンチマーク関数
run_benchmark() {
  local test_name=$1
  local url=$2
  local output_file=$3

  echo -e "${YELLOW}Running: ${test_name}${NC}"

  # Apache Benchを使用
  ab -n ${REQUESTS} -c ${CONCURRENCY} -H "Authorization: Bearer ${TOKEN}" "${url}" > "${output_file}" 2>&1

  # 結果を抽出して表示
  local rps=$(grep "Requests per second" "${output_file}" | awk '{print $4}')
  local mean_time=$(grep "Time per request.*mean\)" | head -1 | awk '{print $4}')
  local p50=$(grep "50%" "${output_file}" | awk '{print $2}')
  local p95=$(grep "95%" "${output_file}" | awk '{print $2}')
  local p99=$(grep "99%" "${output_file}" | awk '{print $2}')

  echo "  Requests/sec: ${rps}"
  echo "  Mean time: ${mean_time}ms"
  echo "  P50: ${p50}ms"
  echo "  P95: ${p95}ms"
  echo "  P99: ${p99}ms"
  echo ""
}

# ベンチマーク実行
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULT_DIR="benchmark_results/${TIMESTAMP}"
mkdir -p "${RESULT_DIR}"

echo "Step 3: Running benchmarks..."
echo "Results will be saved to: ${RESULT_DIR}"
echo ""

# Test 1: GET /tweets (基本的なツイート一覧取得)
run_benchmark \
  "Test 1: GET /tweets (count=20, cursor=0)" \
  "${API_BASE}/tweets?count=20&cursor=0" \
  "${RESULT_DIR}/test1_tweets_page1.txt"

# Test 2: GET /tweets (深いページネーション)
run_benchmark \
  "Test 2: GET /tweets (count=20, cursor=1000)" \
  "${API_BASE}/tweets?count=20&cursor=1000" \
  "${RESULT_DIR}/test2_tweets_deep_page.txt"

# Test 3: GET /tweets (大量取得)
run_benchmark \
  "Test 3: GET /tweets (count=100, cursor=0)" \
  "${API_BASE}/tweets?count=100&cursor=0" \
  "${RESULT_DIR}/test3_tweets_large_count.txt"

# Test 4: GET /users/me/feed (フィード取得)
run_benchmark \
  "Test 4: GET /users/me/feed (count=20, cursor=0)" \
  "${API_BASE}/users/me/feed?count=20&cursor=0" \
  "${RESULT_DIR}/test4_feed_page1.txt"

# Test 5: GET /users/me/feed (深いページネーション)
run_benchmark \
  "Test 5: GET /users/me/feed (count=20, cursor=100)" \
  "${API_BASE}/users/me/feed?count=20&cursor=100" \
  "${RESULT_DIR}/test5_feed_deep_page.txt"

echo "================================"
echo -e "${GREEN}Benchmark completed!${NC}"
echo "Results saved to: ${RESULT_DIR}"
echo ""
echo "Next steps:"
echo "  1. Review the results above"
echo "  2. Add the index: psql postgres://user:password@localhost:5432/mydatabase -c 'CREATE INDEX idx_tweets_created_at_id ON tweets(created_at DESC, id DESC);'"
echo "  3. Run this benchmark again to compare"
echo "================================"
