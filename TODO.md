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

- [x] 配置安全检查
  默认 `jwt.secret=change-me-in-production` 启动时会输出强警告。

- [x] 前端监控表单字段补全
  创建/编辑弹窗已暴露 HTTP header/body/auth、OAuth2、mTLS、重试、重复告警、DNS server/type、Ping count/timeout 等配置。

- [x] 告警收敛语义校准
  `resend_interval` 已改为按秒计算重复 DOWN 告警间隔。

## P1：核心功能完善

- [x] HTTP 高级选项实现完整性
  - [x] `max_redirects` 已按监控项配置生效。
  - [x] `cache_bust`、`save_response`、`save_error_response`、`response_max_length` 已接入 HTTP checker。
  - [x] `retry_only_on_status_code` 已接入重试逻辑。
  - [x] mTLS、OAuth2 client credentials 已实现。
  - [x] NTLM 已通过 `github.com/Azure/go-ntlmssp` 接入。

- [x] DNS 检查增强
  - [x] `dns_resolve_type`、`dns_resolve_server`、`dns_last_result` 已使用。
  - [x] 支持 A/AAAA/CNAME/MX/TXT/NS 类型选择。

- [x] Ping 检查增强
  - [x] 心跳消息已记录丢包率。
  - [x] 已接入单次 timeout 和整体 timeout。

- [x] SLA 准确性增强
  - [x] 单项/整体 SLA 已按 heartbeat 时间线计算真实停机时长。
  - [x] period 已改为自然日/周/月/季度/年。
  - [x] 整体 SLA 查询会保存 `SLAReport`。

- [x] 通知能力增强
  - [x] 邮件发送支持 465 TLS、STARTTLS、多收件人和 CC。
  - [x] 飞书 webhook 返回非 JSON 时会返回明确错误。
  - [x] 邮件通知支持 `subject_template` / `body_template`。

## P2：前端体验与类型整理

- [x] 移除剩余主要 `any`
  主要 Vue 页面、导入弹窗、store catch 分支已改为明确类型或 `unknown`。

- [x] 监控详情实时刷新
  WebSocket client 已支持多订阅者，监控详情页会追加当前监控的 heartbeat。

- [x] 导入导出交互完善
  - [x] 导入预览展示通知配置数量。
  - [x] 覆盖策略增加风险提示。
  - [x] 脱敏通知配置会提示用户补齐密钥。

- [x] 仪表盘完善
  - [x] 监控总数已使用实时状态数据。
  - [x] 已增加 UP/DOWN/PENDING 汇总和平均响应时间。
  - [x] 已增加当前故障列表。

- [x] 用户管理完善
  - [x] 防止管理员停用自己或移除最后一个 active 管理员。
  - [x] 管理员可重置用户密码。

## P3：工程质量

- [x] 增加集成测试
  - [x] handler 测试覆盖认证保护、通知测试、SLA、WebSocket token。
  - [x] checker 测试使用本地 HTTP/TCP fixture，并覆盖 DNS 错误路径。
  - [x] 导入导出增加跨用户隔离测试。

- [x] 前端测试
  已接入 Vitest，并增加 monitor store 基础单测。

- [x] 数据库迁移策略
  已增加 `schema_migrations` 版本记录和启动时 SQL 迁移执行；AutoMigrate 保留用于模型补齐。

- [x] 错误与日志规范
  API 错误响应保留 `error` 文案并新增稳定 `code` 字段；启动配置日志避免输出敏感值。

- [x] API 文档同步
  已新增 `docs/API.md` 并从 README 链接。

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
配置可用 PostgreSQL 连接后：
go run ./cmd/server/
cd web && npm run dev
```

## 暂缓事项

- 新增监控类型暂缓，优先补齐现有 HTTP/TCP/Ping/DNS 能力。
- 公共状态页、维护窗口、告警升级策略等高级功能暂缓。
