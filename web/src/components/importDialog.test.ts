import { describe, expect, it } from 'vitest'
import {
  defaultImportStrategy,
  emptyImportState,
  hasConflicts,
  hasImportErrors,
  hasMaskedNotifications,
  hasNewTags,
  hasNotifications,
  importExecuteRequest,
  importPreviewFromResponse,
  importPreviewRequest,
  importResultFromResponse,
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

  it('builds import API requests', () => {
    const data = { version: '1.0' }

    expect(importPreviewRequest(data)).toEqual({ data, strategy: 'skip' })
    expect(importExecuteRequest(data, 'overwrite')).toEqual({ data, strategy: 'overwrite' })
  })

  it('normalizes import preview responses', () => {
    expect(importPreviewFromResponse(null)).toEqual({
      new_count: 0,
      conflict_count: 0,
      conflicts: [],
      new_monitors: [],
      new_tags: [],
      notifications: 0,
      masked_notifications: 0,
      summary: '',
    })

    expect(importPreviewFromResponse({
      new_count: 1,
      conflict_count: 0,
      new_tags: undefined as unknown as [],
      notifications: 2,
      masked_notifications: 1,
      summary: 'ok',
    })).toMatchObject({
      new_count: 1,
      new_tags: [],
      notifications: 2,
      masked_notifications: 1,
    })
  })

  it('normalizes import result responses', () => {
    expect(importResultFromResponse(null)).toEqual({
      imported: 0,
      created: 0,
      updated: 0,
      skipped: 0,
      errors: [],
    })

    expect(importResultFromResponse({
      imported: 1,
      created: 1,
      updated: 0,
      skipped: 0,
      errors: undefined as unknown as string[],
    })).toMatchObject({
      imported: 1,
      errors: [],
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
