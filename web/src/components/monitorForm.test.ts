import { describe, expect, it } from 'vitest'
import {
  addStatusCodeTag,
  authMethodOptions,
  bodyEncodingOptions,
  defaultMonitorPayload,
  dnsTypeOptions,
  httpMethodOptions,
  monitorDialogTitle,
  monitorPayloadFromMonitor,
  monitorSubmitPayload,
  monitorSubmitText,
  monitorTypeOptions,
  removeStatusCodeTag,
  shouldFillPingHostname,
  statusCodesFromMonitor,
} from './monitorForm'
import type { Monitor } from '@/stores/monitor'

function monitor(overrides: Partial<Monitor> = {}): Monitor {
  return {
    id: 1,
    user_id: 1,
    name: 'site',
    description: '',
    type: 'http',
    group_id: null,
    active: true,
    url: 'https://example.com',
    hostname: '',
    port: 443,
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
    auth_method: 'bearer',
    basic_auth_user: 'user',
    auth_workstation: 'ws',
    auth_domain: 'domain',
    bearer_token: 'server-secret',
    tls_key: 'server-key',
    tls_cert: 'server-cert',
    tls_ca: 'ca',
    oauth_client_id: 'server-client',
    oauth_token_url: 'https://issuer/token',
    oauth_scopes: 'scope',
    oauth_auth_method: 'basic',
    oauth_audience: 'audience',
    dns_resolve_type: 'A',
    dns_resolve_server: '',
    http_body_encoding: 'json',
    retry_only_on_status_code: false,
    cache_bust: false,
    save_response: false,
    save_error_response: false,
    response_max_length: 4096,
    ping_count: 4,
    ping_per_request_timeout: 1000,
    accepted_status_codes: ['200-299'],
    notification_ids: [2],
    tags: [],
    ...overrides,
  }
}

describe('monitorForm helpers', () => {
  it('builds independent default payloads', () => {
    const first = defaultMonitorPayload()
    const second = defaultMonitorPayload()
    first.accepted_status_codes.push('500')
    first.notification_ids.push(1)

    expect(second.accepted_status_codes).toEqual(['200-299'])
    expect(second.notification_ids).toEqual([])
  })

  it('maps existing monitor while clearing secret-only fields', () => {
    const payload = monitorPayloadFromMonitor(monitor())

    expect(payload.name).toBe('site')
    expect(payload.port).toBe(443)
    expect(payload.notification_ids).toEqual([2])
    expect(payload.bearer_token).toBe('')
    expect(payload.tls_key).toBe('')
    expect(payload.tls_cert).toBe('')
    expect(payload.oauth_client_id).toBe('')
    expect(payload.oauth_client_secret).toBe('')
  })

  it('handles status code tag updates immutably', () => {
    const tags = ['200-299']
    expect(addStatusCodeTag(tags, ' 500 ')).toEqual(['200-299', '500'])
    expect(addStatusCodeTag(tags, '200-299')).toBe(tags)
    expect(removeStatusCodeTag(['200-299', '500'], '500')).toEqual(['200-299'])
    expect(statusCodesFromMonitor(null)).toEqual(['200-299'])
  })

  it('exposes stable form option lists', () => {
    expect(monitorTypeOptions.map((option) => option.value)).toEqual(['http', 'tcp', 'ping', 'dns', 'group'])
    expect(httpMethodOptions.map((option) => option.value)).toContain('PATCH')
    expect(authMethodOptions.map((option) => option.value)).toContain('oauth2-cc')
    expect(bodyEncodingOptions.map((option) => option.value)).toEqual(['json', 'form', 'xml', 'raw'])
    expect(dnsTypeOptions.map((option) => option.value)).toEqual(['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS'])
  })

  it('derives dialog labels and form update rules', () => {
    expect(monitorDialogTitle(true)).toBe('编辑监控')
    expect(monitorDialogTitle(false)).toBe('新增监控')
    expect(monitorSubmitText(true)).toBe('保存')
    expect(monitorSubmitText(false)).toBe('创建')
    expect(shouldFillPingHostname('ping', '')).toBe(true)
    expect(shouldFillPingHostname('ping', 'example.com')).toBe(false)
    expect(shouldFillPingHostname('http', '')).toBe(false)
  })

  it('normalizes submit payload transport fields', () => {
    expect(monitorSubmitPayload({ ...defaultMonitorPayload(), port: 0 }).port).toBe(0)
    expect(monitorSubmitPayload({ ...defaultMonitorPayload(), port: 443 }).port).toBe(443)
  })
})
