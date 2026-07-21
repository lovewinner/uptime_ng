<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMonitorStore } from '@/stores/monitor'
import { ElMessage } from 'element-plus'
import { defaultMonitorPayload } from '@/components/monitorForm'
import { apiErrorMessage } from '@/api/errors'

const router = useRouter()
const store = useMonitorStore()

const form = ref(defaultMonitorPayload())
const saving = ref(false)
const result = ref<{ total: number; created: number; errors?: string[] } | null>(null)

const groupOptions = ref(store.groupOptions())

onMounted(() => {
  form.value.type = 'ping'
})

async function handleSubmit() {
  if (!form.value.name) {
    ElMessage.warning('请输入名称')
    return
  }
  if (!form.value.ip_range) {
    ElMessage.warning('请输入IP范围')
    return
  }

  saving.value = true
  result.value = null
  try {
    const res = await store.createPingRange(form.value)
    result.value = res
    ElMessage.success(`成功创建 ${res.created} / ${res.total} 个监控`)
  } catch (e: unknown) {
    ElMessage.error(apiErrorMessage(e, '批量录入失败'))
  } finally {
    saving.value = false
  }
}

function handleReset() {
  form.value = defaultMonitorPayload()
  form.value.type = 'ping'
  result.value = null
}

function goBack() {
  router.push('/monitors')
}
</script>

<template>
  <div>
    <el-page-header @back="goBack" style="margin-bottom: 20px">
      <template #content>
        <span style="font-size: 18px">批量录入</span>
      </template>
    </el-page-header>

    <el-card shadow="never">
      <el-form label-width="120px" label-position="right">
        <el-form-item label="名称" :rules="[{ required: true, message: '请输入名称' }]">
          <el-input v-model="form.name" placeholder="监控项名称前缀" />
        </el-form-item>

        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选描述" />
        </el-form-item>

        <el-form-item label="IP范围" :rules="[{ required: true, message: '请输入IP范围' }]">
          <el-input v-model="form.ip_range" placeholder="如 192.168.1.1-192.168.1.254 或 10.0.0.0/28" />
          <div style="font-size: 11px; color: #999; margin-top: 4px">
            支持: CIDR(10.0.0.0/28)、范围(192.168.1.1-192.168.1.254)、列表(ip1,ip2,ip3)
          </div>
        </el-form-item>

        <el-form-item label="父分组">
          <el-select v-model="form.group_id" clearable placeholder="未分组" style="width: 100%">
            <el-option
              v-for="group in groupOptions"
              :key="group.id"
              :label="store.groupLabel(group)"
              :value="group.id"
            />
          </el-select>
        </el-form-item>

        <el-divider content-position="left">Ping 配置</el-divider>

        <el-form-item label="Ping次数">
          <el-input-number v-model="form.ping_count" :min="1" :max="20" />
        </el-form-item>

        <el-form-item label="单次超时(ms)">
          <el-input-number v-model="form.ping_per_request_timeout" :min="100" :max="30000" />
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

      <div v-if="!result" style="margin-top: 20px; display: flex; gap: 12px">
        <el-button type="primary" :loading="saving" @click="handleSubmit">批量创建</el-button>
        <el-button @click="goBack">取消</el-button>
      </div>

      <div v-else style="margin-top: 24px">
        <el-result :title="`创建结果`" :sub-title="`共 ${result.total} 个 IP，成功创建 ${result.created} 个`" :status="result.errors?.length ? 'warning' : 'success'">
          <template #extra>
            <div v-if="result.errors?.length" style="margin-bottom: 16px; text-align: left">
              <p style="font-weight: 600; color: #F56C6C">失败详情:</p>
              <p v-for="(err, i) in result.errors" :key="i" style="font-size: 13px; color: #666; margin: 4px 0">{{ err }}</p>
            </div>
            <el-button type="primary" @click="handleReset">继续录入</el-button>
            <el-button @click="goBack">返回监控列表</el-button>
          </template>
        </el-result>
      </div>
    </el-card>
  </div>
</template>
