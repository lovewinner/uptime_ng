import { beforeEach, describe, expect, it, vi } from 'vitest'
import { clearAuthSnapshot, emptyAuthSnapshot, loadAuthSnapshot, saveAuthSnapshot } from './authStorage'

function stubLocalStorage() {
  const store = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: vi.fn((key: string) => store.get(key) ?? null),
    setItem: vi.fn((key: string, value: string) => {
      store.set(key, value)
    }),
    removeItem: vi.fn((key: string) => {
      store.delete(key)
    }),
    clear: vi.fn(() => {
      store.clear()
    }),
  })
}

describe('auth storage helpers', () => {
  beforeEach(() => {
    stubLocalStorage()
  })

  it('builds the empty auth snapshot', () => {
    expect(emptyAuthSnapshot()).toEqual({
      token: '',
      username: '',
      role: '',
      userId: 0,
    })
  })

  it('loads, saves, and clears auth snapshots', () => {
    expect(loadAuthSnapshot()).toEqual(emptyAuthSnapshot())

    saveAuthSnapshot({
      token: 'jwt',
      username: 'alice',
      role: 'admin',
      userId: 7,
    })

    expect(loadAuthSnapshot()).toEqual({
      token: 'jwt',
      username: 'alice',
      role: 'admin',
      userId: 7,
    })

    clearAuthSnapshot()
    expect(loadAuthSnapshot()).toEqual(emptyAuthSnapshot())
  })
})
