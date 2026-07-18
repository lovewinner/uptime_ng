import { describe, expect, it } from 'vitest'
import { arrayFromResponse, objectFromResponse } from './responses'

describe('api response helpers', () => {
  it('normalizes nullable list responses', () => {
    const items = [{ id: 1 }]

    expect(arrayFromResponse(items)).toBe(items)
    expect(arrayFromResponse(null)).toEqual([])
    expect(arrayFromResponse(undefined)).toEqual([])
    expect(arrayFromResponse({ id: 1 } as unknown as { id: number }[])).toEqual([])
  })

  it('normalizes nullable object responses', () => {
    const fallback = { count: 0 }
    const data = { count: 3 }

    expect(objectFromResponse(data, fallback)).toBe(data)
    expect(objectFromResponse(null, fallback)).toBe(fallback)
    expect(objectFromResponse(undefined, fallback)).toBe(fallback)
    expect(objectFromResponse([{ count: 3 }] as unknown as { count: number }, fallback)).toBe(fallback)
  })
})
