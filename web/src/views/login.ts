export interface AuthForm {
  username: string
  password: string
}

export interface AuthSubmitState {
  redirect: boolean
  error: string
}

export function emptyAuthForm(): AuthForm {
  return { username: '', password: '' }
}

export function loginFailureMessage(): string {
  return '用户名或密码错误'
}

export function registerFailureMessage(error?: string): string {
  return error || '注册失败'
}

export function nextRegisterVisible(current: boolean): boolean {
  return !current
}

export function loginSubmitState(ok: boolean): AuthSubmitState {
  return {
    redirect: ok,
    error: ok ? '' : loginFailureMessage(),
  }
}

export function registerSubmitState(result: { ok: boolean; error?: string }): AuthSubmitState {
  return {
    redirect: result.ok,
    error: result.ok ? '' : registerFailureMessage(result.error),
  }
}
