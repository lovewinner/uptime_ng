import type { AuthResponse } from '@/api/types'
import type { AuthSnapshot } from './authStorage'

export function authSnapshotFromResponse(data: AuthResponse): AuthSnapshot {
  return {
    token: data.token,
    username: data.username,
    role: data.role,
    userId: data.user_id,
  }
}
