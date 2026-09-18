#!/usr/bin/env bash
# Перезапуск dev-сервисов similar-items.
# Использование: ./scripts/restart-dev.sh [backend|frontend|all]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

TARGET="${1:-all}"

kill_port() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1 && lsof -ti:"$port" >/dev/null 2>&1; then
    echo "Останавливаю процессы на порту $port..."
    lsof -ti:"$port" | xargs kill -9 2>/dev/null || true
    sleep 1
  fi
}

start_backend() {
  kill_port 8080
  echo "Запуск backend → http://localhost:8080"
  DATABASE_URL="${DATABASE_URL:-postgresql://similar:similar@localhost:5434/similar?sslmode=disable}" \
    FEED_SYNC_INTERVAL="${FEED_SYNC_INTERVAL:-1h}" \
    go run ./cmd/similar
}

start_frontend() {
  kill_port 3000
  echo "Запуск frontend → http://localhost:3000 (proxy → :8080)"
  cd frontend && npm start
}

case "$TARGET" in
  backend)
    start_backend
    ;;
  frontend)
    start_frontend
    ;;
  all)
    kill_port 8080
    kill_port 3000
    echo "Порты 8080 и 3000 освобождены. Запустите в двух терминалах:"
    echo "  make run"
    echo "  make frontend-dev"
    ;;
  *)
    echo "Usage: $0 [backend|frontend|all]" >&2
    exit 1
    ;;
esac
