import { defineStore } from 'pinia'
import { reactive, ref } from 'vue'
import api from '@/api/http'
import { arrayFromResponse } from '@/api/responses'
import { wsClient } from '@/api/ws'
import type {
  Heartbeat,
  MonitorPayload,
  MonitorStatus,
  Notification,
} from '@/api/types'
import {
  buildMonitorTree as buildMonitorTreeFromItems,
  groupLabel as groupLabelFromItems,
  groupOptions as groupOptionsFromItems,
  monitorsFromResponses,
  parseCodes,
  statusColor,
  statusText,
  type Monitor,
  type MonitorTreeNode,
} from './monitorHelpers'

export type { MonitorStatus }
export type { Monitor, MonitorTreeNode }

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
      statusList.splice(0, statusList.length, ...arrayFromResponse<MonitorStatus>(res.data))
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
      monitors.value = monitorsFromResponses(res.data)
    } finally {
      loading.value = false
    }
  }

  async function fetchNotifications() {
    try {
      const res = await api.get('/notifications')
      notifications.value = arrayFromResponse<Notification>(res.data)
    } catch {
      // ignore
    }
  }

  async function createMonitor(monitor: MonitorPayload) {
  async function createPingRange(monitor: MonitorPayload) {
    const res = await api.post("/monitors/ping-range", monitor)
    await fetchMonitors()
    return res.data
  }
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

  function buildMonitorTree(items: Monitor[] = monitors.value): MonitorTreeNode[] {
    return buildMonitorTreeFromItems(items)
  }

  function groupOptions(excludeID?: number): Monitor[] {
    return groupOptionsFromItems(monitors.value, excludeID)
  }

  function groupLabel(group: Monitor): string {
    return groupLabelFromItems(monitors.value, group)
  }

  function statusByID(id: number): MonitorStatus | undefined {
    return statusList.find((s) => s.id === id)
  }

  return {
    statusList, monitors, notifications, loading,
    fetchStatus, fetchMonitors, fetchNotifications,
    fetchMonitorStatus, createMonitor, createPingRange, updateMonitor, deleteMonitor, pauseMonitor, resumeMonitor,
    statusColor, statusText, parseCodes, buildMonitorTree, groupOptions, groupLabel, statusByID,
  }
})
