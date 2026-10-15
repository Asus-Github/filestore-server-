# Go 微服务文件存储系统项目需求书

> 文档版本：v1.0  
> 编制日期：2026-10-14  
> 项目定位：开源课程项目二次开发与工程化重构  
> 建设重点：Go 后端、中间件可靠性、大文件传输、可验证工程能力

## 1. 文档目的

本文档用于指导当前文件存储项目的后续开发、测试、演示和简历表述，解决以下问题：

1. 以代码事实为准，区分现有能力、待修复缺陷和规划能力，避免简历技术栈与实现不一致。
2. 将项目从“能上传和下载文件”升级为可展示 MySQL、Redis、RabbitMQ、对象存储等中间件工程能力的后端项目。
3. 明确分块上传中前端和后端的职责，突出后端在协议、状态、一致性、幂等、容错和资源治理方面的价值。
4. 为每一项可写入简历的能力建立需求、代码、测试和运行工件之间的证据链。

本文档不是完成情况说明。标记为“规划增强”的内容，在验收通过前不得写成已经实现。

## 2. 项目定位与范围

### 2.1 推荐定位

> 基于开源课程项目进行二次开发和工程化重构的 Go 智能文件存储系统，围绕 Redis 分块会话、MySQL 元数据一致性、RabbitMQ 可靠异步任务、S3 兼容对象存储、原生 gRPC 控制面和面向用户的个人文件知识 Agent 进行重点建设。

项目重点不是堆叠中间件名称，而是回答四个问题：

- 为什么使用该中间件；
- 哪些数据由它负责，哪些数据不由它负责；
- 依赖故障、并发和重复请求发生时系统如何处理；
- 通过什么测试和工件证明设计有效。

上述能力按独立 Track 验收。完成用户 Agent 不自动证明原生 gRPC、RabbitMQ 或多实例 multipart；只有 `AT-AGENT-INTEGRATION-01` 或各 Track 的独立工件才能支撑完整技术链路声明。

### 2.2 主线建设范围

个人可交付范围采用“共同基线 + 可独立验收 Track”：

1. 共同基线 P0：建立可复现基线并修复当前正确性问题。
2. Track A：实现 MySQL 事务型 Outbox 和 RabbitMQ 可靠任务链路。
3. Track B：使用 MinIO/S3 兼容 multipart 实现无 sticky session 的多实例续传和服务端校验。
4. Track C：将内部控制面从 Go Micro RPC 演进为原生 gRPC 微服务，保留 HTTP 网关和文件数据面边界。
5. Track D：建设面向普通用户的个人文件知识 Agent，提供授权搜索、问答、摘要、比较、结构化提取、标签建议和审批式整理。
6. Track E（可选）：建设内部上传可靠性助手，作为 Track A 之后的运维扩展，不与用户 Agent 混用权限和数据。
7. Track A/B/C/D/E 均补充各自的结构化日志、核心指标和验收工件；Track D/E 不得进入上传/下载同步主链路。

项目基础完成定义为 **P0 全部通过 + Track A 或 Track B 任一完整通过**。Track C 是需要在简历中声称“原生 gRPC 微服务”时的必需门禁；Track D 是用户产品 Agent 主线；Track E 只作为可选的 Agent/SRE 扩展。未通过的 Track 只能写入 roadmap。

以下能力列为远期扩展，不阻塞上述 Track：全链路 OpenTelemetry、真实阿里云 OSS/Ceph 认证、双向对象对账、引用安全 GC、历史 SHA-1 全量迁移、生产级 Kubernetes 和完整供应链安全。

### 2.3 非目标

- 不为了简历关键词强行引入 GORM；现有 `database/sql` 更适合展示事务、Context 和 SQL 能力。
- 原生 gRPC 作为独立 Track C 建设，不要求浏览器文件字节全部改走 gRPC；HTTP 网关和对象存储数据面继续保留。
- 不宣称 exactly-once、MySQL 与对象存储跨系统 ACID、消息零丢失、生产级高可用或生产级 Kubernetes。
- 不在单存储后端的正确性和对账机制完成前实现 `StoreMix`、`StoreAll` 或多云复制。
- 不使用没有固定环境、负载和原始报告支撑的性能提升百分比。

## 3. 状态与证据口径

### 3.1 能力状态

- **现有能力**：源码中存在有效实现，但仍需当前版本运行证据才能升级为简历完成项。
- **待修复**：代码已存在，但有明确的正确性、安全性或部署问题，属于 P0。
- **规划增强**：当前尚未实现，只能出现在需求书或 roadmap；不得进入正式简历 bullet。
- **不支持**：仅在文档、配置常量或简历中出现，代码和运行证据不足。

### 3.2 测试状态

- `PASS`：测试已运行，断言满足，原始工件已保留。
- `FAIL`：测试已运行，但不满足需求。
- `XFAIL`：缺陷已经登记，有可执行测试，且实际失败原因与预期一致。
- `BLOCKED`：测试因凭证、外部环境等前置条件缺失而无法运行。
- `NOT RUN`：测试尚未运行，或规划能力尚无测试。
- `UNVERIFIED`：声称曾执行，但源码快照、命令输出或环境工件不足，无法复现结论。
- `N/A/UNSUPPORTED`：在当前受支持配置中明确不适用，不属于预期失败。

不能把“已知有问题”直接写成 `XFAIL`，不能把规划功能写成 `BLOCKED`，也不能把明确不支持的部署方式写成 `XFAIL`。

### 3.3 当前审计结论

- 基线提交为本地 `4ea64fa`，但它不能证明上游仓库的精确版本。P0 必须补充上游 URL、ref/tree hash、许可证、导入方式及个人变更 diff。
- 当前工作树包含大量未提交和未跟踪内容，提交哈希不能单独复现当前系统。
- 本次审计曾执行 `go test -count=1 ./...`，但未保留满足 `PASS` 定义的完整源码状态、原始输出和环境工件，当前记为 `UNVERIFIED`；可见测试主要位于 `account/handler`、`dbproxy/mapper`、`util`，其余包多为编译检查。
- 仓库存在 API 和浏览器检查脚本，本次未执行，当前状态为 `NOT RUN`；脚本中的计数不代表独立断言数量，更不代表当前通过数量。
- 历史日志和截图只能作为旁证，不等同于当前提交的测试报告。

## 4. 当前系统真值表

| 简历或文档命题 | 当前代码事实 | 状态 | 当前安全表述 |
|---|---|---|---|
| 六服务微服务架构 | 存在 `apigw`、`account`、`dbproxy`、`upload`、`download`、`transfer` 六个可执行程序；默认关闭异步转存时 transfer 不消费任务 | 现有能力 | “六服务代码结构，控制面与文件数据面分离” |
| 服务间使用 gRPC | 当前使用 Protobuf 定义的 Go Micro RPC；生成代码不能证明原生 gRPC transport，Track C 尚未实现 | 规划增强 | 当前只能写“Go Micro RPC”；Track C 通过后才写“原生 gRPC 微服务” |
| etcd 服务发现 | 框架默认仍可使用 mDNS；测试配置可选择 etcd，但缺少当前 PASS 工件 | 部分现有 | “接入并配置 etcd registry” |
| GORM 数据访问层 | 实际使用 `database/sql` 和 MySQL 驱动 | 不支持 | “基于 database/sql 的数据访问层” |
| MySQL 主从读写分离 | 代码只有一个 DSN 和一个连接池，无副本路由、主库粘滞或延迟策略 | 不支持 | 暂不写入简历 |
| MySQL 事务保证元数据一致 | 全局文件与用户关系通过两次独立 RPC 写入；DB-proxy 的 transaction 字段未生效 | 待修复 | “持久化文件元数据和用户关系” |
| Redis 是通用缓存 | Redis 保存有 TTL 的会话、`chkidx_*` 已上传分块标记和恢复索引，并提供尽力而为的合并互斥；成功后删除会话，不保存永久完成事实 | 现有能力 | “Redis 管理分块会话、已上传分块标记和尽力而为的合并互斥” |
| 分布式断点续传 | Redis 状态共享，但分块字节保存在上传节点本地磁盘 | 不支持 | “单节点应用级断点续传” |
| 普通上传为流式处理 | 解析 multipart 文件后以 `io.Copy` 写入应用临时文件并计算服务端 SHA-1；提供 `filehash` 时才进行一致性比较 | 部分现有 | 按该事实描述，不宣称端到端请求流式处理 |
| 去重和秒传安全可靠 | 基于 SHA-1 元数据；远端对象未做 `HEAD`；知道 hash 和大小的其他用户可能建立关系 | 待修复 | “单节点、基于元数据的去重/秒传原型” |
| RabbitMQ 可靠异步存储 | 已启用 mandatory publish、durable queue/message、手动 ACK 和重连，但缺少 publisher confirm、`NotifyReturn` 监听/关联、自动重试、DLQ 重放和 Outbox | 待修复 | “RabbitMQ 异步转存原型” |
| OSS/Ceph 海量云存储 | SDK 适配器部分存在，但缺少当前版本 provider 认证，Ceph 路径还存在全量缓冲；“海量云存储”没有证据 | 部分现有/待运行认证 | “实现 OSS/Ceph 适配器，运行认证待完成” |
| HTTP Range 下载 | 本地存储通过 `http.ServeContent` 支持；OSS 为签名 URL；Ceph 路径不是同等流式实现 | 部分现有 | “本地存储支持 HTTP Range 下载” |
| API 网关负载均衡 | Go Micro 客户端有框架选择器；不等于自研负载均衡，文件字节也不经过网关 | 部分现有 | “服务发现和客户端节点选择” |
| 网关限流与熔断 | 仅覆盖网关到 account 的出站 RPC，参数硬编码，不保护 upload/download 入口 | 部分现有 | 限定作用域后描述 |
| Postman 完成接口测试 | 当前主要可见证据为 Node API/浏览器 E2E 脚本，不能用 Postman 代称 | 不支持 | “编写脚本化 API/E2E 检查” |
| 生产级 Kubernetes | 清单版本较旧，缺少完整探针、资源、安全配置和正确多副本文件语义 | 不支持 | “提供 Kubernetes 示例清单” |

## 5. 现有与目标架构

### 5.1 当前架构

```text
Browser
  |-- HTTP --> apigw
  |              `-- Go Micro RPC --> account --> dbproxy --> MySQL
  |
  |-- file bytes --> upload
  |                    |-- account/dbproxy --> MySQL
  |                    |-- Redis multipart state
  |                    |-- local disk / Ceph / OSS
  |                    `-- RabbitMQ (optional)
  |
  `-- HTTP --> download
                      |-- account/dbproxy --> MySQL
                      `-- local disk / Ceph / OSS

RabbitMQ --> transfer --> OSS --> dbproxy --> MySQL
```

当前默认有效路径是 MySQL、Redis、注册中心和本地文件系统。RabbitMQ、Ceph、OSS 不在默认数据路径中。

### 5.2 目标架构

```text
Browser
  |-- control plane --> apigw --> account/dbproxy --> MySQL
  `-- data plane ----> upload/download
                            |
                            |-- Redis: short-lived coordination cache
                            |-- MySQL: durable upload job/session facts
                            `-- MinIO/S3: unique immutable staging key
                                      |
                              provider complete
                                      |
                                      v
                         T1: job=VERIFY_QUEUED + verify Outbox
                                      |
                                 Outbox Relay
                                      |
                                  RabbitMQ
                                      |
                              Verify Worker (read only)
                                      |
                         stream SHA-256 from exact key/version
                                      |
                                      v
                         T2: file_object + user_file + AVAILABLE

Structured Logs / Prometheus / optional OpenTelemetry
```

目标主线中，upload 服务负责写入每个 job 唯一且不可覆盖的 staging key；worker 只负责读取精确 object key/version 进行校验和状态发布，不再次写入或覆盖该对象。旧的“本地磁盘转存 OSS”保留为兼容路径，不与新主线混用。

Track C 的协议边界如下：

```text
Browser --HTTP/JSON--> apigw --native gRPC--> account / metadata / job services
Browser --HTTP bytes--> upload/download --HTTP/S3--> object storage

User Assistant API --typed gRPC tools--> search/read/proposal APIs
                                      `--> MySQL, Redis, index, object metadata
Optional Ops Assistant --typed gRPC tools--> job/evidence APIs
```

原生 gRPC 只替换服务间控制面调用，不把浏览器上传字节强行搬进内部 RPC。内部大文件搬运优先由对象存储完成；只有确有跨服务流式需求时，才增加经过限流和大小限制的 client/server streaming RPC。

### 5.3 权威数据边界

| 数据 | 权威来源 | 设计约束 |
|---|---|---|
| 用户、密码摘要、令牌摘要 | MySQL | Redis 缓存不能成为授权真相 |
| 用户文件关系 | MySQL `user_file` | 外键关联 `file_object`；有效关系具有用户和对象唯一约束 |
| 文件身份与可见状态 | MySQL `file_object` | 保存 `hash_algo/digest/size/object_key/version_id/status/version`；唯一键为 `(hash_algo,digest,size)`；仅 `AVAILABLE` 可下载和秒传 |
| 上传工作流和恢复最小事实 | MySQL `upload_job` | 保存 `provider_upload_id/geometry/generation/staging_key/version_id/status/version/terminal_response/attempt/next_attempt_at` |
| Outbox 事件 | MySQL `outbox_event` | event ID 唯一；RabbitMQ 只运输消息，不作为任务唯一事实来源 |
| 短期上传协调 | Redis | 缓存会话、part 和租约；必须有 TTL；丢失后由 MySQL 与分页 `ListParts` 重建 |
| 分块和最终对象字节 | MinIO/S3 兼容存储 | 使用 job 唯一、不可覆盖的 staging key 和精确 version；不依赖单个 upload 进程本地磁盘 |
| 摘要与大小 | MySQL 中的服务端验证结果 | 客户端声明只能作为提示，不能直接建立完整性事实 |

最小持久化模型要求如下：

- `file_object(id, hash_algo, digest, size, object_key, version_id, status, version)`：`(hash_algo, digest, size)` 唯一；`object_key + version_id` 指向不可变 canonical object。
- `user_file(user_id, file_object_id, status, created_at, deleted_at)`：外键关联 `file_object`；同一用户和对象的有效关系唯一；P0 只允许解除关系，不物理删除对象。
- `upload_job(id, user_id, provider_upload_id, geometry, generation, staging_key, version_id, status, version, terminal_response, attempt, next_attempt_at)`：保存恢复所需最小事实，不因 Redis 丢失而无法识别任务。
- `outbox_event(id, aggregate_id, schema_version, idempotency_key, status, lease_owner, lease_until, published_at)`：event ID 和业务幂等键唯一；发送状态由 MySQL 管理。

## 6. 用户角色与核心用例

| 角色 | 核心用例 |
|---|---|
| 普通用户 | 注册、登录、上传、下载、管理文件，并通过自然语言搜索、问答、摘要、比较和整理自己的文件 |
| 大文件用户 | 初始化 multipart、暂停、恢复、补传缺失 part、查询完成状态 |
| 运维人员 | 查看依赖健康、Outbox 积压、队列深度、DLQ、失败任务和对象异常 |
| 开发者 | 一键启动环境，执行单元、集成、E2E、故障和性能测试 |
| 面试官/评审者 | 核对上游来源、个人提交、架构决策、故障实验和简历证据 |

核心用例：

- `UC-01` 用户完成注册、登录、注销、令牌过期和撤销。
- `UC-02` 用户流式上传普通文件，服务端建立权威摘要和元数据。
- `UC-03` 用户上传大文件，在浏览器刷新或进程切换后恢复缺失 part。
- `UC-04` 同一用户对已验证对象执行秒传；不同用户仍需证明持有完整内容。
- `UC-05` 用户通过授权和签名 URL 下载完整文件或字节范围。
- `UC-06` 文件异步进入对象存储，失败后可重试、审计和人工重放。
- `UC-07` 上传实例退出后，其他实例继续 multipart 会话。
- `UC-08` 运维人员根据 job ID 定位 HTTP、SQL、Redis、消息和对象状态（可选 Track E）。
- `UC-09` 用户使用自然语言搜索自己有权访问的文件，得到文件、片段和可点击引用。
- `UC-10` 用户对选定或检索出的多个文件进行问答、摘要、比较和结构化信息提取。
- `UC-11` 系统在文件入库后生成候选标签/分类，用户审批后整理文件元数据。

## 7. 核心领域状态与不变量

### 7.1 上传会话与任务状态

```text
OPEN --> COMPLETING --> VERIFY_QUEUED --> VERIFYING --> SUCCEEDED
  |           |                |             |
  |           |                `-----------> RETRY_WAIT
  |           |                                  |
  |           `-------------------------------> FAILED_PERMANENT
  `-------------------------------------------> CANCELED / EXPIRED
```

关键转换如下，详细状态表应在实现 ADR 中继续展开：

| 来源 | 目标 | 执行者与 CAS | 副作用与恢复 |
|---|---|---|---|
| `OPEN` | `COMPLETING` | upload 以 `version + generation` CAS | 冻结 generation，拒绝新 part，等待该 generation 的在途写租约结束 |
| `COMPLETING` | `VERIFY_QUEUED` | upload 执行 provider complete，T1 CAS job 并写 verify Outbox | 响应未知时按精确 staging key/version 与 provider 状态恢复 |
| `VERIFY_QUEUED` | `VERIFYING` | worker 领取带 fencing epoch 的任务 | 旧 epoch 的 worker 不得提交结果 |
| `VERIFYING` | `SUCCEEDED` | verifier 计算 SHA-256，T2 CAS | 原子 upsert `file_object`、创建 `user_file`、保存终态响应并发布 `AVAILABLE` |
| `VERIFYING` | `RETRY_WAIT` | worker 事务内更新 attempt/next time 并写下一次 Outbox | DB 提交后 ACK；DB 失败才 NACK/requeue |
| 非终态 | `CANCELED/EXPIRED` | API 或 sweeper CAS | 仅在没有有效在途租约时 abort；否则延迟重试清理 |

- `OPEN` 才允许申请 `(session_id,generation,part_no)` 写入租约。
- owner token、续租和 compare-and-delete 只能减少重复执行，不能保证唯一执行；正确性依赖单调 fencing epoch、MySQL version/CAS、冻结的 generation 和幂等对象操作。
- complete 冻结 generation 后必须等待在途 part 租约结束，再调用 provider complete。
- `terminal_response` 持久化在 MySQL，Redis 只缓存；客户端在提交后丢失响应时按幂等键查询结果。
- Redis flush/restart 后从 MySQL 的 upload job 和分页 `ListParts` 重建协调状态。

### 7.2 文件对象状态

```text
STAGED --> VERIFYING --> AVAILABLE
   |           |
   `----------> FAILED / QUARANTINED
AVAILABLE --> DELETING --> DELETED
```

- `AVAILABLE` 是下载和秒传的唯一可见状态。
- 文件对象状态是下载可见性的唯一权威；`upload_job` 仅描述工作流执行状态。
- T1 只提交 `VERIFY_QUEUED + verify Outbox`，不得提前创建可用用户关系；T2 在服务端校验成功后原子提交 `file_object + user_file + job SUCCEEDED`。
- 若摘要身份已存在，T2 关联已有 canonical object 并异步清理新 staging object；任何未验证内容都不得覆盖已有对象。
- MySQL 与对象存储不追求跨系统 ACID，使用状态机、幂等副作用、Outbox 和对账收敛。
- DB 事务前创建的 staged 对象可能成为孤儿，因此 staged-prefix 清理属于主线能力，不应推迟到远期。

### 7.3 内容与授权不变量

1. 同一物理对象可关联多个用户，但用户关系必须独立授权。
2. 删除一个用户关系不得删除仍被其他用户引用的对象。
3. 已建立的内容记录不得被后续上传静默覆盖大小、摘要或地址。
4. 新写入强制使用服务端 SHA-256，身份唯一键包含 `hash_algo + digest + size`；历史 SHA-1 仅用于兼容查询，不得与 SHA-256 混用。
5. 跨用户秒传不能只凭公开或可猜测的 hash 和大小授权。主线方案为：保留全局物理去重，但不同用户必须上传内容，由服务端验证后再关联。
6. 统一幂等键为 `(user_id, operation, idempotency_key)`，同时保存 request fingerprint 和终态响应；同键同载荷重放原结果，同键不同载荷返回 `409`。

## 8. 功能需求

### 8.1 P0：可信基线与正确性修复

| ID | 需求 | 验收标准 |
|---|---|---|
| `FR-P0-01` | 记录上游 URL、精确 ref/tree hash、许可证、导入方式和个人增量 | README 和 ADR 可追溯；个人改动可由提交和 diff 复核 |
| `FR-P0-02` | 将 `go.mod`、middleware、E2E、Compose、schema 和启动脚本纳入版本控制 | `git ls-files` 检查通过；干净克隆可启动 E1 环境 |
| `FR-P0-03` | 清理 README 中 Ceph、OSS、RabbitMQ、CI/CD、gRPC、分布式等过度完成声明 | 公开文档与当前真值表一致 |
| `FR-P0-04` | 定义 `local` 和 `single-node-object` 配置档，拒绝未实现的 `StoreMix/StoreAll` | 无效配置启动失败，不再静默回落本地模式 |
| `FR-P0-05` | 新增类型化 `CommitUpload` RPC，使用 `DBTX`/`*sql.Tx` 原子写入文件与用户关系 | 任一 SQL 边界注入失败，两张表均无半提交；同 hash 并发提交结果一致 |
| `FR-P0-06` | RPC 和 SQL 透传 Context、deadline、request ID | 阻塞依赖时请求有界失败，后台查询被取消 |
| `FR-P0-07` | 修复 HTTP 状态语义 | 参数、认证、授权、冲突、超限、依赖故障分别返回 400、401、403、409、413/429、503 |
| `FR-P0-08` | Redis 原子创建 session、resume index 和 TTL | 并发 init 只产生一个有效会话，不存在无 TTL session |
| `FR-P0-09` | multipart 引入显式状态、generation、version/CAS 和 fencing epoch | chunk-vs-complete、cancel-vs-complete 的结果确定；旧 generation/epoch 不得提交副作用 |
| `FR-P0-10` | 完成租约使用随机 owner token、续租和 compare-and-delete，但只用于减少重复执行 | 正确性由 fencing、冻结的 part 集合、DB CAS 和幂等对象操作保证，不以租约唯一性作为安全边界 |
| `FR-P0-11` | 分块注册前验证会话仍存在且为 `OPEN` | 会话在上传中途过期时不会生成残缺 Redis hash |
| `FR-P0-12` | complete 保存幂等结果 | 成功响应丢失后重复 complete 返回同一资源 |
| `FR-P0-13` | 限制单文件大小、chunk 数、活动 session 数和节点暂存总量 | 超限请求拒绝且不创建 DB、Redis、磁盘残留 |
| `FR-P0-14` | 修复秒传和已存在对象校验 | 删除远端对象或篡改同尺寸本地对象后秒传拒绝；普通重传可发现并隔离损坏对象 |
| `FR-P0-15` | 修复跨用户秒传授权 | 仅 hash+size 不能让新用户获得文件关系；跨用户需服务端验证完整内容 |
| `FR-P0-16` | 加固 secret、CORS、trusted proxy、token 传递和 HTTP timeout | 生产模式拒绝默认 secret；query token 返回 401；慢请求有界失败 |
| `FR-P0-17` | 实现优雅退出与连接关闭 | `SIGTERM` 后停止接入，在期限内完成或取消请求并关闭依赖连接 |
| `FR-P0-18` | 明确本地模式单上传副本，或为相关服务配置真实共享卷 | 部署语义与数据可见性一致，不再声称本地模式可水平扩展 |
| `FR-P0-19` | 建立机器可读测试报告 | Go、race、API E2E、浏览器 E2E 的命令、状态、版本、退出码和工件均保留 |
| `FR-P0-20` | 删除只软删除或解除当前用户的 `user_file` 关系 | 引用安全 GC 验收前禁止物理删除共享对象；删除一名用户关系不影响其他用户下载 |

P0 只实现节点级资源上限。并发安全的用户总配额需要 MySQL 原子 reservation，列为 P2，不在 P0 虚假承诺。

### 8.2 Track A：MySQL Outbox 与 RabbitMQ 可靠任务

| ID | 需求 | 验收标准 |
|---|---|---|
| `FR-P1-01` | `upload_job` 记录状态、attempt、next_attempt_at、generation、version 和 last_error | 非法状态跳转被拒绝；CAS 冲突和旧 generation 可识别 |
| `FR-P1-02` | 新增 `outbox_event`；T1 将 job 置为 `VERIFY_QUEUED` 并在同事务写 verify Outbox | DB 提交后进程立即退出，重启 relay 仍能发布；用户关系只在校验后的 T2 创建 |
| `FR-P1-03` | Outbox event 具有唯一 event ID、aggregate ID、schema version 和幂等键 | 重复发布可识别，消息契约可版本演进 |
| `FR-P1-04` | Outbox 使用 `PENDING/LEASED/SENT`、`lease_owner/lease_until/published_at`；relay 以 MySQL 时间批量领取，发布期间不持有 DB 事务 | 多 relay 可恢复领取；崩溃或超时后租约释放并重发 |
| `FR-P1-05` | 监听并按 channel generation、publish sequence 和 event ID 关联 publisher confirm 与 mandatory return | 仅 persistent mandatory publish 获 positive confirm 且无关联 return 才置 `SENT`；return/nack/timeout/channel close/未知结果均重发 |
| `FR-P1-06` | 修复全局共享 AMQP channel 的并发发布问题 | race 测试与并发真实 broker 测试通过 |
| `FR-P1-07` | 消费者使用 job ID、消息 ID、generation 和 DB CAS 幂等处理 | 重复或旧代消息不会创建重复关系或使状态倒退 |
| `FR-P1-08` | MySQL 作为重试权威，事务内更新 `RETRY_WAIT/FAILED_PERMANENT` 并写下一次 Outbox；DLQ 仅用于观察 | 重复 redrive 是带唯一 generation 的 DB 命令；RabbitMQ DLQ 不作为唯一业务事实 |
| `FR-P1-09` | 明确 ACK 边界 | 消费者 DB 事务提交后 ACK；DB 失败才 NACK/requeue；ACK 丢失后的重复投递可收敛 |
| `FR-P1-10` | 异步接口返回 `202 Accepted + job_id`，提供状态查询 | UI 可区分处理中、可下载、失败和可重试；仅 T2 提交后的 AVAILABLE 可下载 |

必须验证以下崩溃窗口：

- DB 已提交、事件未发布；
- broker 已接收、confirm 未到；
- confirm 已到、Outbox 尚未标记 sent；
- return 与 confirm 先后或并发出现；
- 对象写入完成、DB 未更新或消息尚未 ACK。

这些窗口允许重复发布，因此目标语义是 **at-least-once 运输 + 幂等消费**，不是 exactly-once。

### 8.3 Track B：服务端主导、无 sticky session 的多实例 multipart

主线选择一种实现，避免同时维护多个未完成方案：

> 使用 MinIO/S3 兼容原生 multipart。上传服务代理 part 字节并流式调用 `UploadPart`；对象存储保存 part，Redis 管短期协调，MySQL 保存恢复所需事实、generation、状态和终态响应。Track B 不依赖 Track A 时，由 MySQL 可恢复 job 调度 verifier。

| ID | 需求 | 验收标准 |
|---|---|---|
| `FR-P1-11` | 服务端决定 chunk size、part count 和 session owner | 客户端无法通过篡改 geometry 越界上传 |
| `FR-P1-12` | provider 契约支持 create/upload/分页 list/complete/abort multipart | provider upload ID、geometry、generation、staging key、part/ETag 可由 MySQL 和 provider 状态恢复 |
| `FR-P1-13` | part 经后端流式写入对象存储，不落 upload Pod 本地永久目录 | 固定容器资源下完成 `AT-MP-RESOURCE-01`，而不是仅定性声称内存有界 |
| `FR-P1-14` | part 先取得 `(session,generation,part_no)` 写入租约；complete 冻结 generation 并等待在途租约结束 | 旧 generation 被拒绝；同 part 异内容在覆盖对象前返回 `409` |
| `FR-P1-15` | complete 前校验分页 part 列表、大小、generation 和状态 | 缺 part、重复 part、错误大小均不能 complete |
| `FR-P1-16` | provider complete 后执行 T1；verifier 从精确 staging key/version 流式计算 SHA-256，T2 后转 `AVAILABLE` | complete 响应丢失、`NoSuchUpload`、校验后 CAS 失败均能恢复；未验证对象不可下载 |
| `FR-P1-17` | 过期、取消和失败会话执行 provider `AbortMultipartUpload`，staging sweeper 处理未知孤儿 | 清理期限后不存在无解释的 multipart 泄漏 |
| `FR-P1-18` | 至少两个 upload 实例无 sticky session 运行 | 交替上传 part、终止其中一个实例后，另一实例可完成且摘要一致；不据此宣称全部依赖高可用 |
| `FR-P1-19` | 浏览器用用户绑定的 opaque session ID 和本地恢复凭证重新查询会话 | 刷新恢复通过；跨用户猜测、过期凭证、同名同尺寸异内容均不会错误关联 |

可选扩展为客户端直传预签名 part URL，但仍由后端创建会话、签发短期权限、验证 part、执行 complete、最终摘要校验和元数据提交。这里的 bearer/session credential 禁止放入 query；下载签名 URL 可以使用 query 参数，但必须是绑定资源和有效期的短期签名，不得当作登录 token。

### 8.4 P1：HTTP/RPC 中间件与可观测性

| ID | 需求 | 验收标准 |
|---|---|---|
| `FR-P1-20` | 定义 HTTP 入站/出站 middleware 执行模型：认证前做 IP/连接/body 保护，认证后做用户限流；Recovery、request ID、脱敏日志、CORS 和指标边界明确 | CORS 预检不被认证拦截；panic（含 header 已写场景）返回统一可解释错误；日志无 token；慢请求和超大 body 有界结束 |
| `FR-P1-21` | 增加用户/IP 限流和 upload/download 入口并发保护 | 超限返回 `429 + Retry-After`，两个数据入口均受保护 |
| `FR-P1-22` | RPC wrapper 透传 deadline、request ID 和 trace metadata | deadline 可真正取消 SQL/下游调用 |
| `FR-P1-23` | RPC 仅对幂等调用自动重试，副作用调用使用幂等键 | 丢失响应不会造成重复写 |
| `FR-P1-24` | 明确 limiter 和 breaker 的作用范围与错误分类 | 业务错误、依赖错误和 timeout 分开统计，不能把硬编码参数当实测 SLA |
| `FR-P1-25` | 输出结构化日志和 RED 指标 | 可按 service、route、status、error_class 查询率、错误和延迟 |
| `FR-P1-26` | 输出依赖和业务指标 | 包含 DB/Redis 池、Outbox backlog、confirm/return、queue/DLQ depth、staged/failed job |
| `FR-P1-27` | 区分 liveness、readiness 和 feature health | Redis 故障时 upload Pod 可保持 ready，但 multipart 返回明确 503 并上报 feature unhealthy |

### 8.5 P2：扩展能力

| ID | 需求 | 验收标准 |
|---|---|---|
| `FR-P2-01` | MySQL 原子配额 reservation 与释放 | 100 个并发请求不能通过竞争突破总配额，废弃 reservation 可回收 |
| `FR-P2-02` | 引用安全删除和宽限期 GC | 事务确认零有效引用并经过宽限期后，才执行 `AVAILABLE -> DELETING -> DELETED` 物理回收；多用户共享对象时，最后一个引用删除前不得回收 |
| `FR-P2-03` | DB 到对象、对象到 DB 双向对账 | 可发现并审计孤儿、缺失和损坏对象 |
| `FR-P2-04` | SHA-256 历史迁移 | 迁移可重启，历史对象可读，冲突对象被隔离 |
| `FR-P2-05` | OSS 或 Ceph 单一真实 provider 认证 | PUT、HEAD、Range GET、完整 GET、删除、超时、错误凭证均通过 |
| `FR-P2-06` | OpenTelemetry 端到端 Trace | 一次上传可关联 HTTP、RPC、SQL、MQ 和对象存储 |
| `FR-P2-07` | Kubernetes 加固 | 使用受支持 API，具备 Secret、PVC、探针、资源、安全上下文和回滚验证 |
| `FR-P2-08` | 供应链安全 | 依赖、secret、镜像扫描和 SBOM 在 CI 中产生可追踪结果 |

## 9. 为什么分块上传能突出后端能力

前端的 `Blob.slice`、并发请求、进度条和暂停按钮只是数据切分与交互层。真正能体现后端能力的是服务端是否建立了可验证的传输协议和一致性边界。

### 9.1 服务端应承担的能力

| 后端能力 | 关键问题 | 验证方式 |
|---|---|---|
| 会话协议 | 谁能上传、有效期多久、允许多少 part | 越权、过期、重复 init、非法 index 测试 |
| 权威 geometry | chunk 大小和数量由谁决定 | 篡改 chunk size/count 被拒绝 |
| 流式与背压 | 大文件是否占满内存或 goroutine | 1 GiB 文件、内存曲线、并发限制测试 |
| 幂等性 | 网络重试会不会产生重复 part 或重复文件 | 同 part、同 complete 重复请求测试 |
| 并发状态机 | chunk、complete、cancel 同时发生怎么办 | 竞态矩阵与 `go test -race` |
| 分布式协调 | 切换实例后能否继续 | 两实例交替上传和进程终止实验 |
| 服务端完整性 | 是否信任客户端 hash | 完成后服务端流式 SHA-256 验证 |
| 事务提交 | 文件、用户关系、任务能否半提交 | SQL 故障注入与事务回滚测试 |
| 清理与补偿 | 失败后 Redis、part 和 staged 对象怎么办 | TTL、abort、孤儿扫描测试 |
| 安全和配额 | 是否可用超大文件/海量会话耗尽资源 | 大小、会话、并发、速率限制测试 |
| 可观测性 | 卡住时能否回答任务在哪一步 | job ID、指标、结构化日志和 Trace |

### 9.2 推荐面试表述

> 前端负责切片和进度展示，后端负责定义并执行上传协议。服务端决定分块参数和权限，使用 Redis 管理有 TTL 的短期会话和完成租约，使用对象存储保存跨实例可见的 part，使用 MySQL 持久化任务和幂等完成结果，并在 complete 后由服务端校验最终 SHA-256。通过重复请求、并发竞态、Redis 故障和上传实例退出测试验证恢复能力。

这种表述比“前端把文件切成若干块上传”更能体现后端工程深度。

## 10. 中间件专项要求

### 10.1 MySQL

重点展示：

- 类型化事务接口和 `*sql.Tx`；
- 文件元数据、用户关系、job 和 Outbox 的事务边界；
- 唯一约束、CAS/version、防止状态倒退；
- Context、deadline、连接池和慢查询指标；
- 关键查询索引和 `EXPLAIN` 结果；
- 事务无法覆盖对象存储时，如何通过状态机和补偿实现最终一致性。

不建议为了 GORM 关键词重写数据层。手写 SQL 能更清楚地展示事务、索引和并发控制。

### 10.2 Redis

重点展示：

- Lua 或事务命令原子创建 session、resume index 和 TTL；
- Redis 协调 `OPEN -> COMPLETING`，最终状态转换必须由 MySQL version/CAS 和 fencing epoch 兜底；
- owner token 锁、续租和 compare-and-delete；租约只减少并发，不是唯一正确性边界；
- 短期幂等结果、会话限额和速率限制；
- Redis 故障时 multipart 降级，而普通上传不被错误连带下线；
- key 数、TTL、池等待和锁冲突指标。

Redis 不保存永久业务事实，也不因为“用了 Redis”就称为缓存架构。

### 10.3 RabbitMQ

重点展示：

- Outbox 消除“DB 已提交但消息未发出”的丢任务窗口；
- publisher confirm 和 mandatory return；
- channel 并发安全；
- durable exchange/queue/message、manual ACK；
- 有界重试、退避、DLQ 和 redrive；
- job ID、消息 ID、attempt 与幂等消费者；
- broker 和 worker 故障下的状态收敛。

简历应写具体语义，不写泛化的“提高可用性”。

### 10.4 MinIO/S3、OSS 与 Ceph

主线先完整实现一个 S3 兼容 provider，建议 MinIO，契约至少包含：

- `Put/GetRange/Head/Delete/Health`；
- `CreateMultipart/UploadPart/ListParts/CompleteMultipart/AbortMultipart`；
- timeout、重试和不可重试错误分类；
- provider upload ID、part number、ETag/checksum 持久化；
- 不可变对象键、version ID 或等价覆盖保护；
- 对象进入 `AVAILABLE` 前的完整内容校验。

S3 multipart ETag 不是通用 SHA-256，`HEAD` 中的自报 metadata 也不能单独证明对象未被覆盖。

### 10.5 Go Micro RPC 与注册发现

- 正确术语为“Protobuf 定义的 Go Micro RPC”。
- 默认节点选择和重试属于框架能力，不能写成自研负载均衡。
- etcd、Consul、Kubernetes 插件“可配置”不等于都已运行验证。
- 副作用 RPC 必须使用幂等键或禁用不安全自动重试。
- 只有保留了版本、启动参数和运行日志后，才能描述实际 transport 与 registry。

### 10.6 Track C：原生 gRPC 微服务

#### 10.6.1 迁移目标与边界

当前系统是“Protobuf + Go Micro RPC”，不是原生 gRPC。Track C 的目标是新增并验证真正的 `google.golang.org/grpc` 服务端和客户端，迁移内部控制面，同时保留外部 HTTP 接口和文件数据面：

- `apigw` 继续对浏览器提供 Gin HTTP/JSON API，通过 gRPC client 调用 account、metadata 和 job 服务。
- `upload`、`download` 的浏览器文件字节继续走 HTTP；最终对象和 multipart part 继续由对象存储承载。
- account、文件元数据、上传任务、对象校验等控制操作优先使用 unary RPC。
- 只有跨服务确有大对象搬运需求时，才增加 client/server/bidirectional streaming RPC；必须有消息大小、速率、并发和 deadline 限制。
- Go Micro RPC 与原生 gRPC 在过渡期并存，使用配置开关逐服务切换，禁止一次性删除旧路径。

#### 10.6.2 协议与服务设计

- Protobuf package 统一使用 `filestore.v1`，服务和消息按领域拆分，字段编号一经发布不得复用。
- 关键服务至少包括 `AccountService`、`MetadataService`、`UploadJobService` 和 `ObjectVerifyService`；不让通用 DB-proxy 暴露为 Agent 或业务层的长期控制面。
- 所有副作用 RPC 携带 `request_id`、`idempotency_key`、`expected_version` 或 `generation`；重复调用返回相同结果或明确冲突。
- 使用 `google.protobuf.Timestamp`、明确的 enum 和版本字段；删除或重命名字段先 `reserved`，不直接复用旧编号。
- 生成代码、proto lint、兼容性检查和 API 文档进入版本控制；禁止手工修改生成文件来绕过协议变更。

#### 10.6.3 连接治理

- 服务端实现 gRPC health checking；reflection 仅在开发和测试 profile 开启，生产关闭或限制到内部网络。
- gRPC interceptor 统一处理 request ID、认证 metadata、panic recovery、日志脱敏、指标、deadline 和错误分类。
- 外部网关到内部服务优先使用 mTLS 或明确的服务身份；开发环境可用 insecure profile，但配置必须显式标识。
- 通过 etcd resolver 注册服务，使用 gRPC service config 和 `round_robin` 等明确策略；不把 Go Micro 的节点选择器描述成 gRPC 负载均衡。
- 默认不对副作用 RPC 自动重试；只有幂等读操作或携带幂等键的调用允许有界重试。
- 每个 channel 复用并设置 keepalive、连接上限和 dial/RPC deadline；禁止每次请求创建新连接。

#### 10.6.4 兼容迁移流程

1. 固化现有 Go Micro RPC 的行为和 API E2E 基线。
2. 从现有 `.proto` 提取 `filestore.v1` 契约，生成原生 gRPC server/client。
3. 在 account 和 dbproxy/metadata 服务增加 gRPC listener，与旧 RPC 并行运行。
4. 为网关增加 gRPC adapter，按服务和环境开关切换调用路径。
5. 对读请求先灰度，确认状态码、超时和数据结果一致，再迁移幂等写请求。
6. 保留回滚开关；连续两个版本的 gRPC 运行报告通过后，才考虑移除旧 Go Micro 调用。

#### 10.6.5 Track C 功能需求与验收

| ID | 需求 | 验收标准 |
|---|---|---|
| `FR-GRPC-01` | 新增原生 gRPC server/client，不以 Go Micro client.Call 冒充 | `grpcurl`/官方 gRPC client 能完成 health、反射（测试 profile）和一个真实业务 RPC |
| `FR-GRPC-02` | 统一 `filestore.v1` proto、版本、字段兼容和生成流程 | proto lint、生成代码检查、breaking check 通过；字段编号无复用 |
| `FR-GRPC-03` | 将 account、metadata、job 等控制 RPC 迁移到 gRPC adapter | 网关 HTTP/JSON 契约不变；E2E 结果与旧路径一致 |
| `FR-GRPC-04` | unary RPC 传递 request ID、认证 metadata、deadline 和 expected version | 服务端日志、DB 查询和错误响应可关联；超时能取消下游 SQL |
| `FR-GRPC-05` | 建立 gRPC health、服务发现和明确的节点选择策略 | 两个实例注册 etcd；客户端可分布请求；实例摘除后的错误和恢复时间有报告 |
| `FR-GRPC-06` | 实现 interceptor、状态码映射、脱敏日志和指标 | `Unauthenticated/PermissionDenied/InvalidArgument/Aborted/Unavailable` 与 HTTP 400/401/403/409/503 映射一致 |
| `FR-GRPC-07` | 副作用 RPC 使用幂等键/version/CAS，禁止无条件自动重试 | 丢失响应后重试不产生重复关系、任务或对象 |
| `FR-GRPC-08` | 对需要的跨服务大数据操作提供受限 streaming RPC | 大小、速率、并发和 deadline 生效；默认路径仍优先使用对象存储，不把浏览器流量引入 gRPC |
| `FR-GRPC-09` | 旧 Go Micro 与 gRPC 可配置并存并可回滚 | 任一服务可切换旧/新路径；切换期间 API 结果和 DB 不变量一致 |
| `FR-GRPC-10` | mTLS/服务身份和生产反射策略可配置 | 未授权内部调用被拒绝；生产 reflection 不可从外部访问 |

对应验收用例：

- `AT-GRPC-CONTRACT-01`：proto 生成、lint、breaking check、客户端/服务端互操作。
- `AT-GRPC-GATEWAY-01`：HTTP 网关切换 gRPC 后，注册登录、文件元数据和 job 查询结果一致。
- `AT-GRPC-FAILURE-01`：服务端延迟、实例终止、连接重建和 etcd 摘除；请求在 deadline 内返回并可恢复。
- `AT-GRPC-IDEMPOTENCY-01`：写 RPC 在响应丢失后重复调用；DB 只有一个逻辑结果。
- `AT-GRPC-SECURITY-01`：错误 token、错误服务身份、越权 metadata、生产 reflection 暴露均被拒绝。

只有上述需求和用例全部 `PASS`，才允许在简历中写“原生 gRPC 微服务”“gRPC 服务发现/负载均衡”；否则只能写“Protobuf 定义的 Go Micro RPC”。

### 10.7 Track D：个人文件知识 Agent

#### 10.7.1 建设定位与能力分层

Track D 的 Primary Agent 面向已认证普通用户，而不是运维人员。它只在当前用户有效 `user_file` 关系授权的 `AVAILABLE` 文件上提供自然语言搜索、问答、摘要、比较、结构化提取、标签建议和整理 proposal。

MySQL `user_file` 是授权真相；对象存储保存原文件；抽取文本、全文索引、embedding、摘要和自动标签都是可删除、可重建的派生数据。Track D 不进入上传、下载同步链路，不获得任意 SQL、Shell、Redis、RabbitMQ、URL fetch 或对象存储凭证。

必须区分确定性功能、固定模型工作流和真正的 Agent：

| 能力 | 实现类型 | 简历口径 |
|---|---|---|
| 文件名、类型、日期、标签和全文搜索 | 确定性后端检索 | 智能搜索，不称为 Agent 推理 |
| 指定文件问答、摘要、比较和字段提取 | 固定的检索/生成/校验 workflow | 个人文件知识工作流 |
| 自动分类和候选标签 | 上传后的异步推理 workflow | 自动标签/分类 |
| 根据前一步结果自主选择搜索、元数据、授权 chunk 等多个工具 | 有界 read-only tool loop | 通过验收后才称为 Agent |
| 重命名、接受标签和设置分类 | 用户审批后的确定性命令 | Agent 提议、后端执行 |

#### 10.7.2 P0/P1 用户工作流

| 用户能力 | P0 | P1 |
|---|---|---|
| 自然语言找文件 | 用户输入模糊关键词，后端在当前用户索引执行 metadata + FULLTEXT 检索；类型、日期、标签由 UI 作为显式 typed filters 传入，P0 不从自然语言猜测过滤器 | 模型查询改写、日期/类型意图解析、embedding + FULLTEXT 混合检索、rerank、保存搜索 |
| 多文件问答 | 用户显式选择文件或从个人文件库检索候选；每个关键结论引用 `user_file_id + source_version + chunk_id + locator`；证据不足时拒答 | 大文件集分层检索、对话追问和可复用 knowledge space |
| 摘要/比较/提取 | P0 只提供单文件、有引用摘要，作为问答 workflow 的固定模式 | 多文件比较、JSON Schema 字段提取、超长文档 map/reduce、表格感知比较和自定义模板 |
| 自动分类和标签 | 不进入首版完成门禁 | 按 taxonomy 生成候选分类/标签，保存为 `SUGGESTED`；用户自定义 taxonomy 和反馈学习 |
| 文件整理 | P0 只支持一种低风险操作：单文件 `RENAME` proposal；可信 UI 审批后执行 | 接受/拒绝标签、设置分类、虚拟 collection、受限批量整理和 undo |
| 文件格式 | UTF-8 文本、Markdown、可提取文本 PDF | DOCX/PPTX/XLSX、OCR PDF、图片和表格专用解析 |

搜索可以同步返回；模型问答和摘要使用 `202 Accepted + assistant_run_id`，通过轮询或 SSE 获取状态。P0 不流式输出尚未完成授权复检和引用校验的答案正文。首版完成门禁只要求“找文件 + 有引用问答/摘要 + 单文件重命名 proposal”闭环；其余能力单独按 P1 验收。

#### 10.7.3 目标架构与数据权威

```text
T2: file_object AVAILABLE + user_file active
  -> same-transaction ingestion_job
  -> lease worker or optional Outbox transport
  -> isolated extractor
  -> chunk -> per-user_file index generation -> READY

authenticated request
  -> Gateway derives subject/tenant/roles
  -> Assistant API
       |-- deterministic route: metadata + FULLTEXT search
       |-- fixed workflow: retrieve -> pack -> generate -> validate
       `-- agentic route: bounded read-only typed-tool loop
  -> final ACL/version recheck
  -> citation + schema + DLP validation
  -> buffered result

organization suggestion
  -> structured proposal
  -> trusted UI approval
  -> typed metadata command
  -> authorization + fingerprint + CAS + idempotency
  -> verification and audit
```

权威边界：

- 原文件字节和不可变版本由对象存储负责。
- 文件身份、用户关系、ACL version、用户标签和文件名由 MySQL 负责。
- ingestion job、assistant run、proposal、approval 和 execution 由 MySQL 持久化。
- RabbitMQ 只负责可选异步运输，不能成为索引任务或 Assistant run 的唯一事实源。
- Redis 只用于短期 run 状态、限流、缓存和幂等加速，丢失后可从 MySQL 重建。
- 搜索索引、chunk、embedding、摘要和建议标签均为派生数据，必须可按 source/pipeline generation 重建。

#### 10.7.4 异步 ingestion、删除与重建

```text
PENDING -> FETCHING -> EXTRACTING -> CHUNKING -> INDEXING -> READY
                 |          |            |
                 +-------> RETRY_WAIT ----+
                 `-------> UNSUPPORTED / FAILED_PERMANENT
READY -> REVOKED
```

- T2 创建 `user_file` 时，在同一 MySQL 事务插入唯一 ingestion job；唯一键为 `(user_file_id, source_version, pipeline_version)`。
- P0 可由 MySQL lease worker 调度；Track A 完成后可用 Outbox/RabbitMQ 运输，但 MySQL job 始终是权威。
- extractor 在无网络、无云凭证、非 root、只读根文件系统、丢弃 capabilities 的临时沙箱中运行，并限制 CPU、内存、PID、时长、页数、字符数、chunk 数和输出大小。
- 类型由 magic bytes 判定；P0 禁止宏、脚本、外链、嵌入对象和压缩包展开。解析失败不得产生部分可搜索索引。
- pipeline 变化创建 generation `N+1`；完整生成并校验后原子切换 `active_generation`，失败时不暴露混合 generation。
- 删除/撤权必须在 MySQL 中同步使 `user_file` 失效并递增 `acl_version`，随后异步删除 chunk、引用和缓存；物理索引残留不能继续授权。
- 同一物理对象被多个用户引用时，P0 允许按 `user_file` 生成独立索引投影，以简化授权、删除和共享隔离。
- P0 冻结支持边界：UTF-8 文本/Markdown 和无密码、可提取文本的 PDF；扫描 PDF、加密 PDF、复杂表格、空文本、损坏文件和其他格式返回明确 `UNSUPPORTED/FAILED`。最大页数、字符数、单 chunk 大小、chunk 数和解析时长在 E6 manifest 中预先冻结。
- 上线前已有文件由可暂停、可恢复的 backfill job 扫描；按 `user_file_id` 分页，记录游标和成功/失败数。Track D 完成门禁要求历史可支持文件达到已冻结的收敛比例，不能只覆盖上线后的新文件。

规范化事件包括 `user_file.linked.v1`、`user_file.metadata_changed.v1`、`user_file.deleted.v1`、`knowledge.ingestion.requested.v1`、`knowledge.document.ready.v1`、`knowledge.document.failed.v1`、`organization.proposal.created.v1`、`organization.proposal.approved.v1` 和 `organization.proposal.executed.v1`。事件携带 source/ACL version，但不包含 token、签名 URL、凭证或原始内容。

#### 10.7.5 Assistant API 与类型化工具

Gateway 从认证上下文生成 `subject_id/tenant_id/roles/request_id/run_id`，这些字段不能由模型或请求体覆盖。Agent 只能看到有界、类型化工具：

```text
search_my_files(query, selected_user_file_ids, typed_filters, cursor, limit)
get_my_file_metadata(run_scoped_file_refs)
read_authorized_chunks(run_scoped_citation_refs, max_chunks, max_bytes)
get_my_ingestion_status(user_file_id)
```

工具要求：

- `search_my_files` 只能检索后端认证用户的有效关系，并限制候选文件数、结果数和时间窗。
- `read_authorized_chunks` 只接受本次 run 中前一步授权检索返回的引用，不接受任意 chunk/object key。
- Agent tool loop 只读；模型在最终结构化输出中给出重命名建议，由后端校验后创建 proposal。`create/execute_approved_proposal` 均不在模型可见工具集中。
- P0 整理动作只有单文件 `RENAME`，单事务、全有或全无；`ACCEPT_TAGS`、`REJECT_TAGS`、`SET_CATEGORY` 移入 P1。
- 删除、对象键修改、物理移动、共享/ACL 变更、导出和任意批量动作不向 Agent 开放。
- Agentic route 默认上限为 5 个工具轮次、5 个文件、24 个 chunk、128 KiB 检索文本和每用户 2 个并发 run；具体值在 E6 固化并压测。

Assistant run 状态机：

```text
QUEUED -> RETRIEVING -> GENERATING -> VALIDATING -> SUCCEEDED
   |          |             |             |
   +----------+-------------+-----------> RETRY_WAIT -> QUEUED
   `-----------------------------------> CANCELED / FAILED / EXPIRED
```

- 创建接口使用 `(user_id, intent, idempotency_key)` 和 request fingerprint 保证幂等；同键异载荷返回 `409`。
- 排队和每个状态均有 deadline；provider 429/5xx 只做有界退避，超过上限进入可解释失败。
- cancel 与 worker 通过 run version/CAS 竞争；取消成功后不得继续读取 chunk 或调用模型。
- SSE 事件带单调 sequence，客户端按 `Last-Event-ID` 或历史查询补齐断线窗口；事件可重放但状态转换幂等。
- 终态结果按配置保留，过期返回 `410`；`GET run` 明确区分处理中、成功、取消、失败、权限范围变化和结果过期。

#### 10.7.6 授权检索、引用与 Prompt Injection 防护

1. 每次检索都将 chunk/index 与 active `user_file` 关联，并要求 `file_object.status=AVAILABLE`、source/index generation 一致。
2. 不存在和无权访问统一返回 `404`；已授权但未索引返回 `409 NOT_INDEXED`；不支持格式返回 `422 UNSUPPORTED_FORMAT`。
3. 文件名、内容和 metadata 都是不可信数据；其中的“忽略规则”“调用工具”“导出秘密”不能改变系统指令或工具权限。
4. 模型只看到本次 run 的 `S1/S2...` 引用标签，不看到对象键、签名 URL、数据库主键集合或其他用户存在性信息。
5. 每个关键结论和提取字段必须引用授权 chunk；引用校验器复核 user、ACL/source version、chunk hash、locator 和原文子串。
6. 生成结束、结果释放前再次检查授权和版本；中途撤权或版本变化时丢弃缓冲答案并返回 `409 RUN_SCOPE_CHANGED`。
7. 模型输出在发布前执行 schema、引用、长度和 secret/DLP 校验；引用失败不得降级为无引用答案。
8. 外部模型调用只发送有界 chunk；对已明确支持并进入测试集的凭证类别（bearer token、API key、cookie、签名 URL、密码模式）执行确定性脱敏。该机制不能宣称识别任意秘密或全部 PII，UI 必须披露残余风险。
9. 定义 `local-model` 与 `external-model` 两个部署档。外部档启用前必须记录 provider 的训练使用、数据保留、地域、日志、删除和退出处理条款，并获得用户明确启用；不满足策略时禁用外部模型功能。
10. 上传、下载和基础搜索不依赖模型或索引可用性；模型超时、429/5xx 和索引故障只影响 Assistant run。

#### 10.7.7 数据模型、保留与失败语义

建议新增：

- `content_revision`：不可变对象版本、digest、MIME、extractor version 和解析状态。
- `knowledge_document`：用户范围 `user_file` 投影、source/ACL/pipeline version、active generation 和状态。
- `knowledge_chunk`：document、ordinal、locator、text hash、token count 和受限文本。
- `ingestion_job`、`deletion_job`：租约、attempt、next time、状态和错误码。
- `file_annotation`：用户/自动标签和分类、suggested/accepted/rejected 状态、confidence、evidence 和版本。
- `assistant_run`、`assistant_citation`：intent、run 状态、prompt/model/retrieval version、引用、usage 和 cost。
- `organization_proposal/item/approval/execution`：fingerprint、expected version、过期时间、幂等键、审批和执行结果。

失败语义：

- 多文件任务默认要求全部 `READY`；只有显式 `allow_partial=true` 才可返回 `PARTIAL` 并列出省略原因。
- 证据不足必须返回“无法从授权文件确认”，禁止使用模型记忆补全文件事实。
- JSON Schema 提取失败允许有界重试一次，仍失败则整个 run 失败，不返回半合法结构。
- proposal 无审批、审批过期、fingerprint 不匹配、actor 不同或 expected version 变化时返回 `409`，不得产生副作用。
- 同幂等键同载荷重放原结果，同键异载荷返回冲突；取消和 deadline 到期停止后续工具/模型调用。

默认保留策略：抽取临时文件完成即删且最多 1 小时；删除关系的 chunk/引用在 24 小时内清理；prompt 和检索片段默认不持久化；脱敏 run metadata 保留 30 天；审批/动作审计保留 180 天；应用日志保留 14 天且不含文件内容或凭证。备份过期策略必须单独说明。

#### 10.7.8 功能需求与验收

| ID | 需求 | 验收标准 |
|---|---|---|
| `FR-AGENT-P0-01` | 幂等 ingestion、隔离抽取、generation 激活、删除和重建 | 重复事件、worker 重启和 ACK 丢失后只有一个 READY projection |
| `FR-AGENT-P0-02` | 检索、chunk 读取和结果发布前均执行后端授权/version 校验，授权不依赖模型输出 | 对抗集中未授权工具执行为 0，响应不返回未授权内容、标题、数量或直接标识；timing 测试按冻结环境、样本量和统计阈值报告残余风险 |
| `FR-AGENT-P0-03` | 自然语言 metadata/FULLTEXT 搜索返回授权文件、片段和稳定 locator | 固定 corpus 的 Recall@10、MRR@10、nDCG@10 达到冻结门槛 |
| `FR-AGENT-P0-04` | 指定/检索文件问答使用可验证引用，证据不足时拒答 | 关键声明引用覆盖、引用 hash/locator 和拒答准确率通过 |
| `FR-AGENT-P0-05` | 提供单文件、有引用摘要模式 | 摘要范围受限，关键结论引用可验证，证据不足不补写文件外事实 |
| `FR-AGENT-P0-07` | 单文件重命名使用 proposal、可信审批、fingerprint、CAS、幂等和审计 | 无审批、过期、重放、异 actor、改参数和 stale version 均零副作用 |
| `FR-AGENT-P0-08` | 抽取隔离、注入防护、脱敏、保留、资源上限和 provider 故障隔离 | 恶意文档、解析器异常、秘密种子、超限和模型故障均被拦截或有界失败 |
| `FR-AGENT-P0-09` | 建立可回放 corpus/eval、运行和引用工件、延迟/usage/cost 记录 | 每个声明可追溯到 corpus version、run、tool trace 和原始分母 |
| `FR-AGENT-P0-10` | 提供有界 read-only Agent route，自主选择多个 typed tools | 相比固定 workflow 的 held-out 复杂任务有可测增益，且工具参数全部后端校验 |
| `FR-AGENT-P1-01` | embedding/FULLTEXT 混合检索、rerank 和可重建 generation | provider/embedding 变化后可重建，旧/新 generation 不混用 |
| `FR-AGENT-P1-02` | Office/OCR/图片/表格解析提供格式级 locator | 每种格式的解析、引用、超限和错误样本通过独立验收 |
| `FR-AGENT-P1-03` | collection、受限原子批量整理和 undo | 批量操作不越权、全有或全无，并可按审计记录撤销 |
| `FR-AGENT-P1-04` | 多文件比较和 JSON Schema 字段提取 | 首次 schema 合法率、重试率、run 失败率及字段 precision/recall/F1 可复算 |
| `FR-AGENT-P1-05` | 自动标签/分类只产生 suggestion，不覆盖用户元数据 | 接受/拒绝、用户值优先、模型版本变化和重复事件状态规则通过 |

验收用例包括：

- `AT-AGENT-INGEST-01`：重复事件、worker 崩溃、重试和 pipeline 升级后只有一个 active generation。
- `AT-AGENT-EXTRACT-01`：路径穿越名、恶意 PDF、宏文档、压缩炸弹、解析器超时不能逃逸沙箱或生成 READY 索引。
- `AT-AGENT-SEARCH-01`：固定 corpus 测量 Recall@10、MRR@10 和 nDCG@10，并保存原始 query/result。
- `AT-AGENT-RAG-CITE-01`：错误引用、伪造标签、失效版本、无证据问题和中途撤权均不能发布未验证答案。
- `AT-AGENT-SUMMARY-01`：单文件摘要只使用授权 chunk，关键结论引用有效，无证据内容不补写。
- `AT-AGENT-P1-EXTRACT-01`：P1 比较/提取覆盖指定文件，分别报告首次 schema 合法率、重试率、run 失败率和字段 precision/recall/F1。
- `AT-AGENT-TOOLLOOP-01`：复杂任务要求模型按结果选择后续工具；测量工具选择、合法参数、轮次、任务成功和成本。
- `AT-AGENT-TAG-01`：自动标签不覆盖用户标签，重复事件和模型版本变化保持状态规则。
- `AT-AGENT-SCOPE-01`：跨用户、伪造 tenant/file ID 和同名同 hash 样本中，未授权工具调用均不执行，响应不包含对方内容、标题、数量或直接标识；timing 对比固定硬件、并发、样本量和允许统计差异并报告残余风险。
- `AT-AGENT-DELETE-01`：物理 chunk 未清理时，逻辑删除已立即阻止检索、问答和 citation。
- `AT-AGENT-REINDEX-01`：失败 generation 不可见，成功切换只暴露一个完整 generation。
- `AT-AGENT-APPROVAL-01`：缺失、伪造、过期、重放、异 actor、改参数和 stale version 审批均零 mutation。
- `AT-AGENT-INJECTION-01`：版本化攻击 corpus 中，文件名/内容指令导致的未授权工具执行为 0，后端跨用户读取为 0，已植入并声明支持检测的 secret 类别不出现在输出/日志；结果不宣称对未知攻击普遍免疫。
- `AT-AGENT-REDACT-01`：种子 token、签名 URL、密码和 API key 不出现在模型请求快照、输出、Trace 或日志。
- `AT-AGENT-FAILURE-01`：索引、模型、embedding 或 parser 故障不影响上传、下载和基础 metadata API。
- `AT-AGENT-DOS-01`：超大查询、文件数、chunk、抽取输出和并发 run 被 `413/429` 有界拒绝且无孤儿状态。
- `AT-AGENT-EVAL-01`：五次重复 run 生成准确率、引用、工具、延迟、usage/cost 和稳定性报告。
- `AT-AGENT-INTEGRATION-01`：可选全栈集成 trace 证明 Go Assistant orchestration -> 原生 gRPC typed tools -> MySQL ACL/run -> 对象存储 -> RabbitMQ ingestion -> Redis 限流/幂等缓存链路；未通过时不得把这些技术全部归因于 Agent Track。

E6 在实现调优前提交版本化 eval manifest，至少包含 50 条分层搜索 query、40 条问答/拒答任务和 20 条复杂 tool-loop 任务，并固定 train/dev/held-out 划分。建议首版门槛：搜索 Recall@10 >= 0.85、MRR@10 >= 0.70；关键声明 citation precision >= 0.95、citation recall >= 0.90；无证据拒答准确率 >= 0.90；tool-loop held-out task success >= 0.80，且相对固定 workflow 的绝对提升至少 0.10。安全硬门禁为：测试集中未授权工具执行 0 次、未授权文件直接标识/标题/内容/数量返回 0 次、有效 citation identity/hash 比例 100%、无有效审批的 mutation 0 次。所有指标报告原始分子/分母、p50/p95、单 run usage/cost，并明确这些是本地验收目标而非已取得结果。

JSON Schema 提取移入 P1；必须分别报告首次生成合法率、重试率和整次 run 失败率。“发布结果 100% schema-valid”只证明 validator 拦截生效，不作为模型质量指标。

Track D P0 完成要求所有 `FR-AGENT-P0-*` 和对应 `AT-AGENT-*` 为 `PASS`。只完成搜索、问答等固定 workflow 时，简历应写“个人文件知识助手/工作流”；只有 `AT-AGENT-TOOLLOOP-01` 通过后，才能写“个人文件知识 Agent”。

#### 10.7.9 可选项与非目标

P0 不做多 Agent、全盘自动整理、自动删除、共享权限变更、任意 Web 搜索、持久聊天记忆、未授权跨用户知识库、生产级 OCR 或所有 Office 格式支持。多 Agent 只有在存在独立权限域和独立评估标准时才考虑。

### 10.8 可选 Track E：内部上传可靠性助手

原“上传可靠性与审计 Agent”降为内部可选项，不阻塞 Track D，也不与 Track D 共享用户身份、索引、工具或权限。

可选 MVP 仅提供只读 `job_id` 诊断：查询 job、multipart、Outbox/MQ、对象 metadata 和版本化 runbook，输出结构化状态、证据和建议。它需要 Track A 的可靠 job/Outbox 证据后，才能声称 RabbitMQ 诊断有效；第一版不开放自动 remediation。可使用独立的 `FR-OPSAGENT-E-*` 和 `AT-OPSAGENT-*` 验收，未通过时不得写入正式简历。

## 11. 非功能与安全需求

### 11.1 非功能需求

| ID | 要求 |
|---|---|
| `NFR-01` | 文件与用户关系不允许半提交 |
| `NFR-02` | 跨 MySQL、MQ、对象存储的中间状态必须可解释、可重试、可对账 |
| `NFR-03` | init、part、complete、Outbox publish、consumer、redrive 均幂等；幂等键为 `(user_id, operation, idempotency_key)`，保存 request fingerprint 和终态 response；同键异载荷返回 `409` |
| `NFR-04` | 对象进入 AVAILABLE 前完成服务端整对象校验；Range 仅从已验证且不可变的精确 object version 提供，服务端校验 `Content-Range`/长度，测试端比较返回字节 |
| `NFR-05` | 文件大小、会话、暂存、连接池、worker、prefetch 和 HTTP 并发均有界 |
| `NFR-06` | 上传、下载和摘要计算使用流式 I/O，内存不随完整文件大小线性增长 |
| `NFR-07` | 依赖不可用时在配置 deadline 内失败，不无限等待 |
| `NFR-08` | 进程退出后，持久任务和 multipart 可由其他实例恢复 |
| `NFR-09` | schema、状态机、消息契约和配置版本化，并用 ADR 记录关键决策 |
| `NFR-10` | 干净 Linux 环境可复现默认栈，配置档边界清晰 |
| `NFR-11` | Assistant 检索必须以前置授权、chunk 读取复检和结果发布前复检为边界；撤权后旧索引不得继续授权 |
| `NFR-12` | 文本、chunk、全文索引、embedding、摘要和标签均为按 source/pipeline generation 可删除、可重建的派生数据 |
| `NFR-13` | parser、索引、embedding 和模型 provider 的故障、超时或限流不得影响上传、下载和基础文件管理 API |
| `NFR-14` | 模型支持的回答和提取结果必须通过引用、schema、长度和敏感信息校验后才能发布 |

### 11.2 安全需求

- 保留 bcrypt 密码、随机 token、token SHA-256 落库、过期和撤销。
- bearer/session token 仅通过受保护的 header/cookie 传递，不允许 query/form token；短期下载签名可以使用 query 参数，但必须绑定资源、用户或授权范围、过期时间及 nonce/key ID，不能作为 bearer token。
- 生产模式拒绝默认数据库密码、Redis 密码和下载签名 secret。
- CORS 使用允许列表，并明确上传、下载和 Range 所需 header。
- 仅信任配置的反向代理，防止 forwarded host/proto 污染返回 URL。
- 限制文件大小、会话数、暂存空间、用户/IP 速率和并发数。
- 文件名执行路径穿越、控制字符和 XSS 防护。
- 签名下载使用恒定时间比较并设置短有效期；记录签发后无法即时撤销的边界。
- 日志、Trace 和测试工件不得记录明文 token、密码、签名 URL、云凭证或未脱敏文件正文。
- Assistant 的文件抽取器必须运行在无网络、无凭证、非 root、资源有界的临时沙箱中。
- 文件名、文件内容、metadata 和引用片段均按不可信输入处理，不能改变系统指令、工具权限或授权范围。
- 外部模型只接收当前用户已授权、数量有界并完成确定性脱敏的 chunk；不得发送完整文件或其他用户存在性信息。

## 12. 实施路线与完成门禁

### 阶段 A：归属与可复现基线

交付物：上游说明、许可证、个人提交、可跟踪依赖与启动文件、当前测试清单。  
门禁：干净克隆可以启动；公开 README 与真实实现一致。

### 阶段 B：P0 正确性

交付物：类型化事务 RPC、Redis 原子会话、multipart 状态机、幂等 complete、对象校验、安全默认值、优雅退出。  
门禁：P0 测试全部 `PASS`；受支持配置之外的多副本本地存储标记为 `N/A/UNSUPPORTED`，只有已登记且有可执行失败测试的缺陷才标记 `XFAIL`。

### 阶段 C：Outbox 与 RabbitMQ

交付物：job/outbox schema、relay、confirm/return、幂等消费者、retry/DLQ/redrive、202 状态 API。  
门禁：所有定义的发布和消费崩溃窗口通过故障测试。

### 阶段 D：MinIO multipart 与多实例恢复

交付物：S3 multipart provider、跨实例 part、异步校验、abort 清理、浏览器刷新恢复。  
门禁：两个 upload 实例无 sticky session，使用确定性 failpoint 覆盖关键窗口并补充随机终止 smoke test；在固定资源条件下仍完成 1 GiB 文件且 SHA-256 一致。

### 阶段 E：Track C 原生 gRPC

交付物：`filestore.v1` proto、原生 gRPC server/client、interceptor、health、etcd resolver、明确重试与 mTLS profile、HTTP 网关 adapter、旧路径回滚开关。  
门禁：`AT-GRPC-*` 全部 `PASS`；未通过前不得在简历写“原生 gRPC 微服务”。

### 阶段 F：Track D 个人文件知识 Agent

交付物：授权 ingestion/indexing、隔离抽取、全文/metadata 检索、有引用问答/单文件摘要、审批式重命名、Assistant run 生命周期和回放评估集；比较、提取、自动标签、Office/OCR 作为后续 P1 交付。  
门禁：`FR-AGENT-P0-*` 与对应 `AT-AGENT-*` 全部 `PASS`；先 read-only，再开放单文件重命名；Agent 不进入上传/下载同步链路。

### 阶段 F-optional：Track E 内部上传可靠性助手

交付物：只读 job 证据工具和故障诊断 workflow。  
门禁：Track A 证据和 `AT-OPSAGENT-*` 通过后，才可作为额外 Agent/SRE 作品展示；不影响 Track D。

### 阶段 G：观测与容量证据

交付物：结构化日志、request/job/trace/agent_run ID、核心指标、固定负载、测试报告和演示脚本。  
门禁：一次故障可由 job ID 或 agent_run_id 定位；性能和 Agent 评估数字可从原始工件复算。

### 扩展阶段

根据求职时间选择一至两项：OpenTelemetry、OSS 认证、引用安全 GC/对账、Kubernetes 加固、Agent embedding 检索。不要同时铺开所有方向。

## 13. 验证计划

### 13.1 环境

| 环境 | 组成 | 用途 |
|---|---|---|
| `E0` | Go 工具链，无外部依赖 | 单元、静态、race、fuzz |
| `E1` | 六个可执行程序 + MySQL + Redis + etcd + 本地存储 | 默认功能和 P0 验证 |
| `E2` | E1 + RabbitMQ + MinIO | Outbox、消息、对象存储和 multipart 集成 |
| `E3` | 两个以上 upload 实例 + 负载入口 + E2 依赖 | 跨实例恢复和实例故障 |
| `E4` | 独立 OSS 或 Ceph 测试环境 | 可选 provider 认证 |
| `E5` | E2 + 原生 gRPC listener/client + etcd resolver + 双实例 | Track C 协议、超时、发现、负载和回滚验证 |
| `E6` | E2/E3 + User Assistant API + 脱敏个人文件 corpus + 回放集 | Track D ingestion、授权检索、引用、注入防护、审批和成本评估 |
| `E7` | E2 + Optional Ops Assistant + 脱敏 job 故障回放集 | Track E 内部诊断扩展，不作为 Track D 的证据 |

每次运行必须记录：Git commit、dirty/untracked 清单、补丁摘要、Go/Node/浏览器/Docker/中间件版本、CPU、内存、磁盘、容器资源限制和脱敏配置。E6 还必须记录模型/provider 版本、prompt/template 版本、工具 schema 版本和评估集版本。

### 13.2 分层测试

**单元测试**

- 输入校验、签名、Range、geometry、状态机、CAS、重试策略和错误映射。
- `go test ./...`、`go test -race ./...`。
- parser、文件名、Range、part index 和签名使用 fuzz/property test。

**组件集成测试**

- MySQL：事务回滚、并发唯一键、Context 取消、连接池耗尽。
- Redis：Lua 原子性、TTL、状态 CAS、锁 ownership、过期竞争、重启。
- RabbitMQ：confirm、return、断线、重复投递、重试、DLQ/redrive。
- MinIO：create/upload/list/complete/abort、Range、覆盖防护、timeout。
- Go Micro RPC：保留旧路径兼容测试。
- Native gRPC：proto compatibility、health/reflection、interceptor、status mapping、deadline/cancellation、etcd resolver、round_robin、mTLS、streaming limits、retry/idempotency。
- User Assistant：typed tool schema、authorized retrieval、citation validation、schema output、redaction、approval、CAS、audit、event dedupe、provider/model error handling。
- Optional Ops Assistant：job evidence scope、runbook retrieval、read-only diagnosis、event dedupe and audit。

**API E2E**

- 注册、登录、过期、撤销、跨用户越权。
- 普通上传、服务端摘要、同用户秒传和跨用户 proof-of-possession。
- init、resume、重复 part、并发 part、cancel、complete、重复 complete。
- 完整下载、合法/非法 Range、重命名、删除。
- 每次变更后同时核对响应、下载字节、DB、Redis、队列和对象状态。

**浏览器 E2E**

- 上传、暂停、刷新、重新选择同一文件、恢复、下载、重命名、删除。
- Track D：用户凭模糊描述找到目标文件；问答中的引用可打开正确文件版本和页码/文本位置；用户可预览并确认一次重命名 proposal。
- Track D UI 覆盖索引处理中、未索引、不支持格式、模型处理中、取消、失败、结果过期和生成期间撤权状态。
- Agent 答案在授权/引用校验完成前不显示未验证正文；审批必须来自可信 UI 控件，而不是对话文本。
- 桌面和移动视口，无未预期浏览器异常。
- 特殊文件名只作为文本显示。

### 13.3 可执行验收用例目录

每个 `FR-*` 至少关联一个 `AT-*`。测试用例必须固定前置数据、容器资源、并发数、故障注入点、最大完成时间、清理期限和精确断言，不能只写“最终收敛”。建议首批用例如下：

| ID | 覆盖范围 | 固定条件与判定 |
|---|---|---|
| `AT-BASELINE-01` | FR-P0-01/02/19 | 固定 commit 和 source-state，在 E1 从干净克隆启动；记录命令、退出码和所有容器版本 |
| `AT-DB-TX-01` | FR-P0-05 | 在 T2 SQL 边界触发 failpoint；断言 `file_object`、`user_file`、job 无半提交 |
| `AT-REDIS-SESSION-01` | FR-P0-08/09 | 并发 32 次 init，短 TTL 过期竞态；断言只有一个有效 session 且没有无 TTL key |
| `AT-IDEMPOTENCY-01` | FR-P0-12、NFR-03 | 同用户同 operation/key 同载荷返回相同 response；异载荷返回 409；COMMIT 后断连接可查询结果 |
| `AT-MP-RESOURCE-01` | FR-P1-13、NFR-05/06 | E3，1 GiB 文件，16 MiB part，并发 4，upload 容器 512 MiB；峰值 RSS ≤384 MiB、goroutine ≤预热基线+64、FD ≤基线+32，停止流量 60 秒后回到基线+10%，字节和 SHA-256 完全一致 |
| `AT-MP-REPLICA-01` | FR-P1-18/19 | 两 upload 实例、无 sticky session、交替 part、确定性实例退出；断言另一实例完成且用户/对象关系唯一 |
| `AT-MQ-OUTBOX-01` | FR-P1-02/04/05/09 | 覆盖提交后未发布、confirm 前断线、return、confirm 后未标 sent；断言 event 最终 SENT 或明确 retry 状态 |
| `AT-MQ-IDEMPOTENT-01` | FR-P1-07/08 | 重复 delivery、ACK 丢失、DLQ redrive；断言 job 终态唯一，关系和 canonical object 不重复 |
| `AT-RESUME-01` | FR-P1-19 | 用户绑定 opaque session ID；刷新并重新选择同一文件恢复；跨用户、过期凭证、同名同尺寸异内容均拒绝 |
| `AT-GRPC-CONTRACT-01` | FR-GRPC-01/02 | E5 中执行 proto lint、breaking check、health、reflection（测试 profile）和真实业务 RPC 互操作 |
| `AT-GRPC-FAILURE-01` | FR-GRPC-04/05/06/09 | deadline、实例终止、etcd 摘除和连接重建均在有界时间内返回；切回 Go Micro 后结果一致 |
| `AT-GRPC-IDEMPOTENCY-01` | FR-GRPC-07 | 写 RPC 响应丢失后重试；DB 只有一个逻辑结果，旧 version 返回冲突 |
| `AT-GRPC-SECURITY-01` | FR-GRPC-10 | 错误服务身份、越权 metadata、生产 reflection 暴露均被拒绝 |
| `AT-AGENT-INGEST-01` | FR-AGENT-P0-01 | E6 重复事件、worker 重启和 pipeline 升级后只有一个 READY projection |
| `AT-AGENT-SEARCH-01` | FR-AGENT-P0-03 | 固定个人 corpus 测量 Recall@10、MRR@10、nDCG@10 并保存 query/result |
| `AT-AGENT-RAG-CITE-01` | FR-AGENT-P0-04 | 错误引用、伪造标签、失效版本、无证据问题和中途撤权均不能发布未验证答案 |
| `AT-AGENT-SUMMARY-01` | FR-AGENT-P0-05 | 单文件摘要只使用授权 chunk，关键结论引用有效，无证据内容不补写 |
| `AT-AGENT-TOOLLOOP-01` | FR-AGENT-P0-10 | 复杂任务按结果选择后续工具；工具参数合法，轮次、任务成功和成本可测 |
| `AT-AGENT-SCOPE-01` | FR-AGENT-P0-02/08 | 未授权工具执行为 0，响应不含对方内容/标题/数量/直接标识；timing 测试固定环境和统计阈值并披露残余风险 |
| `AT-AGENT-DELETE-01` | FR-AGENT-P0-01/02 | 物理 chunk 未清理时，逻辑删除已立即阻止检索、问答和 citation |
| `AT-AGENT-REINDEX-01` | FR-AGENT-P0-01 | 失败 generation 不可见，成功切换只暴露一个完整 generation |
| `AT-AGENT-APPROVAL-01` | FR-AGENT-P0-07 | 无审批、过期、重放、异 actor、改参数和 stale version 均零 mutation |
| `AT-AGENT-INJECTION-01` | FR-AGENT-P0-08 | 版本化攻击 corpus 中未授权工具执行为 0、跨用户读取为 0；不宣称对未知攻击普遍免疫 |
| `AT-AGENT-REDACT-01` | FR-AGENT-P0-08 | 测试集植入且声明支持的 token、签名 URL、密码和 API key 类别不出现在模型请求、输出、Trace 或日志 |
| `AT-AGENT-FAILURE-01` | FR-AGENT-P0-08 | 索引、模型、embedding 或 parser 故障不影响上传、下载和基础 metadata API |
| `AT-AGENT-EVAL-01` | FR-AGENT-P0-09 | 五次重复 run 产生准确率、引用、工具、延迟、usage/cost 和稳定性报告 |
| `AT-AGENT-INTEGRATION-01` | Track A/C/D 集成门禁 | 一条 trace 证明 Go Assistant、gRPC typed tools、MySQL ACL/run、对象存储、RabbitMQ ingestion 和 Redis 限流/缓存的真实调用关系 |
| `AT-OPSAGENT-DIAG-01` | FR-OPSAGENT-E-* | E7 只读 job 诊断；仅作为可选 Track E 证据 |

资源阈值是本地验收阈值，不是生产 SLO；只有保留原始报告后才能用于简历。

### 13.4 故障注入矩阵

| ID | 故障/注入点 | 允许中间状态与目标结果 |
|---|---|---|
| `AT-DEP-REDIS-01` | Redis 在普通上传时不可用 | 普通上传继续；multipart 返回有界 503，feature health 报警 |
| `AT-DB-TX-01` | T2 两条 SQL 之间失败 | 整个事务回滚，无半提交 |
| `AT-MP-INFLIGHT-01` | part 写入中触发 failpoint 并退出 upload | 不产生完成标记；旧 generation part 可清理 |
| `AT-MP-COMPLETE-01` | provider complete 前后、响应丢失、`NoSuchUpload` | 通过精确 staging key/version 和 provider 状态恢复；最终只有一个可解释终态 |
| `AT-MP-LEASE-01` | lease 过期后旧 worker 恢复 | 旧 fencing epoch 的结果被 DB CAS 拒绝，不得覆盖 canonical object |
| `AT-MP-CANCEL-01` | cancel/expire 与在途 part 并发 | 等待或拒绝在途租约后再 abort；最终状态唯一且清理有期限 |
| `AT-MQ-PUBLISH-01` | broker 发布前后断网 | Outbox 保留或重发，任务不静默丢失 |
| `AT-MQ-CONFIRM-01` | confirm 后、标记 sent 前退出 relay | 允许重复发布，按 event ID 幂等收敛 |
| `AT-MQ-ACK-01` | 重试事务提交后 ACK 丢失 | 重投不重复建立逻辑资源；DB 状态可解释 |
| `AT-MQ-RETURN-01` | 消息不可路由 | 收到并关联 return，Outbox 不标记 sent |
| `AT-OBJECT-REGISTER-01` | UploadPart 成功但 Redis/DB 登记失败 | 通过 provider ListParts 和 MySQL job 恢复，不依赖单次响应 |
| `AT-VERIFY-CAS-01` | 校验完成但 DB CAS 失败 | 对象保持 staging/可重试，不进入 AVAILABLE；不得覆盖已有对象 |
| `AT-DEDUPE-OBJECT-01` | 删除或篡改远端对象后秒传 | 秒传拒绝并产生一致性告警 |
| `AT-MP-REPLICA-01` | 两 upload 实例交替上传 part | 任一实例可 complete，最终摘要一致 |
| `AT-MP-FAILOVER-01` | 中途终止一个 upload 实例 | 另一实例恢复并完成；不据此宣称依赖整体高可用 |
| `AT-RESOURCE-01` | 磁盘或对象存储空间耗尽 | 请求失败，不生成 AVAILABLE 元数据，残留可清理 |
| `AT-LIFECYCLE-01` | `SIGTERM` 时存在请求 | 停止接入，在期限内完成或取消并释放连接 |

### 13.5 性能验证原则

性能指标必须先冻结环境和工作负载，再采集基线和确定目标。每个数字至少包含：

- 硬件与容器限制；
- 文件大小和数据类型；
- 并发、持续时间和样本量；
- p50、p95、p99、吞吐和错误率；
- CPU、RSS、goroutine、FD、DB/Redis pool、queue depth；
- 至少三次独立运行的中位数和波动。

“无持续资源增长”和“队列可清空”必须转成可判定阈值，例如固定测试时长、允许增长斜率和停止流量后的排空期限。阈值是验收标准，不是实测成绩。

### 13.6 证据工件

```text
results/<commit>/<timestamp>/
  manifest.json
  source-state.txt
  environment.txt
  config.redacted.env
  junit/
  coverage/
  benchmarks/
  logs/
  metrics/
  traces/
  screenshots/
  hashes.csv
  db-before.sql
  db-after.sql
  redis-before.txt
  redis-after.txt
  queue-state.json
  object-inventory.json
  fault-timeline.json
  agent/
    corpus-manifest.json
    retrieval-results.jsonl
    citation-validation.jsonl
    tool-traces.jsonl
    provider-payloads.redacted.jsonl
    approvals.jsonl
    eval-report.json
```

`manifest.json` 必须关联 requirement ID、test ID、状态、命令、版本、seed、时长和工件摘要。Agent 工件必须记录 corpus、pipeline、prompt、model/provider、tool schema 和授权策略版本，但不得保存未脱敏文件正文、token 或签名 URL。简历中的每个完成项和数字都应能反查到该 manifest。

## 14. 简历能力建设建议

### 14.1 最值得投入的六项能力

| 排名 | 建设项 | 原因 |
|---|---|---|
| 1 | MySQL Outbox + RabbitMQ 可靠任务 | 同时体现事务边界、消息语义、幂等、重试和故障恢复，面试价值最高 |
| 2 | Redis multipart 状态机与分布式协调 | 能从简单 KV 使用上升到原子脚本、租约、TTL、并发控制和降级 |
| 3 | MinIO/S3 multipart 多实例恢复 | 直接回答“前端切片为什么算后端亮点”，并展示流式 I/O 和对象存储 |
| 4 | 原生 gRPC 控制面迁移 | 体现协议设计、Protobuf 兼容、拦截器、服务发现、超时和回滚，而不是只写一个依赖 |
| 5 | 个人文件知识 Agent | 将 Agent 能力落到授权检索、RAG 引用、文件理解、审批式整理和安全评估 |
| 6 | 故障注入 + 指标 + 证据链 | 把 Agent/SRE 背景转化为开发质量优势，而不是与后端开发割裂 |

### 14.2 不建议优先投入

- 为了关键词从 `database/sql` 整体迁移到 GORM；
- 仅部署 MySQL 主从但不实现路由、读后写一致性和延迟策略；
- 同时接入 OSS、Ceph、MinIO 三个 provider，却没有一个通过完整契约测试；
- 先做 Kubernetes、服务网格或多云复制，再补上传正确性；
- 只增加中间件依赖和 Docker 容器，不建设故障语义和验证过程。

### 14.3 与 Agent/SRE 经历的衔接

无需弱化现有 Agent/SRE 内容，而应把优势落到开发闭环：

- 用 Agent 辅助生成故障用例可以作为效率工具，但需求、状态机和验收由你设计并解释。
- Agent 项目主线优先选择“个人文件知识助手”，因为它直接服务网盘用户，同时能复用对象存储、MySQL 用户关系、Redis 会话、异步索引和 gRPC typed tools。
- 用授权检索、可验证引用、隔离抽取、Prompt Injection 防护、审批、CAS、幂等、post-condition 和审计证明 Agent 是受控的软件工程能力，不是简单接一个聊天接口。
- 上传故障诊断助手仅作为可选 Track E，适合在 Track A 完成后补充 Agent/SRE 特色。
- 用 SRE 思维定义 SLI、依赖健康和故障恢复，但简历先写具体开发机制，再写观测结果。
- 用故障注入证明 Outbox、幂等消费和 multipart 恢复，而不是泛泛写“提升高可用性”。
- 用结构化日志、指标和测试 manifest 证明你不仅会部署，也会实现和验证业务代码。

## 15. 简历声明与证据映射

| 声明 | 当前状态 | 可使用门槛 | 对应验收 | 当前建议措辞 |
|---|---|---|---|---|
| 六服务结构 | 可证明 | 能解释默认活跃路径和数据面边界 | `AT-BASELINE-01` | “梳理六个服务进程的职责，分离网关控制面与文件数据面” |
| Go Micro RPC | 可证明 | 保留当前生成代码和运行报告 | `AT-BASELINE-01` | “基于 Protobuf 定义 Go Micro RPC” |
| etcd 注册发现 | 配置存在 | E1 当前运行工件 PASS | `AT-RPC-REGISTRY-01`（待补） | “接入 etcd registry”，暂不写“已验证” |
| Redis 断点续传 | 单节点原型 | P0 Redis/竞态测试 PASS | `AT-REDIS-SESSION-01`、`AT-IDEMPOTENCY-01` | “Redis 管理分块会话、TTL 和已上传分块标记” |
| 无 sticky session 的多实例 multipart | 当前不成立 | `FR-P1-11` 至 `FR-P1-19` 全部 PASS | `AT-MP-REPLICA-01`、`AT-MP-FAILOVER-01`、`AT-RESUME-01` | 通过 Track B 后再使用 |
| MySQL 事务一致性 | 当前不成立 | `FR-P0-05` PASS | `AT-DB-TX-01` | 暂写“持久化文件和用户关系” |
| RabbitMQ 可靠任务 | 当前不成立 | Outbox、confirm、幂等、重试故障测试 PASS | `AT-MQ-OUTBOX-01`、`AT-MQ-IDEMPOTENT-01` | 当前写“异步转存原型” |
| 云对象存储 | 只有适配器 | provider 契约认证 PASS | `AT-PROVIDER-01`（待补） | “实现适配器”，不写“海量高可用” |
| 去重与秒传 | 有安全缺口 | `FR-P0-14/15` PASS | `AT-DEDUPE-OBJECT-01` | “基于元数据的去重/秒传原型” |
| 原生 gRPC 微服务 | 当前不成立 | Track C 的 `FR-GRPC-*`/`AT-GRPC-*` 全部 PASS | `AT-GRPC-CONTRACT-01`、`AT-GRPC-FAILURE-01` | 当前只能写 Go Micro RPC |
| 个人文件知识 Agent | 当前不成立 | Track D 的 `FR-AGENT-P0-*`/`AT-AGENT-*` 全部 PASS；Agentic claim 还需 `AT-AGENT-TOOLLOOP-01` | `AT-AGENT-SEARCH-01`、`AT-AGENT-RAG-CITE-01`、`AT-AGENT-APPROVAL-01` | 当前只能写“规划个人文件知识助手” |
| 内部上传可靠性助手 | 当前不成立 | 可选 Track E 的 `FR-OPSAGENT-E-*`/`AT-OPSAGENT-*` 全部 PASS | `AT-OPSAGENT-DIAG-01` | 不作为 Track D 的用户 Agent 证据 |
| API/浏览器检查通过 | 当前未运行 | 固定提交机器报告 PASS | `AT-BASELINE-01` 及 E2E manifest | 当前写“存在脚本化检查” |
| 性能提升 X% | 无证据 | 固定基线和原始报告 | `AT-PERF-*`（待补） | 暂不使用 |

## 16. 简历文案建议

### 16.1 候选文案：仅在归属审计和当前验收后使用

以下不是当前正式简历内容，而是通过个人贡献审计和对应 `PASS` 工件后的候选版本：

> **Go 微服务文件存储系统｜开源课程项目二次开发与工程化重构**  
> 技术栈：Go、Gin、Go Micro、Protobuf、MySQL、Redis、RabbitMQ、Docker
>
> - 在已披露上游项目基础上，负责网关、账号、上传、下载、数据库代理和异步转存模块的工程化重构；使用 Protobuf 定义 Go Micro RPC，并以运行报告证明注册发现配置。
> - 设计 Redis multipart 会话协议，服务端负责 session ownership、TTL、part 校验、generation/CAS 和幂等完成；仅在 `AT-*` 报告通过后使用该完成时表述。
> - 设计 MySQL Outbox 与 RabbitMQ 任务链路，使用 publisher confirm、mandatory return、幂等消费者和有界重试；仅在 Track A 全部通过后写入正式简历。
> - 或者，基于 MinIO/S3 multipart 实现无 sticky session 的多实例续传和服务端 SHA-256 校验；仅在 Track B 全部通过后写入正式简历。
> - 若 Track C 通过：将内部控制面迁移到原生 gRPC，补充 Protobuf 兼容检查、interceptor、deadline、etcd resolver、mTLS 和回滚验证；仅在 `AT-GRPC-*` 全部通过后写入。
> - 若 Track D P0 通过：建设面向用户的个人文件知识 Agent，提供授权全文搜索、有引用问答/单文件摘要和审批式重命名；比较、结构化提取、自动标签等仅在对应 P1 验收通过后追加。
> - 若可选 Track E 通过：补充只读上传 job 可靠性诊断助手；该能力单独表述，不与用户侧 Agent 混写。

正式简历 bullet 必须同时具备：个人代码增量、对应 requirement ID、当前 `PASS` 的 `AT-*`、commit 和工件路径。只有仓库中存在的课程原始能力，只能作为项目背景，不能写成个人职责。

### 16.2 完成 P0 后

> - 使用类型化 RPC 和 MySQL 事务原子提交文件元数据与用户关系，透传 Context/deadline，并通过 SQL 边界故障注入验证无半提交。
> - 将 Redis 分块会话改为原子初始化和显式状态机，使用 owner token 租约协调完成操作，覆盖重复请求、过期、取消和并发竞态。
> - 修复秒传的对象存在性和跨用户授权问题，增加服务端对象校验、上传资源上限、HTTP 状态映射和优雅退出。

### 16.3 完成 Track A、Track B、Track C、Track D 或 Track E 后

**Track A：Outbox + RabbitMQ**

> - 设计 MySQL 上传状态机与事务型 Outbox，将可恢复任务事实与待发布事件提交到 MySQL；通过 RabbitMQ publisher confirm、mandatory return、幂等消费者、有界重试和 DLQ 观察实现 at-least-once 异步任务链路。
> - 通过 broker 断网、confirm 前后退出、worker 重复投递和 ACK 丢失实验，验证任务状态可解释并最终收敛。

**Track B：MinIO/S3 multipart**

> - 基于 MinIO/S3 multipart 将分块字节迁移到共享对象存储，使用 Redis 管理短期会话和协调租约、MySQL 保存 generation/状态/终态响应，支持无 sticky session 的多实例续传。
> - 通过浏览器刷新恢复、part 并发竞态、确定性实例退出和服务端 SHA-256 校验，验证上传结果唯一、可恢复且字节一致。

**Track C：原生 gRPC**

> - 将内部控制面从 Go Micro RPC 演进为原生 gRPC，统一 `filestore.v1` Protobuf 契约，补充 interceptor、health、deadline/cancellation、etcd resolver、mTLS profile、幂等重试和 HTTP 网关兼容切换。
> - 通过 proto breaking check、双实例发现、服务退出、响应丢失重试和未授权调用测试，验证协议兼容、故障有界和回滚能力。

**Track D：个人文件知识 Agent**

> - 建设面向用户的个人文件知识 Agent，通过后端授权范围和有界 typed tools 完成全文找文件、有引用问答和单文件摘要；对撤权、版本、引用和 Prompt Injection 进行后端校验。
> - 为单文件重命名设计 proposal -> trusted approval -> typed command -> CAS 执行链路，通过幂等键、过期审批和版本冲突测试保证模型不能直接修改文件元数据。
> - 多文件比较、JSON Schema 提取、自动标签/分类、Office/OCR 和 collection 仅在各自 P1 验收通过后写入简历。

**Track E：可选内部上传可靠性助手**

> - 在 Track A 完成后，补充只读 job 诊断和 runbook 检索；不与用户侧 Agent 共享身份、索引或权限，也不作为 Track D 的完成依据。

只有对应 Track 的 requirement 和 `AT-*` 在 manifest 中全部为 `PASS` 时，才使用对应完成时表述；另一 Track 只能写在 roadmap 中。

### 16.4 原始材料中的待核对或需降级表述

以下是根据当前项目材料和用户提供的简历截图整理的核对项，不代表 PDF 全文逐字引用；正式修改前应以最终提交的简历版本、页码和 SHA-256 为准。

| 待核对表述 | 建议处理 |
|---|---|
| GORM | 删除，当前没有使用 |
| gRPC | 当前代码只能写 Go Micro RPC；完成 Track C 且 `AT-GRPC-*` 全部 PASS 后，才可写“原生 gRPC 微服务” |
| MySQL 主从、读写分离 | 删除，当前只有单 DSN/单连接池 |
| Postman 完成接口联调 | 改为脚本化 API/E2E 检查，除非补充真实 collection 和报告 |
| 支持秒传、分块上传、断点续传 | 限定为“当前单节点原型”；P1 通过后再写多实例 |
| RabbitMQ 提升高可用性 | 改为当前“异步转存原型”；完成 Outbox 后写具体消息语义 |
| OSS 海量、弹性、水平扩展 | 改为“OSS 适配器”；通过认证和扩展测试后再升级 |
| API 网关实现负载均衡 | 改为“服务发现和客户端节点选择”，不要写自研负载均衡 |
| 提升系统可靠性与弹性 | 改成通过的具体故障，例如“worker 退出后任务可恢复” |
| 面向用户的智能网盘 Agent | 当前不能写已实现；Track D P0 通过后可写授权搜索、有引用问答/摘要和审批式重命名，P1 能力逐项验收后追加 |
| 运维 Agent/自动修复 | 降为可选 Track E，不作为用户侧 Agent 的默认能力；只有独立验收后单独表述 |

## 17. 面试演示流程

1. **说明来源**：展示上游 URL/ref、许可证、baseline 和个人提交。
2. **解释架构**：说明六个可执行程序、默认活跃路径、控制面和数据面分离。
3. **普通上传**：展示应用临时文件写入、服务端摘要、事务提交和第二次上传的去重结果；不要把 `FormFile` 路径描述成端到端请求流式。
4. **multipart**：在 Track B 通过后，上传部分 part，刷新页面或切换实例后恢复，再完成并校验 SHA-256。
5. **MySQL/Redis 细节**：展示事务回滚、Redis TTL、状态 CAS、generation/fencing 和 owner token 锁；说明租约不是唯一正确性边界。
6. **RabbitMQ 故障**：在 Track A 通过后停止 broker，展示 Outbox 积压；恢复后发布并完成任务。
7. **消费者故障**：在 Track A 通过后于对象校验或 ACK 窗口终止 worker，展示重复投递通过幂等逻辑收敛。
8. **多实例故障**：在 Track B 通过后终止一个 upload 实例，由另一个实例继续 multipart。
9. **观测证据**：使用 job ID 串联日志、指标、DB、Redis、队列和对象状态。
10. **原生 gRPC**：在 Track C 通过后展示 `grpcurl` health、proto 版本、双实例 etcd 发现、deadline、mTLS 和旧路径回滚。
11. **用户侧 Agent**：在 Track D 通过后，以普通用户身份自然语言搜索文件，展示授权结果、引用片段、多文件比较、结构化提取和标签建议。
12. **审批式整理**：用户确认重命名或接受标签，展示 proposal 指纹、expected version、CAS 执行、幂等重放和审计记录。
13. **可选运维扩展**：仅在 Track E 通过后展示脱敏 job 诊断；它不是默认用户 Agent。
14. **主动说明边界**：明确 at-least-once 不是 exactly-once，MySQL 与对象存储不是跨系统事务；Agent 不是任意运维控制面。

## 18. 主要风险与决策

| 风险或选择 | 决策 |
|---|---|
| 课程项目来源可能削弱个人贡献 | 主动披露来源，以可审查提交、ADR、测试和故障实验证明个人增量 |
| 当前工作树无法复现 | 先整理版本控制和干净环境，再继续扩展功能 |
| Redis 状态与本地 part 形成双权威 | P1 将 part 权威迁移到 MinIO/S3，Redis 仅协调短期状态 |
| MySQL 与对象存储无法原子提交 | 使用状态机、Outbox、幂等和清理，不伪造跨系统 ACID |
| RabbitMQ 会重复投递 | 接受 at-least-once，以 job ID 和 DB CAS 收敛 |
| 服务端最终 SHA-256 增加一次对象读取 | 作为可信完整性和简历证据的成本；后续可使用 provider checksum 优化 |
| 后端代理 part 增加带宽压力 | 主线优先展示后端流式与控制能力；后续可扩展预签名直传模式 |
| gRPC 与 Go Micro 并存增加运维复杂度 | 先双栈灰度，按服务切换并保留回滚；只有 Track C 通过后再移除旧调用 |
| 用户文件内容含恶意 Prompt Injection | 文件内容永远按不可信证据处理；只允许后端 allowlist typed tools，输出做引用、schema、DLP 和权限复检 |
| 用户撤权与 Agent 检索并发 | 每次 retrieval、chunk read 和结果发布前复检 `user_file`/`acl_version`；变化时丢弃答案 |
| Agent 误诊或越权造成副作用 | 用户侧默认只读；整理动作 typed、可信 UI 审批、CAS、幂等、post-condition 和审计 |
| 运维 Agent 越权或误诊 | 仅可选 Track E；与用户 Agent 隔离身份、索引、工具和权限，第一版只读 |
| Agent/SRE 内容挤占开发经历 | 将故障注入、指标和验证用于证明开发机制，而非单列空泛运维描述 |

## 19. 最终完成定义

项目只有同时满足以下条件，才适合在简历中作为重点后端项目：

1. 干净克隆可启动，个人增量可追溯。
2. 简历中与仓库不一致的 GORM、Go Micro/gRPC、主从读写分离等表述已经按实际证据纠正。
3. P0 正确性需求有机器可读的 `PASS` 报告。
4. Track A 或 Track B 至少一条完整中间件主线通过；另一条可以作为 roadmap，不得混写成已完成。
5. 若声明 Track B，则 multipart 必须通过刷新恢复、跨实例上传和实例退出测试；若只完成 Track A，不得把多实例 multipart 写入简历。
6. 若声明原生 gRPC，Track C 的 `FR-GRPC-*` 和 `AT-GRPC-*` 必须全部 `PASS`。
7. 若声明用户侧 Agent，Track D 的 `FR-AGENT-P0-*` 与对应 `AT-AGENT-*` 必须全部 `PASS`；若声明 Agentic tool loop，还必须通过 `AT-AGENT-TOOLLOOP-01`；若把 Go/gRPC/RabbitMQ/Redis/对象存储统一归因于 Agent 链路，还必须通过 `AT-AGENT-INTEGRATION-01`。
8. Track E 只作为可选内部上传诊断扩展，不能替代 Track D 的用户产品 Agent 证据。
9. 所有完成时和性能数字均可映射到 requirement ID、`AT-*`、commit 和工件路径。
10. 面试时能够解释成功路径、失败路径、数据权威、事务边界、用户文件权限、Agent 安全边界和未解决限制。
