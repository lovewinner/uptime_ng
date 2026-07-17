export interface Tag {
  id: number
  name: string
  color: string
}

export interface Monitor {
  id: number
  user_id: number
  name: string
  description: string
  type: 'http' | 'tcp' | 'ping' | 'dns' | 'push'
  active: boolean
  url: string
  hostname: string
  port: number
  method: string
  interval: number
  timeout: number
  max_retries: number
  retry_interval: number
  resend_interval: number
  headers: string
  body: string
  keyword: string
  invert_keyword: boolean
  ignore_tls: boolean
  upside_down: boolean
  max_redirects: number
  auth_method: string
  basic_auth_user: string
  auth_workstation: string
  auth_domain: string
  bearer_token?: string
  tls_key?: string
  tls_cert?: string
  tls_ca: string
  oauth_client_id?: string
  oauth_token_url: string
  oauth_scopes: string
  oauth_auth_method: string
  oauth_audience: string
  dns_resolve_type: string
  dns_resolve_server: string
  http_body_encoding: string
  retry_only_on_status_code: boolean
  cache_bust: boolean
  save_response: boolean
  save_error_response: boolean
  response_max_length: number
  ping_count: number
  ping_per_request_timeout: number
  accepted_status_codes: string[] | string
  tags?: Tag[]
  notification_ids?: number[]
}

export interface MonitorListItem {
  monitor: Omit<Monitor, 'accepted_status_codes'> & { accepted_status_codes: string }
  tags: Tag[]
  notification_ids: number[]
}

export interface MonitorPayload {
  name: string
  description: string
  type: string
  url: string
  hostname: string
  port: number
  method: string
  interval: number
  timeout: number
  max_retries: number
  retry_interval: number
  resend_interval: number
  headers: string
  body: string
  keyword: string
  invert_keyword: boolean
  ignore_tls: boolean
  upside_down: boolean
  max_redirects: number
  auth_method: string
  basic_auth_user: string
  basic_auth_pass: string
  auth_workstation: string
  auth_domain: string
  bearer_token: string
  tls_key: string
  tls_cert: string
  tls_ca: string
  oauth_client_id: string
  oauth_client_secret: string
  oauth_token_url: string
  oauth_scopes: string
  oauth_auth_method: string
  oauth_audience: string
  dns_resolve_type: string
  dns_resolve_server: string
  http_body_encoding: string
  retry_only_on_status_code: boolean
  cache_bust: boolean
  save_response: boolean
  save_error_response: boolean
  response_max_length: number
  ping_count: number
  ping_per_request_timeout: number
  accepted_status_codes: string[]
  notification_ids: number[]
}

export interface Heartbeat {
  id: number
  monitor_id: number
  status: number
  msg: string
  ping_ms: number | null
  http_status: number
  important: boolean
  time: string
}

export interface Incident {
  id: number
  monitor_id: number
  title: string
  status: number
  started_at: string
  ended_at: string | null
  duration_seconds: number
  msg: string
}

export interface UptimeDataPoint {
  timestamp: number
  uptime: number
  avg_ping: number
  min_ping: number
  max_ping: number
  up: number
  down: number
}

export interface UptimeSummary {
  uptime_24h: number
  uptime_30d: number
  uptime_1y: number
}

export interface WSMessage<T = unknown> {
  type: string
  payload: T
}

export interface Notification {
  id: number
  name: string
  type: 'feishu' | 'email'
  config: string
  active: boolean
}
