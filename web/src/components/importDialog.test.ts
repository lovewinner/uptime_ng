import { describe, expect, it } from 'vitest'
import {
  defaultImportStrategy,
  emptyImportState,
  hasConflicts,
  hasImportErrors,
  hasMaskedNotifications,
  hasNewTags,
  hasNotifications,
  initialImportStep,
  maskedNotificationsWarning,
  parseImportJSON,
} from './importDialog'

describe('importDialog helpers', () => {
  it('returns default step and strategy', () => {
    expect(initialImportStep()).toBe('upload')
    expect(defaultImportStrategy()).toBe('copy')
  })

  it('parses import json and surfaces invalid json errors', () => {
    expect(parseImportJSON('{"version":"1.0"}')).toEqual({ version: '1.0' })
    expect(() => parseImportJSON('{')).toThrow()
  })

  it('returns empty import dialog state', () => {
    expect(emptyImportState()).toEqual({
      step: 'upload',
      previewData: null,
      importResult: null,
      error: '',
    })
  })

  it('derives import preview visibility flags and warning text', () => {
    const preview = {
      conflict_count: 1,
      notifications: 2,
      masked_notifications: 1,
      new_tags: [{ name: 'prod', color: '#67C23A' }],
    }

    expect(hasConflicts(preview)).toBe(true)
    expect(hasNotifications(preview)).toBe(true)
    expect(hasMaskedNotifications(preview)).toBe(true)
    expect(hasNewTags(preview)).toBe(true)
    expect(maskedNotificationsWarning(1)).toBe('1 个通知配置包含脱敏密钥，导入后需要手动补齐')
  })

  it('handles absent import preview and result collections', () => {
    expect(hasConflicts(null)).toBe(false)
    expect(hasNotifications(null)).toBe(false)
    expect(hasMaskedNotifications(null)).toBe(false)
    expect(hasNewTags(null)).toBe(false)
    expect(hasImportErrors(null)).toBe(false)
    expect(hasImportErrors({ errors: ['bad row'] })).toBe(true)
  })
})
