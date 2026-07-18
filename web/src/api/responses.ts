export function arrayFromResponse<T>(data: T[] | null | undefined): T[] {
  return Array.isArray(data) ? data : []
}

export function objectFromResponse<T extends object>(data: T | null | undefined, fallback: T): T {
  if (data && typeof data === 'object' && !Array.isArray(data)) return data
  return fallback
}
