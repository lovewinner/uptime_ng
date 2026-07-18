import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { guest: true },
    },
    {
      path: '/',
      name: 'dashboard',
      component: () => import('@/views/DashboardView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/monitors',
      name: 'monitors',
      component: () => import('@/views/MonitorListView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/monitors/:id',
      name: 'monitor-detail',
      component: () => import('@/views/MonitorDetailView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/notifications',
      name: 'notifications',
      component: () => import('@/views/NotificationView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/sla',
      name: 'sla',
      component: () => import('@/views/SLAView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/maintenance',
      name: 'maintenance',
      component: () => import('@/views/MaintenanceView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/users',
      name: 'users',
      component: () => import('@/views/UserManagementView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
  ],
})

router.beforeEach((to, _from, next) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isLoggedIn()) {
    next('/login')
  } else if (to.meta.guest && auth.isLoggedIn()) {
    next('/')
  } else if (to.meta.requiresAdmin && !auth.isAdmin()) {
    next('/')
  } else {
    next()
  }
})

export default router
