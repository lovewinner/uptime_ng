import type { MaintenanceWindow, Monitor } from '@/api/types'
import { localDateTimeText } from './formatters'

type MonitorOption = Pick<Monitor, 'id' | 'name'>

export interface MaintenanceForm {
  name: string
  description: string
  monitor_id: number | null
  start_at: string
  end_at: string
  active: boolean
}

export interface MaintenancePayload extends MaintenanceForm {
  start_at: string
  end_at: string
}

export function defaultMaintenanceForm(): MaintenanceForm {
  return {
    name: '',
    description: '',
    monitor_id: null,
    start_at: '',
    end_at: '',
    active: true,
  }
}

export function maintenanceFormFromWindow(window: MaintenanceWindow): MaintenanceForm {
  return {
    name: window.name,
    description: window.description || '',
    monitor_id: window.monitor_id,
    start_at: window.start_at,
    end_at: window.end_at,
    active: window.active,
  }
}

export function maintenancePayloadFromForm(form: MaintenanceForm): MaintenancePayload {
  return {
    ...form,
    start_at: maintenanceTimePayload(form.start_at),
    end_at: maintenanceTimePayload(form.end_at),
  }
}

export function maintenanceDialogTitle(editing: MaintenanceWindow | null): string {
  return editing ? '编辑维护窗口' : '新增维护窗口'
}

export function maintenanceMonitorName(monitors: MonitorOption[], id: number | null): string {
  if (!id) return '全部监控'
  return monitors.find((monitor) => monitor.id === id)?.name || `#${id}`
}

export function maintenanceTimePayload(value: string): string {
  return new Date(value).toISOString()
}

export function maintenanceTimeText(value: string): string {
  return localDateTimeText(value)
}

export function maintenanceActiveTagType(active: boolean): 'success' | 'info' {
  return active ? 'success' : 'info'
}

export function maintenanceActiveText(active: boolean): string {
  return active ? '启用' : '停用'
}
