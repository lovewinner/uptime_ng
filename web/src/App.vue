<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { computed, ref, onMounted, onUnmounted } from 'vue'

const auth = useAuthStore()
const isLoggedIn = computed(() => auth.isLoggedIn())

const isMobile = ref(false)
const drawerVisible = ref(false)

function checkScreen() {
  isMobile.value = window.innerWidth < 640
}

function handleLogout() {
  auth.logout()
  drawerVisible.value = false
}

function navTo(path: string) {
  drawerVisible.value = false
}

onMounted(() => {
  checkScreen()
  window.addEventListener('resize', checkScreen)
})
onUnmounted(() => {
  window.removeEventListener('resize', checkScreen)
})
</script>

<template>
  <el-container style="min-height: 100vh">
    <el-header v-if="isLoggedIn" class="app-header">
      <div class="header-left">
        <el-button v-if="isMobile" class="menu-btn" @click="drawerVisible = true" link>
          <span style="font-size: 22px">☰</span>
        </el-button>
        <h1 class="app-title" @click="$router.push('/')">uptime_ng</h1>
      </div>
      <nav v-if="!isMobile" class="nav-bar">
        <el-button text style="color: #ccc" @click="$router.push('/')">仪表盘</el-button>
        <el-button text style="color: #ccc" @click="$router.push('/monitors')">监控项</el-button>
        <el-button text style="color: #ccc" @click="$router.push('/notifications')">通知</el-button>
        <el-button text style="color: #ccc" @click="$router.push('/sla')">SLA报表</el-button>
        <el-button text style="color: #ccc" @click="$router.push('/maintenance')">维护窗口</el-button>
        <el-button v-if="auth.isAdmin()" text style="color: #ccc" @click="$router.push('/users')">用户管理</el-button>
      </nav>
      <div class="header-right">
        <span class="username">{{ auth.username }}</span>
        <el-button type="primary" size="small" @click="handleLogout">退出</el-button>
      </div>
    </el-header>

    <el-drawer v-model="drawerVisible" direction="ltr" size="260px" title="uptime_ng" :with-header="true">
      <el-menu mode="vertical" @select="navTo">
        <el-menu-item index="/" @click="$router.push('/')">仪表盘</el-menu-item>
        <el-menu-item index="/monitors" @click="$router.push('/monitors')">监控项</el-menu-item>
        <el-menu-item index="/notifications" @click="$router.push('/notifications')">通知</el-menu-item>
        <el-menu-item index="/sla" @click="$router.push('/sla')">SLA报表</el-menu-item>
        <el-menu-item index="/maintenance" @click="$router.push('/maintenance')">维护窗口</el-menu-item>
        <el-menu-item v-if="auth.isAdmin()" index="/users" @click="$router.push('/users')">用户管理</el-menu-item>
      </el-menu>
    </el-drawer>

    <el-main>
      <RouterView />
    </el-main>
  </el-container>
</template>

<style scoped>
.app-header {
  background: #1a1a2e;
  padding: 0 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: 0 2px 8px rgba(0,0,0,0.3);
  gap: 8px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.app-title {
  color: #fff;
  font-size: 18px;
  margin: 0;
  cursor: pointer;
  white-space: nowrap;
}

.menu-btn {
  color: #ccc;
  padding: 4px;
}

.nav-bar {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  flex: 1;
  justify-content: center;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.username {
  color: #ccc;
  font-size: 13px;
  white-space: nowrap;
}

body {
  margin: 0;
  font-family: 'Helvetica Neue', Arial, sans-serif;
}
</style>
