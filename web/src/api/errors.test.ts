import { describe, expect, it } from 'vitest'
import { apiErrorMessage, isDialogCancel } from './errors'

describe('apiErrorMessage', () => {
  it('extracts api error responses and generic errors', () => {
    expect(apiErrorMessage({ response: { data: { error: 'bad request' } } }, 'fallback')).toBe('bad request')
    expect(apiErrorMessage('plain error', 'fallback')).toBe('plain error')
    expect(apiErrorMessage(new Error('boom'), 'fallback')).toBe('boom')
    expect(apiErrorMessage({}, 'fallback')).toBe('fallback')
  })
})

describe('isDialogCancel', () => {
  it('recognizes element plus dialog cancel results', () => {
    expect(isDialogCancel('cancel')).toBe(true)
    expect(isDialogCancel('close')).toBe(true)
    expect(isDialogCancel(new Error('cancel'))).toBe(false)
  })
})
