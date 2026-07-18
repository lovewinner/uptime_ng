import { arrayFromResponse, objectFromResponse } from '@/api/responses'

export type ImportStrategy = 'skip' | 'overwrite' | 'copy'
export type ImportStep = 'upload' | 'preview' | 'result'

export interface ImportPreview {
  new_count: number
  conflict_count: number
  conflicts?: Array<{ name: string; type: string; existing_id: number; existing_name: string }>
  new_monitors?: unknown[]
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

export interface ImportRequest {
  data: unknown
  strategy: ImportStrategy
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

export function importPreviewRequest(data: unknown): ImportRequest {
  return { data, strategy: 'skip' }
}

export function importExecuteRequest(data: unknown, strategy: ImportStrategy): ImportRequest {
  return { data, strategy }
}

export function emptyImportPreview(): ImportPreview {
  return {
    new_count: 0,
    conflict_count: 0,
    conflicts: [],
    new_monitors: [],
    new_tags: [],
    notifications: 0,
    masked_notifications: 0,
    summary: '',
  }
}

export function importPreviewFromResponse(data: ImportPreview | null | undefined): ImportPreview {
  const preview = objectFromResponse(data, emptyImportPreview())
  return {
    ...emptyImportPreview(),
    ...preview,
    conflicts: arrayFromResponse(preview.conflicts),
    new_monitors: arrayFromResponse(preview.new_monitors),
    new_tags: arrayFromResponse(preview.new_tags),
  }
}

export function emptyImportResult(): ImportResult {
  return {
    imported: 0,
    created: 0,
    updated: 0,
    skipped: 0,
    errors: [],
  }
}

export function importResultFromResponse(data: ImportResult | null | undefined): ImportResult {
  const result = objectFromResponse(data, emptyImportResult())
  return {
    ...emptyImportResult(),
    ...result,
    errors: arrayFromResponse(result.errors),
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
