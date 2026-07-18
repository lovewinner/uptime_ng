export type ImportStrategy = 'skip' | 'overwrite' | 'copy'
export type ImportStep = 'upload' | 'preview' | 'result'

export interface ImportPreview {
  new_count: number
  conflict_count: number
  new_tags: Array<{ name: string; color: string }>
  notifications: number
  masked_notifications: number
  summary: string
}

export interface ImportResult {
  imported: number
  created: number
  updated: number
  skipped: number
  errors: string[]
}

export function parseImportJSON(text: string): unknown {
  return JSON.parse(text)
}

export function initialImportStep(): ImportStep {
  return 'upload'
}

export function defaultImportStrategy(): ImportStrategy {
  return 'copy'
}

export function emptyImportState(): {
  step: ImportStep
  previewData: ImportPreview | null
  importResult: ImportResult | null
  error: string
} {
  return {
    step: initialImportStep(),
    previewData: null,
    importResult: null,
    error: '',
  }
}

export function hasConflicts(preview: Pick<ImportPreview, 'conflict_count'> | null): boolean {
  return (preview?.conflict_count ?? 0) > 0
}

export function hasNotifications(preview: Pick<ImportPreview, 'notifications'> | null): boolean {
  return (preview?.notifications ?? 0) > 0
}

export function hasMaskedNotifications(preview: Pick<ImportPreview, 'masked_notifications'> | null): boolean {
  return (preview?.masked_notifications ?? 0) > 0
}

export function maskedNotificationsWarning(count: number): string {
  return `${count} 个通知配置包含脱敏密钥，导入后需要手动补齐`
}

export function hasNewTags(preview: Pick<ImportPreview, 'new_tags'> | null): boolean {
  return !!preview?.new_tags?.length
}

export function hasImportErrors(result: Pick<ImportResult, 'errors'> | null): boolean {
  return !!result?.errors?.length
}
