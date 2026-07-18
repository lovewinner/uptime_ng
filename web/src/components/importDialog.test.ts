import { describe, expect, it } from 'vitest'
import { defaultImportStrategy, initialImportStep, parseImportJSON } from './importDialog'

describe('importDialog helpers', () => {
  it('returns default step and strategy', () => {
    expect(initialImportStep()).toBe('upload')
    expect(defaultImportStrategy()).toBe('copy')
  })

  it('parses import json and surfaces invalid json errors', () => {
    expect(parseImportJSON('{"version":"1.0"}')).toEqual({ version: '1.0' })
    expect(() => parseImportJSON('{')).toThrow()
  })
})
