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
  keyword: string
  invert_keyword: boolean
  ignore_tls: boolean
  upside_down: boolean
  max_redirects: number
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
  keyword: string
  ignore_tls: boolean
  upside_down: boolean
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
