import { describe, expect, it } from 'vitest'
import {
  localDateTimeText,
  millisecondsText,
  monitorTypeText,
  percentText,
  roundedNumber,
  timestampMonthDayText,
} from './formatters'

describe('shared view formatters', () => {
  it('formats monitor types consistently', () => {
    expect(monitorTypeText('http')).toBe('HTTP')
    expect(monitorTypeText('group')).toBe('GROUP')
  })

  it('formats common numeric and time values', () => {
    expect(percentText(0.9876, 2)).toBe('98.76%')
    expect(percentText(0.99995, 3)).toBe('99.995%')
    expect(millisecondsText(12.34, 1)).toBe('12.3 ms')
    expect(millisecondsText(12.6, 0)).toBe('13 ms')
    expect(roundedNumber(12.6, 0)).toBe(13)
    expect(roundedNumber(12.34, 1)).toBe(12.3)
    expect(localDateTimeText('2024-03-09T16:00:00Z')).toBe(new Date('2024-03-09T16:00:00Z').toLocaleString('zh-CN'))
    expect(timestampMonthDayText(1710000000)).toBe(new Date(1710000000 * 1000).getMonth() + 1 + '月' + new Date(1710000000 * 1000).getDate() + '日')
  })
})
