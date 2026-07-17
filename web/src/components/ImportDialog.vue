<script setup lang="ts">
import { computed, ref } from 'vue'
import api from '@/api/http'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'imported'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

const fileInput = ref<HTMLInputElement | null>(null)
const strategy = ref<'skip' | 'overwrite' | 'copy'>('copy')
const previewData = ref<any>(null)
const importResult = ref<any>(null)
const loading = ref(false)
const error = ref('')
const step = ref<'upload' | 'preview' | 'result'>('upload')

function handleFileChange() {
  error.value = ''
  const file = fileInput.value?.files?.[0]
  if (!file) return

  const reader = new FileReader()
  reader.onload = async (e) => {
    try {
      const data = JSON.parse(e.target?.result as string)
      step.value = 'preview'
      loading.value = true
      const res = await api.post('/monitors/import/preview', { data, strategy: 'skip' })
      previewData.value = res.data
    } catch (err: any) {
      error.value = typeof err === 'string' ? err : (err.message || '无效的JSON文件')
    } finally {
      loading.value = false
    }
  }
  reader.readAsText(file)
}

async function executeImport() {
  loading.value = true
  try {
    const file = fileInput.value?.files?.[0]
    if (!file) return
    const text = await file.text()
    const data = JSON.parse(text)
    const res = await api.post('/monitors/import', { data, strategy: strategy.value })
    importResult.value = res.data
    step.value = 'result'
    emit('imported')
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || '导入失败'
  } finally {
    loading.value = false
  }
}

function close() {
  step.value = 'upload'
  previewData.value = null
  importResult.value = null
  error.value = ''
  visible.value = false
}
</script>

<template>
  <el-dialog v-model="visible" title="导入监控配置" width="min(600px, 95%)" @close="close">
    <div v-if="step === 'upload'">
      <p>选择 uptime_ng 导出的 JSON 文件：</p>
      <input ref="fileInput" type="file" accept=".json" @change="handleFileChange" style="margin-bottom:15px" />
      <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    </div>

    <div v-else-if="step === 'preview'" v-loading="loading">
      <h4>导入预览</h4>
      <div v-if="previewData">
        <p>将导入 <b>{{ previewData.new_count }}</b> 个新监控项</p>
        <p v-if="previewData.conflict_count > 0">发现 <b>{{ previewData.conflict_count }}</b> 个同名冲突</p>
        <div v-if="previewData.conflict_count > 0" style="margin: 15px 0">
          <p>冲突处理策略：</p>
          <el-radio-group v-model="strategy">
            <el-radio value="skip">跳过同名</el-radio>
            <el-radio value="overwrite">覆盖已有</el-radio>
            <el-radio value="copy">复制（添加后缀）</el-radio>
          </el-radio-group>
        </div>

        <div v-if="previewData.new_tags && previewData.new_tags.length > 0" style="margin-top:10px">
          <p>会创建以下新标签：</p>
          <el-tag v-for="t in previewData.new_tags" :key="t.name" :color="t.color" style="margin-right:5px">{{ t.name }}</el-tag>
        </div>

        <p style="color:#999;font-size:12px;margin-top:10px">{{ previewData.summary }}</p>
      </div>
    </div>

    <div v-else-if="step === 'result'" v-loading="loading">
      <h4>导入完成</h4>
      <div v-if="importResult">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="导入">{{ importResult.imported }}</el-descriptions-item>
          <el-descriptions-item label="新建">{{ importResult.created }}</el-descriptions-item>
          <el-descriptions-item label="更新">{{ importResult.updated }}</el-descriptions-item>
          <el-descriptions-item label="跳过">{{ importResult.skipped }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="importResult.errors && importResult.errors.length > 0" style="margin-top:10px">
          <el-alert v-for="err in importResult.errors" :key="err" :title="err" type="warning" show-icon :closable="false" style="margin-bottom:5px" />
        </div>
      </div>
    </div>

    <template #footer>
      <el-button v-if="step === 'preview'" @click="step = 'upload'">重新选择</el-button>
      <el-button @click="close">关闭</el-button>
      <el-button v-if="step === 'preview'" type="primary" @click="executeImport">确认导入</el-button>
    </template>
  </el-dialog>
</template>