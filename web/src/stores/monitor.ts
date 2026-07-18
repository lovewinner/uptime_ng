import { defineStore } from 'pinia'
import { reactive, ref } from 'vue'
import api from '@/api/http'
import { wsClient } from '@/api/ws'
import type {
  Heartbeat,
  Monitor as ApiMonitor,
  MonitorListItem,
  MonitorPayload,
  MonitorStatus,
  Notification,
} from '@/api/types'

export type { MonitorStatus }

export type Monitor = Omit<ApiMonitor, 'accepted_status_codes'> & { accepted_status_codes: string[] }
export type MonitorTreeNode = Monitor & { children?: MonitorTreeNode[] }

export const useMonitorStore = defineStore('monitor', () => {
  const statusList = reactive<MonitorStatus[]>([])
  const monitors = ref<Monitor[]>([])
  const notifications = ref<Notification[]>([])
  const loading = ref(false)

  wsClient.onMessage((msg) => {
    if (msg.type === 'heartbeat' && msg.payload) {
      const beat = msg.payload as Heartbeat
      const idx = statusList.findIndex((s) => s.id === beat.monitor_id)
      if (idx >= 0 && statusList[idx]) {
        statusList[idx]!.status = beat.status
        statusList[idx]!.ping_ms = beat.ping_ms || 0
      }
    }
  })

  async function fetchStatus() {
    try {
      const res = await api.get('/monitors/status')
      statusList.splice(0, statusList.length, ...res.data)
    } catch {
      // ignore
    }
  }

  async function fetchMonitorStatus(id: number) {
    const res = await api.get(`/monitors/${id}/status`)
    upsertStatus(res.data as MonitorStatus)
    return res.data as MonitorStatus
  }

  function upsertStatus(status: MonitorStatus) {
    const idx = statusList.findIndex((s) => s.id === status.id)
    if (idx >= 0) {
      statusList[idx] = status
    } else {
      statusList.push(status)
    }
  }

  async function fetchMonitors() {
    loading.value = true
    try {
      const res = await api.get('/monitors')
      const data = res.data as MonitorListItem[]
      monitors.value = data.map((item) => ({
        ...item.monitor,
        tags: item.tags || [],
        notification_ids: item.notification_ids || [],
        accepted_status_codes: parseCodes(item.monitor.accepted_status_codes),
      }))
    } finally {
      loading.value = false
    }
  }

  async function fetchNotifications() {
    try {
      const res = await api.get('/notifications')
      notifications.value = res.data
    } catch {
      // ignore
    }
  }

  async function createMonitor(monitor: MonitorPayload) {
    const res = await api.post('/monitors', monitor)
    await fetchMonitors()
    return res.data
  }

  async function updateMonitor(id: number, monitor: MonitorPayload) {
    await api.put(`/monitors/${id}`, monitor)
    await fetchMonitors()
  }

  async function deleteMonitor(id: number) {
    await api.delete(`/monitors/${id}`)
    await fetchMonitors()
  }

  async function pauseMonitor(id: number) {
    await api.post(`/monitors/${id}/pause`)
    await fetchMonitors()
  }

  async function resumeMonitor(id: number) {
    await api.post(`/monitors/${id}/resume`)
    await fetchMonitors()
  }

  function parseCodes(raw: string): string[] {
    if (!raw) return ['200-299']
    try {
      return JSON.parse(raw)
    } catch {
      return ['200-299']
    }
  }

  function buildMonitorTree(items: Monitor[] = monitors.value): MonitorTreeNode[] {
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

  function groupOptions(excludeID?: number): Monitor[] {
    const excluded = new Set<number>()
    if (excludeID) {
      collectDescendants(excludeID, excluded)
      excluded.add(excludeID)
    }
    return monitors.value.filter((m) => m.type === 'group' && !excluded.has(m.id))
  }

  function groupLabel(group: Monitor): string {
    const names = [group.name]
    let currentID = group.group_id
    const seen = new Set<number>([group.id])
    while (currentID) {
      if (seen.has(currentID)) break
      seen.add(currentID)
      const parent = monitors.value.find((m) => m.id === currentID)
      if (!parent) break
      names.unshift(parent.name)
      currentID = parent.group_id
    }
    return names.join(' / ')
  }

  function statusByID(id: number): MonitorStatus | undefined {
    return statusList.find((s) => s.id === id)
  }

  function collectDescendants(id: number, out: Set<number>) {
    monitors.value.forEach((m) => {
      if (m.group_id === id && !out.has(m.id)) {
        out.add(m.id)
        collectDescendants(m.id, out)
      }
    })
  }

  function statusColor(status: number): 'success' | 'danger' | 'warning' | 'info' {
    switch (status) {
      case 1: return 'success'
      case 0: return 'danger'
      case 2: return 'warning'
      default: return 'info'
    }
  }

  function statusText(status: number): string {
    switch (status) {
      case 1: return 'UP'
      case 0: return 'DOWN'
      case 2: return 'PENDING'
      default: return 'UNKNOWN'
    }
  }

  return {
    statusList, monitors, notifications, loading,
    fetchStatus, fetchMonitors, fetchNotifications,
    fetchMonitorStatus, createMonitor, updateMonitor, deleteMonitor, pauseMonitor, resumeMonitor,
    statusColor, statusText, parseCodes, buildMonitorTree, groupOptions, groupLabel, statusByID,
  }
})
