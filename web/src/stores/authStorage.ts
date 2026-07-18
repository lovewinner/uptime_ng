export interface AuthSnapshot {
  token: string
  username: string
  role: string
  userId: number
}

const authStorageKeys = {
  token: 'token',
  username: 'username',
  role: 'role',
  userId: 'user_id',
}

export function emptyAuthSnapshot(): AuthSnapshot {
  return {
    token: '',
    username: '',
    role: '',
    userId: 0,
  }
}

export function loadAuthSnapshot(): AuthSnapshot {
  return {
    token: localStorage.getItem(authStorageKeys.token) || '',
    username: localStorage.getItem(authStorageKeys.username) || '',
    role: localStorage.getItem(authStorageKeys.role) || '',
    userId: Number(localStorage.getItem(authStorageKeys.userId) || '0'),
  }
}

export function saveAuthSnapshot(snapshot: AuthSnapshot) {
  localStorage.setItem(authStorageKeys.token, snapshot.token)
  localStorage.setItem(authStorageKeys.username, snapshot.username)
  localStorage.setItem(authStorageKeys.role, snapshot.role)
  localStorage.setItem(authStorageKeys.userId, String(snapshot.userId))
}

export function clearAuthSnapshot() {
  localStorage.removeItem(authStorageKeys.token)
  localStorage.removeItem(authStorageKeys.username)
  localStorage.removeItem(authStorageKeys.role)
  localStorage.removeItem(authStorageKeys.userId)
}
