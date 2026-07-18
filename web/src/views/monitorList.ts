import type { MonitorTreeNode } from '@/stores/monitor'

export function visibleMonitorRows(treeRows: MonitorTreeNode[], expandedIds: Set<number>): MonitorTreeNode[] {
  const rows: MonitorTreeNode[] = []
  const walk = (nodes: MonitorTreeNode[]) => {
    nodes.forEach((node) => {
      rows.push(node)
      if (node.children?.length && expandedIds.has(node.id)) {
        walk(node.children)
      }
    })
  }
  walk(treeRows)
  return rows
}

export function monitorTargetText(monitor: MonitorTreeNode): string {
  if (monitor.type === 'group') {
    return `${monitor.children?.length || 0} 个子项`
  }
  if (monitor.url) return monitor.url
  if (monitor.type === 'ping') return monitor.hostname || '-'
  if (monitor.hostname && monitor.port) return `${monitor.hostname}:${monitor.port}`
  if (monitor.hostname) return monitor.hostname
  return '-'
}

export function intervalText(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  return `${Math.floor(seconds / 60)}m`
}

export function refreshIntervalSeconds(interval: number): number {
  return Math.max(3, interval || 60)
}

export function pauseResumeButtonType(active: boolean): 'warning' | 'success' {
  return active ? 'warning' : 'success'
}

export function pauseResumeText(active: boolean): string {
  return active ? '暂停' : '恢复'
}

export function pauseResumeSuccessText(activeBeforeAction: boolean): string {
  return activeBeforeAction ? '已暂停' : '已恢复'
}

export function monitorStatusValue(status: number | undefined): number {
  return status ?? 2
}

export function exportURL(ids?: number[]): string {
  if (!ids?.length) {
    return '/monitors/export'
  }
  return '/monitors/export?ids=' + JSON.stringify(ids)
}

export function nextExpandedIds(row: MonitorTreeNode, expanded: boolean, current: Set<number>): Set<number> {
  const next = new Set(current)
  if (expanded) {
    next.add(row.id)
    return next
  }
  next.delete(row.id)
  removeDescendantExpansion(row, next)
  return next
}

function removeDescendantExpansion(row: MonitorTreeNode, expanded: Set<number>) {
  row.children?.forEach((child) => {
    expanded.delete(child.id)
    removeDescendantExpansion(child, expanded)
  })
}
