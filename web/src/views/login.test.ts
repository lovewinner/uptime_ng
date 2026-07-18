import { describe, expect, it } from 'vitest'
import {
  emptyAuthForm,
  loginFailureMessage,
  loginSubmitState,
  nextRegisterVisible,
  registerFailureMessage,
  registerSubmitState,
} from './login'

describe('login helpers', () => {
  it('builds independent empty auth forms', () => {
    const first = emptyAuthForm()
    const second = emptyAuthForm()
    first.username = 'alice'

    expect(second).toEqual({ username: '', password: '' })
  })

  it('formats login and registration errors', () => {
    expect(loginFailureMessage()).toBe('用户名或密码错误')
    expect(registerFailureMessage()).toBe('注册失败')
    expect(registerFailureMessage('username exists')).toBe('username exists')
  })

  it('toggles register mode visibility', () => {
    expect(nextRegisterVisible(false)).toBe(true)
    expect(nextRegisterVisible(true)).toBe(false)
  })

  it('derives submit UI state from auth results', () => {
    expect(loginSubmitState(true)).toEqual({ redirect: true, error: '' })
    expect(loginSubmitState(false)).toEqual({ redirect: false, error: '用户名或密码错误' })
    expect(registerSubmitState({ ok: true })).toEqual({ redirect: true, error: '' })
    expect(registerSubmitState({ ok: false })).toEqual({ redirect: false, error: '注册失败' })
    expect(registerSubmitState({ ok: false, error: 'username exists' })).toEqual({
      redirect: false,
      error: 'username exists',
    })
  })
})
