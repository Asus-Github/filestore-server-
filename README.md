# distributed-fileserver

基于golang实现的一种分布式云存储服务

## 关于微服务架构交互逻辑

<img src='/doc/microservice_interact_archi.png' width="800px"></img>

## 关于分支和标签

- 分支说明:
    - master分支: 通过原生net/http实现各接口功能
    - gin分支: 通过Gin框架来改造http协议通讯相关模块代码
    - pkg分支: 通过go-bindata等工具实现静态资源打包的一个示例
    - 其他分支与每章节的内容大致对应(比如第九章代码可以参考[charter9分支](https://git.imooc.com/coding-323/filestore-server/src/charter9))

- 标签
    - tag是分支代码某个阶段的快照。如`master`主分支的`v0.1`是首个发布的版本，该版本接入了阿里OSS。

## 关于源码版本下载

建议按`每章对应的分支代码`来看，`master`主分支代码为最新版本代码。

- master分支为最新提交的代码

```bash
git clone https://git.imooc.com/coding-323/filestore-server.git
```

- 第九章(RabbitMQ异步存储)章节的代码，可以下载标签为`v0.3`的版本:

```bash
git clone --branch v0.3 https://git.imooc.com/coding-323/filestore-server.git
```

- 第八章(OSS)及之前章节的代码，可以下载标签为`v0.2`的版本:

```bash
git clone --branch v0.2 https://git.imooc.com/coding-323/filestore-server.git
```

## 本地快速启动(Go 版本见 go.mod, 使用 go modules)

依赖通过 `go.mod` 管理(go-micro v4、gin、redigo 等), 无需手动 `go get`。国内网络可设置 `GOPROXY=https://goproxy.cn,direct`。

```bash
# 1. 启动 MySQL / Redis / etcd(首次启动自动执行 doc/table.sql 建库建表)
docker compose -f deploy/local/docker-compose.yml up -d

# 2. 编译并启动全部微服务(配置均可通过环境变量覆盖, 见 config/*.go 与 test/e2e/env.sh)
source test/e2e/env.sh && ./service/start-all.sh
# 浏览器访问 http://localhost:18080/static/view/signin.html

# 3. 运行测试
go test ./...
node test/e2e/api.test.mjs                               # 接口端到端测试
(cd test/e2e && npm install && node browser.test.mjs)    # 真实 Chrome 浏览器端到端测试

# 4. 关闭
./service/stop-all.sh
```

常用环境变量: `FS_MYSQL_HOST` `FS_REDIS_HOST` `FS_DATA_DIR` `FS_PART_DIR` `FS_APIGW_ADDR` `FS_UPLOAD_ENTRY` `FS_DOWNLOAD_ENTRY`
`FS_TOKEN_SECRET`(生产环境务必修改) `FS_CHUNK_SIZE` `MICRO_REGISTRY` `MICRO_REGISTRY_ADDRESS`。

## 关于应用启动(历史版本)

-  微服务架构下启动方式(非容器化部署):

    - 一键启动微服务(start-all.sh):
```bash
> cd $GOPATH/filestore-server
> ./service/start-all.sh 
 编译完成:  service/bin/dbproxy
 编译完成:  service/bin/upload
 编译完成:  service/bin/download
 编译完成:  service/bin/transfer
 编译完成:  service/bin/account
 编译完成:  service/bin/apigw
 已启动  dbproxy
 已启动  upload
 已启动  download
 已启动  transfer
 已启动  account
 已启动  apigw
微服务启动完毕.
```

    - 一键关闭微服务(stop-all.sh):
```bash
> cd $GOPATH/filestore-server
> ./service/stop-all.sh 
 已关闭:  apigw
 已关闭:  account
 已关闭:  transfer
 已关闭:  download
 已关闭:  upload
 已关闭:  dbproxy
执行完毕.
```

-  微服务架构下启动方式(容器化部署):
```bash
> cd $GOPATH/filestore-server
# 脚本方式启动容器
> ./deploy/start-all.sh
# 脚本方式关闭容器
> ./deploy/stop-all.sh
# docker-compose方式启动容器
> cd ./deploy/service_dc
> sudo docker-compose up -d
# k8s方式启动微服务
> cd ./deploy/service_k8s
> kubectl apply -f svc_account.yaml
> kubectl apply -f svc_apigw.yaml
> kubectl apply -f svc_dbproxy.yaml
> kubectl apply -f svc_download.yaml
> kubectl apply -f svc_transfer.yaml
> kubectl apply -f svc_upload.yaml
> cd ./deploy/traefik_k8s
> kubectl apply -f service-ingress.yaml
```

## 进度说明：
* [x] 简单的文件上传服务
* [x] mysql存储文件元数据
* [x] 账号系统, 注册/登录/查询用户或文件数据
* [x] 基于帐号的文件操作接口
* [x] 文件秒传功能
* [x] 文件分块上传/断点续传功能
* [x] 搭建及使用Ceph对象存储集群
* [x] 使用阿里云OSS对象存储服务
* [x] 使用RabbitMQ实现异步任务队列
* [x] 微服务化(API网关, 服务注册, RPC通讯)
* [x] CI/CD(持续集成)

## 参考资料

- Go入门: [语言之旅](https://tour.go-zh.org/welcome/1)
- MySQL: [偶然翻到的一位大牛翻译的使用手册](https://chhy2009.github.io/document/mysql-reference-manual.pdf)
- Redis: [命令手册](http://redisdoc.com/)
- Ceph: [中文社区](http://ceph.org.cn/) [中文文档](http://docs.ceph.org.cn/)
- RabbitMQ: [英文官方](http://www.rabbitmq.com/getstarted.html) [一个中文版文档](http://rabbitmq.mr-ping.com/)
- 阿里云OSS: [文档首页](https://help.aliyun.com/product/31815.html?spm=a2c4g.750001.3.1.47287b13LQI3Ah)
- gRPC: [官方文档中文版](http://doc.oschina.net/grpc?t=56831)
- go-micro微服务框架: [github源码](https://github.com/micro/go-micro)
- gin web框架: [github源码](https://github.com/gin-gonic/gin)
- k8s: [中文社区](https://www.kubernetes.org.cn/docs)
- 我的手记博客 [http://www.imooc.com/u/6198190](http://www.imooc.com/u/6198190)
