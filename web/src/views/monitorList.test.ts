import { describe, expect, it } from 'vitest'
import {
  exportURL,
  intervalText,
  monitorStatusValue,
  monitorTargetText,
  nextExpandedIds,
  pauseResumeButtonType,
  pauseResumeSuccessText,
  pauseResumeText,
  refreshIntervalSeconds,
  visibleMonitorRows,
} from './monitorList'
import { monitorTypeText } from './formatters'
import type { MonitorTreeNode } from '@/stores/monitor'

function node(overrides: Partial<MonitorTreeNode>): MonitorTreeNode {
  return {
    id: 1,
    user_id: 1,
    name: 'node',
    description: '',
    type: 'http',
    group_id: null,
    active: true,
    url: '',
    hostname: '',
    port: 0,
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
    auth_workstation: '',
    auth_domain: '',
    tls_ca: '',
    oauth_token_url: '',
    oauth_scopes: '',
    oauth_auth_method: '',
    oauth_audience: '',
    dns_resolve_type: '',
    dns_resolve_server: '',
    http_body_encoding: '',
    retry_only_on_status_code: false,
    cache_bust: false,
    save_response: false,
    save_error_response: false,
    response_max_length: 4096,
    ping_count: 4,
    ping_per_request_timeout: 1000,
    accepted_status_codes: ['200-299'],
    notification_ids: [],
    tags: [],
    ...overrides,
  }
}

describe('monitorList helpers', () => {
  it('flattens visible tree rows by expanded ids', () => {
    const child = node({ id: 2, name: 'child', group_id: 1 })
    const root = node({ id: 1, name: 'root', type: 'group', children: [child] })

    expect(visibleMonitorRows([root], new Set()).map((row) => row.id)).toEqual([1])
    expect(visibleMonitorRows([root], new Set([1])).map((row) => row.id)).toEqual([1, 2])
  })

  it('collapses descendants when a parent collapses', () => {
    const grandchild = node({ id: 3, group_id: 2 })
    const child = node({ id: 2, group_id: 1, children: [grandchild] })
    const root = node({ id: 1, type: 'group', children: [child] })

    expect([...nextExpandedIds(root, false, new Set([1, 2, 3]))]).toEqual([])
    expect([...nextExpandedIds(root, true, new Set())]).toEqual([1])
  })

  it('formats target and interval text', () => {
    expect(monitorTargetText(node({ type: 'group', children: [node({ id: 2 })] }))).toBe('1 个子项')
    expect(monitorTargetText(node({ type: 'http', url: 'https://example.com' }))).toBe('https://example.com')
    expect(monitorTargetText(node({ type: 'tcp', hostname: 'localhost', port: 5432 }))).toBe('localhost:5432')
    expect(monitorTargetText(node({ type: 'ping', hostname: '' }))).toBe('-')
    expect(intervalText(30)).toBe('30s')
    expect(intervalText(120)).toBe('2m')
  })

  it('formats list action and status presentation values', () => {
    expect(monitorTypeText('http')).toBe('HTTP')
    expect(pauseResumeButtonType(true)).toBe('warning')
    expect(pauseResumeButtonType(false)).toBe('success')
    expect(pauseResumeText(true)).toBe('暂停')
    expect(pauseResumeText(false)).toBe('恢复')
    expect(pauseResumeSuccessText(true)).toBe('已暂停')
    expect(pauseResumeSuccessText(false)).toBe('已恢复')
    expect(monitorStatusValue(undefined)).toBe(2)
    expect(monitorStatusValue(0)).toBe(0)
  })

  it('normalizes row refresh intervals', () => {
    expect(refreshIntervalSeconds(0)).toBe(60)
    expect(refreshIntervalSeconds(1)).toBe(3)
    expect(refreshIntervalSeconds(30)).toBe(30)
  })

  it('builds export urls', () => {
    expect(exportURL()).toBe('/monitors/export')
    expect(exportURL([])).toBe('/monitors/export')
    expect(exportURL([1, 2])).toBe('/monitors/export?ids=[1,2]')
  })
})
