export function monitorTypeText(type: string): string {
  return type.toUpperCase()
}

export function percentText(value: number, digits: number): string {
  return (value * 100).toFixed(digits) + '%'
}

export function millisecondsText(value: number, digits: number): string {
  return value.toFixed(digits) + ' ms'
}

export function roundedNumber(value: number, digits: number): number {
  return Number(value.toFixed(digits))
}

export function localDateTimeText(value: string): string {
  return new Date(value).toLocaleString('zh-CN')
}

export function timestampMonthDayText(timestampSeconds: number): string {
  const date = new Date(timestampSeconds * 1000)
  return (date.getMonth() + 1) + '月' + date.getDate() + '日'
}
