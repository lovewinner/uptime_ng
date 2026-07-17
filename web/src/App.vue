<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { computed } from 'vue'

const auth = useAuthStore()
const isLoggedIn = computed(() => auth.isLoggedIn())

function handleLogout() {
  auth.logout()
}
</script>

<template>
  <el-container style="min-height: 100vh">
    <el-header v-if="isLoggedIn" style="background: #1a1a2e; padding: 0 20px; display: flex; align-items: center; justify-content: space-between">
      <div style="display: flex; align-items: center; gap: 30px">
        <h1 style="color: #fff; font-size: 18px; margin: 0; cursor: pointer" @click="$router.push('/')">
          uptime_ng
        </h1>
        <nav style="display: flex; gap: 10px">
          <el-button type="text" style="color: #ccc" @click="$router.push('/')">仪表盘</el-button>
          <el-button type="text" style="color: #ccc" @click="$router.push('/monitors')">监控项</el-button>
          <el-button type="text" style="color: #ccc" @click="$router.push('/notifications')">通知</el-button>
          <el-button type="text" style="color: #ccc" @click="$router.push('/sla')">SLA报表</el-button>
          <el-button v-if="auth.isAdmin()" type="text" style="color: #ccc" @click="$router.push('/users')">用户管理</el-button>
        </nav>
      </div>
      <div style="display: flex; align-items: center; gap: 15px">
        <span style="color: #ccc">{{ auth.username }}</span>
        <el-button type="primary" size="small" @click="handleLogout">退出</el-button>
      </div>
    </el-header>

    <el-main>
      <RouterView />
    </el-main>
  </el-container>
</template>

<style>
body {
  margin: 0;
  font-family: 'Helvetica Neue', Arial, sans-serif;
}
.el-header {
  box-shadow: 0 2px 8px rgba(0,0,0,0.3);
}
</style>