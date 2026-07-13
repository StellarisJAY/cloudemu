import { createRouter, createWebHistory } from 'vue-router'
import { getAccessToken } from '@/utils/token'

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
      path: '/register',
      name: 'register',
      component: () => import('@/views/RegisterView.vue'),
      meta: { guest: true },
    },
    {
      path: '/profile',
      name: 'profile',
      component: () => import('@/views/ProfileView.vue'),
      meta: { auth: true },
    },
    {
      path: '/forgot-password',
      name: 'forgot-password',
      component: () => import('@/views/ForgotPasswordView.vue'),
      meta: { guest: true },
    },
    {
      path: '/reset-password',
      name: 'reset-password',
      component: () => import('@/views/ResetPasswordView.vue'),
      meta: { guest: true },
    },
    {
      path: '/',
      name: 'lobby',
      component: () => import('@/views/LobbyView.vue'),
      meta: { auth: true },
    },
    {
      path: '/play/:roomId',
      name: 'play',
      component: () => import('@/views/PlayView.vue'),
      meta: { auth: true },
    },
  ],
})

/** 全局导航守卫：auth 路由需登录，guest 路由已登录则跳大厅 */
router.beforeEach((to, _from, next) => {
  const token = getAccessToken()

  if (to.meta.auth && !token) {
    next({ name: 'login', query: { redirect: to.fullPath } })
  } else if (to.meta.guest && token) {
    next({ name: 'lobby' })
  } else {
    next()
  }
})

export default router
