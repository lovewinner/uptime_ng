import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import CreateMonitorDialog from './CreateMonitorDialog.vue'
import type { Monitor } from '@/stores/monitor'

vi.mock('@/api/http', () => ({ default: {} }))
vi.mock('@/api/ws', () => ({
  wsClient: {
    onMessage: vi.fn(() => vi.fn()),
  },
}))

function groupMonitor(): Monitor {
  return {
    id: 1,
    user_id: 1,
    name: 'platform',
    description: '',
    type: 'group',
    group_id: null,
    active: true,
    url: '',
    hostname: '',
    port: 0,
    method: '',
    interval: 30,
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
  }
}

describe('CreateMonitorDialog', () => {
  it('shows group interval and notification fields', () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const wrapper = mount(CreateMonitorDialog, {
      props: {
        modelValue: true,
        monitor: groupMonitor(),
      },
      global: {
        plugins: [pinia],
        stubs: {
          'el-dialog': { template: '<div><slot /><slot name="footer" /></div>' },
          'el-form': { template: '<form><slot /></form>' },
          'el-form-item': { template: '<label><span>{{ label }}</span><slot /></label>', props: ['label'] },
          'el-select': { template: '<div><slot /></div>' },
          'el-option': { template: '<div />' },
          'el-input': { template: '<input />' },
          'el-input-number': { template: '<input />' },
          'el-button': { template: '<button><slot /></button>' },
          'el-divider': { template: '<div><slot /></div>' },
          'el-switch': { template: '<input />' },
          'el-tag': { template: '<span><slot /></span>' },
        },
      },
    })
    expect(wrapper.text()).toContain('检查间隔')
    expect(wrapper.text()).toContain('关联通知')
  })
})
