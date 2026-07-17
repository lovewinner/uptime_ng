<script setup lang="ts">
import { onMounted, ref, computed, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import api from '@/api/http'
import { useMonitorStore } from '@/stores/monitor'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'

use([CanvasRenderer, LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent])

const route = useRoute()
const store = useMonitorStore()
const monitorId = computed(() => Number(route.params.id))

const windowWidth = ref(window.innerWidth)

function onResize() {
  windowWidth.value = window.innerWidth
}
onMounted(() => window.addEventListener('resize', onResize))
onUnmounted(() => window.removeEventListener('resize', onResize))

const descColumns = computed(() => windowWidth.value < 640 ? 1 : windowWidth.value < 1024 ? 2 : 3)

const monitor = ref<any>(null)
const heartbeatList = ref<any[]>([])
const incidentList = ref<any[]>([])
const pingChartData = ref<any[]>([])
const uptimeChartData = ref<any[]>([])
const loading = ref(false)

const pingChartOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: ['平均响应 (ms)'] },
  grid: { left: 50, right: 20, top: 30, bottom: 30 },
  xAxis: {
    type: 'time',
  },
  yAxis: {
    type: 'value',
    name: 'ms',
  },
  series: [
    {
      name: '平均响应 (ms)',
      type: 'line',
      data: pingChartData.value
        .filter((d: any) => d.up > 0)
        .map((d: any) => [new Date(d.timestamp * 1000), d.avg_ping !== undefined ? Number(d.avg_ping) : 0]),
      smooth: true,
      areaStyle: { opacity: 0.1 },
    },
  ],
}))

const uptimeChartOption = computed(() => ({
  tooltip: { trigger: 'axis', formatter: (p: any) => `${p[0].name}<br/>可用率: ${(p[0].value[1] * 100).toFixed(2)}%` },
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
      data: uptimeChartData.value.map((d: any) => [
        new Date(d.timestamp * 1000),
        d.uptime !== undefined ? Number(d.uptime) : 0,
      ]),
      itemStyle: {
        color: (params: any) => params.value[1] > 0.99 ? '#67C23A' : params.value[1] > 0.95 ? '#E6A23C' : '#F56C6C',
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

function statusTagType(status: number): string {
  switch (status) {
    case 1: return 'success'
    case 0: return 'danger'
    case 2: return 'warning'
    default: return 'info'
  }
}

function formatDate(ts: number): string {
  return new Date(ts * 1000).toLocaleString('zh-CN', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
  })
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

    const [pingRes, uptimeRes, beatsRes, incidentsRes] = await Promise.all([
      api.get(`/monitors/${id}/uptime/data`, { params: { granularity: 'hourly', num: 24 } }),
      api.get(`/monitors/${id}/uptime/data`, { params: { granularity: 'daily', num: 30 } }),
      api.get(`/monitors/${id}/beats`, { params: { period: 86400 } }),
      api.get(`/monitors/${id}/incidents`),
    ])

    pingChartData.value = pingRes.data || []
    uptimeChartData.value = uptimeRes.data || []
    heartbeatList.value = (beatsRes.data || []).slice(-50).reverse()
    incidentList.value = incidentsRes.data || []
  } catch {
    // 错误处理
  } finally {
    loading.value = false
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
          <el-tag :type="store.statusColor(
            store.statusList.find(s => s.id === monitor.id)?.status ?? 2
          )" size="small" effect="dark">
            {{ store.statusText(
              store.statusList.find(s => s.id === monitor.id)?.status ?? 2
            ) }}
          </el-tag>
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
          <span>最近心跳记录（最多50条）</span>
        </template>
        <el-table :data="heartbeatList" size="small" stripe max-height="400">
          <el-table-column label="时间" width="160">
            <template #default="{ row }">
              {{ new Date(row.time).toLocaleString('zh-CN') }}
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="statusTagType(row.status)" size="small" effect="dark">
                {{ statusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="延迟" width="100">
            <template #default="{ row }">
              {{ formatPing(row.ping_ms) }}
            </template>
          </el-table-column>
          <el-table-column label="HTTP状态码" width="100">
            <template #default="{ row }">
              {{ row.http_status || '-' }}
            </template>
          </el-table-column>
          <el-table-column label="消息" min-width="200">
            <template #default="{ row }">
              <span style="font-size: 12px">{{ row.msg || '-' }}</span>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="heartbeatList.length === 0" description="暂无心跳记录" :image-size="60" />
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