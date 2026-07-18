import type { Monitor } from '@/stores/monitor'
import type { MonitorPayload } from '@/api/types'

export const DEFAULT_STATUS_CODES = ['200-299']

export const monitorTypeOptions = [
  { label: 'HTTP', value: 'http' },
  { label: 'TCP', value: 'tcp' },
  { label: 'PING', value: 'ping' },
  { label: 'DNS', value: 'dns' },
  { label: 'GROUP', value: 'group' },
]

export const httpMethodOptions = [
  { label: 'GET', value: 'GET' },
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'DELETE', value: 'DELETE' },
  { label: 'PATCH', value: 'PATCH' },
  { label: 'HEAD', value: 'HEAD' },
  { label: 'OPTIONS', value: 'OPTIONS' },
]

export const authMethodOptions = [
  { label: '无', value: '' },
  { label: 'Basic', value: 'basic' },
  { label: 'Bearer Token', value: 'bearer' },
  { label: 'OAuth2 Client Credentials', value: 'oauth2-cc' },
  { label: 'NTLM', value: 'ntlm' },
  { label: 'mTLS', value: 'mtls' },
]

export const bodyEncodingOptions = [
  { label: 'JSON', value: 'json' },
  { label: 'Form', value: 'form' },
  { label: 'XML', value: 'xml' },
  { label: 'Raw', value: 'raw' },
]

export const dnsTypeOptions = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS'].map((value) => ({ label: value, value }))

export function defaultMonitorPayload(): MonitorPayload {
  return {
    name: '',
    description: '',
    type: 'ping',
    group_id: null,
    url: '',
    hostname: '',
    port: 80,
    method: 'GET',
    interval: 60,
    timeout: 30,
    max_retries: 0,
    retry_interval: 60,
    resend_interval: 0,
    headers: '',
    body: '',
    keyword: '',
    invert_keyword: false,
    ignore_tls: false,
    upside_down: false,
    max_redirects: 10,
    auth_method: '',
    basic_auth_user: '',
    basic_auth_pass: '',
    auth_workstation: '',
    auth_domain: '',
    bearer_token: '',
    tls_key: '',
    tls_cert: '',
    tls_ca: '',
    oauth_client_id: '',
    oauth_client_secret: '',
    oauth_token_url: '',
    oauth_scopes: '',
    oauth_auth_method: 'body',
    oauth_audience: '',
    dns_resolve_type: 'A',
    dns_resolve_server: '',
    http_body_encoding: 'json',
    retry_only_on_status_code: false,
    cache_bust: false,
    save_response: false,
    save_error_response: true,
    response_max_length: 4096,
    ping_count: 4,
    ping_per_request_timeout: 1000,
    accepted_status_codes: [...DEFAULT_STATUS_CODES],
    notification_ids: [],
  }
}

export function monitorPayloadFromMonitor(monitor: Monitor): MonitorPayload {
  return {
    ...defaultMonitorPayload(),
    name: monitor.name || '',
    description: monitor.description || '',
    type: monitor.type || 'http',
    group_id: monitor.group_id ?? null,
    url: monitor.url || '',
    hostname: monitor.hostname || '',
    port: monitor.port || 80,
    method: monitor.method || 'GET',
    interval: monitor.interval || 60,
    timeout: monitor.timeout || 30,
    max_retries: monitor.max_retries || 0,
    retry_interval: monitor.retry_interval || 60,
    resend_interval: monitor.resend_interval || 0,
    headers: monitor.headers || '',
    body: monitor.body || '',
    keyword: monitor.keyword || '',
    invert_keyword: monitor.invert_keyword || false,
    ignore_tls: monitor.ignore_tls || false,
    upside_down: monitor.upside_down || false,
    max_redirects: monitor.max_redirects || 10,
    auth_method: monitor.auth_method || '',
    basic_auth_user: monitor.basic_auth_user || '',
    basic_auth_pass: '',
    auth_workstation: monitor.auth_workstation || '',
    auth_domain: monitor.auth_domain || '',
    bearer_token: '',
    tls_key: '',
    tls_cert: '',
    tls_ca: monitor.tls_ca || '',
    oauth_client_id: '',
    oauth_client_secret: '',
    oauth_token_url: monitor.oauth_token_url || '',
    oauth_scopes: monitor.oauth_scopes || '',
    oauth_auth_method: monitor.oauth_auth_method || 'body',
    oauth_audience: monitor.oauth_audience || '',
    dns_resolve_type: monitor.dns_resolve_type || 'A',
    dns_resolve_server: monitor.dns_resolve_server || '',
    http_body_encoding: monitor.http_body_encoding || 'json',
    retry_only_on_status_code: monitor.retry_only_on_status_code || false,
    cache_bust: monitor.cache_bust || false,
    save_response: monitor.save_response || false,
    save_error_response: monitor.save_error_response || false,
    response_max_length: monitor.response_max_length || 4096,
    ping_count: monitor.ping_count || 4,
    ping_per_request_timeout: monitor.ping_per_request_timeout || 1000,
    accepted_status_codes: statusCodesFromMonitor(monitor),
    notification_ids: [...(monitor.notification_ids || [])],
  }
}

export function statusCodesFromMonitor(monitor?: Pick<Monitor, 'accepted_status_codes'> | null): string[] {
  if (!monitor?.accepted_status_codes?.length) {
    return [...DEFAULT_STATUS_CODES]
  }
  return [...monitor.accepted_status_codes]
}

export function addStatusCodeTag(tags: string[], value: string): string[] {
  const trimmed = value.trim()
  if (!trimmed || tags.includes(trimmed)) {
    return tags
  }
  return [...tags, trimmed]
}

export function removeStatusCodeTag(tags: string[], tag: string): string[] {
  return tags.filter((item) => item !== tag)
}

export function monitorDialogTitle(isEdit: boolean): string {
  return isEdit ? '编辑监控' : '新增监控'
}

export function monitorSubmitText(isEdit: boolean): string {
  return isEdit ? '保存' : '创建'
}

export function shouldFillPingHostname(type: string, hostname: string): boolean {
  return type === 'ping' && !hostname
}

export function monitorSubmitPayload(form: MonitorPayload): MonitorPayload {
  return {
    ...form,
    port: form.port || 0,
  }
}
