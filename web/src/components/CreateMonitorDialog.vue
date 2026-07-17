<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useMonitorStore } from '@/stores/monitor'
import { ElMessage } from 'element-plus'

const props = defineProps<{
  modelValue: boolean
  monitor?: any | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: []
}>()

const store = useMonitorStore()

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

const formRef = ref()
const saving = ref(false)
const statusTags = ref<string[]>(['200-299'])

const form = ref({
  name: '',
  description: '',
  type: 'http',
  url: '',
  hostname: '',
  port: 80,
  method: 'GET',
  interval: 60,
  timeout: 30,
  max_retries: 0,
  keyword: '',
  ignore_tls: false,
  upside_down: false,
  accepted_status_codes: ['200-299'],
  notification_ids: [] as number[],
})

const isEdit = computed(() => !!props.monitor?.id)

const title = computed(() => isEdit.value ? '编辑监控' : '新增监控')

const typeOptions = [
  { label: 'HTTP', value: 'http' },
  { label: 'TCP', value: 'tcp' },
  { label: 'PING', value: 'ping' },
  { label: 'DNS', value: 'dns' },
]

const methodOptions = [
  { label: 'GET', value: 'GET' },
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'DELETE', value: 'DELETE' },
  { label: 'PATCH', value: 'PATCH' },
  { label: 'HEAD', value: 'HEAD' },
  { label: 'OPTIONS', value: 'OPTIONS' },
]

function addStatusCode(value: string) {
  const trimmed = value.trim()
  if (trimmed && !statusTags.value.includes(trimmed)) {
    statusTags.value.push(trimmed)
    form.value.accepted_status_codes = [...statusTags.value]
  }
}

function removeStatusCode(tag: string) {
  statusTags.value = statusTags.value.filter(t => t !== tag)
  form.value.accepted_status_codes = [...statusTags.value]
}

function handleInputConfirm(value: string) {
  if (value) {
    addStatusCode(value)
  }
}

watch(
  () => props.monitor,
  (val) => {
    if (val) {
      form.value = {
        name: val.name || '',
        description: val.description || '',
        type: val.type || 'http',
        url: val.url || '',
        hostname: val.hostname || '',
        port: val.port || 80,
        method: val.method || 'GET',
        interval: val.interval || 60,
        timeout: val.timeout || 30,
        max_retries: val.max_retries || 0,
        keyword: val.keyword || '',
        ignore_tls: val.ignore_tls || false,
        upside_down: val.upside_down || false,
        accepted_status_codes: [...(val.accepted_status_codes || ['200-299'])],
        notification_ids: [...(val.notification_ids || [])],
      }
      statusTags.value = [...(val.accepted_status_codes || ['200-299'])]
    } else {
      form.value = {
        name: '',
        description: '',
        type: 'http',
        url: '',
        hostname: '',
        port: 80,
        method: 'GET',
        interval: 60,
        timeout: 30,
        max_retries: 0,
        keyword: '',
        ignore_tls: false,
        upside_down: false,
        accepted_status_codes: ['200-299'],
        notification_ids: [],
      }
      statusTags.value = ['200-299']
    }
  },
  { immediate: true },
)

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    const payload = {
      ...form.value,
      port: form.value.port || 0,
    }
    if (isEdit.value) {
      await store.updateMonitor(props.monitor.id, payload)
    } else {
      await store.createMonitor(payload)
    }
    emit('saved')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

function handleClose() {
  dialogVisible.value = false
}
</script>

<template>
  <el-dialog
    v-model="dialogVisible"
    :title="title"
    width="min(680px, 95%)"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <el-form
      ref="formRef"
      :model="form"
      label-width="120px"
      label-position="right"
    >
      <el-form-item label="名称" prop="name" :rules="[{ required: true, message: '请输入名称' }]">
        <el-input v-model="form.name" placeholder="监控项名称" />
      </el-form-item>

      <el-form-item label="描述" prop="description">
        <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选描述" />
      </el-form-item>

      <el-form-item label="类型" prop="type" :rules="[{ required: true }]">
        <el-select v-model="form.type" placeholder="选择监控类型">
          <el-option v-for="opt in typeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
        </el-select>
      </el-form-item>

      <template v-if="form.type === 'http'">
        <el-form-item label="URL" prop="url" :rules="[{ required: true, message: '请输入URL' }]">
          <el-input v-model="form.url" placeholder="https://example.com" />
        </el-form-item>
        <el-form-item label="请求方法" prop="method">
          <el-select v-model="form.method">
            <el-option v-for="opt in methodOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
      </template>

      <template v-if="form.type === 'tcp'">
        <el-form-item label="主机名" prop="hostname" :rules="[{ required: true, message: '请输入主机名' }]">
          <el-input v-model="form.hostname" placeholder="127.0.0.1" />
        </el-form-item>
        <el-form-item label="端口" prop="port" :rules="[{ required: true, message: '请输入端口' }]">
          <el-input-number v-model="form.port" :min="1" :max="65535" />
        </el-form-item>
      </template>

      <template v-if="form.type === 'ping' || form.type === 'dns'">
        <el-form-item label="主机名" prop="hostname" :rules="[{ required: true, message: '请输入主机名' }]">
          <el-input v-model="form.hostname" placeholder="example.com" />
        </el-form-item>
      </template>

      <el-divider content-position="left">检查配置</el-divider>

      <el-form-item label="检查间隔(s)">
        <el-input-number v-model="form.interval" :min="3" :max="86400" />
      </el-form-item>

      <el-form-item label="超时(s)">
        <el-input-number v-model="form.timeout" :min="1" :max="300" :precision="1" />
      </el-form-item>

      <el-form-item label="最大重试">
        <el-input-number v-model="form.max_retries" :min="0" :max="10" />
      </el-form-item>

      <el-form-item label="关键词">
        <el-input v-model="form.keyword" placeholder="响应体中检测的关键词" />
      </el-form-item>

      <el-form-item label="忽略TLS">
        <el-switch v-model="form.ignore_tls" />
      </el-form-item>

      <el-form-item label="翻转状态">
        <el-switch v-model="form.upside_down" />
        <span style="margin-left: 8px; font-size: 12px; color: #999">DOWN视为UP</span>
      </el-form-item>

      <el-form-item label="接受状态码">
        <div style="display: flex; flex-wrap: wrap; gap: 6px; align-items: center">
          <el-tag
            v-for="tag in statusTags"
            :key="tag"
            closable
            size="small"
            @close="removeStatusCode(tag)"
          >
            {{ tag }}
          </el-tag>
          <el-input
            v-if="statusTags.length < 10"
            size="small"
            style="width: 120px"
            placeholder="如 200-299"
            @keyup.enter="handleInputConfirm(($event.target as any).value); ($event.target as any).value = ''"
            @blur="handleInputConfirm(($event.target as any).value); ($event.target as any).value = ''"
          />
        </div>
        <div style="font-size: 11px; color: #999; margin-top: 4px">默认 200-299，回车添加</div>
      </el-form-item>

      <el-form-item label="关联通知">
        <el-select v-model="form.notification_ids" multiple placeholder="选择通知配置" style="width: 100%">
          <el-option
            v-for="n in store.notifications"
            :key="n.id"
            :label="`${n.name} (${n.type})`"
            :value="n.id"
          />
        </el-select>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="saving" @click="handleSubmit">
        {{ isEdit ? '保存' : '创建' }}
      </el-button>
    </template>
  </el-dialog>
</template>