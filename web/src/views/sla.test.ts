import { describe, expect, it } from 'vitest'
import { formatDowntime, uptimeClass, uptimePercent } from './sla'

describe('SLA formatting helpers', () => {
  it('selects uptime class thresholds', () => {
    expect(uptimeClass(0.999)).toBe('uptime-green')
    expect(uptimeClass(0.99)).toBe('uptime-yellow')
    expect(uptimeClass(0.989)).toBe('uptime-red')
  })

  it('formats uptime percentages', () => {
    expect(uptimePercent(null)).toBe('-')
    expect(uptimePercent(undefined)).toBe('-')
    expect(uptimePercent(0.99995)).toBe('99.995%')
  })

  it('formats downtime duration', () => {
    expect(formatDowntime(0)).toBe('-')
    expect(formatDowntime(9)).toBe('9s')
    expect(formatDowntime(75)).toBe('1m 15s')
    expect(formatDowntime(3675)).toBe('1h 1m 15s')
  })
})
