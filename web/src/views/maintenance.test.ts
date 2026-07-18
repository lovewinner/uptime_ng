import { describe, expect, it } from 'vitest'
import {
  maintenanceActiveTagType,
  maintenanceActiveText,
  defaultMaintenanceForm,
  maintenanceFormFromWindow,
  maintenanceDialogTitle,
  maintenanceMonitorName,
  maintenancePayloadFromForm,
  maintenanceTimePayload,
  maintenanceTimeText,
} from './maintenance'
import type { MaintenanceWindow, Monitor } from '@/api/types'

function window(overrides: Partial<MaintenanceWindow> = {}): MaintenanceWindow {
  return {
    id: 1,
    user_id: 1,
    monitor_id: null,
    name: 'Deploy window',
    description: '',
    start_at: '2026-07-19T01:02:03Z',
    end_at: '2026-07-19T02:02:03Z',
    active: true,
    ...overrides,
  }
}

function monitor(overrides: Partial<Monitor>): Monitor {
  return {
    id: 1,
    user_id: 1,
    name: 'API',
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
    ...overrides,
  }
}

describe('maintenance helpers', () => {
  it('selects the dialog title by editing state', () => {
    expect(maintenanceDialogTitle(null)).toBe('新增维护窗口')
    expect(maintenanceDialogTitle(window())).toBe('编辑维护窗口')
  })

  it('resolves monitor names for global, known, and missing targets', () => {
    const monitors = [monitor({ id: 1, name: 'API' })]

    expect(maintenanceMonitorName(monitors, null)).toBe('全部监控')
    expect(maintenanceMonitorName(monitors, 1)).toBe('API')
    expect(maintenanceMonitorName(monitors, 9)).toBe('#9')
  })

  it('serializes date picker values as RFC3339 timestamps', () => {
    expect(maintenanceTimePayload('2026-07-19T01:02:03')).toBe(new Date('2026-07-19T01:02:03').toISOString())
  })

  it('formats maintenance timestamps with the product locale', () => {
    expect(maintenanceTimeText('2026-07-19T01:02:03Z')).toBe(new Date('2026-07-19T01:02:03Z').toLocaleString('zh-CN'))
  })

  it('formats active state presentation values', () => {
    expect(maintenanceActiveTagType(true)).toBe('success')
    expect(maintenanceActiveTagType(false)).toBe('info')
    expect(maintenanceActiveText(true)).toBe('启用')
    expect(maintenanceActiveText(false)).toBe('停用')
  })

  it('builds default and edit forms independently', () => {
    const first = defaultMaintenanceForm()
    const second = defaultMaintenanceForm()
    first.name = 'changed'

    expect(second).toEqual({
      name: '',
      description: '',
      monitor_id: null,
      start_at: '',
      end_at: '',
      active: true,
    })
    expect(maintenanceFormFromWindow(window({ description: '', monitor_id: 7 }))).toMatchObject({
      name: 'Deploy window',
      description: '',
      monitor_id: 7,
      start_at: '2026-07-19T01:02:03Z',
      end_at: '2026-07-19T02:02:03Z',
      active: true,
    })
  })

  it('builds submit payload with RFC3339 timestamps', () => {
    const form = {
      ...defaultMaintenanceForm(),
      name: 'Deploy',
      start_at: '2026-07-19T01:02:03',
      end_at: '2026-07-19T02:02:03',
    }

    expect(maintenancePayloadFromForm(form)).toEqual({
      ...form,
      start_at: new Date('2026-07-19T01:02:03').toISOString(),
      end_at: new Date('2026-07-19T02:02:03').toISOString(),
    })
  })
})
