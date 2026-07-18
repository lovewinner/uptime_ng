import type { Heartbeat, UptimeDataPoint } from '@/api/types'

type TooltipParam = {
  name: string
  value: number
}

export function uptimePercent(value: number): string {
  return (value * 100).toFixed(2) + '%'
}

export function formatPing(ping: number | null): string {
  if (ping == null) return '-'
  return ping.toFixed(1) + ' ms'
}

export function heartbeatStatusText(status: number): string {
  switch (status) {
    case 1: return 'UP'
    case 0: return 'DOWN'
    case 2: return 'PENDING'
    case 3: return 'MAINTENANCE'
    default: return 'UNKNOWN'
  }
}

export function heartbeatColor(status: number): string {
  switch (status) {
    case 1: return '#67C23A'
    case 3: return '#909399'
    default: return '#F56C6C'
  }
}

export function heartbeatTitle(beat: Heartbeat): string {
  return `${new Date(beat.time).toLocaleString('zh-CN')} · ${heartbeatStatusText(beat.status)} · ${formatPing(beat.ping_ms)}`
}

export function incidentDuration(seconds: number): string {
  return seconds ? Math.floor(seconds / 60) + 'm' : '-'
}

export function incidentStatusText(status: number): string {
  return status === 0 ? 'DOWN' : '已恢复'
}

export function incidentStatusType(status: number): 'danger' | 'success' {
  return status === 0 ? 'danger' : 'success'
}

export function pingSeries(data: UptimeDataPoint[], field: 'max_ping' | 'avg_ping' | 'min_ping') {
  return data
    .filter((point) => point.up > 0)
    .map((point) => [new Date(point.timestamp * 1000), Number(point[field] ?? 0)])
}

export function buildPingChartOption(data: UptimeDataPoint[]) {
  return {
    color: ['#E6A23C', '#409EFF', '#67C23A'],
    tooltip: { trigger: 'axis' },
    legend: { data: ['平均响应', '最大响应', '最小响应'], top: 0, right: 0, icon: 'roundRect', itemWidth: 20 },
    grid: { left: 50, right: 20, top: 40, bottom: 30 },
    xAxis: {
      type: 'time',
      minInterval: 3600 * 1000,
      axisLabel: { formatter: '{HH}:00' },
    },
    yAxis: {
      type: 'value',
      name: 'ms',
      nameLocation: 'middle',
      nameGap: 40,
    },
    series: [
      {
        name: '最大响应',
        type: 'line',
        data: pingSeries(data, 'max_ping'),
        smooth: true,
        symbol: 'circle',
      },
      {
        name: '平均响应',
        type: 'line',
        data: pingSeries(data, 'avg_ping'),
        smooth: true,
        areaStyle: { opacity: 0.1 },
        symbol: 'circle',
      },
      {
        name: '最小响应',
        type: 'line',
        data: pingSeries(data, 'min_ping'),
        smooth: true,
        symbol: 'circle',
      },
    ],
  }
}

export function buildUptimeChartOption(data: UptimeDataPoint[]) {
  return {
    tooltip: {
      trigger: 'axis',
      formatter: (params: TooltipParam[]) => {
        const point = params[0]
        if (!point) return ''
        return `${point.name}<br/>可用率: ${(point.value * 100).toFixed(2)}%`
      },
    },
    grid: { left: 55, right: 20, top: 30, bottom: 50 },
    xAxis: {
      type: 'category',
      data: data.map((point) => {
        const date = new Date(point.timestamp * 1000)
        return (date.getMonth() + 1) + '月' + date.getDate() + '日'
      }),
      axisLabel: {
        margin: 10,
      },
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: 1,
      axisLabel: { formatter: (value: number) => (value * 100).toFixed(0) + '%' },
    },
    series: [
      {
        name: '可用率',
        type: 'bar',
        data: data.map((point) => Number(point.uptime ?? 0)),
        itemStyle: {
          color: (params: { value: number }) => uptimeBarColor(params.value),
        },
      },
    ],
  }
}

export function uptimeBarColor(value: number): string {
  if (value > 0.99) return '#67C23A'
  if (value > 0.95) return '#409EFF'
  if (value > 0.8) return '#E6A23C'
  return '#F56C6C'
}
