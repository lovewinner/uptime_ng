<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useMonitorStore } from '@/stores/monitor'
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { ElMessage } from 'element-plus'
import CreateMonitorDialog from '@/components/CreateMonitorDialog.vue'
import ExportDialog from '@/components/ExportDialog.vue'
import ImportDialog from '@/components/ImportDialog.vue'
import api from '@/api/http'

const store = useMonitorStore()
const router = useRouter()

const dialogVisible = ref(false)
const editingMonitor = ref<any>(null)
const exportVisible = ref(false)
const importVisible = ref(false)

function handleCreate() {
  editingMonitor.value = null
  dialogVisible.value = true
}

function handleEdit(monitor: any) {
  editingMonitor.value = { ...monitor }
  dialogVisible.value = true
}

function handleSaved() {
  dialogVisible.value = false
  ElMessage.success('保存成功')
}

async function handleDelete(monitor: any) {
  try {
    await ElMessageBox.confirm(`确定要删除监控项 "${monitor.name}" 吗？`, '确认删除', {
      type: 'warning',
      confirmButtonText: '删除',
    })
    await store.deleteMonitor(monitor.id)
    ElMessage.success('已删除')
  } catch {
    // cancelled
  }
}

async function handlePauseResume(monitor: any) {
  if (monitor.active) {
    await store.pauseMonitor(monitor.id)
    ElMessage.success('已暂停')
  } else {
    await store.resumeMonitor(monitor.id)
    ElMessage.success('已恢复')
  }
}

function goDetail(id: number) {
  router.push(`/monitors/${id}`)
}

function getUrl(monitor: any): string {
  if (monitor.url) return monitor.url
  if (monitor.hostname && monitor.port) return `${monitor.hostname}:${monitor.port}`
  if (monitor.hostname) return monitor.hostname
  return '-'
}

function getIntervalText(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  return `${Math.floor(seconds / 60)}m`
}

onMounted(async () => {
  await store.fetchMonitors()
  await store.fetchStatus()
})

async function handleExport(ids?: number[]) {
  try {
    let url = '/monitors/export'
    if (ids && ids.length > 0) {
      url += '?ids=' + JSON.stringify(ids)
    }
    const res = await api.get(url, { responseType: 'blob' })
    const blob = new Blob([JSON.stringify(res.data, null, 2)], { type: 'application/json' })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = 'uptime_ng_export.json'
    a.click()
    ElMessage.success('导出成功')
  } catch {
    ElMessage.error('导出失败')
  }
}

function handleImported() {
  store.fetchMonitors()
  store.fetchStatus()
}
</script>

<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px">
      <h2>监控列表</h2>
      <div style="display: flex; gap: 10px">
        <el-button @click="exportVisible = true">导出</el-button>
        <el-button @click="importVisible = true">导入</el-button>
        <el-button type="primary" @click="handleCreate">新增监控</el-button>
      </div>
    </div>

    <el-table
      :data="store.monitors"
      v-loading="store.loading"
      stripe
      style="width: 100%"
      @row-click="goDetail"
    >
      <el-table-column prop="name" label="名称" min-width="150" />
      <el-table-column prop="type" label="类型" width="80">
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
            store.statusList.find(s => s.id === row.id)?.status ?? 2
          )" size="small" effect="dark">
            {{ store.statusText(
              store.statusList.find(s => s.id === row.id)?.status ?? 2
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