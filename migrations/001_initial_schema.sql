-- uptime_ng 初始数据库 schema
-- 这个文件是参考用的完整 DDL，实际由 GORM AutoMigrate 自动创建

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    active BOOLEAN DEFAULT true,
    timezone VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 监控项表
CREATE TABLE IF NOT EXISTS monitors (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    description VARCHAR(500),
    type VARCHAR(20) NOT NULL CHECK (type IN ('http', 'tcp', 'ping', 'dns', 'push')),
    active BOOLEAN DEFAULT true,
    url VARCHAR(2000),
    hostname VARCHAR(255),
    port SMALLINT DEFAULT 0,
    method VARCHAR(10) DEFAULT 'GET',
    interval INTEGER DEFAULT 60,
    timeout DOUBLE PRECISION DEFAULT 30,
    max_retries INTEGER DEFAULT 0,
    retry_interval INTEGER DEFAULT 0,
    resend_interval INTEGER DEFAULT 0,
    headers TEXT,
    body TEXT,
    accepted_status_codes TEXT DEFAULT '["200-299"]',
    keyword VARCHAR(255),
    invert_keyword BOOLEAN DEFAULT false,
    ignore_tls BOOLEAN DEFAULT false,
    upside_down BOOLEAN DEFAULT false,
    max_redirects INTEGER DEFAULT 10,
    auth_method VARCHAR(20),
    basic_auth_user VARCHAR(255),
    basic_auth_pass VARCHAR(255),
    bearer_token VARCHAR(2000),
    auth_workstation VARCHAR(255),
    auth_domain VARCHAR(255),
    tls_key TEXT,
    tls_cert TEXT,
    tls_ca TEXT,
    oauth_client_id TEXT,
    oauth_client_secret TEXT,
    oauth_token_url TEXT,
    oauth_scopes TEXT,
    oauth_auth_method TEXT,
    oauth_audience TEXT,
    dns_resolve_type VARCHAR(5),
    dns_resolve_server VARCHAR(255),
    dns_last_result VARCHAR(255),
    push_token VARCHAR(32),
    packet_size INTEGER DEFAULT 56,
    expiry_notification BOOLEAN DEFAULT true,
    http_body_encoding VARCHAR(25) DEFAULT 'json',
    retry_only_on_status_code BOOLEAN DEFAULT false,
    cache_bust BOOLEAN DEFAULT false,
    save_response BOOLEAN DEFAULT false,
    save_error_response BOOLEAN DEFAULT false,
    response_max_length INTEGER DEFAULT 4096,
    ping_numeric BOOLEAN DEFAULT false,
    ping_count INTEGER DEFAULT 4,
    ping_per_request_timeout INTEGER DEFAULT 1000,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_monitors_user ON monitors(user_id);
CREATE INDEX IF NOT EXISTS idx_monitors_active ON monitors(active);

-- 心跳记录表
CREATE TABLE IF NOT EXISTS heartbeats (
    id SERIAL PRIMARY KEY,
    monitor_id INTEGER NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status SMALLINT NOT NULL,
    msg TEXT,
    ping_ms DOUBLE PRECISION,
    http_status SMALLINT DEFAULT 0,
    important BOOLEAN DEFAULT false,
    retries INTEGER DEFAULT 0,
    down_count INTEGER DEFAULT 0,
    time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    duration INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_hb_monitor_time ON heartbeats(monitor_id, time);
CREATE INDEX IF NOT EXISTS idx_hb_important ON heartbeats(important);

-- 可用率统计 - 分钟级 (保留24小时)
CREATE TABLE IF NOT EXISTS stat_minutely (
    id SERIAL PRIMARY KEY,
    monitor_id INTEGER NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    timestamp BIGINT NOT NULL,
    up INTEGER DEFAULT 0,
    down INTEGER DEFAULT 0,
    avg_ping DECIMAL(10,2) DEFAULT 0,
    min_ping DECIMAL(10,2) DEFAULT 0,
    max_ping DECIMAL(10,2) DEFAULT 0,
    UNIQUE(monitor_id, timestamp)
);

CREATE INDEX IF NOT EXISTS idx_statm_monitor ON stat_minutely(monitor_id);

-- 可用率统计 - 小时级 (保留30天)
CREATE TABLE IF NOT EXISTS stat_hourly (
    id SERIAL PRIMARY KEY,
    monitor_id INTEGER NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    timestamp BIGINT NOT NULL,
    up INTEGER DEFAULT 0,
    down INTEGER DEFAULT 0,
    avg_ping DECIMAL(10,2) DEFAULT 0,
    min_ping DECIMAL(10,2) DEFAULT 0,
    max_ping DECIMAL(10,2) DEFAULT 0,
    UNIQUE(monitor_id, timestamp)
);

CREATE INDEX IF NOT EXISTS idx_stath_monitor ON stat_hourly(monitor_id);

-- 可用率统计 - 天级 (保留365天)
CREATE TABLE IF NOT EXISTS stat_daily (
    id SERIAL PRIMARY KEY,
    monitor_id INTEGER NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    timestamp BIGINT NOT NULL,
    up INTEGER DEFAULT 0,
    down INTEGER DEFAULT 0,
    avg_ping DECIMAL(10,2) DEFAULT 0,
    min_ping DECIMAL(10,2) DEFAULT 0,
    max_ping DECIMAL(10,2) DEFAULT 0,
    UNIQUE(monitor_id, timestamp)
);

CREATE INDEX IF NOT EXISTS idx_statd_monitor ON stat_daily(monitor_id);

-- 通知配置表
CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('feishu', 'email')),
    config TEXT NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notif_user ON notifications(user_id);

-- 监控项-通知关联表
CREATE TABLE IF NOT EXISTS monitor_notifications (
    id SERIAL PRIMARY KEY,
    monitor_id INTEGER NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    notification_id INTEGER NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    UNIQUE(monitor_id, notification_id)
);

-- 标签表
CREATE TABLE IF NOT EXISTS tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    color VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 监控项-标签关联表
CREATE TABLE IF NOT EXISTS monitor_tags (
    id SERIAL PRIMARY KEY,
    monitor_id INTEGER NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    value VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_mt_monitor ON monitor_tags(monitor_id);
CREATE INDEX IF NOT EXISTS idx_mt_tag ON monitor_tags(tag_id);

-- 故障事件表
CREATE TABLE IF NOT EXISTS incidents (
    id SERIAL PRIMARY KEY,
    monitor_id INTEGER NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    status SMALLINT DEFAULT 0,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    duration_seconds INTEGER DEFAULT 0,
    msg TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inc_monitor ON incidents(monitor_id);
CREATE INDEX IF NOT EXISTS idx_inc_started ON incidents(started_at);

-- SLA报告表
CREATE TABLE IF NOT EXISTS sla_reports (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period_type VARCHAR(20) NOT NULL,
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    data_json TEXT,
    generated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sla_user ON sla_reports(user_id);

-- 系统设置表
CREATE TABLE IF NOT EXISTS settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(200) NOT NULL UNIQUE,
    value TEXT,
    type VARCHAR(20)
);