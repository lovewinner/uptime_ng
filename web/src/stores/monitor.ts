import { defineStore } from 'pinia'
import { reactive, ref } from 'vue'
import api from '@/api/http'
import { wsClient } from '@/api/ws'

export interface MonitorStatus {
  id: number
  name: string
  type: string
  status: number // 0=DOWN 1=UP 2=PENDING
  ping_ms: number
  uptime_24h: number
  active: boolean
}

export interface Monitor {
  id: number
  user_id: number
  name: string
  description: string
  type: string
  active: boolean
  url: string
  hostname: string
  port: number
  method: string
  interval: number
  timeout: number
  max_retries: number
  retry_interval: number
  resend_interval: number
  keyword: string
  invert_keyword: boolean
  ignore_tls: boolean
  upside_down: boolean
  max_redirects: number
  accepted_status_codes: string[]
  tags: { id: number; name: string; color: string }[]
  notification_ids: number[]
}

export interface Notification {
  id: number
  name: string
  type: string
  config: string
  active: boolean
}

export const useMonitorStore = defineStore('monitor', () => {
  const statusList = reactive<MonitorStatus[]>([])
  const monitors = ref<Monitor[]>([])
  const notifications = ref<Notification[]>([])
  const loading = ref(false)

  wsClient.onMessage((msg) => {
    if (msg.type === 'heartbeat' && msg.payload) {
      const beat = msg.payload
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

  async function fetchMonitors() {
    loading.value = true
    try {
      const res = await api.get('/monitors')
      const data = res.data as any[]
      monitors.value = data.map((item: any) => ({
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

  async function createMonitor(monitor: any) {
    const res = await api.post('/monitors', monitor)
    await fetchMonitors()
    await fetchStatus()
    return res.data
  }

  async function updateMonitor(id: number, monitor: any) {
    await api.put(`/monitors/${id}`, monitor)
    await fetchMonitors()
    await fetchStatus()
  }

  async function deleteMonitor(id: number) {
    await api.delete(`/monitors/${id}`)
    await fetchMonitors()
    await fetchStatus()
  }

  async function pauseMonitor(id: number) {
    await api.post(`/monitors/${id}/pause`)
    await fetchMonitors()
    await fetchStatus()
  }

  async function resumeMonitor(id: number) {
    await api.post(`/monitors/${id}/resume`)
    await fetchMonitors()
    await fetchStatus()
  }

  function parseCodes(raw: string): string[] {
    if (!raw) return ['200-299']
    try {
      return JSON.parse(raw)
    } catch {
      return ['200-299']
    }
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
    createMonitor, updateMonitor, deleteMonitor, pauseMonitor, resumeMonitor,
    statusColor, statusText, parseCodes,
  }
})