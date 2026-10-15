#!/bin/bash

# 工程根目录(脚本所在目录的上一级)
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
services="
dbproxy
upload
download
transfer
account
apigw
"

# 编译service可执行文件
build_service() {
    rm -f ${ROOT_DIR}/deploy/bin/$1
    # 镜像基于linux, 交叉编译静态链接的可执行文件
    CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH:-amd64} go build -o ${ROOT_DIR}/deploy/bin/$1 ./service/$1
    echo -e "\033[32m编译完成: \033[0m ${ROOT_DIR}/deploy/bin/$1"
}

# 打包镜像
build_image() {
    # 替换(hub.fileserver.com/filestore/$service), 自定义镜像名即可
    sudo docker build -t hub.fileserver.com/filestore/$1 -f ./service/$1/Dockerfile .
    echo -e "\033[32m镜像打包完成: \033[0m hub.fileserver.com/filestore/$1\n"
}

# 切换到工程根目录
cd ${ROOT_DIR}

# 静态资源通过 go:embed 在编译时打包进apigw(见 static/static.go), 无需额外处理

# 执行编译service
mkdir -p ${ROOT_DIR}/deploy/bin && rm -f ${ROOT_DIR}/deploy/bin/*
for service in $services
do
    build_service $service
done

echo -e "\033[32m编译完毕, 开始构建docker镜像... \033[0m"

# 打包微服务镜像
cd ${ROOT_DIR}/deploy/
for service in $services
do
    build_image $service
done

echo -e "\033[32mdocker镜像构建完毕.\033[0m"

# 容器启动示例
# 启动account service
# docker run -it -e PARAMS="--registry=consul --registry_address=192.168.200.212:8500" hub.fileserver.com/filestore/account