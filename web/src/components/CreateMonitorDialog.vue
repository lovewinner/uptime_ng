<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useMonitorStore } from '@/stores/monitor'
import { ElMessage } from 'element-plus'
import type { Monitor } from '@/stores/monitor'
import type { MonitorPayload } from '@/api/types'
import { apiErrorMessage } from '@/api/errors'
import {
  DEFAULT_STATUS_CODES,
  addStatusCodeTag,
  authMethodOptions,
  bodyEncodingOptions,
  defaultMonitorPayload,
  dnsTypeOptions,
  httpMethodOptions,
  monitorDialogTitle,
  monitorPayloadFromMonitor,
  monitorSubmitPayload,
  monitorSubmitText,
  monitorTypeOptions,
  removeStatusCodeTag,
  shouldFillPingHostname,
  statusCodesFromMonitor,
} from './monitorForm'

const props = defineProps<{
  modelValue: boolean
  monitor?: Monitor | null
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
const statusTags = ref<string[]>([...DEFAULT_STATUS_CODES])

const form = ref<MonitorPayload>(defaultMonitorPayload())

const isEdit = computed(() => !!props.monitor?.id)

const title = computed(() => monitorDialogTitle(isEdit.value))
const submitText = computed(() => monitorSubmitText(isEdit.value))

const groupOptions = computed(() => store.groupOptions(props.monitor?.id))

function addStatusCode(value: string) {
  statusTags.value = addStatusCodeTag(statusTags.value, value)
  form.value.accepted_status_codes = [...statusTags.value]
}

function removeStatusCode(tag: string) {
  statusTags.value = removeStatusCodeTag(statusTags.value, tag)
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
      form.value = monitorPayloadFromMonitor(val)
      statusTags.value = statusCodesFromMonitor(val)
    } else {
      form.value = defaultMonitorPayload()
      statusTags.value = [...DEFAULT_STATUS_CODES]
    }
  },
  { immediate: true },
)

function handleNameBlur() {
  if (shouldFillPingHostname(form.value.type, form.value.hostname)) {
    form.value.hostname = form.value.name
  }
}

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    const payload = monitorSubmitPayload(form.value)
    if (isEdit.value) {
      await store.updateMonitor(props.monitor!.id, payload)
      ElMessage.success('保存成功')
    } else {
      await store.createMonitor(payload)
      ElMessage.success('创建成功')
    }
    emit('saved')
  } catch (e: unknown) {
    ElMessage.error(apiErrorMessage(e, '保存失败'))
  } finally {
    saving.value = false
  }
}

function handleClose() {
  dialogVisible.value = false
}

function handleStatusCodeInput(event: Event) {
  const target = event.target
  if (!(target instanceof HTMLInputElement)) return
  handleInputConfirm(target.value)
  target.value = ''
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
        <el-input v-model="form.name" placeholder="监控项名称" @blur="handleNameBlur" />
      </el-form-item>

      <el-form-item label="描述" prop="description">
        <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选描述" />
      </el-form-item>

      <el-form-item label="类型" prop="type" :rules="[{ required: true }]">
        <el-select v-model="form.type" placeholder="选择监控类型">
          <el-option v-for="opt in monitorTypeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
        </el-select>
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

      <template v-if="form.type === 'group'">
        <el-form-item label="检查间隔(s)">
          <el-input-number v-model="form.interval" :min="3" :max="86400" />
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
      </template>

      <template v-if="form.type === 'ping' || form.type === 'dns'">
        <el-form-item label="主机名" prop="hostname" :rules="[{ required: true, message: '请输入主机名' }]">
          <el-input v-model="form.hostname" placeholder="example.com" />
        </el-form-item>
      </template>

      <template v-if="form.type !== 'group'">
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

        <el-form-item label="重试间隔(s)">
          <el-input-number v-model="form.retry_interval" :min="0" :max="86400" />
        </el-form-item>

        <el-form-item label="重复告警(s)">
          <el-input-number v-model="form.resend_interval" :min="0" :max="604800" />
        </el-form-item>

        <el-form-item v-if="form.type === 'http'" label="关键词">
          <el-input v-model="form.keyword" placeholder="响应体中检测的关键词" />
        </el-form-item>

        <el-form-item v-if="form.type === 'http'" label="反向关键词">
          <el-switch v-model="form.invert_keyword" />
        </el-form-item>

        <el-form-item v-if="form.type === 'http'" label="忽略TLS">
          <el-switch v-model="form.ignore_tls" />
        </el-form-item>

        <el-form-item v-if="form.type === 'http'" label="最大重定向">
          <el-input-number v-model="form.max_redirects" :min="0" :max="30" />
        </el-form-item>

        <el-form-item v-if="form.type === 'http'" label="仅状态码重试">
          <el-switch v-model="form.retry_only_on_status_code" />
        </el-form-item>

        <el-form-item v-if="form.type === 'http'" label="Cache Bust">
          <el-switch v-model="form.cache_bust" />
        </el-form-item>

        <el-form-item label="翻转状态">
          <el-switch v-model="form.upside_down" />
          <span style="margin-left: 8px; font-size: 12px; color: #999">DOWN视为UP</span>
        </el-form-item>

        <el-form-item v-if="form.type === 'http'" label="接受状态码">
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
              @keyup.enter="handleStatusCodeInput"
              @blur="handleStatusCodeInput"
            />
          </div>
          <div style="font-size: 11px; color: #999; margin-top: 4px">默认 200-299，回车添加</div>
        </el-form-item>

        <template v-if="form.type === 'http'">
          <el-divider content-position="left">HTTP 高级配置</el-divider>

          <el-form-item label="认证方式">
            <el-select v-model="form.auth_method" style="width: 100%">
              <el-option v-for="opt in authMethodOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
          </el-form-item>

          <template v-if="form.auth_method === 'basic' || form.auth_method === 'ntlm'">
            <el-form-item label="用户名">
              <el-input v-model="form.basic_auth_user" autocomplete="off" />
            </el-form-item>
            <el-form-item label="密码">
              <el-input v-model="form.basic_auth_pass" type="password" show-password autocomplete="new-password" />
            </el-form-item>
          </template>

          <template v-if="form.auth_method === 'ntlm'">
            <el-form-item label="域">
              <el-input v-model="form.auth_domain" />
            </el-form-item>
            <el-form-item label="工作站">
              <el-input v-model="form.auth_workstation" />
            </el-form-item>
          </template>

          <el-form-item v-if="form.auth_method === 'bearer'" label="Bearer Token">
            <el-input v-model="form.bearer_token" type="textarea" :rows="2" autocomplete="off" />
          </el-form-item>

          <template v-if="form.auth_method === 'oauth2-cc'">
            <el-form-item label="Token URL">
              <el-input v-model="form.oauth_token_url" placeholder="https://issuer.example.com/oauth/token" />
            </el-form-item>
            <el-form-item label="Client ID">
              <el-input v-model="form.oauth_client_id" autocomplete="off" />
            </el-form-item>
            <el-form-item label="Client Secret">
              <el-input v-model="form.oauth_client_secret" type="password" show-password autocomplete="new-password" />
            </el-form-item>
            <el-form-item label="Scopes">
              <el-input v-model="form.oauth_scopes" placeholder="space separated scopes" />
            </el-form-item>
            <el-form-item label="Audience">
              <el-input v-model="form.oauth_audience" />
            </el-form-item>
            <el-form-item label="认证提交">
              <el-select v-model="form.oauth_auth_method" style="width: 100%">
                <el-option label="Body" value="body" />
                <el-option label="Basic" value="basic" />
              </el-select>
            </el-form-item>
          </template>

          <template v-if="form.auth_method === 'mtls'">
            <el-form-item label="客户端证书">
              <el-input v-model="form.tls_cert" type="textarea" :rows="4" placeholder="PEM certificate" />
            </el-form-item>
            <el-form-item label="客户端私钥">
              <el-input v-model="form.tls_key" type="textarea" :rows="4" placeholder="PEM private key" />
            </el-form-item>
            <el-form-item label="自定义CA">
              <el-input v-model="form.tls_ca" type="textarea" :rows="3" placeholder="可选 PEM CA" />
            </el-form-item>
          </template>

          <el-form-item label="请求头">
            <el-input v-model="form.headers" type="textarea" :rows="3" placeholder="Header-Name: value，每行一个" />
          </el-form-item>

          <el-form-item label="Body编码">
            <el-select v-model="form.http_body_encoding" style="width: 100%">
              <el-option v-for="opt in bodyEncodingOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
          </el-form-item>

          <el-form-item label="请求Body">
            <el-input v-model="form.body" type="textarea" :rows="4" placeholder="POST/PUT/PATCH 请求体" />
          </el-form-item>

          <el-form-item label="保存成功响应">
            <el-switch v-model="form.save_response" />
          </el-form-item>

          <el-form-item label="保存错误响应">
            <el-switch v-model="form.save_error_response" />
          </el-form-item>

          <el-form-item label="响应长度">
            <el-input-number v-model="form.response_max_length" :min="256" :max="65535" />
          </el-form-item>
        </template>

        <template v-if="form.type === 'dns'">
          <el-divider content-position="left">DNS 配置</el-divider>
          <el-form-item label="记录类型">
            <el-select v-model="form.dns_resolve_type" style="width: 100%">
              <el-option v-for="opt in dnsTypeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="DNS服务器">
            <el-input v-model="form.dns_resolve_server" placeholder="可选，如 8.8.8.8:53" />
          </el-form-item>
        </template>

        <template v-if="form.type === 'ping'">
          <el-divider content-position="left">Ping 配置</el-divider>
          <el-form-item label="Ping次数">
            <el-input-number v-model="form.ping_count" :min="1" :max="20" />
          </el-form-item>
          <el-form-item label="单次超时(ms)">
            <el-input-number v-model="form.ping_per_request_timeout" :min="100" :max="30000" />
          </el-form-item>
        </template>

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
      </template>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="saving" @click="handleSubmit">
        {{ submitText }}
      </el-button>
    </template>
  </el-dialog>
</template>
