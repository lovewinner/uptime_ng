type UserRole = 'admin' | 'user' | string

export function nextUserRole(role: UserRole): 'admin' | 'user' {
  return role === 'admin' ? 'user' : 'admin'
}

export function roleTagType(role: UserRole): 'danger' | 'info' {
  return role === 'admin' ? 'danger' : 'info'
}

export function roleToggleText(role: UserRole): string {
  return role === 'admin' ? '降级' : '提升为管理员'
}

export function activeTagType(active: boolean): 'success' | 'danger' {
  return active ? 'success' : 'danger'
}

export function activeText(active: boolean): string {
  return active ? '启用' : '禁用'
}

export function activeToggleButtonType(active: boolean): 'danger' | 'success' {
  return active ? 'danger' : 'success'
}

export function activeToggleText(active: boolean): string {
  return active ? '禁用' : '启用'
}
