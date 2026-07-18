export function apiErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'string') return error
  if (typeof error === 'object' && error !== null && 'response' in error) {
    const response = (error as { response?: { data?: { error?: string } } }).response
    return response?.data?.error || fallback
  }
  if (error instanceof Error) return error.message || fallback
  return fallback
}

export function isDialogCancel(error: unknown): boolean {
  return error === 'cancel' || error === 'close'
}
