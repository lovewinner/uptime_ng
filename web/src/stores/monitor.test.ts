import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/http', () => ({ default: {} }))
vi.mock('@/api/ws', () => ({
  wsClient: {
    onMessage: vi.fn(() => vi.fn()),
  },
}))

import { useMonitorStore, type Monitor } from './monitor'

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

describe('monitor store helpers', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('parses accepted status codes with fallback', () => {
    const store = useMonitorStore()
    expect(store.parseCodes('["200-299","404"]')).toEqual(['200-299', '404'])
    expect(store.parseCodes('bad-json')).toEqual(['200-299'])
    expect(store.parseCodes('')).toEqual(['200-299'])
  })

  it('formats monitor status labels and colors', () => {
    const store = useMonitorStore()
    expect(store.statusText(1)).toBe('UP')
    expect(store.statusText(0)).toBe('DOWN')
    expect(store.statusText(2)).toBe('PENDING')
    expect(store.statusColor(1)).toBe('success')
    expect(store.statusColor(0)).toBe('danger')
    expect(store.statusColor(2)).toBe('warning')
  })

  it('builds group trees and excludes descendants from group options', () => {
    const store = useMonitorStore()
    store.monitors = [
      monitor({ id: 1, name: 'root', type: 'group' }),
      monitor({ id: 2, name: 'child', type: 'group', group_id: 1 }),
      monitor({ id: 3, name: 'site', group_id: 2 }),
      monitor({ id: 4, name: 'loose' }),
    ]

    const tree = store.buildMonitorTree()
    expect(tree[0]?.name).toBe('root')
    expect(tree[0]?.children?.[0]?.name).toBe('child')
    expect(tree[0]?.children?.[0]?.children?.[0]?.name).toBe('site')
    expect(tree[1]?.name).toBe('loose')

    expect(store.groupOptions(1).map((item) => item.id)).toEqual([])
    expect(store.groupLabel(store.monitors[1]!)).toBe('root / child')
  })
})
