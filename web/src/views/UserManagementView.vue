<script setup lang="ts">
import { onMounted, ref } from 'vue'
import api from '@/api/http'

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
  await api.patch(`/auth/users/${user.id}`, { active: !user.active })
  await fetchUsers()
}

async function toggleRole(user: User) {
  const newRole = user.role === 'admin' ? 'user' : 'admin'
  await api.patch(`/auth/users/${user.id}`, { role: newRole })
  await fetchUsers()
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
          <el-tag :type="row.role === 'admin' ? 'danger' : 'info'" size="small">{{ row.role }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.active ? 'success' : 'danger'" size="small">{{ row.active ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200">
        <template #default="{ row }">
          <el-button size="small" @click="toggleRole(row)">{{ row.role === 'admin' ? '降级' : '提升为管理员' }}</el-button>
          <el-button size="small" :type="row.active ? 'danger' : 'success'" @click="toggleActive(row)">
            {{ row.active ? '禁用' : '启用' }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>