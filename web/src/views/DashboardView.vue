<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useMonitorStore } from '@/stores/monitor'
import { useRouter } from 'vue-router'
import { averagePingValue, dashboardSummary, pingText, uptimePercent } from './dashboard'
import { monitorTypeText } from './formatters'

const store = useMonitorStore()
const router = useRouter()

const summary = computed(() => dashboardSummary(store.statusList))

onMounted(async () => {
  await store.fetchStatus()
})

function goMonitor(id: number) {
  router.push(`/monitors/${id}`)
}
</script>

<template>
  <div>
    <h2>仪表盘</h2>

    <div style="margin-bottom: 20px; display: flex; gap: 15px; flex-wrap: wrap; align-items: center">
      <el-statistic title="监控项总数" :value="summary.totalCount" />
      <el-statistic title="分组数" :value="summary.groupCount" />
      <el-statistic title="UP" :value="summary.upCount" />
      <el-statistic title="DOWN" :value="summary.downCount" />
      <el-statistic title="PENDING" :value="summary.pendingCount" />
      <el-statistic title="平均响应(ms)" :value="averagePingValue(summary.avgPing)" />
      <el-button type="primary" @click="router.push('/monitors')" style="margin-left: auto">管理监控项</el-button>
    </div>

    <el-row :gutter="16">
      <el-col v-for="s in summary.realStatuses" :key="s.id" :xs="24" :sm="12" :md="8" :lg="6" style="margin-bottom: 16px">
        <el-card shadow="hover" style="cursor: pointer" @click="goMonitor(s.id)">
          <div style="display: flex; justify-content: space-between; align-items: center">
            <div>
              <h4 style="margin: 0 0 8px 0">{{ s.name }}</h4>
              <el-tag size="small">{{ monitorTypeText(s.type) }}</el-tag>
            </div>
            <div style="text-align: right">
              <el-tag :type="store.statusColor(s.status)" size="large" effect="dark">
                {{ store.statusText(s.status) }}
              </el-tag>
              <div style="margin-top: 8px; font-size: 13px; color: #666">
                {{ pingText(s.ping_ms) }}
              </div>
              <div style="font-size: 12px; color: #999">
                24h: {{ uptimePercent(s.uptime_24h) }}
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card v-if="summary.currentFaults.length > 0" shadow="never" style="margin-top: 8px">
      <template #header>
        <span>当前故障</span>
      </template>
      <el-table :data="summary.currentFaults" size="small" stripe>
        <el-table-column prop="name" label="监控项" min-width="160" />
        <el-table-column prop="type" label="类型" width="90">
          <template #default="{ row }">
            <el-tag size="small">{{ monitorTypeText(row.type) }}</el-tag>
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
            {{ pingText(row.ping_ms) }}
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-empty v-if="summary.realStatuses.length === 0" description="暂无监控项，请先创建" />
  </div>
</template>
