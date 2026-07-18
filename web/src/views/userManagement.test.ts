import { describe, expect, it } from 'vitest'
import {
  activeTagType,
  activeText,
  activeToggleButtonType,
  activeToggleText,
  nextUserRole,
  roleTagType,
  roleToggleText,
} from './userManagement'

describe('user management helpers', () => {
  it('derives role update and presentation state', () => {
    expect(nextUserRole('admin')).toBe('user')
    expect(nextUserRole('user')).toBe('admin')
    expect(roleTagType('admin')).toBe('danger')
    expect(roleTagType('user')).toBe('info')
    expect(roleToggleText('admin')).toBe('降级')
    expect(roleToggleText('user')).toBe('提升为管理员')
  })

  it('derives active state labels and actions', () => {
    expect(activeTagType(true)).toBe('success')
    expect(activeTagType(false)).toBe('danger')
    expect(activeText(true)).toBe('启用')
    expect(activeText(false)).toBe('禁用')
    expect(activeToggleButtonType(true)).toBe('danger')
    expect(activeToggleButtonType(false)).toBe('success')
    expect(activeToggleText(true)).toBe('禁用')
    expect(activeToggleText(false)).toBe('启用')
  })
})
