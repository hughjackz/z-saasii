#!/usr/bin/env bash
# run.sh — 一键启动/停止 CSMS 前端和后端
#
# 用法:  ./run.sh            # 启动前后端（默认 start）
#        ./run.sh start      # 启动
#        ./run.sh stop       # 停止
#        ./run.sh restart    # 重启
#
# 后端:  Go (backend/bin/csms-backend)，工作目录 backend/（依赖 ./config、resource/、mylog/ 相对路径）
#        监听 :8080 (HTTP API) 和 :9101 (OCPP WebSocket)
# 前端:  Vite dev server (frondend/)，监听 :5690，/api 代理到 http://10.0.0.5:8080
# 日志:  logs/backend.log、logs/frontend.log

set -u

ROOT="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$ROOT/backend"
FRONTEND_DIR="$ROOT/frondend"
LOG_DIR="$ROOT/logs"
BACKEND_LOG="$LOG_DIR/backend.log"
FRONTEND_LOG="$LOG_DIR/frontend.log"
BACKEND_PID="$LOG_DIR/backend.pid"
FRONTEND_PID="$LOG_DIR/frontend.pid"
BACKEND_BIN="$BACKEND_DIR/bin/csms-backend"

mkdir -p "$LOG_DIR"

is_running() { # $1: pidfile
  [ -f "$1" ] && kill -0 "$(cat "$1" 2>/dev/null)" 2>/dev/null
}

stop_one() { # $1: pidfile  $2: 服务名
  if [ -f "$1" ]; then
    local pid
    pid="$(cat "$1" 2>/dev/null)"
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
      # 服务经 setsid 运行在自己的会话中（pgid == pid），按进程组终止可覆盖全部子进程
      kill -TERM -- "-$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null
      # 等待退出，超时后强杀
      local i
      for i in 1 2 3 4; do
        kill -0 "$pid" 2>/dev/null || break
        sleep 1
      done
      if kill -0 "$pid" 2>/dev/null; then
        kill -KILL -- "-$pid" 2>/dev/null || kill -KILL "$pid" 2>/dev/null
        echo "stopped $2 (pid $pid, killed)"
      else
        echo "stopped $2 (pid $pid)"
      fi
    fi
    rm -f "$1"
  fi
}

stop_all() {
  stop_one "$FRONTEND_PID" "frontend"
  stop_one "$BACKEND_PID" "backend"
}

# 二进制不存在，或任一 Go 源文件 / 依赖 / EXI 静态库比二进制新 → 需要重新构建
backend_needs_build() {
  [ ! -x "$BACKEND_BIN" ] && return 0
  find "$BACKEND_DIR" -maxdepth 5 \
    \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' -o -name '*.a' \) \
    -newer "$BACKEND_BIN" -print -quit 2>/dev/null | grep -q .
}

start_backend() {
  if is_running "$BACKEND_PID"; then
    echo "backend already running (pid $(cat "$BACKEND_PID"))"
    return 0
  fi
  if backend_needs_build; then
    echo "[backend] building (CGO enabled for EXI lib)..."
    ( cd "$BACKEND_DIR" && make build ) || { echo "[backend] build failed"; exit 1; }
  fi
  echo "[backend] starting — API :8080, OCPP WS :9101"
  # setsid --fork 让服务运行在独立会话（pgid==pid），并由服务自身写入真实 PID，
  # stop 时按进程组终止即可覆盖全部子进程
  ( cd "$BACKEND_DIR" && setsid --fork bash -c \
      'echo $$ > "$1"; exec ./bin/csms-backend >> "$2" 2>&1' \
      _ "$BACKEND_PID" "$BACKEND_LOG" & )
  for _ in $(seq 1 10); do
    sleep 1
    curl -fsS http://127.0.0.1:8080/health >/dev/null 2>&1 && {
      echo "[backend] up — log: $BACKEND_LOG"
      return 0
    }
    is_running "$BACKEND_PID" || {
      echo "[backend] exited early — last log lines:"
      tail -5 "$BACKEND_LOG"
      exit 1
    }
  done
  echo "[backend] /health not responding yet — see $BACKEND_LOG"
}

start_frontend() {
  if is_running "$FRONTEND_PID"; then
    echo "frontend already running (pid $(cat "$FRONTEND_PID"))"
    return 0
  fi
  if [ ! -d "$FRONTEND_DIR/node_modules" ]; then
    echo "[frontend] installing dependencies (first run)..."
    ( cd "$FRONTEND_DIR" && npm install ) || { echo "[frontend] npm install failed"; exit 1; }
  fi
  echo "[frontend] starting vite dev server — http://127.0.0.1:5690"
  ( cd "$FRONTEND_DIR" && setsid --fork bash -c \
      'echo $$ > "$1"; exec npm run dev >> "$2" 2>&1' \
      _ "$FRONTEND_PID" "$FRONTEND_LOG" & )
  for _ in $(seq 1 15); do
    sleep 1
    curl -fsS http://127.0.0.1:5690 >/dev/null 2>&1 && {
      echo "[frontend] up — log: $FRONTEND_LOG"
      return 0
    }
    is_running "$FRONTEND_PID" || {
      echo "[frontend] exited early — last log lines:"
      tail -5 "$FRONTEND_LOG"
      exit 1
    }
  done
  echo "[frontend] not responding yet — see $FRONTEND_LOG"
}

case "${1:-start}" in
  start)
    start_backend
    start_frontend
    echo
    echo "OK — backend  http://127.0.0.1:8080   frontend  http://127.0.0.1:5690"
    echo "logs: $LOG_DIR/   stop: ./run.sh stop"
    ;;
  stop)
    stop_all
    ;;
  restart)
    stop_all
    sleep 1
    start_backend
    start_frontend
    ;;
  *)
    echo "usage: $0 [start|stop|restart]"
    exit 1
    ;;
esac
