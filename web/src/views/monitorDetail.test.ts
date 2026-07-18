import { describe, expect, it } from 'vitest'
import {
  buildPingChartOption,
  buildUptimeChartOption,
  detailColumnCount,
  formatPing,
  heartbeatColor,
  heartbeatStatusText,
  heartbeatTitle,
  incidentDuration,
  incidentEndTimeText,
  incidentStatusText,
  incidentStatusType,
  incidentTimeText,
  monitorTargetLabel,
  monitorTargetText,
  pingSeries,
  uptimeBarColor,
  uptimePercent,
  visibleBeatCount,
} from './monitorDetail'
import { monitorTypeText } from './formatters'
import type { Heartbeat, UptimeDataPoint } from '@/api/types'

const point = (overrides: Partial<UptimeDataPoint> = {}): UptimeDataPoint => ({
  timestamp: 1710000000,
  uptime: 0.99,
  avg_ping: 30,
  min_ping: 10,
  max_ping: 50,
  up: 1,
  down: 0,
  ...overrides,
})

describe('monitorDetail helpers', () => {
  it('formats uptime, ping and heartbeat labels', () => {
    expect(uptimePercent(0.9876)).toBe('98.76%')
    expect(formatPing(null)).toBe('-')
    expect(formatPing(12.34)).toBe('12.3 ms')
    expect(heartbeatStatusText(3)).toBe('MAINTENANCE')
    expect(heartbeatColor(1)).toBe('#67C23A')
    expect(heartbeatColor(3)).toBe('#909399')
    expect(heartbeatColor(0)).toBe('#F56C6C')
  })

  it('derives responsive detail layout values', () => {
    expect(detailColumnCount(639)).toBe(1)
    expect(detailColumnCount(640)).toBe(2)
    expect(detailColumnCount(1023)).toBe(2)
    expect(detailColumnCount(1024)).toBe(3)
    expect(visibleBeatCount(0)).toBe(10)
    expect(visibleBeatCount(280)).toBe(20)
  })

  it('formats monitor type and target fields', () => {
    expect(monitorTypeText('http')).toBe('HTTP')
    expect(monitorTargetLabel('ping')).toBe('主机名')
    expect(monitorTargetLabel('http')).toBe('URL / 主机名')
    expect(monitorTargetText({ url: 'https://example.com', hostname: 'example.com' })).toBe('https://example.com')
    expect(monitorTargetText({ url: '', hostname: 'example.com' })).toBe('example.com')
    expect(monitorTargetText({ url: '', hostname: '' })).toBe('-')
  })

  it('builds heartbeat titles from stable fields', () => {
    const beat: Heartbeat = {
      id: 1,
      monitor_id: 2,
      status: 1,
      msg: 'ok',
      ping_ms: 8,
      http_status: 200,
      important: true,
      time: '2024-03-09T16:00:00Z',
    }
    const title = heartbeatTitle(beat)
    expect(title).toContain('UP')
    expect(title).toContain('8.0 ms')
  })

  it('formats incidents', () => {
    expect(incidentDuration(0)).toBe('-')
    expect(incidentDuration(125)).toBe('2m')
    expect(incidentTimeText('2024-03-09T16:00:00Z')).toBe(new Date('2024-03-09T16:00:00Z').toLocaleString('zh-CN'))
    expect(incidentEndTimeText(null)).toBe('未结束')
    expect(incidentEndTimeText('2024-03-09T16:00:00Z')).toBe(new Date('2024-03-09T16:00:00Z').toLocaleString('zh-CN'))
    expect(incidentStatusText(0)).toBe('DOWN')
    expect(incidentStatusText(1)).toBe('已恢复')
    expect(incidentStatusType(0)).toBe('danger')
    expect(incidentStatusType(1)).toBe('success')
  })

  it('builds chart series without down-only points', () => {
    const data = [point({ up: 0, max_ping: 99 }), point({ max_ping: 50 })]
    expect(pingSeries(data, 'max_ping')).toHaveLength(1)

    const option = buildPingChartOption(data)
    expect(option.series[0]?.data).toHaveLength(1)
    expect(option.series[1]?.name).toBe('平均响应')
  })

  it('builds uptime chart option and colors thresholds', () => {
    const option = buildUptimeChartOption([point({ uptime: 0.5 })])
    expect(option.series[0]?.data).toEqual([0.5])
    expect(option.tooltip.formatter([{ name: '3月10日', value: 0.9876 }])).toBe('3月10日<br/>可用率: 98.76%')
    expect(option.yAxis.axisLabel.formatter(0.9)).toBe('90%')
    expect(uptimeBarColor(1)).toBe('#67C23A')
    expect(uptimeBarColor(0.97)).toBe('#409EFF')
    expect(uptimeBarColor(0.9)).toBe('#E6A23C')
    expect(uptimeBarColor(0.5)).toBe('#F56C6C')
  })
})
