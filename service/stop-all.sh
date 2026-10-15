#!/bin/bash
# 一键关闭由 start-all.sh 启动的微服务

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="$ROOT_DIR/service/run"

stop_process() {
    local pidfile="$RUN_DIR/$1.pid"
    if [[ ! -f $pidfile ]]; then
        echo -e "\033[31m并未启动: \033[0m" "$1"
        return 0
    fi
    local pid
    pid=$(cat "$pidfile")
    if kill -0 "$pid" 2>/dev/null; then
        kill "$pid"
        # 等待进程退出(最多5秒), 超时则强制结束
        for _ in 1 2 3 4 5 6 7 8 9 10; do
            kill -0 "$pid" 2>/dev/null || break
            sleep 0.5
        done
        kill -0 "$pid" 2>/dev/null && kill -9 "$pid"
        echo -e "\033[32m已关闭: \033[0m" "$1"
    else
        echo -e "\033[31m并未启动: \033[0m" "$1"
    fi
    rm -f "$pidfile"
}

services="
apigw
transfer
download
upload
account
dbproxy
"

# 关闭service
for sname in $services; do
    stop_process "$sname"
done

echo "执行完毕."
