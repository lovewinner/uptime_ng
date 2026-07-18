import type { Notification } from '@/api/types'

export type NotificationType = Notification['type']

export interface NotificationForm {
  name: string
  type: NotificationType
  config: string
}

export const notificationTypeOptions: Array<{ label: string; value: NotificationType }> = [
  { label: '飞书', value: 'feishu' },
  { label: '邮件', value: 'email' },
]

export function defaultNotificationForm(): NotificationForm {
  return {
    name: '',
    type: 'feishu',
    config: '{}',
  }
}

export function notificationFormFromNotification(notification: Notification): NotificationForm {
  return {
    name: notification.name,
    type: notification.type,
    config: editableNotificationConfig(notification.config),
  }
}

export function notificationPayloadFromForm(form: NotificationForm): NotificationForm {
  return { ...form }
}

export function notificationConfigHint(type: string): string {
  if (type === 'feishu') {
    return '示例: {"webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"}'
  }
  return '示例: {"email": "ops@example.com"}'
}

export function notificationDialogTitle(isEdit: boolean): string {
  return isEdit ? '编辑通知' : '新增通知'
}

export function notificationSubmitText(isEdit: boolean): string {
  return isEdit ? '保存' : '创建'
}

export function notificationSavedText(isEdit: boolean): string {
  return isEdit ? '已更新' : '已创建'
}

export function notificationTypeLabel(type: string): string {
  return type === 'feishu' ? '飞书' : '邮件'
}

export function notificationActiveText(active: boolean): string {
  return active ? '是' : '否'
}

export function notificationActiveTagType(active: boolean): 'success' | 'info' {
  return active ? 'success' : 'info'
}

export function notificationConfigText(config: unknown): string {
  return typeof config === 'string' ? config : JSON.stringify(config)
}

export function editableNotificationConfig(config: unknown): string {
  return typeof config === 'string' ? config : JSON.stringify(config, null, 2)
}
