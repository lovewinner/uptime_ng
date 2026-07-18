<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import api from '@/api/http'
import { useMonitorStore } from '@/stores/monitor'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { MaintenanceWindow } from '@/api/types'

const store = useMonitorStore()
const windows = ref<MaintenanceWindow[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const editing = ref<MaintenanceWindow | null>(null)
const formRef = ref()
const saving = ref(false)
const form = reactive({
  name: '',
  description: '',
  monitor_id: null as number | null,
  start_at: '',
  end_at: '',
  active: true,
})

const title = computed(() => editing.value ? '编辑维护窗口' : '新增维护窗口')

async function fetchWindows() {
  loading.value = true
  try {
    const res = await api.get('/maintenance')
    windows.value = res.data || []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  form.name = ''
  form.description = ''
  form.monitor_id = null
  form.start_at = ''
  form.end_at = ''
  form.active = true
  dialogVisible.value = true
}

function openEdit(row: MaintenanceWindow) {
  editing.value = row
  form.name = row.name
  form.description = row.description || ''
  form.monitor_id = row.monitor_id
  form.start_at = row.start_at
  form.end_at = row.end_at
  form.active = row.active
  dialogVisible.value = true
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    const payload = {
      ...form,
      start_at: toRFC3339(form.start_at),
      end_at: toRFC3339(form.end_at),
    }
    if (editing.value) {
      await api.put(`/maintenance/${editing.value.id}`, payload)
    } else {
      await api.post('/maintenance', payload)
    }
    dialogVisible.value = false
    ElMessage.success('已保存')
    await fetchWindows()
  } catch (e: unknown) {
    ElMessage.error(errorMessage(e, '保存失败'))
  } finally {
    saving.value = false
  }
}

async function remove(row: MaintenanceWindow) {
  try {
    await ElMessageBox.confirm(`确定删除维护窗口 "${row.name}" 吗？`, '确认删除', { type: 'warning' })
    await api.delete(`/maintenance/${row.id}`)
    ElMessage.success('已删除')
    await fetchWindows()
  } catch {
    // cancelled
  }
}

function monitorName(id: number | null) {
  if (!id) return '全部监控'
  return store.monitors.find((m) => m.id === id)?.name || `#${id}`
}

function toRFC3339(value: string) {
  return new Date(value).toISOString()
}

function errorMessage(e: unknown, fallback: string): string {
  if (typeof e === 'object' && e !== null && 'response' in e) {
    const response = (e as { response?: { data?: { error?: string } } }).response
    return response?.data?.error || fallback
  }
  return fallback
}

onMounted(async () => {
  await Promise.all([store.fetchMonitors(), fetchWindows()])
})
</script>

<template>
  <div>
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:20px">
      <h2>维护窗口</h2>
      <el-button type="primary" @click="openCreate">新增维护窗口</el-button>
    </div>

    <el-table :data="windows" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column label="对象" min-width="160">
        <template #default="{ row }">{{ monitorName(row.monitor_id) }}</template>
      </el-table-column>
      <el-table-column label="开始" width="190">
        <template #default="{ row }">{{ new Date(row.start_at).toLocaleString('zh-CN') }}</template>
      </el-table-column>
      <el-table-column label="结束" width="190">
        <template #default="{ row }">{{ new Date(row.end_at).toLocaleString('zh-CN') }}</template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.active ? 'success' : 'info'" size="small">{{ row.active ? '启用' : '停用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="!loading && windows.length === 0" description="暂无维护窗口" />

    <el-dialog v-model="dialogVisible" :title="title" width="min(620px, 95%)">
      <el-form ref="formRef" :model="form" label-width="100px">
        <el-form-item label="名称" prop="name" :rules="[{ required: true, message: '请输入名称' }]">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="对象">
          <el-select v-model="form.monitor_id" clearable placeholder="全部监控" style="width:100%">
            <el-option v-for="m in store.monitors" :key="m.id" :label="m.name" :value="m.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="开始" prop="start_at" :rules="[{ required: true, message: '请选择开始时间' }]">
          <el-date-picker v-model="form.start_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ss" style="width:100%" />
        </el-form-item>
        <el-form-item label="结束" prop="end_at" :rules="[{ required: true, message: '请选择结束时间' }]">
          <el-date-picker v-model="form.end_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ss" style="width:100%" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.active" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>
