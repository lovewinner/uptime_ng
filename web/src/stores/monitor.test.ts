import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/http', () => ({ default: {} }))
vi.mock('@/api/ws', () => ({
  wsClient: {
    onMessage: vi.fn(() => vi.fn()),
  },
}))

import { useMonitorStore } from './monitor'

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
})
