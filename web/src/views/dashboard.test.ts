import { describe, expect, it } from 'vitest'
import type { MonitorStatus } from '@/api/types'
import { averagePositivePing, countByStatus, dashboardSummary, uptimePercent } from './dashboard'

function status(partial: Partial<MonitorStatus>): MonitorStatus {
  return {
    id: partial.id ?? 1,
    name: partial.name ?? 'site',
    type: partial.type ?? 'http',
    group_id: partial.group_id ?? null,
    status: partial.status ?? 1,
    ping_ms: partial.ping_ms ?? 0,
    uptime_24h: partial.uptime_24h ?? 1,
    active: partial.active ?? true,
  }
}

describe('dashboardSummary', () => {
  it('separates groups from real monitors and counts statuses', () => {
    const summary = dashboardSummary([
      status({ id: 1, type: 'http', status: 1, ping_ms: 10 }),
      status({ id: 2, type: 'ping', status: 0, ping_ms: 30 }),
      status({ id: 3, type: 'dns', status: 2, ping_ms: 0 }),
      status({ id: 4, type: 'group', status: 0, ping_ms: 100 }),
    ])

    expect(summary.totalCount).toBe(3)
    expect(summary.groupCount).toBe(1)
    expect(summary.upCount).toBe(1)
    expect(summary.downCount).toBe(1)
    expect(summary.pendingCount).toBe(1)
    expect(summary.currentFaults.map((item) => item.id)).toEqual([2, 3])
    expect(summary.avgPing).toBe(20)
  })
})

describe('dashboard helpers', () => {
  it('counts statuses and averages positive ping only', () => {
    const items = [
      status({ status: 1, ping_ms: 0 }),
      status({ status: 1, ping_ms: 12 }),
      status({ status: 0, ping_ms: 24 }),
    ]
    expect(countByStatus(items, 1)).toBe(2)
    expect(averagePositivePing(items)).toBe(18)
    expect(averagePositivePing([status({ ping_ms: 0 })])).toBe(0)
  })

  it('formats uptime percent', () => {
    expect(uptimePercent(0.9999)).toBe('99.99%')
  })
})
