<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useMonitorStore } from '@/stores/monitor'
import { useRouter } from 'vue-router'

const store = useMonitorStore()
const router = useRouter()

const realStatuses = computed(() => store.statusList.filter((s) => s.type !== 'group'))
const groupStatuses = computed(() => store.statusList.filter((s) => s.type === 'group'))
const totalCount = computed(() => realStatuses.value.length)
const groupCount = computed(() => groupStatuses.value.length)
const upCount = computed(() => realStatuses.value.filter((s) => s.status === 1).length)
const downCount = computed(() => realStatuses.value.filter((s) => s.status === 0).length)
const pendingCount = computed(() => realStatuses.value.filter((s) => s.status === 2).length)
const currentFaults = computed(() => realStatuses.value.filter((s) => s.status === 0 || s.status === 2))
const avgPing = computed(() => {
  const values = realStatuses.value.map((s) => s.ping_ms).filter((v) => v > 0)
  if (values.length === 0) return 0
  return values.reduce((sum, v) => sum + v, 0) / values.length
})

onMounted(async () => {
  await store.fetchStatus()
})

function goMonitor(id: number) {
  router.push(`/monitors/${id}`)
}

function uptimePercent(v: number) {
  return (v * 100).toFixed(2) + '%'
}
</script>

<template>
  <div>
    <h2>仪表盘</h2>

    <div style="margin-bottom: 20px; display: flex; gap: 15px; flex-wrap: wrap; align-items: center">
      <el-statistic title="监控项总数" :value="totalCount" />
      <el-statistic title="分组数" :value="groupCount" />
      <el-statistic title="UP" :value="upCount" />
      <el-statistic title="DOWN" :value="downCount" />
      <el-statistic title="PENDING" :value="pendingCount" />
      <el-statistic title="平均响应(ms)" :value="Number(avgPing.toFixed(0))" />
      <el-button type="primary" @click="router.push('/monitors')" style="margin-left: auto">管理监控项</el-button>
    </div>

    <el-row :gutter="16">
      <el-col v-for="s in realStatuses" :key="s.id" :xs="24" :sm="12" :md="8" :lg="6" style="margin-bottom: 16px">
        <el-card shadow="hover" style="cursor: pointer" @click="goMonitor(s.id)">
          <div style="display: flex; justify-content: space-between; align-items: center">
            <div>
              <h4 style="margin: 0 0 8px 0">{{ s.name }}</h4>
              <el-tag size="small">{{ s.type.toUpperCase() }}</el-tag>
            </div>
            <div style="text-align: right">
              <el-tag :type="store.statusColor(s.status)" size="large" effect="dark">
                {{ store.statusText(s.status) }}
              </el-tag>
              <div style="margin-top: 8px; font-size: 13px; color: #666">
                {{ s.ping_ms ? s.ping_ms.toFixed(0) + ' ms' : '-' }}
              </div>
              <div style="font-size: 12px; color: #999">
                24h: {{ uptimePercent(s.uptime_24h) }}
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card v-if="currentFaults.length > 0" shadow="never" style="margin-top: 8px">
      <template #header>
        <span>当前故障</span>
      </template>
      <el-table :data="currentFaults" size="small" stripe>
        <el-table-column prop="name" label="监控项" min-width="160" />
        <el-table-column prop="type" label="类型" width="90">
          <template #default="{ row }">
            <el-tag size="small">{{ row.type.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="store.statusColor(row.status)" size="small" effect="dark">
              {{ store.statusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="响应" width="110">
          <template #default="{ row }">
            {{ row.ping_ms ? row.ping_ms.toFixed(0) + ' ms' : '-' }}
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-empty v-if="realStatuses.length === 0" description="暂无监控项，请先创建" />
  </div>
</template>
