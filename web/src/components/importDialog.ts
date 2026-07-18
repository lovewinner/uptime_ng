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
