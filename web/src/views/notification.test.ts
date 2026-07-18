import { describe, expect, it } from 'vitest'
import {
  defaultNotificationForm,
  editableNotificationConfig,
  notificationPayloadFromForm,
  notificationActiveTagType,
  notificationActiveText,
  notificationConfigHint,
  notificationConfigText,
  notificationDialogTitle,
  notificationFormFromNotification,
  notificationSavedText,
  notificationSubmitText,
  notificationTypeLabel,
  notificationTypeOptions,
} from './notification'
import type { Notification } from '@/api/types'

function notification(overrides: Partial<Notification> = {}): Notification {
  return {
    id: 1,
    name: 'Ops',
    type: 'email',
    config: { email: 'ops@example.com' } as unknown as string,
    active: true,
    ...overrides,
  }
}

describe('notification helpers', () => {
  it('provides the supported notification type options', () => {
    expect(notificationTypeOptions).toEqual([
      { label: '飞书', value: 'feishu' },
      { label: '邮件', value: 'email' },
    ])
  })

  it('formats labels and hints by notification state', () => {
    expect(notificationConfigHint('feishu')).toContain('webhook_url')
    expect(notificationConfigHint('email')).toContain('email')
    expect(notificationDialogTitle(true)).toBe('编辑通知')
    expect(notificationDialogTitle(false)).toBe('新增通知')
    expect(notificationSubmitText(true)).toBe('保存')
    expect(notificationSubmitText(false)).toBe('创建')
    expect(notificationSavedText(true)).toBe('已更新')
    expect(notificationSavedText(false)).toBe('已创建')
    expect(notificationTypeLabel('feishu')).toBe('飞书')
    expect(notificationTypeLabel('email')).toBe('邮件')
    expect(notificationActiveText(true)).toBe('是')
    expect(notificationActiveText(false)).toBe('否')
    expect(notificationActiveTagType(true)).toBe('success')
    expect(notificationActiveTagType(false)).toBe('info')
  })

  it('serializes notification config for table display and editing', () => {
    expect(notificationConfigText('{"email":"ops@example.com"}')).toBe('{"email":"ops@example.com"}')
    expect(notificationConfigText({ email: 'ops@example.com' })).toBe('{"email":"ops@example.com"}')
    expect(editableNotificationConfig('{"email":"ops@example.com"}')).toBe('{"email":"ops@example.com"}')
    expect(editableNotificationConfig({ email: 'ops@example.com' })).toBe('{\n  "email": "ops@example.com"\n}')
  })

  it('builds default, edit, and submit forms', () => {
    const first = defaultNotificationForm()
    const second = defaultNotificationForm()
    first.name = 'changed'

    expect(second).toEqual({ name: '', type: 'feishu', config: '{}' })
    expect(notificationFormFromNotification(notification())).toEqual({
      name: 'Ops',
      type: 'email',
      config: '{\n  "email": "ops@example.com"\n}',
    })
    expect(notificationPayloadFromForm({ name: 'Ops', type: 'email', config: '{}' })).toEqual({
      name: 'Ops',
      type: 'email',
      config: '{}',
    })
  })
})
