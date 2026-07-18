import { millisecondsText, percentText } from './formatters'

export interface SLAItem {
  monitor_id: number
  monitor_name: string
  monitor_type: string
  uptime_percentage: number
  total_checks: number
  failed_checks: number
  avg_ping_ms: number
  incidents: number
  total_downtime_seconds: number
}

export function uptimeClass(pct: number): string {
  if (pct >= 0.999) return 'uptime-green'
  if (pct >= 0.99) return 'uptime-yellow'
  return 'uptime-red'
}

export function uptimePercent(pct: number | undefined | null): string {
  if (pct == null) return '-'
  return percentText(pct, 3)
}

export function formatDowntime(seconds: number): string {
  if (seconds <= 0) return '-'
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h > 0) return `${h}h ${m}m ${s}s`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

export function failedChecksColor(count: number): string {
  return count > 0 ? '#F56C6C' : '#67C23A'
}

export function averagePingText(ping: number | undefined | null): string {
  if (!ping) return '-'
  return millisecondsText(ping, 1)
}

export function incidentTagType(count: number): 'danger' | 'success' {
  return count > 0 ? 'danger' : 'success'
}
