<script setup lang="ts">
import { onMounted, ref, computed, onUnmounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import api from '@/api/http'
import { arrayFromResponse, objectFromResponse } from '@/api/responses'
import { useMonitorStore } from '@/stores/monitor'
import { monitorFromResponse } from '@/stores/monitorHelpers'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { wsClient } from '@/api/ws'
import type { Heartbeat, Incident, UptimeDataPoint, UptimeSummary } from '@/api/types'
import type { Monitor } from '@/stores/monitor'
import {
  buildPingChartOption,
  buildUptimeChartOption,
  detailColumnCount,
  heartbeatColor,
  heartbeatTitle,
  incidentDuration,
  incidentEndTimeText,
  incidentStatusText,
  incidentStatusType,
  incidentTimeText,
  monitorTargetLabel,
  monitorTargetText,
  uptimePercent,
  visibleBeatCount,
} from './monitorDetail'
import { monitorTypeText } from './formatters'

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

const descColumns = computed(() => detailColumnCount(windowWidth.value))

const monitor = ref<Monitor | null>(null)
const currentStatus = computed(() => {
  if (!monitor.value) return 2
  return store.statusByID(monitor.value.id)?.status ?? 2
})
const childMonitors = computed(() => {
  if (!monitor.value) return []
  return store.monitors.filter((item) => item.group_id === monitor.value?.id)
})
const incidentList = ref<Incident[]>([])
const pingChartData = ref<UptimeDataPoint[]>([])
const uptimeChartData = ref<UptimeDataPoint[]>([])
const loading = ref(false)
const defaultUptimeSummary = (): UptimeSummary => ({ uptime_24h: 0, uptime_30d: 0, uptime_1y: 0 })
const uptimeSummary = ref<UptimeSummary>(defaultUptimeSummary())

const beatsContainer = ref<{ $el?: HTMLElement } | null>(null)
const containerWidth = ref(0)
const allBeats = ref<Heartbeat[]>([])
const beatCount = computed(() => visibleBeatCount(containerWidth.value))
const heartbeatList = computed(() => allBeats.value.slice(-beatCount.value))

async function loadBeats() {
  const id = monitorId.value
  const res = await api.get(`/monitors/${id}/beats`, { params: { period: 86400 } })
  allBeats.value = arrayFromResponse<Heartbeat>(res.data)
}
const pingChartOption = computed(() => buildPingChartOption(pingChartData.value))
const uptimeChartOption = computed(() => buildUptimeChartOption(uptimeChartData.value))

onMounted(async () => {
  loading.value = true
  try {
    await store.fetchStatus()
    await store.fetchMonitors()
    const id = monitorId.value

    const res = await api.get(`/monitors/${id}`)
    monitor.value = monitorFromResponse(res.data)

    if (monitor.value?.type === 'group') {
      return
    }

    const [pingRes, uptimeRes, incidentsRes, summaryRes] = await Promise.all([
      api.get(`/monitors/${id}/uptime/data`, { params: { granularity: 'hourly', num: 24 } }),
      api.get(`/monitors/${id}/uptime/data`, { params: { granularity: 'daily', num: 30 } }),
      api.get(`/monitors/${id}/incidents`),
      api.get(`/monitors/${id}/uptime/summary`),
    ])

    pingChartData.value = arrayFromResponse<UptimeDataPoint>(pingRes.data)
    uptimeChartData.value = arrayFromResponse<UptimeDataPoint>(uptimeRes.data)
    uptimeSummary.value = objectFromResponse<UptimeSummary>(summaryRes.data, defaultUptimeSummary())
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
    incidentList.value = arrayFromResponse<Incident>(incidentsRes.data)
  } catch {
    // 错误处理
  } finally {
    loading.value = false
  }

  await nextTick()
  if (monitor.value?.type === 'group') return
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
          <el-tag size="small">{{ monitorTypeText(monitor.type) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item v-if="monitor.type !== 'group'" :label="monitorTargetLabel(monitor.type)">
          {{ monitorTargetText(monitor) }}
        </el-descriptions-item>
        <el-descriptions-item v-if="monitor.type !== 'group'" label="检查间隔">
          {{ monitor.interval }}s
        </el-descriptions-item>
        <el-descriptions-item v-if="monitor.type !== 'group'" label="超时">
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

      <el-card v-if="monitor.type === 'group'" shadow="never">
        <template #header>
          <span>子监控</span>
        </template>
        <el-table :data="childMonitors" size="small" stripe>
          <el-table-column prop="name" label="名称" min-width="160" />
          <el-table-column prop="type" label="类型" width="160">
            <template #default="{ row }">
              <el-tag size="small">{{ monitorTypeText(row.type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="store.statusColor(store.statusByID(row.id)?.status ?? 2)" size="small" effect="dark">
                {{ store.statusText(store.statusByID(row.id)?.status ?? 2) }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="childMonitors.length === 0" description="暂无子监控" :image-size="60" />
      </el-card>

      <template v-else>
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
            :title="heartbeatTitle(beat)"
            :style="{
              width: '12px', height: '18px', borderRadius: '2px',
              flexShrink: 0, cursor: 'pointer',
              backgroundColor: heartbeatColor(beat.status),
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
              {{ incidentTimeText(row.started_at) }}
            </template>
          </el-table-column>
          <el-table-column label="结束时间" width="160">
            <template #default="{ row }">
              {{ incidentEndTimeText(row.ended_at) }}
            </template>
          </el-table-column>
          <el-table-column label="持续" width="100">
            <template #default="{ row }">
              {{ incidentDuration(row.duration_seconds) }}
            </template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="incidentStatusType(row.status)" size="small">
                {{ incidentStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="incidentList.length === 0" description="暂无故障事件" :image-size="60" />
      </el-card>
      </template>
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
