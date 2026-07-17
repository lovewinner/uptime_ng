<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useMonitorStore } from '@/stores/monitor'
import { useRouter } from 'vue-router'

const store = useMonitorStore()
const router = useRouter()

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

    <div style="margin-bottom: 20px; display: flex; gap: 15px">
      <el-statistic-card value="待加载" title="监控项总数" />
      <el-button type="primary" @click="router.push('/monitors')" style="margin-left: auto">管理监控项</el-button>
    </div>

    <el-row :gutter="16">
      <el-col v-for="s in store.statusList" :key="s.id" :span="8" style="margin-bottom: 16px">
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

    <el-empty v-if="store.statusList.length === 0" description="暂无监控项，请先创建" />
  </div>
</template>