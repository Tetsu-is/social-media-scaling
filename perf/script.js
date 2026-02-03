import http from 'k6/http'
import { check } from 'k6'
import { Counter, Rate } from 'k6/metrics'

// 目的: MaxConns=1 + インデックス無し の条件で、GET /tweets がどこで壊れるかを特定する。
//
// なぜ sleep しない:
//   sleep があると各 VU のリクエスト頻度が落ち、pgxpool の待機キューが圧迫されにくくなる。
//   sleep なしで連続リクエストを送り、シングル接続のキューを最大限に詰ませる。
//
// なぜ timeout を 5s に設定:
//   pgxpool の Acquire にタイムアウトは設定されていないため、キューが詰まると無視不能で待機する。
//   k6 側で HTTP タイムアウトを設定し、「待機時間がこの閾値を超えた」という壊れるポイントを検出する。

const timeouts = new Counter('timeouts_total')
const errors = new Rate('error_rate')

export const options = {
  // 5段階で負荷を上げていき、各ステージで十分な観測時間を確保する。
  // MaxConns=1 では並行リクエスト数がそのままキュー長になるため、
  // VU 数の増加に連動してレイテンシが線形に伸びていくはず。
  stages: [
    { duration: '15s', target: 10 },   // ウォームアップ       — キューにほぼ待機なし
    { duration: '30s', target: 50 },   // 軽負荷             — レイテンシ上昇の黎明期
    { duration: '45s', target: 150 },  // 中負荷             — レイテンシ急騰の開始を観測
    { duration: '45s', target: 300 },  // 重負荷             — 前回のmax、キュー蓄積の確認
    { duration: '45s', target: 600 },  // ストレス           — キューが詰まり開始を期待
    { duration: '45s', target: 1000 }, // 強ストレス          — タイムアウト・エラー率の急騰
    { duration: '30s', target: 1500 }, // 破綻テスト          — エラーが爆発するポイントを特定
    { duration: '30s', target: 1500 }, // 破綻維持           — キュー最大状態を長く観測
    { duration: '20s', target: 0 },    // クールダウン
  ],

  // 閾値を意図的に厳しくし、ブレーキポイントに到達したら明示的に FAIL を出す。
  // テスト結果サマリで「どのステージで閾値を突破したか」がすぐに読める。
  thresholds: {
    'http_req_duration{status:200}': ['p(95)<3000'], // 成功リクエストの p95 が 3s を超えたら FAIL
    'error_rate': ['rate<0.05'],  // エラー率が 5% を超えたら FAIL
  },
}

export default function () {
  const res = http.get('http://localhost:8080/tweets?count=20', {
    timeout: '5000ms',
  })

  // status === 0 は k6 のタイムアウト（レスポンス自体が返っていない）
  if (res.status === 0) {
    timeouts.add(1)
  }

  errors.add(res.status !== 200 ? 1 : 0)

  check(res, {
    'status 200': (r) => r.status === 200,
    'has tweets array': (r) => {
      if (r.status !== 200) return false
      try {
        return Array.isArray(JSON.parse(r.body).tweets)
      } catch (_) {
        return false
      }
    },
  })
}
