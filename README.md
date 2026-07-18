# uptime_ng

企业 IT 业务监控系统。基于 uptime-kuma 思路重新设计，专为公司内网部署优化的轻量级监控告警平台。

## 技术栈

- **后端**: Go 1.24 + Gin + GORM
- **前端**: Vue 3 + Vite + TypeScript + Element Plus + ECharts
- **数据库**: PostgreSQL 16
- **实时通信**: WebSocket (gorilla/websocket)
- **部署**: Docker Compose

## 功能特性

### 监控类型
- HTTP/HTTPS（状态码、关键词、TLS 证书忽略、重定向、响应保存、Basic/Bearer/NTLM/OAuth2/mTLS）
- TCP 端口连通性
- ICMP Ping（含丢包率）
- DNS 解析（A/AAAA/CNAME/MX/TXT/NS，可指定 DNS server）
- Group 分组（支持多层父子关系，按分组检查间隔汇总子项状态）

### 告警通知
- **飞书**：交互卡片消息（标题、状态、详情、跳转链接）
- **邮件**：SMTP 发送 HTML 邮件
- **告警收敛**：可配置 resend_interval，避免风暴
- **多通知源**：每个监控项可绑定多个通知配置
- **维护窗口**：按时间窗口抑制检查与通知，可作用于单个监控或全部监控

### SLA 报表
- 三级时间桶聚合：分钟级（24h）/小时级（30d）/天级（365d）
- 可用率：基于心跳数据精确计算 up/(up+down)
- 故障事件：自动追踪起止时间、持续时长
- 趋势图表：响应时间 + 可用率

### 监控对象导入导出
- 完整 JSON 格式导出（含 tags、通知关联）
- 分组通过 group_path 导出，导入时恢复层级
- 导入预览：自动检测冲突
- 三种冲突策略：跳过 / 覆盖 / 复制（添加后缀）
- 敏感信息脱敏（密码字段）

### 用户管理
- JWT 认证
- 角色：admin / user
- 多用户隔离，每个用户独立管理自己的监控项

## 监控间隔

- 默认：60 秒
- 最小：3 秒

## 快速开始

### Docker Compose（推荐）

```bash
docker compose up -d
```

访问 http://localhost:3000
首个注册用户自动成为管理员。

### 本地开发

```bash
# 1. 启动 PostgreSQL
docker compose up -d db

# 2. 启动后端
go run ./cmd/server/

# 3. 启动前端（另一终端）
cd web
npm install
npm run dev
```

前端开发服务器： http://localhost:5173 （自动代理 API 请求到后端 3000）

## API 概览

完整请求/响应字段见 [API Reference](docs/API.md)。

### 认证
- `POST /api/auth/register` - 注册
- `POST /api/auth/login` - 登录（返回 JWT）
- `GET  /api/auth/profile` - 当前用户
- `GET  /api/auth/users` - 用户列表（admin）
- `PATCH /api/auth/users/:id` - 更新用户角色/状态（admin）

### 监控项
- `GET    /api/monitors` - 监控项列表
- `POST   /api/monitors` - 创建
- `GET    /api/monitors/:id` - 详情
- `GET    /api/monitors/:id/status` - 单个监控/分组状态
- `PUT    /api/monitors/:id` - 更新
- `DELETE /api/monitors/:id` - 删除
- `POST   /api/monitors/:id/resume` - 恢复
- `POST   /api/monitors/:id/pause` - 暂停

### 心跳 / 故障
- `GET /api/monitors/status` - 仪表盘状态列表
- `GET /api/monitors/:id/beats?period=3600` - 心跳数据
- `GET /api/monitors/:id/beats/important?limit=50` - 重要心跳
- `GET /api/monitors/:id/incidents` - 故障事件

### SLA
- `GET /api/monitors/:id/uptime?period=month` - 单个监控可用率
- `GET /api/monitors/:id/uptime/data?granularity=daily&num=30` - 趋势数据
- `GET /api/monitors/uptime/overall?period=month` - 全部监控可用率

### 通知
- `GET    /api/notifications` - 列表
- `POST   /api/notifications` - 创建
- `PUT    /api/notifications/:id` - 更新
- `DELETE /api/notifications/:id` - 删除
- `POST   /api/notifications/:id/test` - 测试发送

### 维护窗口
- `GET    /api/maintenance` - 列表
- `POST   /api/maintenance` - 创建
- `PUT    /api/maintenance/:id` - 更新
- `DELETE /api/maintenance/:id` - 删除

### 导入导出
- `GET  /api/monitors/export?ids=[1,2,3]` - 导出 JSON
- `POST /api/monitors/import/preview` - 预览导入（检测冲突）
- `POST /api/monitors/import` - 执行导入

### WebSocket
- `ws://host/api/ws?token=<jwt>` - 实时推送心跳状态

## 配置

通过 `config.yaml` 或环境变量（`UPTIME_NG_*` 前缀）配置：

```yaml
server:
  host: 0.0.0.0
  port: 3000

database:
  host: localhost
  port: 5432
  user: uptime
  password: uptime123
  dbname: uptime_ng
  sslmode: disable

jwt:
  secret: change-me-in-production
  expirehours: 72

smtp:
  host: smtp.example.com
  port: 587
  username: ops@example.com
  password: ***
  from: noreply@example.com

feishu:
  webhook_url: https://open.feishu.cn/open-apis/bot/v2/hook/xxx
```

## 数据库迁移

服务启动时会先执行 `migrations/*.sql`，并在 `schema_migrations` 表中记录文件名、校验和与应用时间；如果已应用迁移文件内容发生变化，启动会失败以避免不可追踪的 schema 漂移。随后仍会执行 GORM `AutoMigrate`，用于补齐模型字段。

## 导入导出格式示例

```json
{
  "version": "1.0",
  "exported_at": "2026-07-17T07:30:00Z",
  "exported_by": "admin",
  "monitors": [
    {
      "name": "公司官网",
      "type": "http",
      "group_path": ["生产环境", "官网"],
      "url": "https://www.example.com",
      "interval": 60,
      "timeout": 30,
      "max_retries": 3,
      "accepted_status_codes": ["200-299"],
      "tags": [
        {"name": "生产环境", "color": "#FF0000"}
      ],
      "notification_names": ["飞书告警群"]
    }
  ],
  "notifications": [
    {
      "name": "飞书告警群",
      "type": "feishu",
      "config": "{\"webhook_url\":\"https://...\"}"
    }
  ]
}
```

## 项目结构

```
cmd/server/              # 入口
internal/
  config/                # 配置管理
  model/                 # 数据模型（GORM）
  handler/               # HTTP/WS 处理器
  engine/                # 监控引擎（checker + scheduler + uptime calculator）
  notifier/              # 告警通知（飞书/邮件）
  middleware/            # 中间件
  router/                # 路由注册
web/src/                 # Vue 3 前端
  views/                 # 页面
  components/            # 通用组件
  stores/                # Pinia 状态
  api/                   # HTTP/WS 客户端
migrations/              # 数据库迁移
docker/                  # Dockerfile
```

## License

MIT
