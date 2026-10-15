#!/bin/bash
# 本地开发/测试环境: 使用 docker 中的 MySQL(3316) / Redis(6379), 数据存放在 ./data
# 用法: source test/e2e/env.sh && ./service/start-all.sh
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export FS_MYSQL_HOST=127.0.0.1:3316
export FS_REDIS_HOST=127.0.0.1:6379
export FS_DATA_DIR="$ROOT_DIR/data/files"
export FS_PART_DIR="$ROOT_DIR/data/parts"
export FS_APIGW_ADDR=0.0.0.0:18080
# mdns在部分网络环境(如开启VPN/多网卡的macOS)下服务发现不稳定, 本地测试使用docker中的etcd作为注册中心
export MICRO_REGISTRY=etcd
export MICRO_REGISTRY_ADDRESS=127.0.0.1:2379
# 单机部署时RPC只监听回环地址; 否则会以局域网IP注册, 在开启VPN/防火墙时本机可能无法访问
export MICRO_SERVER_ADDRESS=127.0.0.1:0
# 测试时使用较小的分块, 便于用小文件覆盖多分块/断点续传场景
export FS_CHUNK_SIZE=1048576
export FS_MAX_UPLOAD_SIZE=33554432
