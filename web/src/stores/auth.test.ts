import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const mocks = vi.hoisted(() => ({
  post: vi.fn(),
  connect: vi.fn(),
  disconnect: vi.fn(),
}))

vi.mock('@/api/http', () => ({
  default: { post: mocks.post },
}))

vi.mock('@/api/ws', () => ({
  wsClient: { connect: mocks.connect, disconnect: mocks.disconnect },
}))

import { useAuthStore } from './auth'

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

describe('auth store', () => {
  beforeEach(() => {
    stubLocalStorage()
    mocks.post.mockReset()
    mocks.connect.mockReset()
    mocks.disconnect.mockReset()
    setActivePinia(createPinia())
  })

  it('stores auth state and connects websocket on login', async () => {
    mocks.post.mockResolvedValueOnce({
      data: { token: 'jwt', username: 'alice', role: 'admin', user_id: 7 },
    })

    const store = useAuthStore()
    await expect(store.login('alice', 'password')).resolves.toBe(true)

    expect(store.token).toBe('jwt')
    expect(store.username).toBe('alice')
    expect(store.role).toBe('admin')
    expect(store.userId).toBe(7)
    expect(localStorage.getItem('token')).toBe('jwt')
    expect(mocks.connect).toHaveBeenCalledOnce()
  })

  it('returns api error message on register failure', async () => {
    mocks.post.mockRejectedValueOnce({ response: { data: { error: 'username already exists' } } })

    const store = useAuthStore()
    await expect(store.register('alice', 'password')).resolves.toEqual({
      ok: false,
      error: 'username already exists',
    })
  })

  it('clears auth state and disconnects websocket on logout', () => {
    const store = useAuthStore()
    store.setAuth('jwt', 'alice', 'admin', 7)

    store.logout()

    expect(store.token).toBe('')
    expect(store.username).toBe('')
    expect(store.role).toBe('')
    expect(store.userId).toBe(0)
    expect(localStorage.getItem('token')).toBeNull()
    expect(mocks.disconnect).toHaveBeenCalledOnce()
  })
})
