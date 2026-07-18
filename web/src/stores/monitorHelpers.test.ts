import { describe, expect, it } from 'vitest'
import {
  buildMonitorTree,
  collectDescendants,
  groupLabel,
  groupOptions,
  monitorFromResponse,
  monitorsFromResponses,
  parseCodes,
  statusColor,
  statusText,
  type Monitor,
} from './monitorHelpers'
import type { MonitorResponse } from '@/api/types'

function monitor(overrides: Partial<Monitor>): Monitor {
  return {
    id: 1,
    user_id: 1,
    name: 'monitor',
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
    retry_interval: 0,
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

describe('monitor helpers', () => {
  it('parses accepted status codes with fallback', () => {
    expect(parseCodes('["200-299","404"]')).toEqual(['200-299', '404'])
    expect(parseCodes('bad-json')).toEqual(['200-299'])
    expect(parseCodes('')).toEqual(['200-299'])
  })

  it('formats monitor status labels and colors', () => {
    expect(statusText(1)).toBe('UP')
    expect(statusText(0)).toBe('DOWN')
    expect(statusText(2)).toBe('PENDING')
    expect(statusText(9)).toBe('UNKNOWN')
    expect(statusColor(1)).toBe('success')
    expect(statusColor(0)).toBe('danger')
    expect(statusColor(2)).toBe('warning')
    expect(statusColor(9)).toBe('info')
  })

  it('normalizes monitor list API items', () => {
    const item = {
      monitor: {
        ...monitor({
          id: 12,
          name: 'site',
          accepted_status_codes: ['200-299', '301'],
        }),
        accepted_status_codes: '["200-299","301"]',
      },
      tags: [{ id: 1, name: 'prod', color: '#ff0000' }],
      notification_ids: [2, 3],
    } satisfies MonitorResponse

    expect(monitorFromResponse(item)).toMatchObject({
      id: 12,
      name: 'site',
      accepted_status_codes: ['200-299', '301'],
      tags: [{ id: 1, name: 'prod', color: '#ff0000' }],
      notification_ids: [2, 3],
    })
  })

  it('normalizes missing monitor associations to empty arrays', () => {
    const item = {
      monitor: {
        ...monitor({ id: 12 }),
        accepted_status_codes: 'bad-json',
      },
      tags: undefined,
      notification_ids: undefined,
    } as unknown as MonitorResponse

    expect(monitorFromResponse(item)).toMatchObject({
      accepted_status_codes: ['200-299'],
      tags: [],
      notification_ids: [],
    })
    expect(monitorsFromResponses(null)).toEqual([])
    expect(monitorsFromResponses(undefined)).toEqual([])
  })

  it('builds group trees and excludes descendants from group options', () => {
    const monitors = [
      monitor({ id: 1, name: 'root', type: 'group' }),
      monitor({ id: 2, name: 'child', type: 'group', group_id: 1 }),
      monitor({ id: 3, name: 'site', group_id: 2 }),
      monitor({ id: 4, name: 'loose' }),
    ]

    const tree = buildMonitorTree(monitors)
    expect(tree[0]?.name).toBe('root')
    expect(tree[0]?.children?.[0]?.name).toBe('child')
    expect(tree[0]?.children?.[0]?.children?.[0]?.name).toBe('site')
    expect(tree[1]?.name).toBe('loose')

    expect(groupOptions(monitors, 1).map((item) => item.id)).toEqual([])
    expect(groupLabel(monitors, monitors[1]!)).toBe('root / child')
  })

  it('collects descendants recursively', () => {
    const monitors = [
      monitor({ id: 1, type: 'group' }),
      monitor({ id: 2, type: 'group', group_id: 1 }),
      monitor({ id: 3, group_id: 2 }),
      monitor({ id: 4 }),
    ]
    const out = new Set<number>()
    collectDescendants(monitors, 1, out)
    expect([...out]).toEqual([2, 3])
  })
})
