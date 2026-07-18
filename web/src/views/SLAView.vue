<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import api from '@/api/http'
import { useMonitorStore } from '@/stores/monitor'
import { useRouter } from 'vue-router'
import { formatDowntime, uptimeClass, uptimePercent, type SLAItem } from './sla'

const store = useMonitorStore()
const router = useRouter()
const loading = ref(false)
const period = ref('month')
const slaData = ref<SLAItem[]>([])

const periodOptions = [
  { label: '今天', value: 'day' },
  { label: '本周', value: 'week' },
  { label: '本月', value: 'month' },
  { label: '本季度', value: 'quarter' },
  { label: '本年', value: 'year' },
]

function goDetail(id: number) {
  router.push(`/monitors/${id}`)
}

async function fetchSLA() {
  loading.value = true
  try {
    const res = await api.get('/monitors/uptime/overall', {
      params: { period: period.value },
    })
    slaData.value = res.data || []
  } catch {
    slaData.value = []
  } finally {
    loading.value = false
  }
}

function handlePeriodChange(val: string) {
  period.value = val
  fetchSLA()
}

onMounted(async () => {
  await store.fetchMonitors()
  await fetchSLA()
})
</script>

<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 10px; margin-bottom: 20px">
      <h2>SLA 报表</h2>
      <el-radio-group v-model="period" size="small" @change="handlePeriodChange" style="flex-wrap: wrap">
        <el-radio-button v-for="opt in periodOptions" :key="opt.value" :value="opt.value">
          {{ opt.label }}
        </el-radio-button>
      </el-radio-group>
    </div>

    <el-table :data="slaData" v-loading="loading" stripe style="width: 100%">
      <el-table-column label="监控项" min-width="180">
        <template #default="{ row }">
          <el-link type="primary" :underline="false" @click="goDetail(row.monitor_id)">
            {{ row.monitor_name }}
          </el-link>
        </template>
      </el-table-column>
      <el-table-column label="类型" width="130">
        <template #default="{ row }">
          <el-tag size="small">{{ row.monitor_type.toUpperCase() }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="可用率" width="120" align="center">
        <template #default="{ row }">
          <span :class="uptimeClass(row.uptime_percentage)" style="font-weight: 600">
            {{ uptimePercent(row.uptime_percentage) }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="总检查" width="100" align="center">
        <template #default="{ row }">
          {{ row.total_checks }}
        </template>
      </el-table-column>
      <el-table-column label="失败" width="80" align="center">
        <template #default="{ row }">
          <span :style="{ color: row.failed_checks > 0 ? '#F56C6C' : '#67C23A' }">
            {{ row.failed_checks }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="平均延迟" width="110" align="center">
        <template #default="{ row }">
          {{ row.avg_ping_ms ? row.avg_ping_ms.toFixed(1) + ' ms' : '-' }}
        </template>
      </el-table-column>
      <el-table-column label="故障次数" width="100" align="center">
        <template #default="{ row }">
          <el-tag :type="row.incidents > 0 ? 'danger' : 'success'" size="small">
            {{ row.incidents }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="累计宕机" width="120" align="center">
        <template #default="{ row }">
          {{ formatDowntime(row.total_downtime_seconds) }}
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="!loading && slaData.length === 0" description="暂无SLA数据" />
  </div>
</template>

<style scoped>
.uptime-green {
  color: #67C23A;
}
.uptime-yellow {
  color: #E6A23C;
}
.uptime-red {
  color: #F56C6C;
}
</style>
