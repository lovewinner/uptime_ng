<script setup lang="ts">
import { onMounted, ref } from 'vue'
import api from '@/api/http'
import { apiErrorMessage, isDialogCancel } from '@/api/errors'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  activeTagType,
  activeText,
  activeToggleButtonType,
  activeToggleText,
  nextUserRole,
  roleTagType,
  roleToggleText,
} from './userManagement'

interface User {
  id: number
  username: string
  role: string
  active: boolean
}

const users = ref<User[]>([])
const loading = ref(false)

async function fetchUsers() {
  loading.value = true
  try {
    const res = await api.get('/auth/users')
    users.value = res.data
  } catch {
    // ignore
  } finally {
    loading.value = false
  }
}

async function toggleActive(user: User) {
  try {
    await api.patch(`/auth/users/${user.id}`, { active: !user.active })
    await fetchUsers()
  } catch (e: unknown) {
    ElMessage.error(apiErrorMessage(e, '操作失败'))
  }
}

async function toggleRole(user: User) {
  try {
    await api.patch(`/auth/users/${user.id}`, { role: nextUserRole(user.role) })
    await fetchUsers()
  } catch (e: unknown) {
    ElMessage.error(apiErrorMessage(e, '操作失败'))
  }
}

async function resetPassword(user: User) {
  try {
    const result = await ElMessageBox.prompt(`请输入 ${user.username} 的新密码`, '重置密码', {
      inputType: 'password',
      inputPattern: /^.{6,}$/,
      inputErrorMessage: '密码至少 6 个字符',
      confirmButtonText: '重置',
      cancelButtonText: '取消',
    })
    await api.patch(`/auth/users/${user.id}`, { password: result.value })
    ElMessage.success('密码已重置')
  } catch (e: unknown) {
    if (isDialogCancel(e)) return
    ElMessage.error(apiErrorMessage(e, '重置失败'))
  }
}

onMounted(fetchUsers)
</script>

<template>
  <div>
    <h2>用户管理</h2>

    <el-table :data="users" v-loading="loading" border stripe style="width:100%">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="username" label="用户名" />
      <el-table-column label="角色" width="120">
        <template #default="{ row }">
          <el-tag :type="roleTagType(row.role)" size="small">{{ row.role }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="activeTagType(row.active)" size="small">{{ activeText(row.active) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="250">
        <template #default="{ row }">
          <el-button size="small" @click="toggleRole(row)">{{ roleToggleText(row.role) }}</el-button>
          <el-button size="small" :type="activeToggleButtonType(row.active)" @click="toggleActive(row)">
            {{ activeToggleText(row.active) }}
          </el-button>
          <el-button size="small" @click="resetPassword(row)">重置密码</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>
