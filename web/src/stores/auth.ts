import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api/http'
import { apiErrorMessage } from '@/api/errors'
import { wsClient } from '@/api/ws'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const username = ref(localStorage.getItem('username') || '')
  const role = ref(localStorage.getItem('role') || '')
  const userId = ref(Number(localStorage.getItem('user_id') || '0'))

  function setAuth(t: string, uname: string, r: string, uid: number) {
    token.value = t
    username.value = uname
    role.value = r
    userId.value = uid
    localStorage.setItem('token', t)
    localStorage.setItem('username', uname)
    localStorage.setItem('role', r)
    localStorage.setItem('user_id', String(uid))
    wsClient.connect()
  }

  function logout() {
    token.value = ''
    username.value = ''
    role.value = ''
    userId.value = 0
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    localStorage.removeItem('role')
    localStorage.removeItem('user_id')
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
