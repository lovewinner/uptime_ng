import type { Monitor as ApiMonitor, MonitorResponse } from '@/api/types'
import { arrayFromResponse } from '@/api/responses'

export type Monitor = Omit<ApiMonitor, 'accepted_status_codes'> & { accepted_status_codes: string[] }
export type MonitorTreeNode = Monitor & { children?: MonitorTreeNode[] }
export type StatusColor = 'success' | 'danger' | 'warning' | 'info'

export function parseCodes(raw: string): string[] {
  if (!raw) return ['200-299']
  try {
    return JSON.parse(raw)
  } catch {
    return ['200-299']
  }
}

export function monitorFromResponse(item: MonitorResponse): Monitor {
  return {
    ...item.monitor,
    tags: item.tags || [],
    notification_ids: item.notification_ids || [],
    accepted_status_codes: parseCodes(item.monitor.accepted_status_codes),
  }
}

export function monitorsFromResponses(items: MonitorResponse[] | null | undefined): Monitor[] {
  return arrayFromResponse(items).map(monitorFromResponse)
}

export function buildMonitorTree(items: Monitor[]): MonitorTreeNode[] {
  const nodes = new Map<number, MonitorTreeNode>()
  items.forEach((monitor) => {
    nodes.set(monitor.id, { ...monitor, children: [] })
  })

  const roots: MonitorTreeNode[] = []
  nodes.forEach((node) => {
    if (node.group_id && nodes.has(node.group_id)) {
      nodes.get(node.group_id)!.children!.push(node)
    } else {
      roots.push(node)
    }
  })

  return roots
}

export function groupOptions(items: Monitor[], excludeID?: number): Monitor[] {
  const excluded = new Set<number>()
  if (excludeID) {
    collectDescendants(items, excludeID, excluded)
    excluded.add(excludeID)
  }
  return items.filter((monitor) => monitor.type === 'group' && !excluded.has(monitor.id))
}

export function groupLabel(items: Monitor[], group: Monitor): string {
  const names = [group.name]
  let currentID = group.group_id
  const seen = new Set<number>([group.id])
  while (currentID) {
    if (seen.has(currentID)) break
    seen.add(currentID)
    const parent = items.find((monitor) => monitor.id === currentID)
    if (!parent) break
    names.unshift(parent.name)
    currentID = parent.group_id
  }
  return names.join(' / ')
}

export function collectDescendants(items: Monitor[], id: number, out: Set<number>) {
  items.forEach((monitor) => {
    if (monitor.group_id === id && !out.has(monitor.id)) {
      out.add(monitor.id)
      collectDescendants(items, monitor.id, out)
    }
  })
}

export function statusColor(status: number): StatusColor {
  switch (status) {
    case 1: return 'success'
    case 0: return 'danger'
    case 2: return 'warning'
    default: return 'info'
  }
}

export function statusText(status: number): string {
  switch (status) {
    case 1: return 'UP'
    case 0: return 'DOWN'
    case 2: return 'PENDING'
    default: return 'UNKNOWN'
  }
}
