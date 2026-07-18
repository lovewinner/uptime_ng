<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed, watch } from 'vue'
import { useMonitorStore } from '@/stores/monitor'
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { ElMessage } from 'element-plus'
import CreateMonitorDialog from '@/components/CreateMonitorDialog.vue'
import ExportDialog from '@/components/ExportDialog.vue'
import ImportDialog from '@/components/ImportDialog.vue'
import api from '@/api/http'
import type { Monitor, MonitorTreeNode } from '@/stores/monitor'
import {
  exportURL,
  intervalText,
  monitorTargetText,
  nextExpandedIds,
  visibleMonitorRows,
} from './monitorList'

const store = useMonitorStore()
const router = useRouter()

const dialogVisible = ref(false)
const editingMonitor = ref<Monitor | null>(null)
const exportVisible = ref(false)
const importVisible = ref(false)
const treeRows = computed(() => store.buildMonitorTree())
const expandedIds = ref(new Set<number>())
const refreshTimers = new Map<number, { timer: ReturnType<typeof setInterval>, interval: number }>()
const visibleRows = computed(() => visibleMonitorRows(treeRows.value, expandedIds.value))

function handleCreate() {
  editingMonitor.value = null
  dialogVisible.value = true
}

function handleEdit(monitor: Monitor) {
  editingMonitor.value = { ...monitor }
  dialogVisible.value = true
}

function handleSaved() {
  dialogVisible.value = false
  syncVisibleRefresh()
  ElMessage.success('保存成功')
}

async function handleDelete(monitor: Monitor) {
  try {
    await ElMessageBox.confirm(`确定要删除监控项 "${monitor.name}" 吗？`, '确认删除', {
      type: 'warning',
      confirmButtonText: '删除',
    })
    await store.deleteMonitor(monitor.id)
    syncVisibleRefresh()
    ElMessage.success('已删除')
  } catch {
    // cancelled
  }
}

async function handlePauseResume(monitor: Monitor) {
  if (monitor.active) {
    await store.pauseMonitor(monitor.id)
    syncVisibleRefresh()
    ElMessage.success('已暂停')
  } else {
    await store.resumeMonitor(monitor.id)
    syncVisibleRefresh()
    ElMessage.success('已恢复')
  }
}

function goDetail(id: number) {
  router.push(`/monitors/${id}`)
}

function getUrl(monitor: MonitorTreeNode): string {
  return monitorTargetText(monitor)
}

function getIntervalText(seconds: number): string {
  return intervalText(seconds)
}

onMounted(async () => {
  await Promise.all([
    store.fetchMonitors(),
    store.fetchNotifications(),
  ])
  syncVisibleRefresh()
})

onUnmounted(() => {
  refreshTimers.forEach(({ timer }) => clearInterval(timer))
  refreshTimers.clear()
})

watch(visibleRows, () => syncVisibleRefresh())

async function handleExport(ids?: number[]) {
  try {
    const res = await api.get(exportURL(ids), { responseType: 'blob' })
    const blob = new Blob([res.data], { type: 'application/json' })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = 'uptime_ng_export.json'
    a.click()
    ElMessage.success('导出成功')
  } catch {
    ElMessage.error('导出失败')
  }
}

async function handleImported() {
  await store.fetchMonitors()
  syncVisibleRefresh()
}

function handleExpandChange(row: MonitorTreeNode, expanded: boolean) {
  expandedIds.value = nextExpandedIds(row, expanded, expandedIds.value)
}

function syncVisibleRefresh() {
  const visibleIDs = new Set(visibleRows.value.map((row) => row.id))
  refreshTimers.forEach(({ timer }, id) => {
    if (!visibleIDs.has(id)) {
      clearInterval(timer)
      refreshTimers.delete(id)
    }
  })
  visibleRows.value.forEach((row) => {
    const interval = Math.max(3, row.interval || 60)
    const existing = refreshTimers.get(row.id)
    if (existing && existing.interval === interval) return
    if (existing) {
      clearInterval(existing.timer)
    }
    refreshRowStatus(row)
    refreshTimers.set(row.id, {
      interval,
      timer: setInterval(() => refreshRowStatus(row), interval * 1000),
    })
  })
}

async function refreshRowStatus(row: MonitorTreeNode) {
  try {
    await store.fetchMonitorStatus(row.id)
  } catch {
    // ignore row refresh errors
  }
}
</script>

<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 10px; margin-bottom: 20px">
      <h2>监控列表</h2>
      <div style="display: flex; gap: 8px; flex-wrap: wrap">
        <el-button @click="exportVisible = true">导出</el-button>
        <el-button @click="importVisible = true">导入</el-button>
        <el-button type="primary" @click="handleCreate">新增监控</el-button>
      </div>
    </div>

    <el-table
      :data="treeRows"
      v-loading="store.loading"
      stripe
      row-key="id"
      :tree-props="{ children: 'children' }"
      style="width: 100%"
      @row-click="(row: Monitor) => goDetail(row.id)"
      @expand-change="handleExpandChange"
    >
      <el-table-column label="名称" min-width="150">
        <template #default="{ row }">
          <el-link type="primary" :underline="false" @click.stop="goDetail(row.id)">
            {{ row.name }}
          </el-link>
        </template>
      </el-table-column>
      <el-table-column prop="type" label="类型" width="130">
        <template #default="{ row }">
          <el-tag size="small">{{ row.type.toUpperCase() }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="URL / 主机名" min-width="200">
        <template #default="{ row }">
          {{ getUrl(row) }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="store.statusColor(
            store.statusByID(row.id)?.status ?? 2
          )" size="small" effect="dark">
            {{ store.statusText(
              store.statusByID(row.id)?.status ?? 2
            ) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="间隔" width="80">
        <template #default="{ row }">
          {{ getIntervalText(row.interval) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click.stop="handleEdit(row)">编辑</el-button>
          <el-button
            size="small"
            :type="row.active ? 'warning' : 'success'"
            @click.stop="handlePauseResume(row)"
          >
            {{ row.active ? '暂停' : '恢复' }}
          </el-button>
          <el-button size="small" type="danger" @click.stop="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="!store.loading && store.monitors.length === 0" description="暂无监控项" />

    <CreateMonitorDialog
      v-model="dialogVisible"
      :monitor="editingMonitor"
      @saved="handleSaved"
    />

    <ExportDialog
      v-model="exportVisible"
      :monitors="store.monitors"
      @export="handleExport"
    />

    <ImportDialog
      v-model="importVisible"
      @imported="handleImported"
    />
  </div>
</template>
