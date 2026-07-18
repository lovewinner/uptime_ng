# uptime_ng 开发进度与 TODO

更新时间：2026-07-18

本文档根据 `README.md`、`docs/API.md` 和当前代码实现整理，用于跟踪后续开发优先级。

## 当前状态

### 已完成 / 基本可用

- 后端基础：Go 1.24 + Gin + GORM，PostgreSQL 数据模型，启动时执行 SQL 迁移并记录 `schema_migrations`，随后执行 AutoMigrate 补齐模型字段。
- 认证与用户：注册、登录、JWT、管理员用户管理、多用户数据隔离、禁止停用自己或移除最后一个 active 管理员、管理员重置密码。
- 监控 CRUD：HTTP/TCP/Ping/DNS/Group 监控项创建、更新、删除、暂停、恢复。
- 监控分组：支持 `type=group`、`group_id` 多层父分组、分组循环校验、当前状态接口可递归汇总、导入导出 `group_path` 恢复层级。
- 运行时调度：服务启动加载 active 监控项，监控项变更后同步启动/停止/重启调度器；Push 类型不会执行 checker；Group 按自身检查间隔递归汇总子项状态并写入心跳。
- HTTP 检查：状态码范围、关键词、反向关键词、Basic/Bearer/NTLM/OAuth2 client credentials/mTLS、Header、Body、忽略 TLS、最大重定向、cache bust、响应保存与截断。
- TCP/DNS/Ping 检查：TCP 连通性、DNS A/AAAA/CNAME/MX/TXT/NS 与自定义 DNS server、Ping 丢包率与单次 timeout。
- 心跳与 WebSocket：心跳写入、按用户推送实时状态，浏览器通过 `?token=` 认证连接。
- 通知：飞书卡片、邮件 HTML 告警、通知测试接口、每个监控/分组绑定多个通知源、重复 DOWN 告警按秒级 `resend_interval` 收敛，维护窗口期间抑制告警。
- SLA/统计：分钟/小时/天聚合、趋势数据、故障事件创建与恢复、单项/整体 SLA 按 heartbeat 时间线计算，整体 SLA 会保存 `SLAReport`。
- 导入导出：JSON 导出、冲突预览、跳过/覆盖/复制策略、通知与标签关联、敏感配置脱敏、分组路径导入恢复。
- 前端页面：登录/注册、仪表盘、树形监控列表、监控/分组详情、通知管理、SLA、维护窗口、用户管理。
- 测试基础：Go 测试覆盖调度器、checker、handler、导入导出、SLA、WebSocket、分组校验与状态聚合、维护窗口、API 文档路径；前端已接入 Vitest 并覆盖 monitor store helper 与创建监控弹窗。

## P0：上线前必须处理

- [x] 配置安全检查
  默认 `jwt.secret=change-me-in-production` 启动时会输出强警告。

- [x] 前端监控表单字段补全
  创建/编辑弹窗已暴露 HTTP header/body/auth、OAuth2、mTLS、重试、重复告警、DNS server/type、Ping count/timeout、父分组等配置。

- [x] 告警收敛语义校准
  `resend_interval` 已改为按秒计算重复 DOWN 告警间隔。

- [x] 数据库迁移策略
  已增加 `schema_migrations` 版本记录和启动时 SQL 迁移执行；AutoMigrate 保留用于模型补齐。

- [x] README 与当前功能同步
  README 已补充分组、HTTP 高级配置、NTLM/OAuth2/mTLS、DNS 类型选择、维护窗口、按需状态 API 和 `group_path` 导入导出示例。

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

- [x] 监控分组
  - [x] 新增 `group` 类型和 `group_id` 父分组属性。
  - [x] 支持多层分组和循环校验。
  - [x] 当前状态接口支持按子项递归汇总，空分组为 PENDING。
  - [x] 导入导出使用 `group_path` 恢复层级。

- [x] 分组周期检查
  分组对象已使用自身“检查间隔”属性独立调度；每到间隔由后台递归检查子对象状态，只有全部子对象为 UP 时父分组才为 UP。分组检查结果会持久化为 heartbeat、更新统计桶、推送 WebSocket，并按状态变更触发通知；空分组为 PENDING。

- [x] 按需状态计算 API
  已新增 `GET /api/monitors/:id/status`，后台按请求对象计算单个监控或分组状态；请求分组时递归汇总子对象状态。

- [x] 分组 SLA 聚合
  分组周期检查会生成分组 heartbeat 和统计桶，单项/整体 SLA 已支持 group 类型，不再返回 `group_sla_unsupported`。

- [x] 维护窗口
  已新增维护窗口模型、迁移、CRUD API、前端页面和调度器抑制逻辑；窗口期写入 `StatusMaintenance` 心跳并抑制通知。

## P2：前端体验与类型整理

- [x] 移除剩余主要 `any`
  主要 Vue 页面、导入弹窗、store catch 分支已改为明确类型或 `unknown`。

- [x] 监控详情实时刷新
  WebSocket client 已支持多订阅者，普通监控详情页会追加当前监控的 heartbeat。

- [x] 导入导出交互完善
  - [x] 导入预览展示通知配置数量。
  - [x] 覆盖策略增加风险提示。
  - [x] 脱敏通知配置会提示用户补齐密钥。

- [x] 仪表盘完善
  - [x] 监控总数已使用实时状态数据。
  - [x] 已增加 UP/DOWN/PENDING 汇总和平均响应时间。
  - [x] 已增加当前故障列表。

- [x] 监控列表分组展示
  已改为树形折叠列表，分组可嵌套。

- [x] 移除“未分组”虚拟节点
  未分组的监控对象已直接显示在监控列表根层级，不再划归到“未分组”节点下。

- [x] 用户管理完善
  - [x] 防止管理员停用自己或移除最后一个 active 管理员。
  - [x] 管理员可重置用户密码。

- [x] 仪表盘口径区分分组与真实监控
  仪表盘已区分真实监控和分组，UP/DOWN/PENDING、当前故障和平均响应时间只统计非 group 监控，同时单独显示分组数量。

- [x] 展开态驱动的状态刷新
  监控列表已按可见行和各自“检查间隔”调用 `GET /api/monitors/:id/status`；折叠后不可见的子对象会清理刷新定时器，不再发出状态请求。

- [x] 分组批量操作
  暂停/恢复分组已递归联动所有子分组和子监控，并同步停止或重启调度器。

## P3：工程质量

- [x] 增加集成测试
  - [x] handler 测试覆盖认证保护、通知测试、SLA、WebSocket token。
  - [x] checker 测试使用本地 HTTP/TCP fixture，并覆盖 DNS 错误路径。
  - [x] 导入导出增加跨用户隔离测试。
  - [x] 分组测试覆盖创建校验、调度跳过、状态递归聚合、导入导出层级恢复。

- [x] 前端测试
  已接入 Vitest，并增加 monitor store 基础单测和分组树 helper 测试。

- [x] 错误与日志规范
  API 错误响应保留 `error` 文案并新增稳定 `code` 字段；启动配置日志避免输出敏感值。

- [x] API 文档同步
  已新增 `docs/API.md` 并从 README 链接；分组 API 行为已记录。

- [x] 前端组件测试扩展
  已增加创建监控弹窗组件测试，覆盖 group 类型的检查间隔和通知配置展示；monitor store 测试同步覆盖无“未分组”虚拟节点的树形结构。

- [x] API 文档自动校验
  已增加 router 级测试，校验 `docs/API.md` 覆盖关键 API 路径，并验证 `/api/monitors/status` 与 `/api/monitors/:id/status` 的真实路由行为。

## 已知验证方式

后端：

```bash
CGO_ENABLED=1 go test ./...
```

前端：

```bash
cd web
npm ci
npm run test:unit
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

- 公共状态页、告警升级策略等高级功能暂缓。
