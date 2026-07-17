<script setup lang="ts">
import { onMounted, ref, computed, onUnmounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import api from '@/api/http'
import { useMonitorStore } from '@/stores/monitor'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { wsClient } from '@/api/ws'
import type { Heartbeat, Incident, UptimeDataPoint, UptimeSummary } from '@/api/types'
import type { Monitor } from '@/stores/monitor'

use([CanvasRenderer, LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent])

const route = useRoute()
const store = useMonitorStore()
const monitorId = computed(() => Number(route.params.id))

const windowWidth = ref(window.innerWidth)
let beatsObserver: ResizeObserver | null = null
let refreshTimer: ReturnType<typeof setInterval> | null = null
let unsubscribeWS: (() => void) | null = null

function onResize() {
  windowWidth.value = window.innerWidth
}
onMounted(() => window.addEventListener('resize', onResize))
onUnmounted(() => {
  window.removeEventListener('resize', onResize)
  beatsObserver?.disconnect()
  if (refreshTimer) clearInterval(refreshTimer)
  unsubscribeWS?.()
})

const descColumns = computed(() => windowWidth.value < 640 ? 1 : windowWidth.value < 1024 ? 2 : 3)

const monitor = ref<Monitor | null>(null)
const currentStatus = computed(() => {
  if (!monitor.value) return 2
  return store.statusList.find((s) => s.id === monitor.value?.id)?.status ?? 2
})
const incidentList = ref<Incident[]>([])
const pingChartData = ref<UptimeDataPoint[]>([])
const uptimeChartData = ref<UptimeDataPoint[]>([])
const loading = ref(false)
const uptimeSummary = ref<UptimeSummary>({ uptime_24h: 0, uptime_30d: 0, uptime_1y: 0 })

function uptimePercent(v: number): string {
  return (v * 100).toFixed(2) + '%'
}

const beatsContainer = ref<{ $el?: HTMLElement } | null>(null)
const containerWidth = ref(0)
const allBeats = ref<Heartbeat[]>([])
const beatCount = computed(() => Math.max(10, Math.floor(containerWidth.value / 14)))
const heartbeatList = computed(() => allBeats.value.slice(-beatCount.value))

async function loadBeats() {
  const id = monitorId.value
  const res = await api.get(`/monitors/${id}/beats`, { params: { period: 86400 } })
  allBeats.value = res.data || []
}
const pingChartOption = computed(() => ({
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
      data: pingChartData.value
        .filter((d) => d.up > 0)
        .map((d) => [new Date(d.timestamp * 1000), d.max_ping !== undefined ? Number(d.max_ping) : 0]),
      smooth: true,
      symbol: 'circle',
    },
    {
      name: '平均响应',
      type: 'line',
      data: pingChartData.value
        .filter((d) => d.up > 0)
        .map((d) => [new Date(d.timestamp * 1000), d.avg_ping !== undefined ? Number(d.avg_ping) : 0]),
      smooth: true,
      areaStyle: { opacity: 0.1 },
      symbol: 'circle',
    },
    {
      name: '最小响应',
      type: 'line',
      data: pingChartData.value
        .filter((d) => d.up > 0)
        .map((d) => [new Date(d.timestamp * 1000), d.min_ping !== undefined ? Number(d.min_ping) : 0]),
      smooth: true,
      symbol: 'circle',
    },
  ],
}))

const uptimeChartOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  grid: { left: 50, right: 20, top: 30, bottom: 30 },
  xAxis: {
    type: 'time',
  },
  yAxis: {
    type: 'value',
    min: 0,
    max: 1,
    axisLabel: { formatter: (v: number) => (v * 100).toFixed(0) + '%' },
  },
  series: [
    {
      name: '可用率',
      type: 'bar',
      data: uptimeChartData.value.map((d) => [
        new Date(d.timestamp * 1000),
        d.uptime !== undefined ? Number(d.uptime) : 0,
      ]),
      itemStyle: {
        color: (params: { value: [Date, number] }) => params.value[1] > 0.99 ? '#67C23A' : params.value[1] > 0.95 ? '#E6A23C' : '#F56C6C',
      },
    },
  ],
}))

function statusText(status: number): string {
  switch (status) {
    case 1: return 'UP'
    case 0: return 'DOWN'
    case 2: return 'PENDING'
    default: return 'UNKNOWN'
  }
}

function formatPing(ping: number | null): string {
  if (ping == null) return '-'
  return ping.toFixed(1) + ' ms'
}

onMounted(async () => {
  loading.value = true
  try {
    await store.fetchStatus()
    const id = monitorId.value

    const res = await api.get(`/monitors/${id}`)
    monitor.value = res.data.monitor

    const [pingRes, uptimeRes, incidentsRes, summaryRes] = await Promise.all([
      api.get(`/monitors/${id}/uptime/data`, { params: { granularity: 'hourly', num: 24 } }),
      api.get(`/monitors/${id}/uptime/data`, { params: { granularity: 'daily', num: 30 } }),
      api.get(`/monitors/${id}/incidents`),
      api.get(`/monitors/${id}/uptime/summary`),
    ])

    pingChartData.value = pingRes.data || []
    uptimeChartData.value = uptimeRes.data || []
    uptimeSummary.value = summaryRes.data || {}
    await loadBeats()
    unsubscribeWS = wsClient.onMessage((msg) => {
      if (msg.type !== 'heartbeat') return
      const beat = msg.payload as Heartbeat
      if (beat.monitor_id !== id) return
      allBeats.value = [...allBeats.value.filter((item) => item.id !== beat.id), beat].slice(-500)
    })
    const interval = monitor.value?.interval
    if (interval && interval > 0) {
      refreshTimer = setInterval(() => loadBeats(), interval * 1000)
    }
    incidentList.value = incidentsRes.data || []
  } catch {
    // 错误处理
  } finally {
    loading.value = false
  }

  await nextTick()
  if (beatsContainer.value?.$el) {
    containerWidth.value = beatsContainer.value.$el.clientWidth
    beatsObserver = new ResizeObserver(entries => {
      containerWidth.value = entries[0]?.contentRect?.width ?? containerWidth.value
    })
    beatsObserver.observe(beatsContainer.value.$el)
  }
})
</script>

<template>
  <div v-loading="loading">
    <template v-if="monitor">
      <el-page-header @back="$router.push('/monitors')" style="margin-bottom: 20px">
        <template #content>
          <span style="font-size: 18px">{{ monitor.name }}</span>
        </template>
      </el-page-header>

      <el-descriptions :column="descColumns" border style="margin-bottom: 24px">
        <el-descriptions-item label="名称">{{ monitor.name }}</el-descriptions-item>
        <el-descriptions-item label="类型">
          <el-tag size="small">{{ monitor.type.toUpperCase() }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="monitor.type === 'ping' ? '主机名' : 'URL / 主机名'">
          {{ monitor.url || monitor.hostname || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="检查间隔">
          {{ monitor.interval }}s
        </el-descriptions-item>
        <el-descriptions-item label="超时">
          {{ monitor.timeout }}s
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="store.statusColor(currentStatus)" size="small" effect="dark">
            {{ store.statusText(currentStatus) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item v-if="monitor.type === 'ping'" label="在线时间 (24 小时)">
          {{ uptimePercent(uptimeSummary.uptime_24h) }}
        </el-descriptions-item>
        <el-descriptions-item v-if="monitor.type === 'ping'" label="在线时间 (30 天)">
          {{ uptimePercent(uptimeSummary.uptime_30d) }}
        </el-descriptions-item>
        <el-descriptions-item v-if="monitor.type === 'ping'" label="在线时间 (1 年)">
          {{ uptimePercent(uptimeSummary.uptime_1y) }}
        </el-descriptions-item>
      </el-descriptions>

      <el-row :gutter="20" style="margin-bottom: 24px">
        <el-col :xs="24" :md="12">
          <el-card shadow="never">
            <template #header>
              <span>24小时响应时间</span>
            </template>
            <VChart v-if="pingChartData.length > 0" :option="pingChartOption" style="height: 280px" autoresize />
            <el-empty v-else description="暂无响应时间数据" :image-size="80" />
          </el-card>
        </el-col>
        <el-col :xs="24" :md="12">
          <el-card shadow="never">
            <template #header>
              <span>30天可用率</span>
            </template>
            <VChart v-if="uptimeChartData.length > 0" :option="uptimeChartOption" style="height: 280px" autoresize />
            <el-empty v-else description="暂无可用率数据" :image-size="80" />
          </el-card>
        </el-col>
      </el-row>

      <el-card shadow="never" style="margin-bottom: 24px">
        <template #header>
          <span>最近心跳记录（{{ beatCount }}格）</span>
        </template>
        <TransitionGroup ref="beatsContainer" name="beat" tag="div" v-if="heartbeatList.length > 0"
          style="display: flex; flex-wrap: nowrap; overflow: hidden; gap: 2px; padding: 8px 0; justify-content: flex-end; position: relative">
          <div
            v-for="(beat, i) in heartbeatList"
            :key="beat.id ?? i"
            :class="['beat-cell', { 'beat-pop': i === heartbeatList.length - 1 }]"
            :title="`${new Date(beat.time).toLocaleString('zh-CN')} · ${statusText(beat.status)} · ${formatPing(beat.ping_ms)}`"
            :style="{
              width: '12px', height: '18px', borderRadius: '2px',
              flexShrink: 0, cursor: 'pointer',
              backgroundColor: beat.status === 1 ? '#67C23A' : '#F56C6C',
            }"
          />
        </TransitionGroup>
        <el-empty v-else description="暂无心跳记录" :image-size="60" />
      </el-card>

      <el-card shadow="never">
        <template #header>
          <span>近期故障事件</span>
        </template>
        <el-table :data="incidentList" size="small" stripe>
          <el-table-column label="标题" min-width="180">
            <template #default="{ row }">
              {{ row.title }}
            </template>
          </el-table-column>
          <el-table-column label="开始时间" width="160">
            <template #default="{ row }">
              {{ new Date(row.started_at).toLocaleString('zh-CN') }}
            </template>
          </el-table-column>
          <el-table-column label="结束时间" width="160">
            <template #default="{ row }">
              {{ row.ended_at ? new Date(row.ended_at).toLocaleString('zh-CN') : '未结束' }}
            </template>
          </el-table-column>
          <el-table-column label="持续" width="100">
            <template #default="{ row }">
              {{ row.duration_seconds ? Math.floor(row.duration_seconds / 60) + 'm' : '-' }}
            </template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.status === 0 ? 'danger' : 'success'" size="small">
                {{ row.status === 0 ? 'DOWN' : '已恢复' }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="incidentList.length === 0" description="暂无故障事件" :image-size="60" />
      </el-card>
    </template>

    <el-empty v-else description="监控项不存在" />
  </div>
</template>

<style scoped>
.beat-move {
  transition: transform 0.35s ease;
}
.beat-enter-active {
  transition: all 0.35s ease;
}
.beat-leave-active {
  transition: all 0.35s ease;
  position: absolute !important;
}
.beat-enter-from {
  transform: translateX(14px);
  opacity: 0;
}
.beat-leave-to {
  transform: translateX(-14px) !important;
  opacity: 0;
}

.beat-cell {
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}
.beat-cell:hover {
  transform: scaleY(1.6);
  z-index: 10;
  box-shadow: 0 2px 8px rgba(0,0,0,0.25);
}

@keyframes beatPop {
  0% { transform: scale(0.8); box-shadow: 0 0 0 3px rgba(103,194,58,0.4); }
  50% { transform: scale(1.2); box-shadow: 0 0 0 8px rgba(103,194,58,0.15); }
  100% { transform: scale(1); box-shadow: 0 0 0 0 rgba(103,194,58,0); }
}
.beat-pop {
  animation: beatPop 0.35s ease-out;
}
</style>
