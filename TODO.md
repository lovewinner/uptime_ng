# uptime_ng 开发进度与 TODO

更新时间：2026-07-18

本文档根据 `README.md` 中承诺的功能、当前代码实现和近期修复状态整理，用于跟踪后续开发优先级。

## 当前状态

### 已完成 / 基本可用

- 后端基础：Go + Gin + GORM，PostgreSQL 数据模型，启动时 AutoMigrate。
- 认证与用户：注册、登录、JWT、管理员用户管理、多用户数据隔离。
- 监控 CRUD：HTTP/TCP/Ping/DNS 监控项创建、更新、删除、暂停、恢复。
- 运行时调度：服务启动加载 active 监控项，监控项变更后同步启动/停止/重启调度器。
- HTTP 检查：状态码范围、关键词、反向关键词、Basic/Bearer 认证、Header、Body、忽略 TLS。
- TCP/DNS/Ping 检查：基础连通性检测可用。
- 心跳与 WebSocket：心跳写入、按用户推送实时状态，浏览器通过 `?token=` 认证连接。
- 通知：飞书卡片、邮件 HTML 告警、通知测试接口、每个监控绑定多个通知源。
- SLA/统计：分钟/小时/天聚合、趋势数据、故障事件创建与恢复、基础 SLA 列表。
- 导入导出：JSON 导出、冲突预览、跳过/覆盖/复制策略、通知与标签关联、敏感配置脱敏。
- 前端基础页面：登录/注册、仪表盘、监控列表、监控详情、通知管理、SLA、用户管理。
- 测试基础：新增 Go 单元测试覆盖调度器联动、导入导出、脱敏、状态码解析、统计清理。

## P0：上线前必须处理

- [ ] Docker 部署闭环  
  当前 Dockerfile 只构建后端二进制，没有构建/复制 Vue 前端 `dist`，但后端路由会从 `./dist` 服务静态文件。需要在功能稳定后统一设计镜像构建方案。

- [ ] Docker Ping 依赖  
  Ping 检查当前调用系统 `ping` 命令。Alpine 镜像需要确认安装 `iputils` 或替换为不依赖外部命令的实现。

- [ ] 配置安全检查  
  生产启动时应拒绝默认 `jwt.secret=change-me-in-production`，或至少输出强警告。

- [ ] 前端监控表单字段补全  
  后端支持 `headers`、`body`、`auth_method`、`basic_auth_*`、`bearer_token`、`retry_interval`、`resend_interval`、DNS server/type、Ping count/timeout 等字段，但创建/编辑弹窗只暴露了基础字段。

- [ ] 告警收敛语义校准  
  当前 `resend_interval` 基于连续 down 次数，不是 README 容易理解的时间间隔。需要明确改名、改 UI 文案，或改为按秒/分钟计时。

## P1：核心功能完善

- [ ] HTTP 高级选项实现完整性
  - [ ] `max_redirects` 目前 HTTP client 默认固定 10，需要按监控项配置生效。
  - [ ] `cache_bust`、`save_response`、`save_error_response`、`response_max_length` 模型字段存在但未完整实现。
  - [ ] `retry_only_on_status_code` 模型字段存在但未实现。
  - [ ] mTLS、OAuth2 client credentials、NTLM 相关模型字段存在，但 checker 未实现。

- [ ] DNS 检查增强
  - [ ] `dns_resolve_type`、`dns_resolve_server`、`dns_last_result` 尚未完整使用。
  - [ ] 支持 A/AAAA/CNAME/MX/TXT 等类型选择。

- [ ] Ping 检查增强
  - [ ] README 提到“含丢包率”，当前结果未记录/展示丢包率。
  - [ ] 跨平台解析 `ping` 输出需要更稳健，或改用 Go ICMP 库。

- [ ] SLA 准确性增强
  - [ ] 当前 SLA 主要基于检查次数 `up/(up+down)`，未按真实持续时间加权。
  - [ ] 当前 period 是最近 N 天，不是“本日/本周/本月”的自然周期，需要明确产品语义。
  - [ ] 增加 SLA 报表生成/保存能力，`SLAReport` 模型目前未形成完整业务闭环。

- [ ] 通知能力增强
  - [ ] 邮件发送未支持 TLS/STARTTLS 配置差异、抄送、多收件人。
  - [ ] 飞书 webhook 返回非标准 JSON 时当前可能假定成功，需要更明确的错误策略。
  - [ ] 通知模板自定义尚未实现。

## P2：前端体验与类型整理

- [ ] 移除剩余主要 `any`
  `MonitorDetailView.vue`、`ImportDialog.vue`、部分 catch 分支仍使用 `any`，需要继续抽类型。

- [ ] 监控详情实时刷新
  WebSocket 目前更新仪表盘状态；监控详情页的心跳列表和图表不会实时追加。

- [ ] 导入导出交互完善
  - [ ] 导入预览展示通知配置是否会被创建/跳过。
  - [ ] 覆盖策略增加更明确的风险提示。
  - [ ] 导出敏感字段脱敏后，导入时需要提示用户补齐密钥。

- [ ] 仪表盘完善
  - [ ] 监控总数卡片当前仍是静态“待加载”。
  - [ ] 增加 UP/DOWN/PENDING 汇总、最近故障、平均响应时间。

- [ ] 用户管理完善
  - [ ] 防止管理员停用自己或移除最后一个管理员。
  - [ ] 增加密码修改/重置流程。

## P3：工程质量

- [ ] 增加集成测试
  - [ ] handler 测试继续覆盖认证、通知测试、SLA、WebSocket token。
  - [ ] checker 测试使用本地 HTTP/TCP/DNS fixture。
  - [ ] 导入导出增加跨用户隔离测试。

- [ ] 前端测试
  当前没有前端单测或 E2E。建议先增加关键 store 和主要表单的测试。

- [ ] 数据库迁移策略
  当前以 GORM AutoMigrate 为主，`migrations/001_initial_schema.sql` 是参考文件。正式部署前应决定是否引入版本化迁移。

- [ ] 错误与日志规范
  统一错误码、错误信息、日志字段，避免将敏感 URL/token 写入日志。

- [ ] API 文档同步
  README 目前是概览，后续需要补齐请求/响应结构、字段含义、错误码。

## 已知验证方式

后端：

```bash
CGO_ENABLED=1 go test ./...
```

前端：

```bash
cd web
npm ci
npm run type-check
npm run build-only
```

本地开发：

```bash
docker compose up -d db
go run ./cmd/server/
cd web && npm run dev
```

## 暂缓事项

- Docker 镜像构建和 Compose 拆分暂缓，等代码功能完善后统一处理。
- 新增监控类型暂缓，优先补齐现有 HTTP/TCP/Ping/DNS 能力。
- 公共状态页、维护窗口、告警升级策略等高级功能暂缓。
