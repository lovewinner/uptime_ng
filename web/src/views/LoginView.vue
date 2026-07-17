<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

const form = reactive({ username: '', password: '' })
const registering = reactive({ username: '', password: '' })

const loginLoading = ref(false)
const registerLoading = ref(false)
const loginError = ref('')
const registerError = ref('')
const showRegister = ref(false)

async function handleLogin() {
  loginLoading.value = true
  loginError.value = ''
  const ok = await auth.login(form.username, form.password)
  loginLoading.value = false
  if (ok) {
    router.push('/')
  } else {
    loginError.value = '用户名或密码错误'
  }
}

async function handleRegister() {
  registerLoading.value = true
  registerError.value = ''
  const result = await auth.register(registering.username, registering.password)
  registerLoading.value = false
  if (result.ok) {
    router.push('/')
  } else {
    registerError.value = result.error || '注册失败'
  }
}
</script>

<template>
  <div style="display:flex;justify-content:center;align-items:center;min-height:80vh">
    <el-card style="width:400px">
      <template #header>
        <h2 style="text-align:center;margin:0">uptime_ng</h2>
      </template>

      <template v-if="!showRegister">
        <el-form @submit.prevent="handleLogin">
          <el-form-item label="用户名">
            <el-input v-model="form.username" placeholder="请输入用户名" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="form.password" type="password" show-password placeholder="请输入密码" />
          </el-form-item>
          <el-alert v-if="loginError" :title="loginError" type="error" show-icon :closable="false" style="margin-bottom:15px" />
          <el-button type="primary" native-type="submit" :loading="loginLoading" style="width:100%">登录</el-button>
        </el-form>
        <div style="text-align:center;margin-top:15px">
          <el-button type="text" @click="showRegister = true">没有账号？注册</el-button>
        </div>
      </template>

      <template v-else>
        <el-form @submit.prevent="handleRegister">
          <el-form-item label="用户名">
            <el-input v-model="registering.username" placeholder="至少3个字符" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="registering.password" type="password" show-password placeholder="至少6个字符" />
          </el-form-item>
          <el-alert v-if="registerError" :title="registerError" type="error" show-icon :closable="false" style="margin-bottom:15px" />
          <el-button type="primary" native-type="submit" :loading="registerLoading" style="width:100%">注册</el-button>
        </el-form>
        <div style="text-align:center;margin-top:15px">
          <el-button type="text" @click="showRegister = false">已有账号？返回登录</el-button>
        </div>
      </template>
    </el-card>
  </div>
</template>