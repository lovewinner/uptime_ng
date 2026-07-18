<script setup lang="ts">
import { onMounted, ref, reactive, computed } from 'vue'
import api from '@/api/http'
import { apiErrorMessage } from '@/api/errors'
import { arrayFromResponse } from '@/api/responses'
import { ElMessageBox } from 'element-plus'
import { ElMessage } from 'element-plus'
import type { Notification } from '@/api/types'
import {
  defaultNotificationForm,
  notificationActiveTagType,
  notificationActiveText,
  notificationConfigHint,
  notificationConfigText,
  notificationDialogTitle,
  notificationFormFromNotification,
  notificationPayloadFromForm,
  notificationSavedText,
  notificationSubmitText,
  notificationTypeLabel,
  notificationTypeOptions,
  type NotificationForm,
} from './notification'

const notifications = ref<Notification[]>([])
const loading = ref(false)

const dialogVisible = ref(false)
const editingNotif = ref<Notification | null>(null)
const formRef = ref()
const saving = ref(false)

const form = reactive<NotificationForm>(defaultNotificationForm())

const configHint = computed(() => notificationConfigHint(form.type))

const isEdit = computed(() => !!editingNotif.value?.id)
const dialogTitle = computed(() => notificationDialogTitle(isEdit.value))
const submitText = computed(() => notificationSubmitText(isEdit.value))

async function fetchNotifications() {
  loading.value = true
  try {
    const res = await api.get('/notifications')
    notifications.value = arrayFromResponse<Notification>(res.data)
  } catch {
    // ignore
  } finally {
    loading.value = false
  }
}

function handleCreate() {
  editingNotif.value = null
  Object.assign(form, defaultNotificationForm())
  dialogVisible.value = true
}

function handleEdit(notif: Notification) {
  editingNotif.value = { ...notif }
  Object.assign(form, notificationFormFromNotification(notif))
  dialogVisible.value = true
}

async function handleDelete(notif: Notification) {
  try {
    await ElMessageBox.confirm(
      `确定要删除通知 "${notif.name}" 吗？相关监控项会失去此通知。`,
      '确认删除',
      { type: 'warning', confirmButtonText: '删除' },
    )
    await api.delete(`/notifications/${notif.id}`)
    ElMessage.success('已删除')
    await fetchNotifications()
  } catch {
    // cancelled
  }
}

async function handleTest(notif: Notification) {
  try {
    const res = await api.post(`/notifications/${notif.id}/test`)
    ElMessage.success(res.data?.message || '测试消息已发送')
  } catch (e: unknown) {
    ElMessage.error(apiErrorMessage(e, '测试失败'))
  }
}

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    if (isEdit.value) {
      await api.put(`/notifications/${editingNotif.value!.id}`, notificationPayloadFromForm(form))
    } else {
      await api.post('/notifications', notificationPayloadFromForm(form))
    }
    dialogVisible.value = false
    ElMessage.success(notificationSavedText(isEdit.value))
    await fetchNotifications()
  } catch (e: unknown) {
    ElMessage.error(apiErrorMessage(e, '保存失败'))
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  fetchNotifications()
})
</script>

<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 10px; margin-bottom: 20px">
      <h2>通知管理</h2>
      <el-button type="primary" @click="handleCreate">新增通知</el-button>
    </div>

    <el-table :data="notifications" v-loading="loading" stripe style="width: 100%">
      <el-table-column prop="name" label="名称" min-width="150" />
      <el-table-column prop="type" label="类型" width="100">
        <template #default="{ row }">
          <el-tag size="small">{{ notificationTypeLabel(row.type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="active" label="启用" width="80">
        <template #default="{ row }">
          <el-tag :type="notificationActiveTagType(row.active)" size="small">
            {{ notificationActiveText(row.active) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="配置" min-width="200">
        <template #default="{ row }">
          <span style="font-size: 12px; color: #666; word-break: break-all">
            {{ notificationConfigText(row.config) }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="handleEdit(row)">编辑</el-button>
          <el-button size="small" type="warning" @click="handleTest(row)">测试</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="!loading && notifications.length === 0" description="暂无通知配置" />

    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="min(500px, 95%)"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" label-width="80px" label-position="right">
        <el-form-item label="名称" prop="name" :rules="[{ required: true, message: '请输入名称' }]">
          <el-input v-model="form.name" placeholder="通知名称" />
        </el-form-item>
        <el-form-item label="类型" prop="type" :rules="[{ required: true }]">
          <el-select v-model="form.type" style="width: 100%">
            <el-option v-for="opt in notificationTypeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="配置" prop="config" :rules="[{ required: true, message: '请输入配置' }]">
          <el-input
            v-model="form.config"
            type="textarea"
            :rows="5"
            placeholder="JSON配置"
            :autosize="{ minRows: 3, maxRows: 8 }"
          />
          <div style="font-size: 11px; color: #999; margin-top: 4px">{{ configHint }}</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSubmit">
          {{ submitText }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>
