#!/bin/bash
# 一键编译并启动全部微服务
# 用法: ./service/start-all.sh [额外参数, 如 --registry=consul --registry_address=127.0.0.1:8500]
# 可通过环境变量覆盖配置(见 config/*.go), 例如:
#   FS_MYSQL_HOST=127.0.0.1:3306 FS_REDIS_HOST=127.0.0.1:6379 FS_DATA_DIR=./data/files ./service/start-all.sh

set -u

# 切换到工程根目录(脚本所在目录的上一级)
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR" || exit 1

BIN_DIR="$ROOT_DIR/service/bin"
RUN_DIR="$ROOT_DIR/service/run"
LOG_DIR="${FS_LOG_DIR:-$ROOT_DIR/service/log}"
EXTRA_ARGS=("$@")

# 微服务可以用supervisor做进程管理工具；
# 或者也可以通过docker/k8s进行部署
# 按依赖顺序启动: dbproxy -> account -> upload/download/transfer -> apigw
services="
dbproxy
account
upload
download
transfer
apigw
"

mkdir -p "$BIN_DIR" "$RUN_DIR" "$LOG_DIR"

# 编译service可执行文件
build_service() {
    if go build -o "$BIN_DIR/$1" "./service/$1"; then
        echo -e "\033[32m 编译完成: \033[0m service/bin/$1"
    else
        echo -e "\033[31m 编译失败: \033[0m $1"
        exit 1
    fi
}

# 检查service进程
check_process() {
    local pid=$1
    sleep 1
    if kill -0 "$pid" 2>/dev/null; then
        echo -e "\033[32m 已启动 \033[0m $2 (pid $pid)"
        return 0
    fi
    echo -e "\033[31m 启动失败 \033[0m $2, 请查看日志 $LOG_DIR/$2.log"
    return 1
}

# 启动service
run_service() {
    local pidfile="$RUN_DIR/$1.pid"
    if [[ -f $pidfile ]] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
        echo -e "\033[33m 已在运行 \033[0m $1 (pid $(cat "$pidfile"))"
        return 0
    fi
    nohup "$BIN_DIR/$1" "${EXTRA_ARGS[@]+"${EXTRA_ARGS[@]}"}" >> "$LOG_DIR/$1.log" 2>&1 &
    local pid=$!
    echo "$pid" > "$pidfile"
    check_process "$pid" "$1"
}

# 执行编译service
for service in $services; do
    build_service "$service"
done

# 执行启动service
for service in $services; do
    run_service "$service"
done

echo '微服务启动完毕.'
APIGW_ADDR="${FS_APIGW_ADDR:-0.0.0.0:8080}"
echo "访问 http://localhost:${APIGW_ADDR##*:}/static/view/signin.html"
