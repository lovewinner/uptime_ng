import type { MonitorStatus } from '@/api/types'

export interface DashboardSummary {
  realStatuses: MonitorStatus[]
  groupStatuses: MonitorStatus[]
  totalCount: number
  groupCount: number
  upCount: number
  downCount: number
  pendingCount: number
  currentFaults: MonitorStatus[]
  avgPing: number
}

export function dashboardSummary(statuses: MonitorStatus[]): DashboardSummary {
  const realStatuses = statuses.filter((status) => status.type !== 'group')
  const groupStatuses = statuses.filter((status) => status.type === 'group')
  const currentFaults = realStatuses.filter((status) => status.status === 0 || status.status === 2)

  return {
    realStatuses,
    groupStatuses,
    totalCount: realStatuses.length,
    groupCount: groupStatuses.length,
    upCount: countByStatus(realStatuses, 1),
    downCount: countByStatus(realStatuses, 0),
    pendingCount: countByStatus(realStatuses, 2),
    currentFaults,
    avgPing: averagePositivePing(realStatuses),
  }
}

export function countByStatus(statuses: MonitorStatus[], statusCode: number): number {
  return statuses.filter((status) => status.status === statusCode).length
}

export function averagePositivePing(statuses: MonitorStatus[]): number {
  const values = statuses.map((status) => status.ping_ms).filter((value) => value > 0)
  if (values.length === 0) return 0
  return values.reduce((sum, value) => sum + value, 0) / values.length
}

export function uptimePercent(value: number): string {
  return (value * 100).toFixed(2) + '%'
}
