import { describe, expect, it } from 'vitest'
import {
  averagePingText,
  failedChecksColor,
  formatDowntime,
  incidentTagType,
  uptimeClass,
  uptimePercent,
} from './sla'
import { monitorTypeText } from './formatters'

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

  it('formats remaining table presentation values', () => {
    expect(monitorTypeText('http')).toBe('HTTP')
    expect(failedChecksColor(0)).toBe('#67C23A')
    expect(failedChecksColor(2)).toBe('#F56C6C')
    expect(averagePingText(null)).toBe('-')
    expect(averagePingText(undefined)).toBe('-')
    expect(averagePingText(0)).toBe('-')
    expect(averagePingText(12.34)).toBe('12.3 ms')
    expect(incidentTagType(0)).toBe('success')
    expect(incidentTagType(1)).toBe('danger')
  })
})
