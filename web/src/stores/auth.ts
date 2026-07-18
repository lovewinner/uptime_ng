import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api/http'
import { apiErrorMessage } from '@/api/errors'
import { wsClient } from '@/api/ws'
import { clearAuthSnapshot, emptyAuthSnapshot, loadAuthSnapshot, saveAuthSnapshot } from './authStorage'

export const useAuthStore = defineStore('auth', () => {
  const stored = loadAuthSnapshot()
  const token = ref(stored.token)
  const username = ref(stored.username)
  const role = ref(stored.role)
  const userId = ref(stored.userId)

  function setAuth(t: string, uname: string, r: string, uid: number) {
    token.value = t
    username.value = uname
    role.value = r
    userId.value = uid
    saveAuthSnapshot({ token: t, username: uname, role: r, userId: uid })
    wsClient.connect()
  }

  function logout() {
    const empty = emptyAuthSnapshot()
    token.value = empty.token
    username.value = empty.username
    role.value = empty.role
    userId.value = empty.userId
    clearAuthSnapshot()
    wsClient.disconnect()
  }

  async function login(username_: string, password_: string): Promise<boolean> {
    try {
      const res = await api.post('/auth/login', { username: username_, password: password_ })
      const data = res.data
      setAuth(data.token, data.username, data.role, data.user_id)
      return true
    } catch {
      return false
    }
  }

  async function register(username_: string, password_: string): Promise<{ ok: boolean; error?: string }> {
    try {
      const res = await api.post('/auth/register', { username: username_, password: password_ })
      const data = res.data
      setAuth(data.token, data.username, data.role, data.user_id)
      return { ok: true }
    } catch (e: unknown) {
      return { ok: false, error: apiErrorMessage(e, '注册失败') }
    }
  }

  const isLoggedIn = () => !!token.value
  const isAdmin = () => role.value === 'admin'

  return { token, username, role, userId, login, register, logout, setAuth, isLoggedIn, isAdmin }
})
